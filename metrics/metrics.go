package metrics

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Prometheus struct {
	reqCnt                  *prometheus.CounterVec
	reqDur                  *prometheus.HistogramVec
	ReqCntURLLabelMappingFn func(c *gin.Context) string
}

func NewPrometheus(port int, cs ...prometheus.Collector) *Prometheus {
	pMux := http.NewServeMux()
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
	}
	reg.MustRegister(p.reqCnt)
	reg.MustRegister(p.reqDur)
	for _, c := range cs {
		reg.MustRegister(c)
	}
	pMux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	go func() {
		listenAddr := fmt.Sprintf(":%d", port)

		server := &http.Server{
			Addr:              listenAddr,
			ReadHeaderTimeout: 3 * time.Second,
		}

		err := server.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()
	return p
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

		p.reqDur.WithLabelValues(status, c.Request.Method, url).Observe(elapsed)
		p.reqCnt.WithLabelValues(status, c.Request.Method, c.HandlerName(), c.Request.Host, url).Inc()
	}
}
