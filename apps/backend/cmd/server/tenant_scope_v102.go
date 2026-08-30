package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/lyrahcle/lyrahcle/apps/backend/internal/auth"
	"github.com/lyrahcle/lyrahcle/apps/backend/internal/config"
	"github.com/lyrahcle/lyrahcle/apps/backend/internal/store"
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
			agents, err := scope.CountAgents(r.Context(), orgID)
			if err != nil {
				http.Error(w, "tenant summary failed", http.StatusInternalServerError)
				return
			}
			assets, err := scope.CountAssets(r.Context(), orgID)
			if err != nil {
				http.Error(w, "tenant summary failed", http.StatusInternalServerError)
				return
			}
			incidents, err := scope.CountIncidents(r.Context(), orgID)
			if err != nil {
				http.Error(w, "tenant summary failed", http.StatusInternalServerError)
				return
			}
			auditEvents, err := scope.CountAuditEvents(r.Context(), orgID)
			if err != nil {
				http.Error(w, "tenant summary failed", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"organization_id": orgID,
				"agents":          agents,
				"assets":          assets,
				"incidents":       incidents,
				"audit_events":    auditEvents,
			})
		})
	})
}
