package metrics

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/elisasre/go-common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestPrometheus(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	p := NewPrometheus(65001)
	reg := p.GetRegistry()
	r.Use(p.HandlerFunc())
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	type testCase struct {
		name        string
		url         []byte
		code        int
		count       int
		ignoreCount bool
	}

	testCases := []testCase{
		{
			name:  "valid request to /ping",
			url:   []byte("/ping"),
			code:  http.StatusOK,
			count: 3,
		},
		{
			name:  "request to /notfound path",
			url:   []byte("/notfound"),
			code:  http.StatusNotFound,
			count: 2,
		},
		{
			name:        "request to non utf8 path",
			url:         []byte("/\xc0"),
			code:        http.StatusNotFound,
			count:       3,
			ignoreCount: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for i := 0; i < tc.count; i++ {
				w := httptest.NewRecorder()
				req := httptest.NewRequest("GET", string(tc.url), nil)
				r.ServeHTTP(w, req)
				assert.Equal(t, tc.code, w.Code)
			}
		})
	}

	countsByCode := make(map[string]int, len(testCases))
	for _, tc := range testCases {
		if tc.ignoreCount {
			continue
		}
		countsByCode[strconv.Itoa(tc.code)] += tc.count
	}

	metricFamily, err := reg.Gather()
	assert.NoError(t, err)
	for _, mf := range metricFamily {
		if common.StringValue(mf.Name) == "http_requests_total" {
			for _, m := range mf.GetMetric() {
				for _, l := range m.Label {
					if common.StringValue(l.Name) == "code" {
						value := common.StringValue(l.Value)
						assert.Equal(t, countsByCode[value], int(common.Float64Value(m.Counter.Value)))
					}
				}
			}
		}
	}
}
