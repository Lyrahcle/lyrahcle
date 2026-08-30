package main

import (
	"net/http"

	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
)

func requireRequestPermission(w http.ResponseWriter, r *http.Request, checker auth.PermissionChecker, permission string) bool {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	allowed, err := checker.HasPermission(r.Context(), claims.UserID, claims.OrganizationID, permission)
	if err != nil {
		http.Error(w, "authorization check failed", http.StatusInternalServerError)
		return false
	}
	if !allowed {
		http.Error(w, "forbidden", http.StatusForbidden)
		return false
	}
	return true
}
