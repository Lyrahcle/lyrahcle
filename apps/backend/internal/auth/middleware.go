package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const claimsContextKey contextKey = "auth_claims"
const agentIDContextKey contextKey = "auth_agent_id"

// RevocationCheck reports whether the token identified by jti has been
// revoked (logout, password change) before its natural expiry.
type RevocationCheck func(ctx context.Context, jti string) bool

// RequireRole authenticates a bearer JWT and requires at least the given
// role. isRevoked is optional (pass nil, or omit it entirely — both
// existing call sites and new ones work): when provided, a token whose jti
// is revoked is rejected even though its signature and expiry are still
// valid.
func RequireRole(secret []byte, minimum Role, isRevoked ...RevocationCheck) func(http.Handler) http.Handler {
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

			claims, err := ParseToken(secret, tokenString)
			if err != nil {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			if check != nil && check(r.Context(), claims.ID) {
				http.Error(w, "token has been revoked", http.StatusUnauthorized)
				return
			}

			if !RoleAtLeast(claims.Role, minimum) {
				http.Error(w, "insufficient role", http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), claimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type AgentKeyLookup func(ctx context.Context, keyHash string) (agentID string, err error)

func RequireAgentKeyOrRole(secret []byte, minimum Role, lookup AgentKeyLookup, isRevoked ...RevocationCheck) func(http.Handler) http.Handler {
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

			RequireRole(secret, minimum, isRevoked...)(next).ServeHTTP(w, r)
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
