package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"net/http"
	"time"
)

func (a *apiRoutes) registerAuditThreatRoutes(r chi.Router) {
	auditLog := a.auditLog
	cache := a.cache
	cfg := a.cfg
	isTokenRevoked := a.isTokenRevoked
	rbac := a.rbac
	spaceCVEs := a.spaceCVEs

	r.With(auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "audit.read", isTokenRevoked)).Get("/audit-log", func(w http.ResponseWriter, r *http.Request) {
		limit, offset := pageParams(r)
		orgID, ok := requestOrganizationID(w, r)
		if !ok {
			return
		}
		page, err := auditLog.ListByOrganization(r.Context(), orgID, limit, offset)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(page)
	})

	r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Get("/threat-intel/space-cves", func(w http.ResponseWriter, r *http.Request) {
		limit, offset := pageParams(r)
		filter := parseSpaceCVEFilter(r)

		cacheKey := fmt.Sprintf("lyrahcle:cache:space-cves:%d:%d:%s:%s:%s:%s:%s:%t",
			limit, offset, filter.Severity, filter.Keyword,
			filter.PublishedFrom.Format(time.RFC3339), filter.PublishedTo.Format(time.RFC3339),
			filter.SortBy, filter.SortDesc)
		var page store.Page[store.SpaceCVE]
		if hit, _ := cache.Get(r.Context(), cacheKey, &page); hit {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(page)
			return
		}

		page, err := spaceCVEs.List(r.Context(), limit, offset, filter)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cache.Set(r.Context(), cacheKey, page, 30*time.Second)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(page)
	})

	r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Get("/threat-intel/space-cves/keywords", func(w http.ResponseWriter, r *http.Request) {
		const cacheKey = "lyrahcle:cache:space-cves:keywords"
		var keywords []string
		if hit, _ := cache.Get(r.Context(), cacheKey, &keywords); hit {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(keywords)
			return
		}

		keywords, err := spaceCVEs.DistinctKeywords(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cache.Set(r.Context(), cacheKey, keywords, 5*time.Minute)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(keywords)
	})
}
