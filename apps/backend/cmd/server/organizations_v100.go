package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
)

var organizationSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}$`)

func registerOrganizationRoutes(r chi.Router, cfg config.Config, organizations *store.OrganizationStore, users *store.UserStore, auditLog *store.AuditStore, isTokenRevoked func(context.Context, *auth.Claims) (bool, error)) {
	r.Route("/organizations", func(r chi.Router) {
		r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Get("/me", func(w http.ResponseWriter, r *http.Request) {
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
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]bool{"platform_admin": allowed})
		})
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
				next.ServeHTTP(w, r)
			}))
		}
		r.With(requirePlatformAdmin).Get("/", func(w http.ResponseWriter, r *http.Request) {
			list, err := organizations.List(r.Context())
			if err != nil {
				http.Error(w, "organization listing failed", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(list)
		})
		r.With(requirePlatformAdmin).Post("/", func(w http.ResponseWriter, r *http.Request) {
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
				logAuditInOrganization(r.Context(), auditLog, org.ID, audit.ActorUser, claims.UserID.String(), audit.ActionOrganizationCreated, clientIP(r), map[string]interface{}{
					"organization_id":       org.ID.String(),
					"actor_organization_id": claims.OrganizationID.String(),
					"slug":                  org.Slug,
					"name":                  org.Name,
				})
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(org)
		})

		r.With(requirePlatformAdmin).Get("/platform-users", func(w http.ResponseWriter, r *http.Request) {
			allUsers, err := users.List(r.Context())
			if err != nil {
				http.Error(w, "platform user listing failed", http.StatusInternalServerError)
				return
			}
			adminIDs, err := users.ListPlatformAdministratorIDs(r.Context())
			if err != nil {
				http.Error(w, "platform administrator listing failed", http.StatusInternalServerError)
				return
			}
			adminSet := map[uuid.UUID]bool{}
			for _, id := range adminIDs {
				adminSet[id] = true
			}
			items := make([]map[string]interface{}, 0, len(allUsers))
			for _, user := range allUsers {
				items = append(items, map[string]interface{}{
					"id": user.ID, "organization_id": user.OrganizationID, "email": user.Email,
					"username": user.Username, "enabled": user.Enabled, "platform_admin": adminSet[user.ID],
				})
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(items)
		})

		r.With(requirePlatformAdmin).Post("/platform-admins/{userID}", func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.FromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			userID, err := uuid.Parse(chi.URLParam(r, "userID"))
			if err != nil {
				http.Error(w, "invalid user id", http.StatusBadRequest)
				return
			}
			if userID == claims.UserID {
				http.Error(w, "self-grant is not allowed", http.StatusConflict)
				return
			}
			if err := users.GrantPlatformAdministrator(r.Context(), userID); err != nil {
				http.Error(w, "platform administrator grant failed", http.StatusBadRequest)
				return
			}
			logAuditInOrganization(r.Context(), auditLog, claims.OrganizationID, audit.ActorUser, claims.UserID.String(), audit.ActionPlatformAdminGranted, clientIP(r), map[string]interface{}{"target_user_id": userID.String()})
			w.WriteHeader(http.StatusNoContent)
		})

		r.With(requirePlatformAdmin).Delete("/platform-admins/{userID}", func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.FromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			userID, err := uuid.Parse(chi.URLParam(r, "userID"))
			if err != nil {
				http.Error(w, "invalid user id", http.StatusBadRequest)
				return
			}
			if userID == claims.UserID {
				http.Error(w, "self-revocation is not allowed", http.StatusConflict)
				return
			}
			if err := users.RevokePlatformAdministrator(r.Context(), userID); err != nil {
				http.Error(w, err.Error(), http.StatusConflict)
				return
			}
			logAuditInOrganization(r.Context(), auditLog, claims.OrganizationID, audit.ActorUser, claims.UserID.String(), audit.ActionPlatformAdminRevoked, clientIP(r), map[string]interface{}{"target_user_id": userID.String()})
			w.WriteHeader(http.StatusNoContent)
		})
	})
}
