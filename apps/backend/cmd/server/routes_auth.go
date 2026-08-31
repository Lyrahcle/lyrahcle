package main

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/metrics"
	"github.com/mlko01/lyrahcle/apps/backend/internal/sessionstore"
	"github.com/mlko01/lyrahcle/apps/backend/internal/validate"
	"net/http"
	"strings"
	"time"
)

func (a *apiRoutes) registerAuthenticationRoutes(r chi.Router) {
	auditLog := a.auditLog
	blocklist := a.blocklist
	cfg := a.cfg
	isTokenRevoked := a.isTokenRevoked
	loginLimiter := a.loginLimiter
	mailClient := a.mailClient
	passwordResets := a.passwordResets
	redisClient := a.redisClient
	sessions := a.sessions
	users := a.users
	webAuthnCredentials := a.webAuthnCredentials
	webAuthnFlows := a.webAuthnFlows
	webAuthnHandler := a.webAuthnHandler

	r.Route("/auth", func(r chi.Router) {
		r.With(loginLimiter).Post("/login", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Identifier string `json:"identifier"`
				Password   string `json:"password"`
				TOTPCode   string `json:"totp_code"`
			}
			if err := decodeStrictJSON(r, &body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			body.Identifier = strings.TrimSpace(strings.ToLower(body.Identifier))
			if body.Identifier == "" || body.Password == "" {
				http.Error(w, "invalid credentials", http.StatusUnauthorized)
				return
			}

			user, err := users.GetByIdentifier(r.Context(), body.Identifier)
			if err != nil || !auth.CheckPassword(user.PasswordHash, body.Password) || !user.Enabled {
				metrics.AuthLoginsTotal.WithLabelValues("invalid_credentials").Inc()
				logAudit(r.Context(), auditLog, audit.ActorAnonymous, body.Identifier, audit.ActionLoginFailed, clientIP(r), nil)
				http.Error(w, "invalid credentials", http.StatusUnauthorized)
				return
			}

			webAuthnCount, err := webAuthnCredentialCount(r.Context(), webAuthnCredentials, user.ID)
			if err != nil {
				http.Error(w, "authentication state unavailable", http.StatusServiceUnavailable)
				return
			}
			if webAuthnCount > 0 {
				metrics.AuthLoginsTotal.WithLabelValues("webauthn_required").Inc()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "webauthn_required"})
				return
			}

			if user.TOTPEnabled {
				if body.TOTPCode == "" {
					metrics.AuthLoginsTotal.WithLabelValues("totp_required").Inc()
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					json.NewEncoder(w).Encode(map[string]string{"error": "totp_required"})
					return
				}
				if !auth.VerifyTOTP(user.TOTPSecret, body.TOTPCode) {
					metrics.AuthLoginsTotal.WithLabelValues("invalid_totp").Inc()
					logAudit(r.Context(), auditLog, audit.ActorAnonymous, body.Identifier, audit.ActionLoginFailed, clientIP(r), map[string]interface{}{"reason": "invalid_totp"})
					http.Error(w, "invalid two-factor code", http.StatusUnauthorized)
					return
				}
			}

			token, tokenClaims, err := auth.IssueOrganizationUserTokenWithState(cfg.JWTSecret, user.ID, user.OrganizationID, auth.Role(user.Role), user.MustChangePassword, user.AuthVersion, cfg.JWTAccessTTL)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if redisClient != nil {
				if err := sessions.Register(r.Context(), sessionstore.Session{
					ID:        tokenClaims.ID,
					UserID:    user.ID.String(),
					CreatedAt: tokenClaims.IssuedAt.Time.UTC(),
					ExpiresAt: tokenClaims.ExpiresAt.Time.UTC(),
					IP:        clientIP(r),
					UserAgent: r.UserAgent(),
				}); err != nil {
					http.Error(w, "session registration failed", http.StatusServiceUnavailable)
					return
				}
			}

			metrics.AuthLoginsTotal.WithLabelValues("success").Inc()
			logAudit(r.Context(), auditLog, audit.ActorUser, user.ID.String(), audit.ActionLogin, clientIP(r), map[string]interface{}{"email": user.Email})

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"token": token, "role": user.Role, "must_change_password": user.MustChangePassword})
		})

		registerWebAuthnRoutes(r, cfg, webAuthnHandler, webAuthnFlows, webAuthnCredentials, users, sessions, blocklist, auditLog, redisClient, isTokenRevoked, loginLimiter)
		registerPasswordRecoveryRoutes(r, cfg, users, passwordResets, webAuthnCredentials, sessions, blocklist, auditLog, mailClient, redisClient, loginLimiter)

		r.Route("/totp", func(r chi.Router) {
			r.Use(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked))

			r.Post("/setup", func(w http.ResponseWriter, r *http.Request) {
				claims, ok := auth.FromContext(r.Context())
				if !ok {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				user, err := users.GetByID(r.Context(), claims.UserID)
				if err != nil {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				if user.TOTPEnabled {
					http.Error(w, "two-factor authentication is already enabled", http.StatusConflict)
					return
				}

				secret, err := auth.GenerateTOTPSecret()
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if err := users.SetPendingTOTPSecret(r.Context(), user.ID, secret); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				uri := auth.TOTPProvisioningURI("Lyrahcle", user.Email, secret)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{"provider": "google_authenticator", "issuer": "Lyrahcle", "account_name": user.Email, "secret": secret, "otpauth_url": uri})
			})

			r.Post("/enable", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Code string `json:"code"`
				}
				if err := decodeStrictJSON(r, &body); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				claims, ok := auth.FromContext(r.Context())
				if !ok {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				user, err := users.GetByID(r.Context(), claims.UserID)
				if err != nil {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				if user.TOTPSecret == "" {
					http.Error(w, "call /auth/totp/setup first", http.StatusBadRequest)
					return
				}
				if !auth.VerifyTOTP(user.TOTPSecret, body.Code) {
					http.Error(w, "invalid two-factor code", http.StatusUnauthorized)
					return
				}
				if err := blocklist.RevokeUser(r.Context(), user.ID.String(), time.Now(), cfg.JWTAccessTTL+time.Minute); err != nil {
					http.Error(w, "authorization state update failed", http.StatusServiceUnavailable)
					return
				}
				if redisClient != nil {
					if err := sessions.DeleteUser(r.Context(), user.ID.String()); err != nil {
						http.Error(w, "session revocation failed", http.StatusServiceUnavailable)
						return
					}
				}
				if err := users.EnableTOTP(r.Context(), user.ID); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				logAudit(r.Context(), auditLog, audit.ActorUser, user.ID.String(), audit.ActionTOTPEnabled, clientIP(r), nil)
				w.WriteHeader(http.StatusNoContent)
			})

			r.Post("/disable", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Password string `json:"password"`
				}
				if err := decodeStrictJSON(r, &body); err != nil {
					http.Error(w, "invalid JSON body", http.StatusBadRequest)
					return
				}
				claims, ok := auth.FromContext(r.Context())
				if !ok {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				user, err := users.GetByID(r.Context(), claims.UserID)
				if err != nil {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				if !auth.CheckPassword(user.PasswordHash, body.Password) {
					http.Error(w, "invalid password", http.StatusUnauthorized)
					return
				}
				if err := blocklist.RevokeUser(r.Context(), user.ID.String(), time.Now(), cfg.JWTAccessTTL+time.Minute); err != nil {
					http.Error(w, "authorization state update failed", http.StatusServiceUnavailable)
					return
				}
				if redisClient != nil {
					if err := sessions.DeleteUser(r.Context(), user.ID.String()); err != nil {
						http.Error(w, "session revocation failed", http.StatusServiceUnavailable)
						return
					}
				}
				if err := users.DisableTOTP(r.Context(), user.ID); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				logAudit(r.Context(), auditLog, audit.ActorUser, user.ID.String(), audit.ActionTOTPDisabled, clientIP(r), nil)
				w.WriteHeader(http.StatusNoContent)
			})
		})

		r.Route("/sessions", func(r chi.Router) {
			r.Use(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked))

			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				claims, ok := auth.FromContext(r.Context())
				if !ok {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				if redisClient == nil {
					http.Error(w, "session management unavailable", http.StatusServiceUnavailable)
					return
				}
				list, err := sessions.List(r.Context(), claims.UserID.String())
				if err != nil {
					http.Error(w, "session management unavailable", http.StatusServiceUnavailable)
					return
				}
				type sessionResponse struct {
					ID        string    `json:"id"`
					CreatedAt time.Time `json:"created_at"`
					ExpiresAt time.Time `json:"expires_at"`
					IP        string    `json:"ip"`
					UserAgent string    `json:"user_agent"`
					Current   bool      `json:"current"`
				}
				out := make([]sessionResponse, 0, len(list))
				for _, item := range list {
					out = append(out, sessionResponse{ID: item.ID, CreatedAt: item.CreatedAt, ExpiresAt: item.ExpiresAt, IP: item.IP, UserAgent: item.UserAgent, Current: item.ID == claims.ID})
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(out)
			})

			r.Delete("/{sessionID}", func(w http.ResponseWriter, r *http.Request) {
				claims, ok := auth.FromContext(r.Context())
				if !ok {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				if redisClient == nil {
					http.Error(w, "session management unavailable", http.StatusServiceUnavailable)
					return
				}
				sessionID := strings.TrimSpace(chi.URLParam(r, "sessionID"))
				if _, err := uuid.Parse(sessionID); err != nil {
					http.Error(w, "sessionID must be a valid UUID", http.StatusBadRequest)
					return
				}
				item, err := sessions.Get(r.Context(), sessionID)
				if err != nil || item.UserID != claims.UserID.String() {
					http.Error(w, "session not found", http.StatusNotFound)
					return
				}
				remaining := time.Until(item.ExpiresAt)
				if remaining > 0 {
					if err := blocklist.Revoke(r.Context(), item.ID, remaining); err != nil {
						http.Error(w, "session revocation failed", http.StatusServiceUnavailable)
						return
					}
				}
				if err := sessions.Delete(r.Context(), claims.UserID.String(), item.ID); err != nil {
					http.Error(w, "session revocation failed", http.StatusServiceUnavailable)
					return
				}
				logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionSessionRevoked, clientIP(r), map[string]interface{}{"session_id": item.ID, "current": item.ID == claims.ID})
				w.WriteHeader(http.StatusNoContent)
			})

			r.Delete("/", func(w http.ResponseWriter, r *http.Request) {
				claims, ok := auth.FromContext(r.Context())
				if !ok {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				if redisClient == nil {
					http.Error(w, "session management unavailable", http.StatusServiceUnavailable)
					return
				}
				if err := blocklist.RevokeUser(r.Context(), claims.UserID.String(), time.Now(), cfg.JWTAccessTTL+time.Minute); err != nil {
					http.Error(w, "session revocation failed", http.StatusServiceUnavailable)
					return
				}
				if err := sessions.DeleteUser(r.Context(), claims.UserID.String()); err != nil {
					http.Error(w, "session revocation failed", http.StatusServiceUnavailable)
					return
				}
				logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionSessionsRevoked, clientIP(r), nil)
				w.WriteHeader(http.StatusNoContent)
			})
		})

		r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Get("/me", func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.FromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			user, err := users.GetByID(r.Context(), claims.UserID)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":                   user.ID.String(),
				"email":                user.Email,
				"username":             user.Username,
				"role":                 user.Role,
				"totp_enabled":         user.TOTPEnabled,
				"must_change_password": user.MustChangePassword,
				"enabled":              user.Enabled,
			})
		})

		r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Post("/logout", func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.FromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			remaining := time.Until(claims.ExpiresAt.Time)
			if err := blocklist.Revoke(r.Context(), claims.ID, remaining); err != nil {
				http.Error(w, "logout could not revoke the session", http.StatusServiceUnavailable)
				return
			}
			if redisClient != nil {
				if err := sessions.Delete(r.Context(), claims.UserID.String(), claims.ID); err != nil {
					http.Error(w, "logout could not remove the session", http.StatusServiceUnavailable)
					return
				}
			}

			logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionLogout, clientIP(r), nil)
			w.WriteHeader(http.StatusNoContent)
		})

		r.With(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked)).Post("/change-password", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				CurrentPassword string `json:"current_password"`
				NewPassword     string `json:"new_password"`
			}
			if err := decodeStrictJSON(r, &body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}

			claims, ok := auth.FromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			user, err := users.GetByID(r.Context(), claims.UserID)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if !auth.CheckPassword(user.PasswordHash, body.CurrentPassword) {
				http.Error(w, "current password is incorrect", http.StatusUnauthorized)
				return
			}
			if err := validate.Password(body.NewPassword); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if auth.CheckPassword(user.PasswordHash, body.NewPassword) {
				http.Error(w, "new password must differ from the current password", http.StatusBadRequest)
				return
			}

			newHash, err := auth.HashPassword(body.NewPassword)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := blocklist.RevokeUser(r.Context(), user.ID.String(), time.Now(), cfg.JWTAccessTTL+time.Minute); err != nil {
				http.Error(w, "authorization state update failed", http.StatusServiceUnavailable)
				return
			}
			if redisClient != nil {
				if err := sessions.DeleteUser(r.Context(), user.ID.String()); err != nil {
					http.Error(w, "session revocation failed", http.StatusServiceUnavailable)
					return
				}
			}
			if err := users.UpdatePassword(r.Context(), user.ID, newHash); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			logAudit(r.Context(), auditLog, audit.ActorUser, user.ID.String(), audit.ActionPasswordChanged, clientIP(r), map[string]interface{}{"email": user.Email})

			w.WriteHeader(http.StatusNoContent)
		})
	})
}
