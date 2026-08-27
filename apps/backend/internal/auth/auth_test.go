package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("supersecret123")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if !CheckPassword(hash, "supersecret123") {
		t.Error("CheckPassword should succeed with the correct password")
	}
	if CheckPassword(hash, "wrongpassword") {
		t.Error("CheckPassword should fail with an incorrect password")
	}
}

func TestIssueAndParseToken(t *testing.T) {
	secret := []byte("test-secret")
	userID := uuid.New()

	token, err := IssueToken(secret, userID, RoleOperator, time.Hour)
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}

	claims, err := ParseToken(secret, token)
	if err != nil {
		t.Fatalf("ParseToken returned error: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}
	if claims.Role != RoleOperator {
		t.Errorf("Role = %v, want %v", claims.Role, RoleOperator)
	}
}

func TestParseTokenWrongSecret(t *testing.T) {
	token, err := IssueToken([]byte("secret-a"), uuid.New(), RoleAdmin, time.Hour)
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}
	if _, err := ParseToken([]byte("secret-b"), token); err == nil {
		t.Error("ParseToken should reject a token signed with a different secret")
	}
}

func TestParseTokenExpired(t *testing.T) {
	token, err := IssueToken([]byte("secret"), uuid.New(), RoleViewer, -time.Hour)
	if err != nil {
		t.Fatalf("IssueToken returned error: %v", err)
	}
	if _, err := ParseToken([]byte("secret"), token); err == nil {
		t.Error("ParseToken should reject an expired token")
	}
}

func TestParseTokenMalformed(t *testing.T) {
	if _, err := ParseToken([]byte("secret"), "not-a-jwt"); err == nil {
		t.Error("ParseToken should reject a malformed token string")
	}
}

func TestRoleAtLeast(t *testing.T) {
	cases := []struct {
		role    Role
		minimum Role
		want    bool
	}{
		{RoleAdmin, RoleViewer, true},
		{RoleAdmin, RoleAdmin, true},
		{RoleOperator, RoleAdmin, false},
		{RoleViewer, RoleOperator, false},
		{RoleOperator, RoleOperator, true},
	}
	for _, c := range cases {
		got := RoleAtLeast(c.role, c.minimum)
		if got != c.want {
			t.Errorf("RoleAtLeast(%v, %v) = %v, want %v", c.role, c.minimum, got, c.want)
		}
	}
}

func TestValidRole(t *testing.T) {
	valid := []string{"admin", "operator", "viewer"}
	for _, r := range valid {
		if !ValidRole(r) {
			t.Errorf("ValidRole(%q) = false, want true", r)
		}
	}
	invalid := []string{"superadmin", "", "Admin", "root"}
	for _, r := range invalid {
		if ValidRole(r) {
			t.Errorf("ValidRole(%q) = true, want false", r)
		}
	}
}
