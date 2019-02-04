package nana

import (
	"fmt"
	"strings"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/Sirupsen/logrus"
)

// MetricsClient allows applications to record StatsD metrics.
type MetricsClient interface {
	Gauge(name string, value float64, tags []string, rate float64) error
	Incr(name string, tags []string, rate float64) error
	ServiceCheck(sc *statsd.ServiceCheck) error
	Timing(name string, value time.Duration, tags []string, rate float64) error
	Close() error
}

// NewDataDogMetricsClient initialises and returns the DataDog metrics client.
func NewDataDogMetricsClient(config AppConfigurer) (MetricsClient, error) {
	client, err := statsd.New(config.GetDataDogStatsDURL())
	if err != nil {
		return nil, err
	}
	client.Namespace = fmt.Sprintf("%s.", strings.ToLower(config.GetName()))
	client.Tags = append(client.Tags, config.GetDataDogStatsDURL(), fmt.Sprintf("version:%s", config.GetVersion()), fmt.Sprintf("environment:%s", config.GetEnvironment()))
	return client, nil
}

// NullMetricsClient blackholes any metrics.
type NullMetricsClient struct {
	Log *logrus.Logger
}

// Gauge does nothing.
func (client *NullMetricsClient) Gauge(name string, value float64, tags []string, rate float64) error {
	if client.Log != nil {
		client.Log.WithFields(logrus.Fields{
			"name":  name,
			"tags":  tags,
			"rate":  rate,
			"value": value,
		}).Debug("Gauge metric blackholed (Datadog not configured)")
	}
	return nil
}

// Incr does nothing.
func (client *NullMetricsClient) Incr(name string, tags []string, rate float64) error {
	if client.Log != nil {
		client.Log.WithFields(logrus.Fields{
			"name": name,
			"tags": tags,
			"rate": rate,
		}).Debug("Incr metric blackholed (Datadog not configured)")
	}
	return nil
}

// ServiceCheck does nothing.
func (client *NullMetricsClient) ServiceCheck(sc *statsd.ServiceCheck) error {
	if client.Log != nil {
		client.Log.WithFields(logrus.Fields{
			"sc": sc,
		}).Debug("ServiceCheck metric blackholed (Datadog not configured)")
	}
	return nil
}

// Timing does nothing.
func (client *NullMetricsClient) Timing(name string, value time.Duration, tags []string, rate float64) error {
	if client.Log != nil {
		client.Log.WithFields(logrus.Fields{
			"name":  name,
			"value": value,
			"tags":  tags,
			"rate":  rate,
		}).Debug("Timing metric blackholed (Datadog not configured)")
	}
	return nil
}

// Close does nothing.
func (NullMetricsClient) Close() error {
	return nil
}
