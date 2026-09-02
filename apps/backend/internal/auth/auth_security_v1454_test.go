package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestParseTokenRejectsWrongSecretAndWrongSigningMethod(t *testing.T) {
	userID := uuid.New()
	token, err := IssueToken([]byte("correct-secret"), userID, RoleOperator, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken([]byte("wrong-secret"), token); err == nil {
		t.Fatal("expected token signed with another secret to be rejected")
	}

	claims := Claims{
		UserID:          userID,
		Role:            RoleOperator,
		SessionIssuedAt: time.Now().UnixNano(),
		AuthVersion:     1,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Issuer:    TokenIssuer,
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{TokenAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	rsToken := jwt.NewWithClaims(jwt.SigningMethodRS256, &claims)
	signed, err := rsToken.SignedString(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseToken([]byte("unused"), signed); err == nil {
		t.Fatal("expected non-HS256 signing method to be rejected")
	}
}

func TestParseTokenRejectsInvalidClaims(t *testing.T) {
	secret := []byte("claims-secret")
	now := time.Now()

	tests := []struct {
		name   string
		mutate func(*Claims)
	}{
		{"nil user", func(c *Claims) { c.UserID = uuid.Nil }},
		{"subject mismatch", func(c *Claims) { c.Subject = uuid.NewString() }},
		{"invalid role", func(c *Claims) { c.Role = Role("root") }},
		{"missing token id", func(c *Claims) { c.ID = "" }},
		{"missing session issue time", func(c *Claims) { c.SessionIssuedAt = 0 }},
		{"invalid auth version", func(c *Claims) { c.AuthVersion = 0 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			userID := uuid.New()
			claims := &Claims{
				UserID:          userID,
				Role:            RoleAnalyst,
				SessionIssuedAt: now.UnixNano(),
				AuthVersion:     1,
				RegisteredClaims: jwt.RegisteredClaims{
					ID:        uuid.NewString(),
					Issuer:    TokenIssuer,
					Subject:   userID.String(),
					Audience:  jwt.ClaimStrings{TokenAudience},
					ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
					IssuedAt:  jwt.NewNumericDate(now),
					NotBefore: jwt.NewNumericDate(now),
				},
			}
			tc.mutate(claims)
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			signed, err := token.SignedString(secret)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ParseToken(secret, signed); err == nil {
				t.Fatalf("expected %s claims to be rejected", tc.name)
			}
		})
	}
}

func TestParseTokenWithFallbackUsesPreviousSecretOnlyWhenNeeded(t *testing.T) {
	current := []byte("current-secret")
	previous := []byte("previous-secret")
	userID := uuid.New()

	oldToken, err := IssueToken(previous, userID, RoleViewer, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseTokenWithFallback(current, previous, oldToken); err != nil {
		t.Fatalf("previous-secret token should survive rotation grace: %v", err)
	}
	if _, err := ParseTokenWithFallback(current, nil, oldToken); err == nil {
		t.Fatal("previous-secret token must fail when no fallback is configured")
	}
}

func TestRequireRoleRevocationAndPasswordChangePaths(t *testing.T) {
	secret := []byte("middleware-secret")
	userID := uuid.New()

	token, _, err := IssueUserTokenWithState(secret, userID, RoleAdmin, true, 1, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := FromContext(r.Context()); !ok {
			t.Fatal("claims missing from request context")
		}
		w.WriteHeader(http.StatusNoContent)
	})

	t.Run("password change blocks unrelated API", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		RequireRole(secret, RoleViewer)(next).ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status=%d want %d", rec.Code, http.StatusForbidden)
		}
	})

	for _, path := range []string{"/api/v1/auth/me", "/api/v1/auth/change-password", "/api/v1/auth/logout"} {
		t.Run("allowed "+path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			RequireRole(secret, RoleViewer)(next).ServeHTTP(rec, req)
			if rec.Code != http.StatusNoContent {
				t.Fatalf("status=%d want %d", rec.Code, http.StatusNoContent)
			}
		})
	}

	t.Run("revocation backend failure fails closed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		check := func(context.Context, *Claims) (bool, error) {
			return false, context.DeadlineExceeded
		}
		RequireRole(secret, RoleViewer, check)(next).ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status=%d want %d", rec.Code, http.StatusServiceUnavailable)
		}
	})

	t.Run("revoked token rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		check := func(context.Context, *Claims) (bool, error) { return true, nil }
		RequireRole(secret, RoleViewer, check)(next).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}

func TestRequireAgentCredentialFailsClosedOnPresentedCertificate(t *testing.T) {
	keyCalls := 0
	certCalls := 0

	keyLookup := func(context.Context, string) (string, error) {
		keyCalls++
		return "agent-from-key", nil
	}
	certLookup := func(context.Context, string) (string, error) {
		certCalls++
		return "", context.Canceled
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("request with invalid presented certificate must not reach handler")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/config", nil)
	req.Header.Set("X-Agent-Key", "lyr_agent_valid")
	req.TLS = &tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{{Raw: []byte("presented-cert")}},
	}
	rec := httptest.NewRecorder()

	RequireAgentCredential(keyLookup, certLookup)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want %d", rec.Code, http.StatusUnauthorized)
	}
	if certCalls != 1 || keyCalls != 0 {
		t.Fatalf("certCalls=%d keyCalls=%d; certificate must take precedence", certCalls, keyCalls)
	}
}

type securityPermissionChecker struct {
	allowed map[string]bool
	errFor  string
	calls   []string
}

func (c *securityPermissionChecker) HasPermission(_ context.Context, _, _ uuid.UUID, permission string) (bool, error) {
	c.calls = append(c.calls, permission)
	if permission == c.errFor {
		return false, context.DeadlineExceeded
	}
	return c.allowed[permission], nil
}

func TestRequireAnyPermissionShortCircuitsAndFailsClosed(t *testing.T) {
	claims := &Claims{UserID: uuid.New(), OrganizationID: uuid.New(), Role: RoleViewer}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })

	t.Run("granted permission short-circuits", func(t *testing.T) {
		checker := &securityPermissionChecker{allowed: map[string]bool{"incidents.read": true}}
		req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(WithClaims(context.Background(), claims))
		rec := httptest.NewRecorder()
		RequireAnyPermission(checker, "incidents.read", "incidents.manage")(next).ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status=%d want %d", rec.Code, http.StatusNoContent)
		}
		if strings.Join(checker.calls, ",") != "incidents.read" {
			t.Fatalf("calls=%v want first permission only", checker.calls)
		}
	})

	t.Run("checker error fails closed", func(t *testing.T) {
		checker := &securityPermissionChecker{
			allowed: map[string]bool{},
			errFor:  "incidents.read",
		}
		req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(WithClaims(context.Background(), claims))
		rec := httptest.NewRecorder()
		RequireAnyPermission(checker, "incidents.read", "incidents.manage")(next).ServeHTTP(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status=%d want %d", rec.Code, http.StatusInternalServerError)
		}
	})
}

func TestRoleAndOrganizationEdgeCases(t *testing.T) {
	if ValidRole("root") {
		t.Fatal("unknown role must be rejected")
	}
	if RoleAtLeast(RoleViewer, RoleAnalyst) {
		t.Fatal("viewer must not satisfy analyst minimum")
	}
	if !RoleAtLeast(RoleAdmin, RoleOperator) {
		t.Fatal("admin should satisfy operator minimum")
	}

	if _, err := OrganizationID(nil); err == nil {
		t.Fatal("nil claims must require organization context")
	}
	if _, err := OrganizationID(&Claims{}); err == nil {
		t.Fatal("nil organization UUID must require organization context")
	}
	orgID := uuid.New()
	got, err := OrganizationID(&Claims{OrganizationID: orgID})
	if err != nil || got != orgID {
		t.Fatalf("OrganizationID()=(%s,%v) want (%s,nil)", got, err, orgID)
	}
}

func TestBearerParsingFailsClosedForAmbiguousHeaders(t *testing.T) {
	secret := []byte("bearer-format-secret")
	token, err := IssueToken(secret, uuid.New(), RoleAdmin, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	for _, header := range []string{
		"bearer " + token,
		"Bearer",
		"Bearer  " + token,
		"Basic " + token,
		" Bearer " + token,
	} {
		t.Run(header[:min(len(header), 16)], func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents", nil)
			req.Header.Set("Authorization", header)
			rec := httptest.NewRecorder()
			RequireRole(secret, RoleViewer)(newTestHandler()).ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("header %q status=%d want %d", header, rec.Code, http.StatusUnauthorized)
			}
		})
	}
}

func TestOrganizationClaimsReachAuthorizationContext(t *testing.T) {
	secret := []byte("organization-context-secret")
	userID, organizationID := uuid.New(), uuid.New()
	token, _, err := IssueOrganizationUserTokenWithState(secret, userID, organizationID, RoleViewer, false, 2, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	checker := &securityPermissionChecker{allowed: map[string]bool{"incidents.read": true}}
	var gotUser, gotOrganization uuid.UUID
	capture := permissionCheckerFunc(func(_ context.Context, user, organization uuid.UUID, permission string) (bool, error) {
		gotUser, gotOrganization = user, organization
		return checker.HasPermission(context.Background(), user, organization, permission)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/incidents", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	RequirePermissionWithFallback(secret, nil, capture, "incidents.read", nil)(newTestHandler()).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want %d", rec.Code, http.StatusOK)
	}
	if gotUser != userID || gotOrganization != organizationID {
		t.Fatalf("authorization context user=%s org=%s, want user=%s org=%s", gotUser, gotOrganization, userID, organizationID)
	}
}
