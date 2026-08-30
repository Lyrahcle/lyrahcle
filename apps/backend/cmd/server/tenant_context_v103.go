package main

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
)

func requestOrganizationID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return uuid.Nil, false
	}
	orgID, err := auth.OrganizationID(claims)
	if err != nil {
		http.Error(w, "organization context required", http.StatusForbidden)
		return uuid.Nil, false
	}
	return orgID, true
}
