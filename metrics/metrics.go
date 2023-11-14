package metrics

import (
	"fmt"
	"net/http"
	"strconv"
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
	reg                     *prometheus.Registry
}

func initRegistry(cs ...prometheus.Collector) *Prometheus {
	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(),
	)
	p := &Prometheus{
		reqCnt: reqCnt,
		reqDur: reqDur,
		ReqCntURLLabelMappingFn: func(c *gin.Context) string {
			return c.Request.URL.Path
		},
		reg: reg,
	}
	reg.MustRegister(p.reqCnt)
	reg.MustRegister(p.reqDur)
	for _, c := range cs {
		reg.MustRegister(c)
	}
	return p
}

func NewPrometheus(port int, cs ...prometheus.Collector) *Prometheus {
	pMux := http.NewServeMux()
	p := initRegistry(cs...)
	pMux.Handle("/metrics", promhttp.HandlerFor(p.reg, promhttp.HandlerOpts{}))
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

func (p *Prometheus) GetRegistry() *prometheus.Registry {
	return p.reg
}

func (p *Prometheus) AddURLMappingFn(fn func(c *gin.Context) string) {
	p.ReqCntURLLabelMappingFn = fn
}

func (p *Prometheus) HandlerFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		status := strconv.Itoa(c.Writer.Status())
		elapsed := float64(time.Since(start)) / float64(time.Second)

		url := p.ReqCntURLLabelMappingFn(c)

		if utf8.ValidString(url) {
			p.reqDur.WithLabelValues(status, c.Request.Method, url).Observe(elapsed)
			p.reqCnt.WithLabelValues(status, c.Request.Method, c.HandlerName(), c.Request.Host, url).Inc()
		}
	}
}
