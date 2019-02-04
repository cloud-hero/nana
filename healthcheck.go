package nana

import (
	"encoding/json"

	"github.com/valyala/fasthttp"
)

// HealthCheckHandler provides a basic health check which returns JSON payload including app name and version
func HealthCheckHandler(appName string, version string) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		message := map[string]interface{}{
			"name":    appName,
			"version": version,
			"healthy": true,
		}

		body, _ := json.Marshal(message)
		ctx.Success("application/json", body)
	}
}

// ExtendedHealthCheckHandler provides an extended health check which returns extra details in json.RawMessage form
func ExtendedHealthCheckHandler(appName string, version string, verifier func() (bool, *json.RawMessage)) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		healthy, data := verifier()

		message := map[string]interface{}{
			"name":    appName,
			"version": version,
			"healthy": healthy,
			"data":    data,
		}

		body, _ := json.Marshal(message)
		ctx.Success("application/json", body)
	}
}
