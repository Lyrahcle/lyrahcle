package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/mlko01/lyrahcle/apps/backend/internal/keyprovider"
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

	SigningKeyStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lyrahcle_signing_keys",
		Help: "Number of managed remote-config signing keys by lifecycle status",
	}, []string{"domain", "status"})

	SigningKeyRotationDue = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lyrahcle_signing_key_rotation_due",
		Help: "Whether a managed signing-key rotation is due",
	}, []string{"domain"})

	SigningKeyStagedPending = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lyrahcle_signing_key_staged_pending",
		Help: "Whether a staged signing key is awaiting distribution and activation",
	}, []string{"domain"})

	SigningKeyWarnings = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "lyrahcle_signing_key_warnings",
		Help: "Current signing-key lifecycle warnings by code",
	}, []string{"domain", "code"})

	RateLimitDecisions = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lyrahcle_rate_limit_decisions_total",
		Help: "Authenticated tenant-aware rate-limit decisions by scope, class, and result",
	}, []string{"scope", "class", "result"})

	RateLimitBackendFallback = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lyrahcle_rate_limit_backend_fallback_total",
		Help: "Rate-limit backend fallbacks by backend and scope",
	}, []string{"backend", "scope"})

	BackgroundJobEventsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lyrahcle_background_job_events_total",
		Help: "Background-job delivery and processing outcomes by job and result",
	}, []string{"job", "result"})

	ScheduledJobLeadershipTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lyrahcle_scheduled_job_leadership_total",
		Help: "Distributed scheduled-job lock decisions by job and result",
	}, []string{"job", "result"})
	SigningKeyAutomationTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "lyrahcle_signing_key_automation_total",
		Help: "Automated signing-key lifecycle checks by result",
	}, []string{"domain", "result"})
)

func init() {
	prometheus.MustRegister(RequestsTotal, RequestDuration, AuthLoginsTotal, SSEConnections, TLEAgeSeconds, TLEStale, TLECorrelationEligible, OrbitalCorrelationSuppressedTotal, CelestrakSyncFailuresTotal, RateLimitDecisions, RateLimitBackendFallback, BackgroundJobEventsTotal, ScheduledJobLeadershipTotal, SigningKeyStatus, SigningKeyRotationDue, SigningKeyStagedPending, SigningKeyWarnings, SigningKeyAutomationTotal)
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

func RecordSigningKeyLifecycle(domain string, keys []keyprovider.KeyMetadata, health keyprovider.RotationHealth) {
	if domain == "" {
		domain = "unknown"
	}
	for _, status := range []keyprovider.KeyStatus{keyprovider.KeyStatusActive, keyprovider.KeyStatusStaged, keyprovider.KeyStatusGrace, keyprovider.KeyStatusRevoked} {
		SigningKeyStatus.WithLabelValues(domain, string(status)).Set(0)
	}
	for _, key := range keys {
		SigningKeyStatus.WithLabelValues(domain, string(key.Status)).Inc()
	}
	due := 0.0
	if health.RotationDue {
		due = 1
	}
	SigningKeyRotationDue.WithLabelValues(domain).Set(due)
	staged := 0.0
	if health.StagedPending {
		staged = 1
	}
	SigningKeyStagedPending.WithLabelValues(domain).Set(staged)
	for _, code := range []string{"no_active_key", "rotation_due", "staged_pending", "grace_expiring"} {
		SigningKeyWarnings.WithLabelValues(domain, code).Set(0)
	}
	for _, warning := range health.Warnings {
		SigningKeyWarnings.WithLabelValues(domain, warning.Code).Set(1)
	}
}
