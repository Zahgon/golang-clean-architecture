package app_gin

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/DoWithLogic/golang-clean-architecture/pkg/binder"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	_           = iota // ignore first value by assigning to blank identifier
	bKB float64 = 1 << (10 * iota)
	bMB
)

// sizeBuckets is the buckets for request/response size. Here we define a spectrum from 1KB through 1NB up to 10MB.
var sizeBuckets = []float64{1.0 * bKB, 2.0 * bKB, 5.0 * bKB, 10.0 * bKB, 100 * bKB, 500 * bKB, 1.0 * bMB, 2.5 * bMB, 5.0 * bMB, 10.0 * bMB}

// prometheusMiddleware collects request count, latency and request/response size metrics in the
// prometheus default registry, labelled by status code, method, host and route path.
func prometheusMiddleware(subsystem string) gin.HandlerFunc {
	labelNames := []string{"code", "method", "host", "url"}

	requestCount := prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: subsystem,
		Name:      "requests_total",
		Help:      "How many HTTP requests processed, partitioned by status code and HTTP method.",
	}, labelNames)
	requestDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: subsystem,
		Name:      "request_duration_seconds",
		Help:      "The HTTP request latencies in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, labelNames)
	responseSize := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: subsystem,
		Name:      "response_size_bytes",
		Help:      "The HTTP response sizes in bytes.",
		Buckets:   sizeBuckets,
	}, labelNames)
	requestSize := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: subsystem,
		Name:      "request_size_bytes",
		Help:      "The HTTP request sizes in bytes.",
		Buckets:   sizeBuckets,
	}, labelNames)
	prometheus.MustRegister(requestCount, requestDuration, responseSize, requestSize)

	return func(c *gin.Context) {
		reqSz := computeApproximateRequestSize(c.Request)

		start := time.Now()
		c.Next()
		elapsed := float64(time.Since(start)) / float64(time.Second)

		url := c.FullPath() // contains route path ala `/users/:id`
		if url == "" {
			// path is empty for 404 cases (when router did not find any matching routes)
			// in this case we use actual path from request to have some distinction in Prometheus
			url = c.Request.URL.Path
		}

		status := c.Writer.Status()
		if err := c.Errors.Last(); err != nil {
			var httpErr *binder.HTTPError
			if errors.As(err.Err, &httpErr) {
				status = httpErr.Code
			}
			if status == 0 || status == http.StatusOK {
				status = http.StatusInternalServerError
			}
		}

		values := []string{strconv.Itoa(status), c.Request.Method, c.Request.Host, url}

		requestDuration.WithLabelValues(values...).Observe(elapsed)
		requestCount.WithLabelValues(values...).Inc()
		requestSize.WithLabelValues(values...).Observe(float64(reqSz))
		responseSize.WithLabelValues(values...).Observe(float64(max(c.Writer.Size(), 0)))
	}
}

func computeApproximateRequestSize(r *http.Request) int {
	s := 0
	if r.URL != nil {
		s = len(r.URL.Path)
	}

	s += len(r.Method)
	s += len(r.Proto)
	for name, values := range r.Header {
		s += len(name)
		for _, value := range values {
			s += len(value)
		}
	}
	s += len(r.Host)

	// N.B. r.Form and r.MultipartForm are assumed to be included in r.URL.

	if r.ContentLength != -1 {
		s += int(r.ContentLength)
	}
	return s
}
