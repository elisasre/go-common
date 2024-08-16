package pprof_test

import (
	"github.com/elisasre/go-common/v2/service/module/httpserver"
	"github.com/elisasre/go-common/v2/service/module/httpserver/pprof"
)

func ExampleWithProfiling() {
	httpserver.New(
		httpserver.WithAddr(":6062"),
		pprof.WithProfiling(),
	)
}
