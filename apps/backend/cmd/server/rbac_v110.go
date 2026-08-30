package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
)

func registerRBACRoutes(r chi.Router, cfg config.Config, rbac *store.RBACStore, auditLog *store.AuditStore, isTokenRevoked func(context.Context, *auth.Claims) (bool, error)) {
	r.Route("/rbac", func(r chi.Router) {
		r.Use(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked))

		r.Get("/permissions", func(w http.ResponseWriter, r *http.Request) {
			if !requireRequestPermission(w, r, rbac, "rbac.read") {
				return
			}
			items, err := rbac.ListPermissions(r.Context())
			if err != nil {
				http.Error(w, "permissions unavailable", http.StatusInternalServerError)
				return
			}
			writeRBACJSON(w, http.StatusOK, items)
		})

		r.Get("/roles", func(w http.ResponseWriter, r *http.Request) {
			if !requireRequestPermission(w, r, rbac, "rbac.read") {
				return
			}
			orgID, ok := requestOrganizationID(w, r)
			if !ok {
				return
			}
			items, err := rbac.ListRoles(r.Context(), orgID)
			if err != nil {
				http.Error(w, "roles unavailable", http.StatusInternalServerError)
				return
			}
			writeRBACJSON(w, http.StatusOK, items)
		})

		r.Post("/roles", func(w http.ResponseWriter, r *http.Request) {
			if !requireRequestPermission(w, r, rbac, "rbac.manage") {
				return
			}
			orgID, ok := requestOrganizationID(w, r)
			if !ok {
				return
			}
			var body struct {
				Name        string   `json:"name"`
				Description string   `json:"description"`
				Permissions []string `json:"permissions"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid json", http.StatusBadRequest)
				return
			}
			if strings.TrimSpace(body.Name) == "" {
				http.Error(w, "name is required", http.StatusBadRequest)
				return
			}
			item, err := rbac.CreateRole(r.Context(), orgID, body.Name, body.Description, body.Permissions)
			if errors.Is(err, store.ErrOrganizationRoleNameTaken) {
				http.Error(w, "role name already exists", http.StatusConflict)
				return
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if claims, ok := auth.FromContext(r.Context()); ok {
				logAuditInOrganization(r.Context(), auditLog, orgID, audit.ActorUser, claims.UserID.String(), audit.ActionRBACRoleCreated, clientIP(r), map[string]interface{}{
					"organization_id": orgID.String(),
					"role_id":         item.ID.String(),
					"role_name":       item.Name,
					"permissions":     body.Permissions,
				})
			}
			writeRBACJSON(w, http.StatusCreated, item)
		})

		r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.FromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			perms, err := rbac.EffectivePermissions(r.Context(), claims.UserID, claims.OrganizationID)
			if err != nil {
				http.Error(w, "permissions unavailable", http.StatusInternalServerError)
				return
			}
			writeRBACJSON(w, http.StatusOK, map[string]interface{}{"organization_id": claims.OrganizationID, "permissions": perms})
		})

		r.Get("/users/{userID}/roles", func(w http.ResponseWriter, r *http.Request) {
			if !requireRequestPermission(w, r, rbac, "rbac.read") {
				return
			}
			orgID, ok := requestOrganizationID(w, r)
			if !ok {
				return
			}
			userID, err := uuid.Parse(chi.URLParam(r, "userID"))
			if err != nil {
				http.Error(w, "invalid user id", http.StatusBadRequest)
				return
			}
			items, err := rbac.AssignedRoles(r.Context(), orgID, userID)
			if err != nil {
				http.Error(w, "role assignments unavailable", http.StatusInternalServerError)
				return
			}
			writeRBACJSON(w, http.StatusOK, items)
		})

		r.Post("/users/{userID}/roles/{roleID}", func(w http.ResponseWriter, r *http.Request) {
			if !requireRequestPermission(w, r, rbac, "rbac.manage") {
				return
			}
			orgID, ok := requestOrganizationID(w, r)
			if !ok {
				return
			}
			userID, err := uuid.Parse(chi.URLParam(r, "userID"))
			if err != nil {
				http.Error(w, "invalid user id", http.StatusBadRequest)
				return
			}
			roleID, err := uuid.Parse(chi.URLParam(r, "roleID"))
			if err != nil {
				http.Error(w, "invalid role id", http.StatusBadRequest)
				return
			}
			if err := rbac.AssignRole(r.Context(), orgID, userID, roleID); err != nil {
				http.Error(w, "role assignment failed", http.StatusBadRequest)
				return
			}
			if claims, ok := auth.FromContext(r.Context()); ok {
				logAuditInOrganization(r.Context(), auditLog, orgID, audit.ActorUser, claims.UserID.String(), audit.ActionRBACRoleAssigned, clientIP(r), map[string]interface{}{
					"organization_id": orgID.String(),
					"target_user_id":  userID.String(),
					"role_id":         roleID.String(),
				})
			}
			w.WriteHeader(http.StatusNoContent)
		})

		r.Delete("/users/{userID}/roles/{roleID}", func(w http.ResponseWriter, r *http.Request) {
			if !requireRequestPermission(w, r, rbac, "rbac.manage") {
				return
			}
			orgID, ok := requestOrganizationID(w, r)
			if !ok {
				return
			}
			userID, err := uuid.Parse(chi.URLParam(r, "userID"))
			if err != nil {
				http.Error(w, "invalid user id", http.StatusBadRequest)
				return
			}
			roleID, err := uuid.Parse(chi.URLParam(r, "roleID"))
			if err != nil {
				http.Error(w, "invalid role id", http.StatusBadRequest)
				return
			}
			if err := rbac.RemoveRole(r.Context(), orgID, userID, roleID); err != nil {
				http.Error(w, "role assignment not found", http.StatusNotFound)
				return
			}
			if claims, ok := auth.FromContext(r.Context()); ok {
				logAuditInOrganization(r.Context(), auditLog, orgID, audit.ActorUser, claims.UserID.String(), audit.ActionRBACRoleRemoved, clientIP(r), map[string]interface{}{
					"organization_id": orgID.String(),
					"target_user_id":  userID.String(),
					"role_id":         roleID.String(),
				})
			}
			w.WriteHeader(http.StatusNoContent)
		})
	})
}

func writeRBACJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
