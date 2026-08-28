package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	RequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lyrahcle_http_requests_total",
		Help: "Total HTTP requests handled by the backend, by method/route/status",
	}, []string{"method", "route", "status"})

	RequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "lyrahcle_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds, by method/route",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route"})
)

func init() {
	prometheus.MustRegister(RequestsTotal, RequestDuration)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}

		RequestsTotal.WithLabelValues(r.Method, route, strconv.Itoa(rec.status)).Inc()
		RequestDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
	})
}
