package metrics

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Prometheus struct {
	reqCnt                  *prometheus.CounterVec
	reqDur                  *prometheus.HistogramVec
	ReqCntURLLabelMappingFn func(c *gin.Context) string
	SkipMetricsURLFn        func(c *gin.Context) bool
	reg                     *prometheus.Registry
	init                    sync.Once
	cs                      []prometheus.Collector
}

// New creates Prometheus instance with given collectors.
// Before usage p.Init() must be called.
func New(cs ...prometheus.Collector) *Prometheus {
	return &Prometheus{cs: cs}
}

// NewPrometheus creates registers collectors and starts metrics server.
// Deprecated: This function will panic instead of returning error. Use metrics module instead.
func NewPrometheus(port int, cs ...prometheus.Collector) *Prometheus {
	p := New(cs...)
	if err := p.Init(); err != nil {
		panic(err)
	}

	pMux := http.NewServeMux()
	pMux.Handle("/metrics", NewPrometheusHandler(p))
	go func() {
		listenAddr := fmt.Sprintf(":%d", port)

		srv := &http.Server{
			Addr:              listenAddr,
			Handler:           pMux,
			ReadHeaderTimeout: 3 * time.Second,
		}
		err := srv.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()
	return p
}

func (p *Prometheus) Init() (err error) {
	p.init.Do(func() {
		p.reqCnt = reqCnt
		p.reqDur = reqDur
		p.ReqCntURLLabelMappingFn = func(c *gin.Context) string { return c.Request.URL.Path }
		p.SkipMetricsURLFn = func(c *gin.Context) bool { return false }
		p.reg = prometheus.NewPedanticRegistry()

		collectors := []prometheus.Collector{
			collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
			collectors.NewGoCollector(),
			p.reqCnt,
			p.reqDur,
		}

		collectors = append(collectors, p.cs...)
		for _, c := range collectors {
			if err = p.reg.Register(c); err != nil {
				return
			}
		}
	})

	return err
}

func (p *Prometheus) GetRegistry() *prometheus.Registry {
	return p.reg
}

func (p *Prometheus) AddURLMappingFn(fn func(c *gin.Context) string) {
	p.ReqCntURLLabelMappingFn = fn
}

func (p *Prometheus) AddSkipMetricsURLFn(fn func(c *gin.Context) bool) {
	p.SkipMetricsURLFn = fn
}

func (p *Prometheus) HandlerFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		status := strconv.Itoa(c.Writer.Status())
		elapsed := float64(time.Since(start)) / float64(time.Second)

		url := p.ReqCntURLLabelMappingFn(c)

		if utf8.ValidString(url) && !p.SkipMetricsURLFn(c) {
			p.reqDur.WithLabelValues(status, c.Request.Method, url).Observe(elapsed)
			p.reqCnt.WithLabelValues(status, c.Request.Method, c.HandlerName(), c.Request.Host, url).Inc()
		}
	}
}

// NewPrometheusHandler creates http.Handler with Prometheus registry.
func NewPrometheusHandler(p *Prometheus) http.Handler {
	return promhttp.HandlerFor(p.reg, promhttp.HandlerOpts{})
}
