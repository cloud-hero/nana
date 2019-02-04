package nana

import (
	"fmt"
	"strings"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

	"github.com/Sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

// Middleware provides a Handler which logs and records metrics for every HTTP request.
type Middleware struct {
	log    *logrus.Logger
	tracer opentracing.Tracer
}

// NewMiddleware configures and returns a Middleware.
func NewMiddleware(log *logrus.Logger, tracer opentracing.Tracer) *Middleware {
	return &Middleware{
		log:    log,
		tracer: tracer,
	}
}

// Handler is a wrapper for HTTP handlers which logs and records metrics.
func (me *Middleware) Handler(handler func(*fasthttp.RequestCtx)) func(*fasthttp.RequestCtx) {
	return me.tracerMiddleware(me.logMiddleware(handler))
}

func (me *Middleware) tracerMiddleware(handler func(*fasthttp.RequestCtx)) func(*fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Path()) != "/healthcheck" {
			requestHeaders := make(map[string][]string)
			responseHeaders := make(map[string][]string)
			ctx.Request.Header.VisitAll(func(key, value []byte) {
				var array []string
				array = append(array, string(value))
				requestHeaders[string(key)] = array
			})
			ctx.Response.Header.VisitAll(func(key, value []byte) {
				var array []string
				array = append(array, string(value))
				requestHeaders[string(key)] = array
			})
			spanCtx, _ := me.tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(requestHeaders))
			span := me.tracer.StartSpan(string(ctx.Path()), ext.RPCServerOption(spanCtx))
			defer span.Finish()
			ext.SpanKindRPCClient.Set(span)
			ext.HTTPUrl.Set(span, string(ctx.Path()))
			ext.HTTPMethod.Set(span, string(ctx.Method()))
			span.Tracer().Inject(
				span.Context(),
				opentracing.HTTPHeaders,
				opentracing.HTTPHeadersCarrier(responseHeaders),
			)
			me.log.Error(responseHeaders)
			for k, v := range responseHeaders {
				ctx.Response.Header.Add(k, v[0])
			}
		}
		handler(ctx)
	}
}

func (me *Middleware) metricsMiddleware(handler func(*fasthttp.RequestCtx)) func(*fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Path()) != "/healthcheck" {

			handler(ctx)

			tags := []string{fmt.Sprintf("method:%s", ctx.Method()), fmt.Sprintf("path:%s", getMetricsPath(ctx)), fmt.Sprintf("status:%v", ctx.Response.StatusCode())}
			metricTags := ctx.UserValue("metricTags")
			if metricTags != nil {
				for _, tag := range metricTags.([]string) {
					tags = append(tags, tag)
				}
			}
		}
	}
}

// Currently metrics records the first path part, eg:
// "/" => "/"
// "/handler" => "/handler"
func getMetricsPath(ctx *fasthttp.RequestCtx) string {
	fullPath := fmt.Sprintf("%s", ctx.Path())
	return fmt.Sprintf("/%s", strings.Split(fullPath, "/")[1])
}

func (me *Middleware) logMiddleware(handler func(*fasthttp.RequestCtx)) func(*fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Path()) != "/healthcheck" {
			handler(ctx)

			fields := logrus.Fields{
				"method": string(ctx.Method()),
				"path":   string(ctx.Path()),
				"status": ctx.Response.StatusCode(),
			}

			logFields := ctx.UserValue("logFields")
			if logFields != nil {
				for key, val := range logFields.(map[string]interface{}) {
					fields[key] = val
				}
			}

			log := me.log.WithFields(fields)
			switch {
			case ctx.Response.StatusCode() < 400:
				log.Info(fasthttp.StatusMessage(ctx.Response.StatusCode()))
			case ctx.Response.StatusCode() < 500:
				log.Warn(string(ctx.Response.Body()))
			default:
				log.Error(string(ctx.Response.Body()))
			}
		}
	}
}
