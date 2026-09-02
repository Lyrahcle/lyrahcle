package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/validate"
)

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusCapturingResponseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusCapturingResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(body)
	w.bytes += n
	return n, err
}

func (w *statusCapturingResponseWriter) Flush() {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func structuredRequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		captured := &statusCapturingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(captured, r)
		status := captured.status
		if status == 0 {
			status = http.StatusOK
		}
		event := map[string]interface{}{
			"event":       "http_request",
			"method":      r.Method,
			"path":        r.URL.Path,
			"status":      status,
			"bytes":       captured.bytes,
			"request_id":  middleware.GetReqID(r.Context()),
			"remote_ip":   clientIP(r),
			"duration_ms": time.Since(start).Milliseconds(),
		}
		if encoded, err := json.Marshal(event); err == nil {
			log.Print(string(encoded))
		}
	})
}

func sanitizeRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID != "" && !validRequestID(requestID) {
			r.Header.Del("X-Request-ID")
		}
		next.ServeHTTP(w, r)
	})
}

func validRequestID(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		switch r {
		case '-', '_', '.', ':':
			continue
		default:
			return false
		}
	}
	return true
}

func requestIDResponseHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestID := middleware.GetReqID(r.Context()); requestID != "" {
			w.Header().Set("X-Request-ID", requestID)
		}
		next.ServeHTTP(w, r)
	})
}

func maxBodySize(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, n)
			next.ServeHTTP(w, r)
		})
	}
}

func pageParams(r *http.Request) (int, int) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	return limit, offset
}

func optionalUUID(value, field string) (*uuid.UUID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("%s must be a valid UUID", field)
	}
	return &id, nil
}

func parseIncidentFilter(r *http.Request) (store.IncidentFilter, error) {
	q := r.URL.Query()
	filter := store.IncidentFilter{
		Status:   strings.ToLower(strings.TrimSpace(q.Get("status"))),
		Severity: strings.ToLower(strings.TrimSpace(q.Get("severity"))),
		AgentID:  strings.TrimSpace(q.Get("agent_id")),
	}
	if filter.Status != "" {
		if err := validate.IncidentStatus(filter.Status); err != nil {
			return store.IncidentFilter{}, err
		}
	}
	if filter.Severity != "" {
		if err := validate.IncidentSeverity(filter.Severity); err != nil {
			return store.IncidentFilter{}, err
		}
	}
	if filter.AgentID != "" {
		if err := validate.AgentID(filter.AgentID); err != nil {
			return store.IncidentFilter{}, err
		}
	}
	assignedTo, err := optionalUUID(q.Get("assigned_to"), "assigned_to")
	if err != nil {
		return store.IncidentFilter{}, err
	}
	filter.AssignedTo = assignedTo
	satelliteID, err := optionalUUID(q.Get("satellite_id"), "satellite_id")
	if err != nil {
		return store.IncidentFilter{}, err
	}
	filter.SatelliteID = satelliteID
	return filter, nil
}

func incidentsCacheKey(orgID uuid.UUID, limit, offset int, filter store.IncidentFilter) string {
	assignedTo := ""
	if filter.AssignedTo != nil {
		assignedTo = filter.AssignedTo.String()
	}
	satelliteID := ""
	if filter.SatelliteID != nil {
		satelliteID = filter.SatelliteID.String()
	}
	return fmt.Sprintf("lyrahcle:cache:incidents:%s:%d:%d:%s:%s:%s:%s:%s",
		orgID, limit, offset, filter.Status, filter.Severity, filter.AgentID, assignedTo, satelliteID)
}

func incidentsCacheKeyDefault(orgID uuid.UUID) string {
	return incidentsCacheKey(orgID, 50, 0, store.IncidentFilter{})
}

func assetAlertsCacheKey(orgID uuid.UUID, limit, offset int) string {
	return fmt.Sprintf("lyrahcle:cache:asset-alerts:%s:%d:%d", orgID, limit, offset)
}

func assetAlertsCacheKeyDefault(orgID uuid.UUID) string {
	return assetAlertsCacheKey(orgID, 50, 0)
}

func rfReadingsCacheKey(orgID uuid.UUID) string {
	return fmt.Sprintf("lyrahcle:cache:rf:readings:%s", orgID)
}

func rfAnomaliesCacheKey(orgID uuid.UUID) string {
	return fmt.Sprintf("lyrahcle:cache:rf:anomalies:%s", orgID)
}

func parseSpaceCVEFilter(r *http.Request) store.SpaceCVEFilter {
	q := r.URL.Query()
	filter := store.SpaceCVEFilter{
		Severity: strings.TrimSpace(q.Get("severity")),
		Keyword:  strings.TrimSpace(q.Get("keyword")),
		SortBy:   strings.TrimSpace(q.Get("sort")),
		SortDesc: strings.ToLower(strings.TrimSpace(q.Get("order"))) != "asc",
	}
	if from := strings.TrimSpace(q.Get("published_from")); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			filter.PublishedFrom = t
		}
	}
	if to := strings.TrimSpace(q.Get("published_to")); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			filter.PublishedTo = t.Add(24*time.Hour - time.Second)
		}
	}
	return filter
}

func securityHeaders(tlsEnabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			h.Set("Cross-Origin-Resource-Policy", "same-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), usb=()")
			h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			if tlsEnabled {
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}
