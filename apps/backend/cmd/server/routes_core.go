package main

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"net/http"
	"strings"
)

func (a *apiRoutes) registerCoreRoutes(r chi.Router) {
	agentCA := a.agentCA
	agentConfigKeyProvider := a.agentConfigKeyProvider
	agentConfigSigner := a.agentConfigSigner
	agentConfigs := a.agentConfigs
	agentIdentities := a.agentIdentities
	agentKeys := a.agentKeys
	auditLog := a.auditLog
	cfg := a.cfg
	fleetRollouts := a.fleetRollouts
	isTokenRevoked := a.isTokenRevoked
	rbac := a.rbac
	satellites := a.satellites
	searchStore := a.searchStore
	tenantLimiter := a.tenantLimiter

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"message":"pong from lyrahcle"}`))
	})

	registerAgentIdentityRoutes(r, cfg, agentCA, agentIdentities, agentKeys, auditLog, rbac, isTokenRevoked, tenantLimiter)
	registerAgentConfigRoutes(r, cfg, agentConfigSigner, agentConfigKeyProvider, agentConfigs, agentIdentities, satellites, auditLog, rbac, isTokenRevoked, tenantLimiter)
	registerFleetRolloutRoutes(r, cfg, agentConfigSigner, fleetRollouts, agentConfigs, satellites, auditLog, rbac, isTokenRevoked)

	r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Get("/search", func(w http.ResponseWriter, r *http.Request) {
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if len(query) < 2 || len(query) > 96 {
			http.Error(w, "q must contain 2-96 characters", http.StatusBadRequest)
			return
		}
		orgID, ok := requestOrganizationID(w, r)
		if !ok {
			return
		}
		results, err := searchStore.SearchByOrganization(r.Context(), orgID, query, 12)
		if err != nil {
			http.Error(w, "search failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	})

	r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Get("/system-info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"environment":             cfg.Environment,
			"license":                 "AGPLv3 (core)",
			"celestrak_groups":        cfg.CelestrakGroups,
			"celestrak_sync_schedule": "every 6 hours",
			"nvd_sync_note":           "managed by the cve-sync worker (apps/cve-sync)",
		})
	})
}
