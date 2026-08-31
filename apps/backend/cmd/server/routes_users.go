package main

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/validate"
	"net/http"
	"strings"
	"time"
)

func (a *apiRoutes) registerUserRoutes(r chi.Router) {
	auditLog := a.auditLog
	blocklist := a.blocklist
	cfg := a.cfg
	isTokenRevoked := a.isTokenRevoked
	rbac := a.rbac
	redisClient := a.redisClient
	sessions := a.sessions
	users := a.users

	r.Route("/users", func(r chi.Router) {
		r.Use(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked))
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				claims, ok := auth.FromContext(r.Context())
				if !ok || claims.OrganizationID == uuid.Nil {
					http.Error(w, "organization context required", http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
			})
		})

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			claims, _ := auth.FromContext(r.Context())
			if !requireRequestPermission(w, r, rbac, "users.read") {
				return
			}
			list, err := users.ListByOrganization(r.Context(), claims.OrganizationID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(list)
		})

		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			if !requireRequestPermission(w, r, rbac, "users.manage") {
				return
			}
			var body struct {
				Email    string `json:"email"`
				Password string `json:"password"`
				Role     string `json:"role"`
			}
			if err := decodeStrictJSON(r, &body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			body.Email = strings.TrimSpace(strings.ToLower(body.Email))
			if err := validate.Email(body.Email); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := validate.Password(body.Password); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if body.Role == "" {
				body.Role = string(auth.RoleViewer)
			}
			if !auth.ValidRole(body.Role) {
				http.Error(w, "role must be one of: viewer, auditor, analyst, operator, admin", http.StatusBadRequest)
				return
			}

			hash, err := auth.HashPassword(body.Password)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			claims, _ := auth.FromContext(r.Context())
			user, err := users.CreateInOrganization(r.Context(), claims.OrganizationID, body.Email, hash, body.Role)
			if err != nil {
				if err == store.ErrEmailTaken {
					http.Error(w, "email already registered", http.StatusConflict)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if claims, ok := auth.FromContext(r.Context()); ok {
				logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionUserCreated, clientIP(r), map[string]interface{}{"organization_id": claims.OrganizationID.String(), "target_email": user.Email, "role": user.Role})
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{"id": user.ID.String(), "email": user.Email, "username": user.Username, "role": user.Role, "enabled": user.Enabled, "must_change_password": user.MustChangePassword})
		})

		r.Patch("/{userID}/role", func(w http.ResponseWriter, r *http.Request) {
			if !requireRequestPermission(w, r, rbac, "users.manage") {
				return
			}
			userID, err := uuid.Parse(chi.URLParam(r, "userID"))
			if err != nil {
				http.Error(w, "userID must be a valid UUID", http.StatusBadRequest)
				return
			}

			var body struct {
				Role string `json:"role"`
			}
			if err := decodeStrictJSON(r, &body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			if !auth.ValidRole(body.Role) {
				http.Error(w, "role must be one of: viewer, auditor, analyst, operator, admin", http.StatusBadRequest)
				return
			}

			claims, ok := auth.FromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			target, err := users.GetByIDInOrganization(r.Context(), claims.OrganizationID, userID)
			if err != nil {
				if err == store.ErrUserNotFound {
					http.Error(w, "user not found", http.StatusNotFound)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if target.Role == string(auth.RoleAdmin) && target.Enabled && body.Role != string(auth.RoleAdmin) {
				adminCount, err := users.CountAdminsInOrganization(r.Context(), claims.OrganizationID)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if adminCount <= 1 {
					http.Error(w, "cannot demote the last remaining admin", http.StatusConflict)
					return
				}
			}

			if err := blocklist.RevokeUser(r.Context(), userID.String(), time.Now(), cfg.JWTAccessTTL+time.Minute); err != nil {
				http.Error(w, "authorization state update failed", http.StatusServiceUnavailable)
				return
			}
			if redisClient != nil {
				if err := sessions.DeleteUser(r.Context(), userID.String()); err != nil {
					http.Error(w, "session revocation failed", http.StatusServiceUnavailable)
					return
				}
			}

			updated, err := users.UpdateRoleInOrganization(r.Context(), claims.OrganizationID, userID, body.Role)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if claims, ok := auth.FromContext(r.Context()); ok {
				logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionRoleChanged, clientIP(r), map[string]interface{}{"organization_id": claims.OrganizationID.String(), "target_user_id": updated.ID.String(), "target_email": updated.Email, "new_role": updated.Role})
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"id": updated.ID.String(), "email": updated.Email, "role": updated.Role, "enabled": updated.Enabled})
		})

		r.Patch("/{userID}/status", func(w http.ResponseWriter, r *http.Request) {
			if !requireRequestPermission(w, r, rbac, "users.manage") {
				return
			}
			userID, err := uuid.Parse(chi.URLParam(r, "userID"))
			if err != nil {
				http.Error(w, "userID must be a valid UUID", http.StatusBadRequest)
				return
			}
			claims, ok := auth.FromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			var body struct {
				Enabled bool `json:"enabled"`
			}
			if err := decodeStrictJSON(r, &body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			target, err := users.GetByIDInOrganization(r.Context(), claims.OrganizationID, userID)
			if err != nil {
				http.Error(w, "user not found", http.StatusNotFound)
				return
			}
			if claims.UserID == userID && !body.Enabled {
				http.Error(w, "cannot disable the current user", http.StatusConflict)
				return
			}
			if target.Role == string(auth.RoleAdmin) && target.Enabled && !body.Enabled {
				count, err := users.CountAdminsInOrganization(r.Context(), claims.OrganizationID)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if count <= 1 {
					http.Error(w, "cannot disable the last remaining admin", http.StatusConflict)
					return
				}
			}
			if !body.Enabled {
				if err := blocklist.RevokeUser(r.Context(), userID.String(), time.Now(), cfg.JWTAccessTTL+time.Minute); err != nil {
					http.Error(w, "authorization state update failed", http.StatusServiceUnavailable)
					return
				}
				if redisClient != nil {
					if err := sessions.DeleteUser(r.Context(), userID.String()); err != nil {
						http.Error(w, "session revocation failed", http.StatusServiceUnavailable)
						return
					}
				}
			}
			updated, err := users.SetEnabledInOrganization(r.Context(), claims.OrganizationID, userID, body.Enabled)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionUserStatusChanged, clientIP(r), map[string]interface{}{"organization_id": claims.OrganizationID.String(), "target_user_id": userID.String(), "enabled": body.Enabled})
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(updated)
		})

		r.Delete("/{userID}", func(w http.ResponseWriter, r *http.Request) {
			if !requireRequestPermission(w, r, rbac, "users.manage") {
				return
			}
			userID, err := uuid.Parse(chi.URLParam(r, "userID"))
			if err != nil {
				http.Error(w, "userID must be a valid UUID", http.StatusBadRequest)
				return
			}
			claims, ok := auth.FromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if claims.UserID == userID {
				http.Error(w, "cannot delete the current user", http.StatusConflict)
				return
			}
			target, err := users.GetByIDInOrganization(r.Context(), claims.OrganizationID, userID)
			if err != nil {
				http.Error(w, "user not found", http.StatusNotFound)
				return
			}
			if target.Role == string(auth.RoleAdmin) && target.Enabled {
				count, err := users.CountAdminsInOrganization(r.Context(), claims.OrganizationID)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if count <= 1 {
					http.Error(w, "cannot delete the last remaining admin", http.StatusConflict)
					return
				}
			}
			if err := blocklist.RevokeUser(r.Context(), userID.String(), time.Now(), cfg.JWTAccessTTL+time.Minute); err != nil {
				http.Error(w, "authorization state update failed", http.StatusServiceUnavailable)
				return
			}
			if redisClient != nil {
				if err := sessions.DeleteUser(r.Context(), userID.String()); err != nil {
					http.Error(w, "session revocation failed", http.StatusServiceUnavailable)
					return
				}
			}
			if err := users.SoftDeleteInOrganization(r.Context(), claims.OrganizationID, userID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionUserDeleted, clientIP(r), map[string]interface{}{"organization_id": claims.OrganizationID.String(), "target_user_id": userID.String(), "target_email": target.Email})
			w.WriteHeader(http.StatusNoContent)
		})
	})
}
