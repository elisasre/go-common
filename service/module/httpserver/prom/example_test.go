package prom_test

import (
	"github.com/elisasre/go-common/metrics"
	"github.com/elisasre/go-common/service/module/httpserver"
	"github.com/elisasre/go-common/service/module/httpserver/prom"
)

func ExampleWithMetrics() {
	httpserver.New(
		httpserver.WithAddr(":6062"),
		prom.WithMetrics(metrics.New()),
	)
}
