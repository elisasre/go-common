package httpserver_test

import (
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
