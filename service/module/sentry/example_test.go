package sentry_test

import (
	"fmt"

	"github.com/elisasre/go-common/service"
	"github.com/elisasre/go-common/service/module/sentry"
)

func ExampleNew() {
	s := sentry.New(
		sentry.WithDSN("some-dsn"),
	)
	err := service.Run(service.Modules{s})
	if err != nil {
		fmt.Println(err)
	}
	// Output: failed to initialize module sentry.Sentry: [Sentry] DsnParseError: invalid scheme
}
