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
