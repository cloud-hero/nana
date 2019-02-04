package nana

import (
	"bytes"
	"testing"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNullMetricsClient(t *testing.T) {
	assert := assert.New(t)

	log := logrus.New()
	log.Level = logrus.DebugLevel
	log.Formatter = &logrus.TextFormatter{
		DisableColors: true,
	}

	var output string
	logBuffer := bytes.NewBufferString(output)
	log.Out = logBuffer

	metrics := &NullMetricsClient{log}
	metrics.Gauge("gauge", float64(123), []string{"tag1", "tag2"}, 1)
	assert.Contains(logBuffer.String(), "name=gauge rate=1 tags=\"[tag1 tag2]\" value=123")

	metrics.Incr("incr", []string{"tag1", "tag2"}, 1)
	assert.Contains(logBuffer.String(), "name=incr rate=1 tags=\"[tag1 tag2]\"")

	metrics.ServiceCheck(&statsd.ServiceCheck{
		Name:   "sc",
		Status: statsd.Critical,
	})
	assert.Contains(logBuffer.String(), "sc 2")

	metrics.Timing("timing", time.Duration(45)*time.Millisecond, []string{"tag1", "tag2"}, 1)
	assert.Contains(logBuffer.String(), "name=timing rate=1 tags=\"[tag1 tag2]\"")
}
