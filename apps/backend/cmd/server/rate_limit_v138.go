package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/ratelimit"
)

func authenticatedTenantRateLimit(currentSecret, previousSecret []byte, limiter *ratelimit.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter == nil {
				next.ServeHTTP(w, r)
				return
			}
			header := strings.TrimSpace(r.Header.Get("Authorization"))
			if !strings.HasPrefix(header, "Bearer ") {
				next.ServeHTTP(w, r)
				return
			}
			claims, err := auth.ParseTokenWithFallback(currentSecret, previousSecret, strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
			if err != nil || claims.OrganizationID == uuid.Nil || claims.UserID == uuid.Nil {
				// Authentication middleware remains authoritative for invalid tokens.
				next.ServeHTTP(w, r)
				return
			}

			class := ratelimit.ClassRead
			switch r.Method {
			case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
				class = ratelimit.ClassWrite
			}
			decision, err := limiter.AllowHuman(r.Context(), claims.OrganizationID, claims.UserID, class)
			if err != nil {
				http.Error(w, "rate limiting temporarily unavailable", http.StatusServiceUnavailable)
				return
			}
			if !decision.Allowed {
				writeRateLimitExceeded(w, decision)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func enforceAgentRateLimit(w http.ResponseWriter, r *http.Request, limiter *ratelimit.Limiter, organizationID uuid.UUID, agentID string) bool {
	if limiter == nil {
		return true
	}
	decision, err := limiter.AllowAgent(r.Context(), organizationID, agentID)
	if err != nil {
		http.Error(w, "rate limiting temporarily unavailable", http.StatusServiceUnavailable)
		return false
	}
	if !decision.Allowed {
		writeRateLimitExceeded(w, decision)
		return false
	}
	return true
}

func writeRateLimitExceeded(w http.ResponseWriter, decision ratelimit.Decision) {
	retryAfter := int(decision.RetryAfter.Seconds())
	if retryAfter < 1 {
		retryAfter = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(decision.Limit))
	w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(decision.Remaining))
	w.Header().Set("X-RateLimit-Scope", decision.Scope)
	http.Error(w, "too many requests", http.StatusTooManyRequests)
}
