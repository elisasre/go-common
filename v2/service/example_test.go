package service_test

import (
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/elisasre/go-common/v2/service"
	"github.com/elisasre/go-common/v2/service/module/httpserver"
	"github.com/elisasre/go-common/v2/service/module/httpserver/pprof"
	"github.com/elisasre/go-common/v2/service/module/siglistener"
)

func ExampleRun() {
	// Send SIGINT after 5 second.
	go func() {
		time.Sleep(time.Second * 5)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT) //nolint: errcheck
	}()

	err := service.Run(service.Modules{
		siglistener.New(os.Interrupt),
		httpserver.New(
			httpserver.WithAddr(":6062"),
			pprof.WithProfiling(),
		),
		httpserver.New(
			httpserver.WithServer(&http.Server{ReadHeaderTimeout: time.Second}),
			httpserver.WithAddr(":6060"),
			httpserver.WithHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "Hello")
			})),
		),
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Service exited successfully")
	// Output: Service exited successfully
}
