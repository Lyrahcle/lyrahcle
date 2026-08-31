package dbscope

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const (
	requestKey              contextKey = "dbscope_request"
	organizationKey         contextKey = "dbscope_organization"
	platformControlPlaneKey contextKey = "dbscope_platform_control_plane"
)

func MarkRequest(ctx context.Context) context.Context {
	return context.WithValue(ctx, requestKey, true)
}

func IsRequest(ctx context.Context) bool {
	marked, _ := ctx.Value(requestKey).(bool)
	return marked
}

func WithOrganization(ctx context.Context, organizationID uuid.UUID) context.Context {
	return context.WithValue(ctx, organizationKey, organizationID)
}

func WithPlatformControlPlane(ctx context.Context) context.Context {
	return context.WithValue(ctx, platformControlPlaneKey, true)
}

func IsPlatformControlPlane(ctx context.Context) bool {
	allowed, _ := ctx.Value(platformControlPlaneKey).(bool)
	return allowed
}

func OrganizationID(ctx context.Context) (uuid.UUID, bool) {
	organizationID, ok := ctx.Value(organizationKey).(uuid.UUID)
	return organizationID, ok && organizationID != uuid.Nil
}

// Middleware marks HTTP-originated database work. Unauthenticated request
// contexts are fail-closed by PostgreSQL RLS until authentication adds an
// organization scope.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(MarkRequest(r.Context())))
	})
}
