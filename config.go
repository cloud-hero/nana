package nana

import (
	"fmt"

	"github.com/Sirupsen/logrus"
)

// AppConfigurer interface allows individual app config structs to inherit Fields
// from AppConfig and still be used by nana.
type AppConfigurer interface {
	GetDataDogHost() string
	GetDataDogAPMURL() string
	GetDataDogStatsDURL() string
	GetEnvironment() string
	GetLogLevel() logrus.Level
	GetName() string
	GetKafkaBrokers() []string
	GetPort() string
	GetVersion() string
}

// AppConfig contains the base configuration fields required for an nana app.
type AppConfig struct {
	DataDogHost       string   `long:"datadog-host" env:"DATADOG_HOST" default:"localhost"`
	DataDogAPMPort    string   `long:"datadog-apm-port" env:"DATADOG_APM_PORT" default:"8126"`
	DataDogStatsDPort string   `long:"datadog-statsd-port" env:"DATADOG_STATSF_PORT" default:"8125"`
	Environment       string   `long:"env" env:"ENVIRONMENT" value-name:"development" default:"development"`
	KafkaBrokers      []string `long:"kafka-brokers" env:"KAFKA_BROKERS" env-delim:"," default:"localhost:9092"`
	LogLevel          string   `long:"log-level" env:"LOG_LEVEL" default:"info"`
	Name              string
	Port              string `long:"port" env:"APP_PORT" default:":8000"`
	Version           string
}

// GetDataDogHost returns the DataDog host.
func (c *AppConfig) GetDataDogHost() string {
	return c.DataDogHost
}

// GetDataDogAPMURL returns the DataDog APM port.
func (c *AppConfig) GetDataDogAPMURL() string {
	return fmt.Sprintf("%s:%s", c.DataDogHost, c.DataDogAPMPort)
}

// GetDataDogStatsDURL returns the DataDog host.
func (c *AppConfig) GetDataDogStatsDURL() string {
	return fmt.Sprintf("%s:%s", c.DataDogHost, c.DataDogStatsDPort)
}

// GetEnvironment returns the current environment (production, development).
func (c *AppConfig) GetEnvironment() string {
	return c.Environment
}

// GetKafkaBrokers returns the Kafka brokers.
func (c *AppConfig) GetKafkaBrokers() []string {
	return c.KafkaBrokers
}

// GetLogLevel parses and returns the log level, defaulting to Info.
func (c *AppConfig) GetLogLevel() logrus.Level {
	level, _ := logrus.ParseLevel(c.LogLevel)
	if level == 0 {
		level = logrus.InfoLevel
	}
	return level
}

// GetName returns the application name.
func (c *AppConfig) GetName() string {
	return c.Name
}

// GetPort returns the port to use for a web server in this app.
func (c *AppConfig) GetPort() string {
	return c.Port
}

// GetVersion returns the application version.
func (c *AppConfig) GetVersion() string {
	return c.Version
}
