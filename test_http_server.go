package nana

import (
	"fmt"
	"io"
	"net"
	"sync"
	"testing"

	"github.com/valyala/fasthttp"
)

// TestHTTPServer provides a way for apps to test using a real HTTP server on a random port.
type TestHTTPServer struct {
	listener io.Closer
	Port     int
}

// Addr provides the address of the test server.
func (server *TestHTTPServer) Addr() string {
	return fmt.Sprintf("localhost:%d", server.Port)
}

// Close stops the test HTTP server.
func (server *TestHTTPServer) Close() {
	server.listener.Close()
}

var ports chan int

func init() {
	ports = make(chan int, 9999)
	for i := 10000; i < 19999; i++ {
		ports <- i
	}
}

var portsPool = sync.Pool{
	New: func() interface{} {
		return <-ports
	},
}

// NewTestHTTPServer spins up a new HTTP server on a dynamically assigned port for use in testing your app.
func NewTestHTTPServer(t *testing.T, handler fasthttp.RequestHandler) *TestHTTPServer {
	server := &TestHTTPServer{
		Port: portsPool.Get().(int),
	}

	ln, err := net.Listen("tcp", server.Addr())
	if err != nil {
		t.Fatalf("Cannot start tcp server at %s: %s", server.Addr(), err)
	}
	server.listener = ln
	go fasthttp.Serve(ln, handler)

	return server
}
