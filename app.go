package nana

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Shopify/sarama"
	"github.com/Sirupsen/logrus"
	"github.com/buaazp/fasthttprouter"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/valyala/fasthttp"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/opentracer"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// App provides a default app structure with Logger, Metrics
// and optional Router and Receivers
type App struct {
	closers []io.Closer

	Config    AppConfigurer
	Log       *logrus.Logger
	Receivers []Receiver
	Router    *fasthttprouter.Router
	Tracer    opentracing.Tracer
}

// NewApp configures and returns an App
func NewApp(config AppConfigurer) (*App, error) {
	app := &App{
		closers: []io.Closer{},
		Config:  config,
		Router:  fasthttprouter.New(),
	}
	tracer := opentracer.New(
		tracer.WithServiceName(config.GetName()),
		tracer.WithGlobalTag("env", config.GetEnvironment()),
		tracer.WithGlobalTag("version", config.GetVersion()),
		tracer.WithAgentAddr(config.GetDataDogAPMURL()),
	)
	opentracing.SetGlobalTracer(tracer)
	app.Tracer = tracer
	var err error

	app.Log, err = newLogger(config)
	if err != nil {
		return nil, err
	}

	app.Router.GET("/healthcheck", app.Middleware(app.NewHealthCheckHandler()))

	return app, nil
}

func newMetrics(config AppConfigurer) (MetricsClient, error) {
	if len(config.GetDataDogHost()) == 0 {
		return &NullMetricsClient{}, nil
	}

	return NewDataDogMetricsClient(config)
}

func newLogger(config AppConfigurer) (*logrus.Logger, error) {
	log := logrus.New()
	log.Formatter = NewLogFormatter(config, &logrus.JSONFormatter{TimestampFormat: time.RFC3339Nano})
	log.Level = config.GetLogLevel()
	log.Out = os.Stdout
	return log, nil
}

// StartWebServer starts a web server for the App's router on the configured port.
func (app *App) StartWebServer() error {
	server, err := net.Listen("tcp", fmt.Sprintf(":%s", app.Config.GetPort()))
	if err != nil {
		return err
	}
	app.DeferClose(server)
	go fasthttp.Serve(server, app.Router.Handler)
	app.Log.Infof("Web server started on port %v", app.Config.GetPort())
	return nil
}

// StartReceivers starts any registered receivers receiving messages.
func (app *App) StartReceivers() {
	for _, receiver := range app.Receivers {
		go receiver.Start()
	}
}

// StopReceivers stops any registered receivers from receiving messages.
func (app *App) StopReceivers() {
	for _, receiver := range app.Receivers {
		receiver.Stop()
	}
}

// WaitForInterrupt starts an infinite loop only broken by Ctrl-C
func (app *App) WaitForInterrupt() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	for _ = range signals {
		return
	}
}

// Close closes any resources such as subscribers, publishers, etc.
func (app *App) Close() error {
	for _, closer := range app.closers {
		err := closer.Close()
		if err != nil {
			return err
		}
	}
	defer tracer.Stop()
	return nil
}

// DeferClose adds a closer to the list of resources to be closed when Close() is called.
func (app *App) DeferClose(closer io.Closer) {
	app.closers = append(app.closers, closer)
}

// NewHealthCheckHandler provides a basic health check which returns JSON payload including app name and version
func (app *App) NewHealthCheckHandler() fasthttp.RequestHandler {
	return HealthCheckHandler(app.Config.GetName(), app.Config.GetVersion())
}

// NewExtendedHealthCheckHandler provides an extended health check which returns extra details in json.RawMessage form
func (app *App) NewExtendedHealthCheckHandler(verifier func() (bool, *json.RawMessage)) fasthttp.RequestHandler {
	return ExtendedHealthCheckHandler(app.Config.GetName(), app.Config.GetVersion(), verifier)
}

// Middleware provides a wrapper around your HTTP handler which logs every
// request, and records count and latency metrics for every request.
func (app *App) Middleware(handler func(*fasthttp.RequestCtx)) fasthttp.RequestHandler {
	middleware := NewMiddleware(app.Log, app.Tracer)
	return middleware.Handler(handler)
}

// RegisterKafkaReceiver configures a receiver for this app which can receive and process messages from a Kafka topic
func (app *App) RegisterKafkaReceiver(topic string, processor KafkaMessageProcessor, messageType interface{}) error {
	subscriber, err := NewKafkaSubscriber(app.Log, app.Config.GetKafkaBrokers(), app.Config.GetName(), []string{topic})
	if err != nil {
		return err
	}
	receiver := NewKafkaReceiver(app.Config.GetName(), app.Log, app.Tracer, subscriber, processor, messageType)
	app.DeferClose(receiver)
	app.Receivers = append(app.Receivers, receiver)
	return nil
}

// NewKafkaPublisher gives you a way of publishing messages to a Kafka topic
func (app *App) NewKafkaPublisher() (*KafkaPublisher, error) {
	return app.NewSecureKafkaPublisher(nil, "", "")
}

// NewSecureKafkaPublisher gives you a way of publishing messages to Kafka over TLS with SASL
func (app *App) NewSecureKafkaPublisher(tlsConfig *tls.Config, saslUser string, saslPassword string) (*KafkaPublisher, error) {
	publisher, err := NewKafkaPublisher(app.Log, app.Config.GetKafkaBrokers(), tlsConfig, saslUser, saslPassword)
	if err != nil {
		return nil, err
	}
	app.DeferClose(publisher)
	return publisher, nil
}

// NewCustomKafkaPublisher gives you a way of publishing messages to Kafka over TLS with SASL
func (app *App) NewCustomKafkaPublisher(config *sarama.Config) (*KafkaPublisher, error) {
	publisher, err := NewCustomKafkaPublisher(app.Log, app.Config.GetKafkaBrokers(), config)
	if err != nil {
		return nil, err
	}
	app.DeferClose(publisher)
	return publisher, nil
}
