package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/dbscope"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
)

type secretProviderStatusResponse struct {
	Provider    string   `json:"provider"`
	External    bool     `json:"external"`
	Location    string   `json:"location,omitempty"`
	Healthy     bool     `json:"healthy"`
	ManagedKeys []string `json:"managed_keys"`
}

func registerSecretStatusRoutes(r chi.Router, cfg config.Config, users *store.UserStore, isTokenRevoked func(context.Context, *auth.Claims) (bool, error)) {
	requirePlatformAdmin := func(next http.Handler) http.Handler {
		return auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.FromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			allowed, err := users.IsPlatformAdministrator(r.Context(), claims.UserID)
			if err != nil {
				http.Error(w, "platform authorization unavailable", http.StatusInternalServerError)
				return
			}
			if !allowed {
				http.Error(w, "platform administrator required", http.StatusForbidden)
				return
			}
			ctx := dbscope.WithPlatformControlPlane(r.Context())
			next.ServeHTTP(w, r.WithContext(ctx))
		}))
	}

	r.With(requirePlatformAdmin).Get("/platform/secrets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(secretProviderStatusResponse{
			Provider: cfg.SecretProviderName, External: cfg.SecretProviderExternal,
			Location: cfg.SecretProviderLocation, Healthy: cfg.SecretProviderHealthy,
			ManagedKeys: []string{"DATABASE_URL", "REDIS_URL", "JWT_SECRET", "JWT_SECRET_PREVIOUS", "SEED_ADMIN_PASSWORD", "SMTP_PASSWORD", "AGENT_CONFIG_SIGNER_TOKEN"},
		})
	})
}
