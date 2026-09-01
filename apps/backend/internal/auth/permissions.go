package auth

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type PermissionChecker interface {
	HasPermission(context.Context, uuid.UUID, uuid.UUID, string) (bool, error)
}

func RequirePermission(checker PermissionChecker, permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := FromContext(r.Context())
			if !ok || claims.UserID == uuid.Nil || claims.OrganizationID == uuid.Nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			allowed, err := checker.HasPermission(r.Context(), claims.UserID, claims.OrganizationID, permission)
			if err != nil {
				http.Error(w, "authorization check failed", http.StatusInternalServerError)
				return
			}
			if !allowed {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireAnyPermission(checker PermissionChecker, permissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := FromContext(r.Context())
			if !ok || claims.UserID == uuid.Nil || claims.OrganizationID == uuid.Nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			for _, permission := range permissions {
				allowed, err := checker.HasPermission(r.Context(), claims.UserID, claims.OrganizationID, permission)
				if err != nil {
					http.Error(w, "authorization check failed", http.StatusInternalServerError)
					return
				}
				if allowed {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, "forbidden", http.StatusForbidden)
		})
	}
}
