// Package prom provides prometheus metrics handler options for httpserver module.
package prom

import (
	"fmt"

	"github.com/elisasre/go-common/metrics"
	"github.com/elisasre/go-common/service/module/httpserver"
)

// WithMetrics replaces servers handler with http.Handler which is instrumented with /metrics endpoint.
// This option is meant be used with stand alone metrics server, not embedded inside application server.
// For serving metrics endpoint inside your application web server check lower level functionalities from metrics.
func WithMetrics(p *metrics.Prometheus) httpserver.Opt {
	return func(s *httpserver.Server) error {
		if err := p.Init(); err != nil {
			return fmt.Errorf("failed to initialize prometheus handler: %w", err)
		}
		return httpserver.WithHandler(metrics.NewPrometheusHandler(p))(s)
	}
}
