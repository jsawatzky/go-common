package metrics

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/jsawatzky/go-common/internal"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/spf13/viper"
)

var (
	httpRequestsTotal    *prometheus.CounterVec
	httpRequestsDuration *prometheus.HistogramVec
	httpResponseSize     *prometheus.HistogramVec
)
var once sync.Once

func Middleware(h http.Handler) http.Handler {

	once.Do(func() {
		httpRequestsTotal = promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: viper.GetString("metric_namespace"),
				Subsystem: "http",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"method", "path", "code"},
		)
		httpRequestsDuration = promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: viper.GetString("metric_namespace"),
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "HTTP request duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "path", "code"},
		)
		httpResponseSize = promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: viper.GetString("metric_namespace"),
				Subsystem: "http",
				Name:      "response_size_bytes",
				Help:      "Size of HTTP responses in bytes",
				Buckets:   prometheus.ExponentialBuckets(10, 5, 6),
			},
			[]string{"method", "path", "code"},
		)
	})

	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		path := "unknown"
		if route != nil {
			if pt, err := route.GetPathTemplate(); err == nil {
				path = pt
			}
		}

		start := time.Now()

		rec := internal.RecordResponse(rw)
		h.ServeHTTP(rec, r)

		labels := []string{r.Method, path, strconv.Itoa(rec.Status())}
		httpRequestsDuration.WithLabelValues(labels...).Observe(float64(time.Since(start).Seconds()))
		httpResponseSize.WithLabelValues(labels...).Observe(float64(rec.ResponseSize()))
		httpRequestsTotal.WithLabelValues(labels...).Inc()
	})
}
