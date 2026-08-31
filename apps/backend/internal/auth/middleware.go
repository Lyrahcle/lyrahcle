package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mlko01/lyrahcle/apps/backend/internal/dbscope"
)

type contextKey string

const claimsContextKey contextKey = "auth_claims"
const agentIDContextKey contextKey = "auth_agent_id"

type RevocationCheck func(ctx context.Context, claims *Claims) (bool, error)

func RequireRole(secret []byte, minimum Role, isRevoked ...RevocationCheck) func(http.Handler) http.Handler {
	return requireRole(secret, nil, minimum, isRevoked...)
}

func RequireRoleWithFallback(current, previous []byte, minimum Role, isRevoked ...RevocationCheck) func(http.Handler) http.Handler {
	return requireRole(current, previous, minimum, isRevoked...)
}

func requireRole(current, previous []byte, minimum Role, isRevoked ...RevocationCheck) func(http.Handler) http.Handler {
	var check RevocationCheck
	if len(isRevoked) > 0 {
		check = isRevoked[0]
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}
			tokenString := strings.TrimPrefix(header, "Bearer ")

			claims, err := ParseTokenWithFallback(current, previous, tokenString)
			if err != nil {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			if check != nil {
				revoked, err := check(r.Context(), claims)
				if err != nil {
					http.Error(w, "authorization temporarily unavailable", http.StatusServiceUnavailable)
					return
				}
				if revoked {
					http.Error(w, "token has been revoked", http.StatusUnauthorized)
					return
				}
			}

			if claims.PasswordChangeRequired && !passwordChangePathAllowed(r.URL.Path) {
				http.Error(w, "password change required", http.StatusForbidden)
				return
			}

			if !RoleAtLeast(claims.Role, minimum) {
				http.Error(w, "insufficient role", http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), claimsContextKey, claims)
			if claims.OrganizationID != uuid.Nil {
				ctx = dbscope.WithOrganization(ctx, claims.OrganizationID)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireAnyRoleWithFallback(current, previous []byte, roles []Role, isRevoked ...RevocationCheck) func(http.Handler) http.Handler {
	allowed := make(map[Role]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return requireRole(current, previous, RoleViewer, isRevoked...)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := FromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if _, ok := allowed[claims.Role]; !ok {
				http.Error(w, "insufficient role", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		}))
	}
}

func passwordChangePathAllowed(path string) bool {
	switch path {
	case "/api/v1/auth/me", "/api/v1/auth/change-password", "/api/v1/auth/logout":
		return true
	default:
		return false
	}
}

type AgentKeyLookup func(ctx context.Context, keyHash string) (agentID string, err error)
type AgentCertificateLookup func(ctx context.Context, fingerprint string) (agentID string, err error)

func hasPresentedAgentCredential(r *http.Request) bool {
	return (r.TLS != nil && len(r.TLS.PeerCertificates) > 0) || r.Header.Get("X-Agent-Key") != ""
}

// RequireAgentCredential authenticates machine-only endpoints with either an mTLS
// certificate or an agent API key. If a certificate is presented, it always takes
// precedence; a revoked certificate must never downgrade to an API key.
func RequireAgentCredential(keyLookup AgentKeyLookup, certificateLookup AgentCertificateLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
				fingerprint := sha256.Sum256(r.TLS.PeerCertificates[0].Raw)
				agentID, err := certificateLookup(r.Context(), hex.EncodeToString(fingerprint[:]))
				if err != nil {
					http.Error(w, "invalid or inactive agent certificate", http.StatusUnauthorized)
					return
				}
				ctx := context.WithValue(r.Context(), agentIDContextKey, agentID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			if agentKey := r.Header.Get("X-Agent-Key"); agentKey != "" {
				agentID, err := keyLookup(r.Context(), HashAgentKey(agentKey))
				if err != nil {
					http.Error(w, "invalid or revoked agent key", http.StatusUnauthorized)
					return
				}
				ctx := context.WithValue(r.Context(), agentIDContextKey, agentID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			http.Error(w, "missing agent credential", http.StatusUnauthorized)
		})
	}
}

func RequireAgentCredentialOrRoleWithFallback(current, previous []byte, minimum Role, keyLookup AgentKeyLookup, certificateLookup AgentCertificateLookup, isRevoked ...RevocationCheck) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if hasPresentedAgentCredential(r) {
				RequireAgentCredential(keyLookup, certificateLookup)(next).ServeHTTP(w, r)
				return
			}
			requireRole(current, previous, minimum, isRevoked...)(next).ServeHTTP(w, r)
		})
	}
}

func RequireAgentKeyOrRole(secret []byte, minimum Role, lookup AgentKeyLookup, isRevoked ...RevocationCheck) func(http.Handler) http.Handler {
	return requireAgentKeyOrRole(secret, nil, minimum, lookup, isRevoked...)
}

func RequireAgentKeyOrRoleWithFallback(current, previous []byte, minimum Role, lookup AgentKeyLookup, isRevoked ...RevocationCheck) func(http.Handler) http.Handler {
	return requireAgentKeyOrRole(current, previous, minimum, lookup, isRevoked...)
}

func requireAgentKeyOrRole(current, previous []byte, minimum Role, lookup AgentKeyLookup, isRevoked ...RevocationCheck) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if agentKey := r.Header.Get("X-Agent-Key"); agentKey != "" {
				agentID, err := lookup(r.Context(), HashAgentKey(agentKey))
				if err != nil {
					http.Error(w, "invalid or revoked agent key", http.StatusUnauthorized)
					return
				}
				ctx := context.WithValue(r.Context(), agentIDContextKey, agentID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			requireRole(current, previous, minimum, isRevoked...)(next).ServeHTTP(w, r)
		})
	}
}

func FromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*Claims)
	return claims, ok
}

func AgentIDFromContext(ctx context.Context) (string, bool) {
	agentID, ok := ctx.Value(agentIDContextKey).(string)
	return agentID, ok
}
