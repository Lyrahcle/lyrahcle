package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
)

func testRouter(t *testing.T) (http.Handler, *pgxpool.Pool) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping HTTP integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)

	tables := []string{
		"rf_anomalies", "rf_readings", "audit_log", "agent_keys",
		"space_cves", "satellites", "assets", "programs", "users",
	}
	for _, tbl := range tables {
		if _, err := pool.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tbl)); err != nil {
			t.Fatalf("failed to drop table %s: %v", tbl, err)
		}
	}

	entries, err := os.ReadDir("../../migrations")
	if err != nil {
		t.Fatalf("failed to read migrations dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		sqlBytes, err := os.ReadFile("../../migrations/" + e.Name())
		if err != nil {
			t.Fatalf("failed to read migration %s: %v", e.Name(), err)
		}
		if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
			t.Fatalf("failed to apply migration %s: %v", e.Name(), err)
		}
	}

	cfg := config.Config{
		JWTSecret: []byte("test-jwt-secret-for-http-tests-only"),
	}

	assets := store.NewAssetStore(pool)
	programs := store.NewProgramStore(pool)
	rf := store.NewRFStore(pool)
	rfAnomalies := store.NewAnomalyStore(pool)
	satellites := store.NewSatelliteStore(pool)
	users := store.NewUserStore(pool)
	agentKeys := store.NewAgentKeyStore(pool)
	auditLog := store.NewAuditStore(pool)
	spaceCVEs := store.NewSpaceCVEStore(pool)
	assetAlerts := store.NewAssetAlertStore(pool)

	router := newRouter(cfg, assets, programs, rf, rfAnomalies, satellites, users, agentKeys, auditLog, spaceCVEs, assetAlerts, nil)
	return router, pool
}

func doJSON(t *testing.T, router http.Handler, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()

	var reqBody *bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		reqBody = bytes.NewBuffer(b)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func registerAndLogin(t *testing.T, router http.Handler, email, password string) string {
	t.Helper()

	rec := doJSON(t, router, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"email":    email,
		"password": password,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, router, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email":    email,
		"password": password,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("login: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal login response: %v", err)
	}
	if resp.Token == "" {
		t.Fatal("login response did not include a token")
	}
	return resp.Token
}

func TestHealthzIsPublic(t *testing.T) {
	router, _ := testRouter(t)

	rec := doJSON(t, router, http.MethodGet, "/healthz", "", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("GET /healthz: status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestProtectedEndpointRejectsMissingToken(t *testing.T) {
	router, _ := testRouter(t)

	rec := doJSON(t, router, http.MethodGet, "/api/v1/satellites/", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /satellites without a token: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestProtectedEndpointRejectsGarbageToken(t *testing.T) {
	router, _ := testRouter(t)

	rec := doJSON(t, router, http.MethodGet, "/api/v1/satellites/", "not-a-real-jwt", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /satellites with a garbage token: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestFirstRegisteredUserBecomesAdmin(t *testing.T) {
	router, pool := testRouter(t)

	token := registerAndLogin(t, router, "first@example.com", "correct-horse-battery")

	rec := doJSON(t, router, http.MethodGet, "/api/v1/users/", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /users as first user: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var users []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &users); err != nil {
		t.Fatalf("failed to unmarshal users response: %v", err)
	}
	if len(users) != 1 || users[0]["role"] != "admin" {
		t.Errorf("first registered user role = %v, want admin (users=%v)", users[0]["role"], users)
	}

	_ = pool
}

func TestViewerCannotAccessAdminEndpoints(t *testing.T) {
	router, pool := testRouter(t)
	ctx := context.Background()

	adminToken := registerAndLogin(t, router, "admin@example.com", "correct-horse-battery")

	rec := doJSON(t, router, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"email":    "viewer@example.com",
		"password": "correct-horse-battery",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register second user: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	users := store.NewUserStore(pool)
	u, err := users.GetByEmail(ctx, "viewer@example.com")
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if u.Role != "viewer" {
		t.Fatalf("second registered user role = %q, want %q (bootstrap-admin rule should only apply to the first user)", u.Role, "viewer")
	}

	viewerToken := registerAndLogin_loginOnly(t, router, "viewer@example.com", "correct-horse-battery")

	rec = doJSON(t, router, http.MethodGet, "/api/v1/users/", viewerToken, nil)
	if rec.Code != http.StatusForbidden {
		t.Errorf("GET /users as viewer: status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	rec = doJSON(t, router, http.MethodGet, "/api/v1/audit-log", viewerToken, nil)
	if rec.Code != http.StatusForbidden {
		t.Errorf("GET /audit-log as viewer: status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	rec = doJSON(t, router, http.MethodPatch, "/api/v1/users/00000000-0000-0000-0000-000000000000/role", viewerToken, map[string]string{"role": "admin"})
	if rec.Code != http.StatusForbidden {
		t.Errorf("PATCH /users/{id}/role as viewer: status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	rec = doJSON(t, router, http.MethodGet, "/api/v1/satellites/", adminToken, nil)
	if rec.Code != http.StatusOK {
		t.Errorf("GET /satellites as admin: status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func registerAndLogin_loginOnly(t *testing.T, router http.Handler, email, password string) string {
	t.Helper()

	rec := doJSON(t, router, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"email":    email,
		"password": password,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("login: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal login response: %v", err)
	}
	return resp.Token
}

func TestSatelliteCreateAndListRoundTrip(t *testing.T) {
	router, _ := testRouter(t)
	token := registerAndLogin(t, router, "operator@example.com", "correct-horse-battery")

	rec := doJSON(t, router, http.MethodPost, "/api/v1/satellites/", token, map[string]interface{}{
		"norad_id":  25544,
		"name":      "ISS (ZARYA)",
		"tle_line1": "1 25544U 98067A   24001.00000000  .00000000  00000-0  00000-0 0  9999",
		"tle_line2": "2 25544  51.6400   0.0000 0000000   0.0000   0.0000 15.50000000000000",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /satellites: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, router, http.MethodGet, "/api/v1/satellites/", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /satellites: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var list []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("failed to unmarshal satellites response: %v", err)
	}
	if len(list) != 1 || list[0]["name"] != "ISS (ZARYA)" {
		t.Errorf("satellites list = %v, want one entry named ISS (ZARYA)", list)
	}
}

func TestDecodeCSPEndpoint(t *testing.T) {
	router, _ := testRouter(t)
	token := registerAndLogin(t, router, "decoder@example.com", "correct-horse-battery")

	rec := doJSON(t, router, http.MethodPost, "/api/v1/tmtc/decode/csp", token, map[string]string{
		"packet_hex": "5012bf0154454d503d32312e34",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /tmtc/decode/csp: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to unmarshal decode response: %v", err)
	}
	if decoded["payload"] != "TEMP=21.4" {
		t.Errorf("decoded payload = %v, want %q", decoded["payload"], "TEMP=21.4")
	}
}

func TestDecodeEndpointsRequireAuth(t *testing.T) {
	router, _ := testRouter(t)

	paths := []string{
		"/api/v1/tmtc/decode/ax25",
		"/api/v1/tmtc/decode/ccsds",
		"/api/v1/tmtc/decode/pus",
		"/api/v1/tmtc/decode/csp",
	}
	for _, path := range paths {
		rec := doJSON(t, router, http.MethodPost, path, "", map[string]string{"frame_hex": "00", "packet_hex": "00"})
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("POST %s without a token: status = %d, want 401", path, rec.Code)
		}
	}
}

func TestRFIngestFlagsAnomaly(t *testing.T) {
	router, _ := testRouter(t)
	token := registerAndLogin(t, router, "rf@example.com", "correct-horse-battery")

	base := -60.0
	for i := 0; i < 6; i++ {
		rec := doJSON(t, router, http.MethodPost, "/api/v1/rf/ingest", token, map[string]interface{}{
			"agent_id":     "sdr-http-test",
			"frequency_hz": 437500000,
			"power_dbm":    base - float64(i%3),
			"captured_at":  time.Now().UTC().Format(time.RFC3339),
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("POST /rf/ingest (baseline #%d): status = %d, body = %s", i, rec.Code, rec.Body.String())
		}
	}

	rec := doJSON(t, router, http.MethodPost, "/api/v1/rf/ingest", token, map[string]interface{}{
		"agent_id":     "sdr-http-test",
		"frequency_hz": 437500000,
		"power_dbm":    -10.0,
		"captured_at":  time.Now().UTC().Format(time.RFC3339),
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /rf/ingest (anomalous): status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, router, http.MethodGet, "/api/v1/rf/anomalies", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /rf/anomalies: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var anomalies []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &anomalies); err != nil {
		t.Fatalf("failed to unmarshal anomalies response: %v", err)
	}
	if len(anomalies) == 0 {
		t.Fatal("expected at least one anomaly to be flagged for the -10 dBm outlier, got none")
	}
}

func TestSecurityHeadersArePresent(t *testing.T) {
	router, _ := testRouter(t)

	rec := doJSON(t, router, http.MethodGet, "/healthz", "", nil)

	for _, h := range []string{"X-Content-Type-Options", "X-Frame-Options", "Referrer-Policy", "Content-Security-Policy"} {
		if rec.Header().Get(h) == "" {
			t.Errorf("response is missing security header %q", h)
		}
	}
}
