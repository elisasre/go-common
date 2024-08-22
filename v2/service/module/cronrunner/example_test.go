package cronrunner_test

import (
	"fmt"
	"os"
	"syscall"

	"github.com/elisasre/go-common/v2/service"
	"github.com/elisasre/go-common/v2/service/module/cronrunner"
	"github.com/elisasre/go-common/v2/service/module/siglistener"
	"github.com/robfig/cron/v3"
)

func ExampleNew() {
	runner := cronrunner.New(
		cronrunner.WithCron(cron.New(cron.WithSeconds())),
		cronrunner.WithFunc("@every 1s", func() {
			fmt.Println("cron job executed")
			syscall.Kill(syscall.Getpid(), syscall.SIGINT) //nolint: errcheck
		}),
	)

	err := service.Run(service.Modules{siglistener.New(os.Interrupt), runner})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Output: cron job executed
}

func ExampleNew_customAddFunc() {
	var (
		id  cron.EntryID
		err error
	)

	c := cron.New(cron.WithSeconds())
	id, err = c.AddFunc("@every 1s", func() {
		fmt.Printf("hello from job with id: %d\n", id)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT) //nolint: errcheck
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	err = service.Run(service.Modules{
		siglistener.New(os.Interrupt),
		cronrunner.New(cronrunner.WithCron(c)),
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Output: hello from job with id: 1
}
