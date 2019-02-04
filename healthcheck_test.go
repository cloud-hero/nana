package nana

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasicHealthCheckHandler(t *testing.T) {
	assert := assert.New(t)

	server := NewTestHTTPServer(t, HealthCheckHandler("app name", "0.0.0-test"))
	defer server.Close()

	resp, err := http.Get(fmt.Sprintf("http://%s/healthcheck", server.Addr()))
	if err != nil {
		t.Fatalf("Error consuming health check: %s", err)
	}

	body, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(`{"healthy":true,"name":"app name","version":"0.0.0-test"}`, string(body))
}

func TestExtendedHealthCheckHandler(t *testing.T) {
	assert := assert.New(t)

	server := NewTestHTTPServer(t, ExtendedHealthCheckHandler("app name", "0.0.0-test", testHealthVerifier))
	defer server.Close()

	resp, err := http.Get(fmt.Sprintf("http://%s/healthcheck", server.Addr()))
	if err != nil {
		t.Fatalf("Error consuming health check: %s", err)
	}

	body, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(`{"data":[{"name":"Nats","isHealthy":true}],"healthy":true,"name":"app name","version":"0.0.0-test"}`, string(body))
}

func testHealthVerifier() (bool, *json.RawMessage) {
	raw := json.RawMessage(`[{"name": "Nats", "isHealthy": true}]`)
	return true, &raw
}
