package nana

import (
	"bytes"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewApp(t *testing.T) {
	assert := assert.New(t)

	config := &AppConfig{
		Name:        "app name",
		Environment: "test",
		Version:     "1.2.3",
	}

	app, err := NewApp(config)

	assert.Nil(err)
	assert.Equal(config, app.Config)
	assert.Equal(logrus.InfoLevel, app.Log.Level)

	app.Log.Level = logrus.DebugLevel

	var output string
	logBuffer := bytes.NewBufferString(output)
	app.Log.Out = logBuffer

	app.Log.Info("Logger should work")
	assert.Contains(logBuffer.String(), "\"application\":\"app name\"")
	assert.Contains(logBuffer.String(), "\"environment\":\"test\"")
	assert.Contains(logBuffer.String(), "\"msg\":\"Logger should work\"")
	assert.Contains(logBuffer.String(), "\"version\":\"1.2.3\"")

}
