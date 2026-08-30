package auth

import (
	"errors"

	"github.com/google/uuid"
)

var ErrOrganizationContextRequired = errors.New("auth: organization context required")

func OrganizationID(claims *Claims) (uuid.UUID, error) {
	if claims == nil || claims.OrganizationID == uuid.Nil {
		return uuid.Nil, ErrOrganizationContextRequired
	}
	return claims.OrganizationID, nil
}
