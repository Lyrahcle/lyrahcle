package main

import (
	"os"
	"strings"
	"testing"
)

func TestPlatformUsersUsesMinimalProjectionV144(t *testing.T) {
	routeBytes, err := os.ReadFile("organizations.go")
	if err != nil {
		t.Fatal(err)
	}
	route := string(routeBytes)
	if !strings.Contains(route, `users.ListPlatformUsers(r.Context())`) {
		t.Fatal("platform-users route must use the minimal platform user projection")
	}
	if strings.Contains(route, `allUsers, err := users.List(r.Context())`) {
		t.Fatal("platform-users route must not load full User records with password/TOTP material")
	}

	storeBytes, err := os.ReadFile("../../internal/store/users.go")
	if err != nil {
		t.Fatal(err)
	}
	store := string(storeBytes)
	for _, token := range []string{
		"func (s *UserStore) ListPlatformUsers",
		"LEFT JOIN platform_administrators",
		"(pa.user_id IS NOT NULL) AS platform_admin",
		"u.deleted_at IS NULL",
	} {
		if !strings.Contains(store, token) {
			t.Fatalf("missing platform user regression token %q", token)
		}
	}
	for _, forbidden := range []string{"password_hash", "totp_secret"} {
		start := strings.Index(store, "func (s *UserStore) ListPlatformUsers")
		end := strings.Index(store[start:], "func (s *UserStore) ListPlatformAdministratorIDs")
		if start >= 0 && end > 0 && strings.Contains(store[start:start+end], forbidden) {
			t.Fatalf("platform user projection must not select %s", forbidden)
		}
	}
}
