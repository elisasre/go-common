// Package pprof provides pprof handler options for httpserver module.
package pprof

import (
	"net/http"

	//nolint:gosec
	_ "net/http/pprof"

	"github.com/elisasre/go-common/service/module/httpserver"
)

// WithProfiling replaces servers handler with http.DefaultServeMux which is instrumented with profiling endpoints by net/http/pprof.
// This option is meant be used with stand alone profiling server, not embedded inside application server.
// For serving profiling endpoints inside your application web server see https://pkg.go.dev/net/http/pprof.
func WithProfiling() httpserver.Opt {
	return httpserver.WithHandler(http.DefaultServeMux)
}
