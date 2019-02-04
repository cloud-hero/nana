package nana

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/opentracing/opentracing-go"

	"github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func testHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetUserValue("logFields", map[string]interface{}{
		"correlationId": "abcdef",
	})
	switch string(ctx.Path()) {
	case "/":
		ctx.SetBodyString("this is a successful request")
	case "/bad":
		ctx.Error("bad request occurred", fasthttp.StatusBadRequest)
	default:
		ctx.Error("internal server error occurred", fasthttp.StatusInternalServerError)
	}
}

func testMiddleware() (*Middleware, *bytes.Buffer) {
	var output string
	buf := bytes.NewBufferString(output)

	log := logrus.New()
	tracer := opentracing.GlobalTracer()
	log.Level = logrus.DebugLevel
	log.Formatter = &logrus.TextFormatter{
		DisableColors: true,
	}
	log.Out = buf

	return NewMiddleware(log, tracer), buf
}

func TestMiddleware200OK(t *testing.T) {
	assert := assert.New(t)

	middleware, logBuffer := testMiddleware()
	server := NewTestHTTPServer(t, middleware.Handler(testHandler))
	defer server.Close()

	resp, err := http.Get(fmt.Sprintf("http://%s/", server.Addr()))
	logOutput := logBuffer.String()

	assert.Nil(err)
	assert.Equal(200, resp.StatusCode)

	// Metric tags
	// assert.Contains(logOutput, "tags=\"[method:GET path:/ status:200]\"")

	// Log fields
	assert.Contains(logOutput, "level=info")
	assert.Contains(logOutput, "msg=OK")
	assert.Contains(logOutput, "correlationId=abcdef")
}

func TestMiddleware400BadRequest(t *testing.T) {
	assert := assert.New(t)

	middleware, logBuffer := testMiddleware()
	server := NewTestHTTPServer(t, middleware.Handler(testHandler))
	defer server.Close()

	resp, err := http.Get(fmt.Sprintf("http://%s/bad", server.Addr()))
	logOutput := logBuffer.String()

	assert.Nil(err)
	assert.Equal(400, resp.StatusCode)

	// Metric tags
	// assert.Contains(logOutput, "tags=\"[method:GET path:/bad status:400]\"")

	// Log fields
	assert.Contains(logOutput, "level=warn")
	assert.Contains(logOutput, "msg=\"bad request occurred\"")
	assert.Contains(logOutput, "correlationId=abcdef")
}

func TestMiddleware500InternalServerError(t *testing.T) {
	assert := assert.New(t)

	middleware, logBuffer := testMiddleware()
	server := NewTestHTTPServer(t, middleware.Handler(testHandler))
	defer server.Close()

	resp, err := http.Get(fmt.Sprintf("http://%s/error", server.Addr()))
	logOutput := logBuffer.String()

	assert.Nil(err)
	assert.Equal(500, resp.StatusCode)

	// Metric tags
	// assert.Contains(logOutput, "tags=\"[method:GET path:/error status:500]\"")

	// Log fields
	assert.Contains(logOutput, "level=error")
	assert.Contains(logOutput, "msg=\"internal server error occurred\"")
	assert.Contains(logOutput, "correlationId=abcdef")
}
