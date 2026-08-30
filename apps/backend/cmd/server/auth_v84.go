package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/mlko01/lyrahcle/apps/backend/internal/audit"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/mailer"
	"github.com/mlko01/lyrahcle/apps/backend/internal/metrics"
	"github.com/mlko01/lyrahcle/apps/backend/internal/sessionstore"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/tokenblocklist"
	"github.com/mlko01/lyrahcle/apps/backend/internal/validate"
	"github.com/mlko01/lyrahcle/apps/backend/internal/webauthnflow"
)

func loadWebAuthnUser(ctx context.Context, users *store.UserStore, credentials *store.WebAuthnStore, userID uuid.UUID) (webauthnflow.User, store.User, error) {
	user, err := users.GetByID(ctx, userID)
	if err != nil {
		return webauthnflow.User{}, store.User{}, err
	}
	handle, err := credentials.Handle(ctx, userID)
	if err != nil {
		return webauthnflow.User{}, store.User{}, err
	}
	creds, err := credentials.Credentials(ctx, userID)
	if err != nil {
		return webauthnflow.User{}, store.User{}, err
	}
	return webauthnflow.User{Handle: handle, Name: user.Email, DisplayName: user.Username, Credentials: creds}, user, nil
}

func registerWebAuthnRoutes(r chi.Router, cfg config.Config, handler *webauthn.WebAuthn, flows *webauthnflow.Store, credentials *store.WebAuthnStore, users *store.UserStore, sessions *sessionstore.Store, blocklist *tokenblocklist.Blocklist, auditLog *store.AuditStore, redisClient *redis.Client, isTokenRevoked auth.RevocationCheck, loginLimiter func(http.Handler) http.Handler) {
	r.With(loginLimiter).Post("/webauthn/login/start", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Identifier string `json:"identifier"`
			Password   string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		body.Identifier = strings.TrimSpace(strings.ToLower(body.Identifier))
		if len(body.Identifier) < 1 || len(body.Identifier) > 254 || len(body.Password) > 128 {
			metrics.AuthLoginsTotal.WithLabelValues("invalid_credentials").Inc()
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		user, err := users.GetByIdentifier(r.Context(), body.Identifier)
		if err != nil || !user.Enabled || !auth.CheckPassword(user.PasswordHash, body.Password) {
			metrics.AuthLoginsTotal.WithLabelValues("invalid_credentials").Inc()
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		waUser, _, err := loadWebAuthnUser(r.Context(), users, credentials, user.ID)
		if err != nil {
			http.Error(w, "authentication state unavailable", http.StatusServiceUnavailable)
			return
		}
		if len(waUser.Credentials) == 0 {
			http.Error(w, "security key is not configured", http.StatusConflict)
			return
		}
		assertion, session, err := handler.BeginLogin(waUser, webauthn.WithUserVerification(protocol.VerificationRequired))
		if err != nil {
			http.Error(w, "unable to start security-key authentication", http.StatusInternalServerError)
			return
		}
		flowID, err := flows.Create(r.Context(), webauthnflow.Ceremony{UserID: user.ID, Purpose: "login", Session: *session})
		if err != nil {
			http.Error(w, "security-key authentication unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"flow_id": flowID, "options": assertion})
	})

	r.With(loginLimiter).Post("/webauthn/login/finish", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			FlowID     string          `json:"flow_id"`
			Credential json.RawMessage `json:"credential"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Credential) == 0 {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		ceremony, err := flows.Consume(r.Context(), body.FlowID, "login")
		if err != nil {
			http.Error(w, "security-key ceremony expired or invalid", http.StatusUnauthorized)
			return
		}
		waUser, user, err := loadWebAuthnUser(r.Context(), users, credentials, ceremony.UserID)
		if err != nil || !user.Enabled {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		parsed, err := protocol.ParseCredentialRequestResponseBytes(body.Credential)
		if err != nil {
			http.Error(w, "invalid security-key response", http.StatusBadRequest)
			return
		}
		credential, err := handler.ValidateLogin(waUser, ceremony.Session, parsed)
		if err != nil {
			logAudit(r.Context(), auditLog, audit.ActorAnonymous, user.ID.String(), audit.ActionLoginFailed, clientIP(r), map[string]interface{}{"reason": "webauthn"})
			http.Error(w, "security-key verification failed", http.StatusUnauthorized)
			return
		}
		if credential.Authenticator.CloneWarning {
			logAudit(r.Context(), auditLog, audit.ActorUser, user.ID.String(), audit.ActionLoginFailed, clientIP(r), map[string]interface{}{"reason": "webauthn_clone_warning"})
			http.Error(w, "security-key verification failed", http.StatusUnauthorized)
			return
		}
		if err := credentials.UpdateCredential(r.Context(), user.ID, credential); err != nil {
			http.Error(w, "credential update failed", http.StatusInternalServerError)
			return
		}
		token, claims, err := auth.IssueOrganizationUserTokenWithState(cfg.JWTSecret, user.ID, user.OrganizationID, auth.Role(user.Role), user.MustChangePassword, user.AuthVersion, cfg.JWTAccessTTL)
		if err != nil {
			http.Error(w, "token issuance failed", http.StatusInternalServerError)
			return
		}
		if redisClient != nil {
			if err := sessions.Register(r.Context(), sessionstore.Session{ID: claims.ID, UserID: user.ID.String(), CreatedAt: claims.IssuedAt.Time.UTC(), ExpiresAt: claims.ExpiresAt.Time.UTC(), IP: clientIP(r), UserAgent: r.UserAgent()}); err != nil {
				http.Error(w, "session registration failed", http.StatusServiceUnavailable)
				return
			}
		}
		metrics.AuthLoginsTotal.WithLabelValues("success").Inc()
		logAudit(r.Context(), auditLog, audit.ActorUser, user.ID.String(), audit.ActionWebAuthnLogin, clientIP(r), nil)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"token": token, "role": user.Role, "must_change_password": user.MustChangePassword})
	})

	r.Route("/webauthn/credentials", func(r chi.Router) {
		r.Use(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked))

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			claims, _ := auth.FromContext(r.Context())
			list, err := credentials.List(r.Context(), claims.UserID)
			if err != nil {
				http.Error(w, "failed to list security keys", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(list)
		})

		r.Post("/register/start", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				Label    string `json:"label"`
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			body.Label = strings.TrimSpace(body.Label)
			if body.Label == "" || len(body.Label) > 80 {
				http.Error(w, "label must contain 1-80 characters", http.StatusBadRequest)
				return
			}
			claims, _ := auth.FromContext(r.Context())
			currentUser, err := users.GetByID(r.Context(), claims.UserID)
			if err != nil || !auth.CheckPassword(currentUser.PasswordHash, body.Password) {
				http.Error(w, "current password is required", http.StatusUnauthorized)
				return
			}
			waUser, _, err := loadWebAuthnUser(r.Context(), users, credentials, claims.UserID)
			if err != nil {
				http.Error(w, "unable to load security-key profile", http.StatusInternalServerError)
				return
			}
			creation, session, err := handler.BeginRegistration(waUser, webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{AuthenticatorAttachment: protocol.CrossPlatform, ResidentKey: protocol.ResidentKeyRequirementDiscouraged, UserVerification: protocol.VerificationRequired}))
			if err != nil {
				http.Error(w, "unable to start security-key registration", http.StatusInternalServerError)
				return
			}
			flowID, err := flows.Create(r.Context(), webauthnflow.Ceremony{UserID: claims.UserID, Purpose: "register", Label: body.Label, Session: *session})
			if err != nil {
				http.Error(w, "security-key registration unavailable", http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"flow_id": flowID, "options": creation})
		})

		r.Post("/register/finish", func(w http.ResponseWriter, r *http.Request) {
			var body struct {
				FlowID     string          `json:"flow_id"`
				Credential json.RawMessage `json:"credential"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Credential) == 0 {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			claims, _ := auth.FromContext(r.Context())
			ceremony, err := flows.Consume(r.Context(), body.FlowID, "register")
			if err != nil || ceremony.UserID != claims.UserID {
				http.Error(w, "security-key ceremony expired or invalid", http.StatusUnauthorized)
				return
			}
			waUser, _, err := loadWebAuthnUser(r.Context(), users, credentials, claims.UserID)
			if err != nil {
				http.Error(w, "unable to load security-key profile", http.StatusInternalServerError)
				return
			}
			parsed, err := protocol.ParseCredentialCreationResponseBytes(body.Credential)
			if err != nil {
				http.Error(w, "invalid security-key response", http.StatusBadRequest)
				return
			}
			credential, err := handler.CreateCredential(waUser, ceremony.Session, parsed)
			if err != nil {
				http.Error(w, "security-key registration failed", http.StatusUnauthorized)
				return
			}
			if err := credentials.Add(r.Context(), claims.UserID, ceremony.Label, credential); err != nil {
				http.Error(w, "security-key registration failed", http.StatusConflict)
				return
			}
			_ = blocklist.RevokeUser(r.Context(), claims.UserID.String(), time.Now(), cfg.JWTAccessTTL+time.Minute)
			if redisClient != nil {
				_ = sessions.DeleteUser(r.Context(), claims.UserID.String())
			}
			logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionWebAuthnRegistered, clientIP(r), map[string]interface{}{"label": ceremony.Label})
			w.WriteHeader(http.StatusNoContent)
		})

		r.Delete("/{credentialID}", func(w http.ResponseWriter, r *http.Request) {
			claims, _ := auth.FromContext(r.Context())
			var body struct {
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			currentUser, err := users.GetByID(r.Context(), claims.UserID)
			if err != nil || !auth.CheckPassword(currentUser.PasswordHash, body.Password) {
				http.Error(w, "current password is required", http.StatusUnauthorized)
				return
			}
			credentialID, err := uuid.Parse(chi.URLParam(r, "credentialID"))
			if err != nil {
				http.Error(w, "credentialID must be a valid UUID", http.StatusBadRequest)
				return
			}
			if err := credentials.Delete(r.Context(), claims.UserID, credentialID); err != nil {
				http.Error(w, "security key not found", http.StatusNotFound)
				return
			}
			_ = blocklist.RevokeUser(r.Context(), claims.UserID.String(), time.Now(), cfg.JWTAccessTTL+time.Minute)
			if redisClient != nil {
				_ = sessions.DeleteUser(r.Context(), claims.UserID.String())
			}
			logAudit(r.Context(), auditLog, audit.ActorUser, claims.UserID.String(), audit.ActionWebAuthnRemoved, clientIP(r), map[string]interface{}{"credential_id": credentialID.String()})
			w.WriteHeader(http.StatusNoContent)
		})
	})
}

func registerPasswordRecoveryRoutes(r chi.Router, cfg config.Config, users *store.UserStore, resets *store.PasswordResetStore, credentials *store.WebAuthnStore, sessions *sessionstore.Store, blocklist *tokenblocklist.Blocklist, auditLog *store.AuditStore, mail *mailer.Client, redisClient *redis.Client, limiter func(http.Handler) http.Handler) {
	r.With(limiter).Post("/password-reset/request", func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		defer func() {
			if remaining := 180*time.Millisecond - time.Since(started); remaining > 0 {
				time.Sleep(remaining)
			}
		}()
		var body struct {
			Identifier string `json:"identifier"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		body.Identifier = strings.TrimSpace(strings.ToLower(body.Identifier))
		if len(body.Identifier) < 1 || len(body.Identifier) > 254 {
			json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
			return
		}
		if !cfg.PasswordRecoveryEnabled {
			json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
			return
		}
		user, err := users.GetByIdentifier(r.Context(), body.Identifier)
		if err != nil || !user.Enabled || !strings.Contains(user.Email, "@") {
			json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
			return
		}
		tokenBytes := make([]byte, 32)
		if _, err := rand.Read(tokenBytes); err != nil {
			json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
			return
		}
		token := base64.RawURLEncoding.EncodeToString(tokenBytes)
		sum := sha256.Sum256(tokenBytes)
		if err := resets.Create(r.Context(), user.ID, sum[:], time.Now().Add(15*time.Minute)); err != nil {
			json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
			return
		}
		resetURL, err := url.Parse(cfg.PasswordResetBaseURL)
		if err == nil {
			query := resetURL.Query()
			query.Set("token", token)
			resetURL.RawQuery = query.Encode()
			email := user.Email
			userID := user.ID.String()
			ip := clientIP(r)
			link := resetURL.String()
			go func() {
				if err := mail.SendPasswordReset(email, link); err != nil {
					return
				}
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				logAudit(ctx, auditLog, audit.ActorAnonymous, userID, audit.ActionPasswordResetRequested, ip, nil)
			}()
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
	})

	r.With(limiter).Post("/password-reset/complete", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Token       string `json:"token"`
			NewPassword string `json:"new_password"`
			TOTPCode    string `json:"totp_code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if err := validate.Password(body.NewPassword); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(body.Token))
		if err != nil || len(raw) != 32 {
			http.Error(w, "invalid or expired reset token", http.StatusUnauthorized)
			return
		}
		sum := sha256.Sum256(raw)
		userID, err := resets.Lookup(r.Context(), sum[:])
		if err != nil {
			http.Error(w, "invalid or expired reset token", http.StatusUnauthorized)
			return
		}
		user, err := users.GetByID(r.Context(), userID)
		if err != nil || !user.Enabled {
			http.Error(w, "invalid or expired reset token", http.StatusUnauthorized)
			return
		}
		if user.TOTPEnabled {
			if !auth.VerifyTOTP(user.TOTPSecret, body.TOTPCode) {
				http.Error(w, "two-factor verification required", http.StatusUnauthorized)
				return
			}
		} else {
			count, err := credentials.Count(r.Context(), user.ID)
			if err != nil {
				http.Error(w, "authentication state unavailable", http.StatusServiceUnavailable)
				return
			}
			if count > 0 {
				http.Error(w, "security-key protected accounts require administrator-assisted recovery", http.StatusConflict)
				return
			}
		}
		if auth.CheckPassword(user.PasswordHash, body.NewPassword) {
			http.Error(w, "new password must differ from the current password", http.StatusBadRequest)
			return
		}
		hash, err := auth.HashPassword(body.NewPassword)
		if err != nil {
			http.Error(w, "password reset failed", http.StatusInternalServerError)
			return
		}
		if _, err := resets.ConsumeAndUpdatePassword(r.Context(), sum[:], hash); err != nil {
			http.Error(w, "invalid or expired reset token", http.StatusUnauthorized)
			return
		}
		_ = blocklist.RevokeUser(r.Context(), user.ID.String(), time.Now(), cfg.JWTAccessTTL+time.Minute)
		if redisClient != nil {
			_ = sessions.DeleteUser(r.Context(), user.ID.String())
		}
		logAudit(r.Context(), auditLog, audit.ActorUser, user.ID.String(), audit.ActionPasswordResetCompleted, clientIP(r), nil)
		w.WriteHeader(http.StatusNoContent)
	})
}

func webAuthnCredentialCount(ctx context.Context, credentials *store.WebAuthnStore, userID uuid.UUID) (int, error) {
	return credentials.Count(ctx, userID)
}

func buildWebAuthnHandler(cfg config.Config) *webauthn.WebAuthn {
	handler, err := webauthn.New(&webauthn.Config{RPDisplayName: "Lyrahcle", RPID: cfg.WebAuthnRPID, RPOrigins: cfg.WebAuthnOrigins})
	if err != nil {
		panic(err)
	}
	return handler
}
