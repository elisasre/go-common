package siglistener_test

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/elisasre/go-common/service"
	"github.com/elisasre/go-common/service/module/siglistener"
)

func ExampleNew() {
	// Send SIGINT after 1 second.
	go func() {
		time.Sleep(time.Second)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT) //nolint: errcheck
	}()

	s := siglistener.New(os.Interrupt)
	err := service.Run(service.Modules{s})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Service exited successfully")
	// Output: Service exited successfully
}
