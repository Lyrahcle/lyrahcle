package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
)

func registerTenantScopeRoutes(r chi.Router, cfg config.Config, scope *store.TenantScopeStore, isTokenRevoked func(context.Context, *auth.Claims) (bool, error)) {
	r.Route("/organization-scope", func(r chi.Router) {
		r.Use(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleAdmin, isTokenRevoked))
		r.Get("/summary", func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.FromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			orgID, err := auth.OrganizationID(claims)
			if err != nil {
				http.Error(w, "organization context required", http.StatusForbidden)
				return
			}
			snapshot, err := scope.Snapshot(r.Context(), orgID)
			if err != nil {
				http.Error(w, "tenant summary failed", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(snapshot)
		})
	})
}
