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

	AuthLoginsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lyrahcle_auth_logins_total",
		Help: "Authentication attempts by result",
	}, []string{"result"})

	SSEConnections = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lyrahcle_sse_connections",
		Help: "Currently open SSE connections by stream",
	}, []string{"stream"})

	TLEAgeSeconds = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lyrahcle_tle_age_seconds",
		Help: "Age of the current TLE epoch in seconds by NORAD ID",
	}, []string{"norad_id"})

	TLEStale = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lyrahcle_tle_stale",
		Help: "Whether the current TLE is stale or unusable by NORAD ID",
	}, []string{"norad_id"})

	TLECorrelationEligible = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lyrahcle_tle_correlation_eligible",
		Help: "Whether the current TLE satisfies the automatic orbital-correlation freshness gate by NORAD ID",
	}, []string{"norad_id"})

	OrbitalCorrelationSuppressedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lyrahcle_orbital_correlation_suppressed_total",
		Help: "Orbital correlation suppressions by reason",
	}, []string{"reason"})

	CelestrakSyncFailuresTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lyrahcle_celestrak_sync_failures_total",
		Help: "CelesTrak synchronization failures by group",
	}, []string{"group"})
)

func init() {
	prometheus.MustRegister(RequestsTotal, RequestDuration, AuthLoginsTotal, SSEConnections, TLEAgeSeconds, TLEStale, TLECorrelationEligible, OrbitalCorrelationSuppressedTotal, CelestrakSyncFailuresTotal)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

type flushableStatusRecorder struct {
	*statusRecorder
	flusher http.Flusher
}

func (r *flushableStatusRecorder) Flush() {
	r.flusher.Flush()
}

func routePattern(r *http.Request) string {
	routeContext := chi.RouteContext(r.Context())
	if routeContext == nil {
		return "unmatched"
	}
	route := routeContext.RoutePattern()
	if route == "" {
		return "unmatched"
	}
	return route
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		writer := http.ResponseWriter(recorder)
		if flusher, ok := w.(http.Flusher); ok {
			writer = &flushableStatusRecorder{statusRecorder: recorder, flusher: flusher}
		}

		next.ServeHTTP(writer, r)

		route := routePattern(r)

		RequestsTotal.WithLabelValues(r.Method, route, strconv.Itoa(recorder.status)).Inc()
		RequestDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
	})
}
