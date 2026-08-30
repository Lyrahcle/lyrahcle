package auth

import (
	"context"
	"net/http"
)

func RequirePermissionWithFallback(
	secret []byte,
	previousSecret []byte,
	checker PermissionChecker,
	permission string,
	isTokenRevoked func(context.Context, *Claims) (bool, error),
) func(http.Handler) http.Handler {
	authn := RequireRoleWithFallback(secret, previousSecret, RoleViewer, isTokenRevoked)
	authz := RequirePermission(checker, permission)
	return func(next http.Handler) http.Handler {
		return authn(authz(next))
	}
}

func RequireAnyPermissionWithFallback(
	secret []byte,
	previousSecret []byte,
	checker PermissionChecker,
	permissions []string,
	isTokenRevoked func(context.Context, *Claims) (bool, error),
) func(http.Handler) http.Handler {
	authn := RequireRoleWithFallback(secret, previousSecret, RoleViewer, isTokenRevoked)
	authz := RequireAnyPermission(checker, permissions...)
	return func(next http.Handler) http.Handler {
		return authn(authz(next))
	}
}

// RequireAgentCredentialOrPermissionWithFallback authenticates agent traffic with
// mTLS/API keys and human traffic with the requested tenant capability. Presented
// but invalid agent credentials fail closed and are never downgraded to JWT auth.
func RequireAgentCredentialOrPermissionWithFallback(
	secret []byte,
	previousSecret []byte,
	checker PermissionChecker,
	permission string,
	keyLookup AgentKeyLookup,
	certificateLookup AgentCertificateLookup,
	isTokenRevoked func(context.Context, *Claims) (bool, error),
) func(http.Handler) http.Handler {
	authz := RequirePermissionWithFallback(secret, previousSecret, checker, permission, isTokenRevoked)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if hasPresentedAgentCredential(r) {
				RequireAgentCredential(keyLookup, certificateLookup)(next).ServeHTTP(w, r)
				return
			}
			authz(next).ServeHTTP(w, r)
		})
	}
}
