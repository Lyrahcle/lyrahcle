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
