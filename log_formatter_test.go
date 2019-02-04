package nana

import (
	"bytes"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestLogFormatter(t *testing.T) {
	assert := assert.New(t)

	config := &AppConfig{
		Name:        "application name",
		Environment: "test",
		Version:     "0.0.0-test",
	}

	var output string
	buf := bytes.NewBufferString(output)

	log := logrus.New()
	log.Out = buf
	underlyingFormatter := &logrus.TextFormatter{
		DisableColors: true,
	}
	log.Formatter = NewLogFormatter(config, underlyingFormatter)

	log.Info("entry with default fields")
	assert.Contains(buf.String(), "level=info msg=\"entry with default fields\" application=\"application name\" environment=test version=0.0.0-test")

	log.WithFields(logrus.Fields{
		"correlationId": "123456789",
	}).Warn("entry with more fields")
	assert.Contains(buf.String(), "level=warning msg=\"entry with more fields\" application=\"application name\" correlationId=123456789 environment=test version=0.0.0-test")
}
