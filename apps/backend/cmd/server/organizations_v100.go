package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/lyrahcle/lyrahcle/apps/backend/internal/audit"
	"github.com/lyrahcle/lyrahcle/apps/backend/internal/auth"
	"github.com/lyrahcle/lyrahcle/apps/backend/internal/config"
	"github.com/lyrahcle/lyrahcle/apps/backend/internal/store"
)

var organizationSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}$`)

func registerOrganizationRoutes(r chi.Router, cfg config.Config, organizations *store.OrganizationStore, auditLog *store.AuditStore, isTokenRevoked func(context.Context, *auth.Claims) (bool, error)) {
	r.Route("/organizations", func(r chi.Router) {
		r.Use(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleAdmin, isTokenRevoked))
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			list, err := organizations.List(r.Context())
			if err != nil {
				http.Error(w, "organization listing failed", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(list)
		})
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Slug string `json:"slug"`
				Name string `json:"name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			body.Slug, body.Name = strings.TrimSpace(strings.ToLower(body.Slug)), strings.TrimSpace(body.Name)
			if !organizationSlugPattern.MatchString(body.Slug) || body.Name == "" || len(body.Name) > 120 {
				http.Error(w, "invalid organization slug or name", http.StatusBadRequest)
				return
			}
			org, err := organizations.Create(r.Context(), body.Slug, body.Name)
			if err != nil {
				if errors.Is(err, store.ErrOrganizationSlugTaken) {
					http.Error(w, "organization slug already registered", http.StatusConflict)
					return
				}
				http.Error(w, "organization creation failed", http.StatusInternalServerError)
				return
			}
			if claims, ok := auth.FromContext(r.Context()); ok {
				logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), "organization.created", clientIP(r), map[string]interface{}{"organization_id": org.ID.String(), "slug": org.Slug})
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(org)
		})
	})
}
