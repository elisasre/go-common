package httpserver_test

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/elisasre/go-common/v2/service/module/httpserver"
)

func ExampleNew() {
	httpserver.New(
		httpserver.WithServer(&http.Server{ReadHeaderTimeout: time.Second}),
		httpserver.WithAddr("127.0.0.1:0"),
		httpserver.WithHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "Hello")
		})),
	)
}

func ExampleNew_https() {
	// Load your certificate and key files
	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		panic(err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	httpserver.New(
		httpserver.WithAddr("127.0.0.1:8443"),
		httpserver.WithTLSConfig(tlsConfig),
		httpserver.WithHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "Hello HTTPS")
		})),
	)
}
