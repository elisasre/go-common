package metrics

import "github.com/prometheus/client_golang/prometheus"

var reqCnt = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name:      "requests_total",
		Subsystem: "http",
		Help:      "How many HTTP requests processed, partitioned by status code and HTTP method.",
	},
	[]string{"code", "method", "handler", "host", "url"},
)

var reqDur = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:      "request_duration_seconds",
		Subsystem: "http",
		Help:      "The HTTP request latencies in seconds.",
	},
	[]string{"code", "method", "url"},
)
