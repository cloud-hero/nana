# nana

This library contains a number of useful standard functions used in all golang apps.

## Getting started

```go
import "github.com/cloud-hero/nana"

func main() {
  // Your specific application config struct can inherit fields from AppConfig
	config := &nana.AppConfig{
		DataDogHost:       "localhost",
		DataDogAPMPort:    "8126",
		DataDogStatsDPort: "8125",
		Environment:       "development",
		KafkaBrokers:      []string{"localhost:9092"},
		LogLevel:          "info",
		Name:              "publisher",
		Port:              "8000",
		Version:           "1.0.0",
	}

  // Configures basic app with health check
  app, err := nana.NewApp(config)
  if err != nil {
    panic(err)
  }
  defer app.Close()

  // Starts the web server
  app.StartWebServer()

  // Wait until Ctrl-C
  app.WaitForInterrupt()
}
```

## Functions

* [Logging](#logging)
* [Middleware](#middleware)
* [Healthcheck Handler](#healthcheck-handler)
* [Pause Handler](#pause-handler)
* [Test HTTP Server](#test-http-server)
* [Nats](#kafka)

### Logging

By using the logger provided in `app.Log` your log entries will always contain certain standard fields, such as `application`, `environment` and `version`.

```
{"application":"renderer","environment":"production","level":"info","msg":"Your log message","time":"2016-12-06T14:06:15.118985034+11:00","version":"1.0.0"}
```

### Middleware

The nana middleware ensures all HTTP requests are logged and metrics collected in a standard format.

```go
router.GET("/yourendpoint", app.Middleware(YourHandler)))
```

You can include fields in logs or tags in metrics by adding them to the request context:

```go
func YourHandler(ctx *fasthttp.RequestCtx) {
  ctx.SetUserValue("logFields", map[string]interface{}{
    "origin":        origin,
  })

  ctx.SetUserValue("metricTags", []string{
    "origin:renderer",
  })
}
```

Log output:

```
{"application":"renderer","environment":"production","level":"info","method":"GET","msg":"Successful request","path":"/healthcheck","status":200,"time":"2016-12-06T14:06:15.118985034+11:00","version":"123"}
```

Metrics:

```
```

### Basic Healthcheck Handler
All HTTP applications require a healthcheck endpoint, this provides the most basic one.

```go
router.GET("/healthcheck", app.NewHealthCheckHandler())
```

### Extended healthcheck Handler
For more advanced health checking you could use ``` app.NewExtendedHealthCheckHandler(func() (boo, *json.RawMessage)) ```

```go
func healthVerifier() (bool, *json.RawMessage) {
  raw := json.RawMessage(`[{"name": "Kafka", "isHealthy": true}]`)
  // Feel free to include any custom data related to health probes, stats, latency values etc
  return true, &raw
}
router.GET("/healthcheck", app.NewExtendedHealthCheckHandler(healthVerifier))
```

### Test HTTP Server
This can be used to spin up your fasthttp application and run HTTP requests against it.  Ports are dynamically allocated.

```go
func TestMyApp(t *testing.T) {
  server := nana.NewTestHTTPServer(t, YourHandler)
  defer server.Close()

  resp, err := http.Get(fmt.Sprintf("http://%s/yourendpoint", server.Addr()))
}
```

### Messages

#### Kafka Receiver
Provides a receiver which can continuously receive messages from a Kafka topic, and process them using a custom processor.

```go
type MyProcessor struct {}

//Message Do something on message
func (p *Processor) Message(msg *stan.Msg, rootSpan opentracing.Span) error {
	p.Log.Info(msg)
	span := rootSpan.Tracer().StartSpan(
		"formatString",
		opentracing.ChildOf(rootSpan.Context()),
	)
	defer span.Finish()
	return nil
}

processor := &MyProcessor{}

topic := "render-html"

app.RegisterKafkaReceiver(topic, processor, processor.MessageType)

app.StartReceivers()

app.WaitForInterrupt()
```

#### Kafka Publisher
Provides a publisher that uses [Kafka Sarama](https://github.com/Shopify/sarama) to publish messages to a Kafka topic.  Note that you can specify a key when you publish, which will be used to partition messages. If a blank key is provided, random partitioning will occur.

```go
publisher, err := app.NewKafkaPublisher()
if err != nil {
  panic(err)
}
defer publisher.Close()

topic := "my-topic"
span := app.Tracer.StartSpan("publishSetup")

time.Sleep(10 * time.Millisecond)

span.Finish()

mapc := opentracing.TextMapCarrier(make(map[string]string))

err = app.Tracer.Inject(span.Context(), opentracing.TextMap, mapc)
if err != nil {
  panic(err)
}

topic := "render-html"
msgType := &pb.BulkIncomingMessage{
  Baggage: baggae,
}

msg, _ := proto.Marshal(msgType)

publisher.Publish("", msg, topic)

time.Sleep(100 * time.Millisecond)

app.Close()
```
