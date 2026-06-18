package metrics

import (
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	registerOnce sync.Once

	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "crm_http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"service", "method", "route", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "crm_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method", "route"},
	)
)

func Register() {
	registerOnce.Do(func() {
		prometheus.MustRegister(httpRequestsTotal, httpRequestDuration)
	})
}

func Middleware(service string) gin.HandlerFunc {
	Register()
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		status := strconv.Itoa(c.Writer.Status())
		httpRequestsTotal.WithLabelValues(service, c.Request.Method, route, status).Inc()
		httpRequestDuration.WithLabelValues(service, c.Request.Method, route).Observe(time.Since(started).Seconds())
	}
}

func Handler() gin.HandlerFunc {
	Register()
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
