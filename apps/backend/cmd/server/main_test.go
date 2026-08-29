package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/testdb"
)

func testRouter(t *testing.T) (http.Handler, *pgxpool.Pool, *store.UserStore) {
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
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("failed to ping test database: %v", err)
	}
	if err := testdb.ResetAndMigrate(ctx, pool, "../../migrations"); err != nil {
		pool.Close()
		t.Fatalf("failed to prepare test database: %v", err)
	}
	t.Cleanup(pool.Close)

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
	return router, pool, users
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

func seedUserAndLogin(t *testing.T, router http.Handler, users *store.UserStore, email, password, role string) string {
	t.Helper()

	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if _, err := users.Create(context.Background(), email, hash, role); err != nil {
		t.Fatalf("seed user %s: %v", email, err)
	}

	rec := doJSON(t, router, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"identifier": email,
		"password":   password,
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

func registerAndLogin(t *testing.T, router http.Handler, users *store.UserStore, email, password string) string {
	t.Helper()
	return seedUserAndLogin(t, router, users, email, password, string(auth.RoleAdmin))
}

func TestHealthzIsPublic(t *testing.T) {
	router, _, _ := testRouter(t)

	rec := doJSON(t, router, http.MethodGet, "/healthz", "", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("GET /healthz: status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestProtectedEndpointRejectsMissingToken(t *testing.T) {
	router, _, _ := testRouter(t)

	rec := doJSON(t, router, http.MethodGet, "/api/v1/satellites/", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /satellites without a token: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestProtectedEndpointRejectsGarbageToken(t *testing.T) {
	router, _, _ := testRouter(t)

	rec := doJSON(t, router, http.MethodGet, "/api/v1/satellites/", "not-a-real-jwt", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /satellites with a garbage token: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAdminOnlyCreateEndpointCreatesUsersWithRequestedRole(t *testing.T) {
	router, _, users := testRouter(t)

	adminToken := seedUserAndLogin(t, router, users, "first@example.com", "correct-horse-battery", "admin")

	rec := doJSON(t, router, http.MethodPost, "/api/v1/users/", adminToken, map[string]string{
		"email":    "operator@example.com",
		"password": "correct-horse-battery",
		"role":     "operator",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /users as admin: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, router, http.MethodGet, "/api/v1/users/", adminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /users as admin: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var list []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("failed to unmarshal users response: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 users, got %d (%v)", len(list), list)
	}
}

func TestPublicSelfRegistrationIsDisabled(t *testing.T) {
	router, _, _ := testRouter(t)

	rec := doJSON(t, router, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"email":    "anyone@example.com",
		"password": "correct-horse-battery",
	})
	if rec.Code == http.StatusCreated {
		t.Fatalf("POST /auth/register: status = %d, want anything but 201 (public registration must be disabled)", rec.Code)
	}
}

func TestViewerCannotAccessAdminEndpoints(t *testing.T) {
	router, _, users := testRouter(t)

	adminToken := seedUserAndLogin(t, router, users, "admin@example.com", "correct-horse-battery", "admin")

	rec := doJSON(t, router, http.MethodPost, "/api/v1/users/", adminToken, map[string]string{
		"email":    "viewer@example.com",
		"password": "correct-horse-battery",
		"role":     "viewer",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("admin create viewer: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	viewerToken := registerAndLogin_loginOnly(t, router, "viewer@example.com", "correct-horse-battery")

	rec = doJSON(t, router, http.MethodGet, "/api/v1/users/", viewerToken, nil)
	if rec.Code != http.StatusForbidden {
		t.Errorf("GET /users as viewer: status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	rec = doJSON(t, router, http.MethodPost, "/api/v1/users/", viewerToken, map[string]string{
		"email":    "other@example.com",
		"password": "correct-horse-battery",
	})
	if rec.Code != http.StatusForbidden {
		t.Errorf("POST /users as viewer: status = %d, want %d", rec.Code, http.StatusForbidden)
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
		"identifier": email,
		"password":   password,
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
	router, _, users := testRouter(t)
	token := registerAndLogin(t, router, users, "operator@example.com", "correct-horse-battery")

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
	router, _, users := testRouter(t)
	token := registerAndLogin(t, router, users, "decoder@example.com", "correct-horse-battery")

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
	router, _, _ := testRouter(t)

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
	router, _, users := testRouter(t)
	token := registerAndLogin(t, router, users, "rf@example.com", "correct-horse-battery")

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
	router, _, _ := testRouter(t)

	rec := doJSON(t, router, http.MethodGet, "/healthz", "", nil)

	for _, h := range []string{"X-Content-Type-Options", "X-Frame-Options", "Referrer-Policy", "Content-Security-Policy"} {
		if rec.Header().Get(h) == "" {
			t.Errorf("response is missing security header %q", h)
		}
	}
}

func TestTOTPSetupEnableAndLoginFlow(t *testing.T) {
	router, _, users := testRouter(t)

	token := seedUserAndLogin(t, router, users, "totp@example.com", "correct-horse-battery", "viewer")

	rec := doJSON(t, router, http.MethodPost, "/api/v1/auth/totp/setup", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /auth/totp/setup: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var setupResp struct {
		Secret     string `json:"secret"`
		OtpauthURL string `json:"otpauth_url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &setupResp); err != nil {
		t.Fatalf("failed to unmarshal setup response: %v", err)
	}
	if setupResp.Secret == "" || setupResp.OtpauthURL == "" {
		t.Fatalf("setup response missing fields: %+v", setupResp)
	}

	badCode := "000000"
	if auth.VerifyTOTP(setupResp.Secret, badCode) {
		badCode = "111111"
	}
	rec = doJSON(t, router, http.MethodPost, "/api/v1/auth/totp/enable", token, map[string]string{"code": badCode})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("POST /auth/totp/enable with wrong code: status = %d, want 401", rec.Code)
	}

	validCode := currentTOTPCode(t, setupResp.Secret)
	rec = doJSON(t, router, http.MethodPost, "/api/v1/auth/totp/enable", token, map[string]string{"code": validCode})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("POST /auth/totp/enable with valid code: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, router, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"identifier": "totp@example.com",
		"password":   "correct-horse-battery",
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("login without totp_code: status = %d, want 401", rec.Code)
	}
	var errResp struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
	if errResp.Error != "totp_required" {
		t.Errorf("login without totp_code: error = %q, want %q", errResp.Error, "totp_required")
	}

	rec = doJSON(t, router, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"identifier": "totp@example.com",
		"password":   "correct-horse-battery",
		"totp_code":  currentTOTPCode(t, setupResp.Secret),
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("login with valid totp_code: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &loginResp); err != nil || loginResp.Token == "" {
		t.Fatalf("login with valid totp_code did not return a token: %s", rec.Body.String())
	}

	rec = doJSON(t, router, http.MethodPost, "/api/v1/auth/totp/disable", loginResp.Token, map[string]string{"password": "correct-horse-battery"})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("POST /auth/totp/disable: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, router, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"identifier": "totp@example.com",
		"password":   "correct-horse-battery",
	})
	if rec.Code != http.StatusOK {
		t.Errorf("login after disabling 2FA: status = %d, want 200, body = %s", rec.Code, rec.Body.String())
	}
}

func currentTOTPCode(t *testing.T, secret string) string {
	t.Helper()
	code, err := auth.CurrentTOTPCode(secret)
	if err != nil {
		t.Fatalf("CurrentTOTPCode: %v", err)
	}
	return code
}

func TestAuthMeReportsTOTPStatus(t *testing.T) {
	router, _, users := testRouter(t)
	token := seedUserAndLogin(t, router, users, "me@example.com", "correct-horse-battery", "viewer")

	rec := doJSON(t, router, http.MethodGet, "/api/v1/auth/me", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /auth/me: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var me struct {
		Email       string `json:"email"`
		TOTPEnabled bool   `json:"totp_enabled"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatalf("failed to unmarshal /auth/me response: %v", err)
	}
	if me.Email != "me@example.com" || me.TOTPEnabled {
		t.Errorf("GET /auth/me = %+v, want email me@example.com and totp_enabled=false", me)
	}
}

func TestSystemInfoEndpoint(t *testing.T) {
	router, _, users := testRouter(t)
	token := seedUserAndLogin(t, router, users, "sysinfo@example.com", "correct-horse-battery", "viewer")

	rec := doJSON(t, router, http.MethodGet, "/api/v1/system-info", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /system-info: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var info map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil {
		t.Fatalf("failed to unmarshal system-info response: %v", err)
	}
	if info["license"] == nil || info["celestrak_sync_schedule"] == nil {
		t.Errorf("system-info response missing fields: %+v", info)
	}
}

func TestSatellitesStreamEndpointSendsSSEEvent(t *testing.T) {
	router, _, users := testRouter(t)
	operatorToken := seedUserAndLogin(t, router, users, "stream-operator@example.com", "correct-horse-battery", "operator")
	viewerToken := seedUserAndLogin(t, router, users, "stream-viewer@example.com", "correct-horse-battery", "viewer")

	rec := doJSON(t, router, http.MethodPost, "/api/v1/satellites/", operatorToken, map[string]interface{}{
		"norad_id":  25544,
		"name":      "ISS (ZARYA)",
		"tle_line1": "1 25544U 98067A   24001.00000000  .00000000  00000-0  00000-0 0  9999",
		"tle_line2": "2 25544  51.6400   0.0000 0000000   0.0000   0.0000 15.50000000000000",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("seed satellite: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/satellites/stream", nil).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+viewerToken)
	streamRec := httptest.NewRecorder()

	router.ServeHTTP(streamRec, req)

	if streamRec.Code != http.StatusOK {
		t.Fatalf("GET /satellites/stream: status = %d, body = %s", streamRec.Code, streamRec.Body.String())
	}
	if ct := streamRec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
	body := streamRec.Body.String()
	if !strings.Contains(body, "event: satellites") || !strings.Contains(body, "ISS (ZARYA)") {
		t.Errorf("stream body missing expected event/data: %s", body)
	}
}

func TestAgentDeviceSelfReportFlow(t *testing.T) {
	router, _, users := testRouter(t)
	adminToken := seedUserAndLogin(t, router, users, "agentadmin@example.com", "correct-horse-battery", "admin")

	rec := doJSON(t, router, http.MethodPost, "/api/v1/agent-keys/", adminToken, map[string]string{"agent_id": "sdr-agent-test"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /agent-keys: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var created struct {
		AgentID string `json:"agent_id"`
		APIKey  string `json:"api_key"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to unmarshal agent-key response: %v", err)
	}

	body, _ := json.Marshal(map[string]interface{}{
		"driver":       "rtlsdr",
		"hardware":     "RTL-SDR Blog V4",
		"manufacturer": "RTL-SDR Blog",
		"product":      "RTL2832U",
		"serial":       "00000001",
		"rx":           true,
		"tx":           false,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/me/device", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Key", created.APIKey)
	deviceRec := httptest.NewRecorder()
	router.ServeHTTP(deviceRec, req)
	if deviceRec.Code != http.StatusNoContent {
		t.Fatalf("POST /agents/me/device: status = %d, body = %s", deviceRec.Code, deviceRec.Body.String())
	}

	rec = doJSON(t, router, http.MethodGet, "/api/v1/agent-keys/", adminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /agent-keys: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var list []struct {
		AgentID string `json:"agent_id"`
		Device  *struct {
			Hardware string `json:"hardware"`
			RX       bool   `json:"rx"`
		} `json:"device"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("failed to unmarshal agent-keys list: %v", err)
	}
	var found bool
	for _, a := range list {
		if a.AgentID != "sdr-agent-test" {
			continue
		}
		found = true
		if a.Device == nil || a.Device.Hardware != "RTL-SDR Blog V4" || !a.Device.RX {
			t.Errorf("agent device info = %+v, want hardware=RTL-SDR Blog V4 rx=true", a.Device)
		}
	}
	if !found {
		t.Fatal("seeded agent not found in agent-keys list")
	}

	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/agents/me/device", bytes.NewReader(body))
	badReq.Header.Set("Content-Type", "application/json")
	badReq.Header.Set("X-Agent-Key", "lyr_agent_totally-invalid")
	badRec := httptest.NewRecorder()
	router.ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusUnauthorized {
		t.Errorf("POST /agents/me/device with invalid key: status = %d, want 401", badRec.Code)
	}
}

func TestSpaceCVEsFilterAndSort(t *testing.T) {
	router, pool, users := testRouter(t)
	token := seedUserAndLogin(t, router, users, "cvefilter@example.com", "correct-horse-battery", "viewer")

	spaceCVEs := store.NewSpaceCVEStore(pool)
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("seed CVE: %v", err)
		}
	}
	must(spaceCVEs.Upsert(ctx, "CVE-2024-0001", "low sev satellite issue", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), "low", 2.1, "satellite"))
	must(spaceCVEs.Upsert(ctx, "CVE-2024-0002", "critical ground station issue", time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), "critical", 9.8, "ground station"))
	must(spaceCVEs.Upsert(ctx, "CVE-2024-0003", "medium satellite issue", time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), "medium", 5.5, "satellite"))

	rec := doJSON(t, router, http.MethodGet, "/api/v1/threat-intel/space-cves?severity=critical", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET space-cves severity filter: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var page struct {
		Items []struct {
			CVEID string `json:"cve_id"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if page.Total != 1 || page.Items[0].CVEID != "CVE-2024-0002" {
		t.Errorf("severity=critical filter = %+v, want only CVE-2024-0002", page)
	}

	rec = doJSON(t, router, http.MethodGet, "/api/v1/threat-intel/space-cves?keyword=satellite", token, nil)
	json.Unmarshal(rec.Body.Bytes(), &page)
	if page.Total != 2 {
		t.Errorf("keyword=satellite filter total = %d, want 2", page.Total)
	}

	rec = doJSON(t, router, http.MethodGet, "/api/v1/threat-intel/space-cves?sort=score&order=asc", token, nil)
	json.Unmarshal(rec.Body.Bytes(), &page)
	if len(page.Items) != 3 || page.Items[0].CVEID != "CVE-2024-0001" {
		t.Errorf("sort=score&order=asc first item = %+v, want CVE-2024-0001 first", page.Items)
	}

	rec = doJSON(t, router, http.MethodGet, "/api/v1/threat-intel/space-cves?published_from=2024-02-01&published_to=2024-12-31", token, nil)
	json.Unmarshal(rec.Body.Bytes(), &page)
	if page.Total != 2 {
		t.Errorf("date range filter total = %d, want 2", page.Total)
	}

	rec = doJSON(t, router, http.MethodGet, "/api/v1/threat-intel/space-cves/keywords", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET space-cves/keywords: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var keywords []string
	json.Unmarshal(rec.Body.Bytes(), &keywords)
	if len(keywords) != 2 {
		t.Errorf("keywords = %v, want 2 distinct values", keywords)
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	router, _, _ := testRouter(t)

	t.Run("generated", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if requestID := rec.Header().Get("X-Request-ID"); requestID == "" {
			t.Fatal("X-Request-ID response header is empty")
		}
	})

	t.Run("preserved", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		req.Header.Set("X-Request-ID", "mission-ops:1234")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if got := rec.Header().Get("X-Request-ID"); got != "mission-ops:1234" {
			t.Fatalf("X-Request-ID = %q, want %q", got, "mission-ops:1234")
		}
	})

	t.Run("unsafe-replaced", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		req.Header.Set("X-Request-ID", "unsafe request id")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		got := rec.Header().Get("X-Request-ID")
		if got == "" || got == "unsafe request id" {
			t.Fatalf("X-Request-ID = %q, want a generated safe identifier", got)
		}
	})
}

func TestAuthenticatedAuthTrafficDoesNotConsumeLoginLimit(t *testing.T) {
	router, _, users := testRouter(t)
	password := "correct-horse-battery"
	token := seedUserAndLogin(t, router, users, "relogin@example.com", password, string(auth.RoleAdmin))

	for i := 0; i < 20; i++ {
		rec := doJSON(t, router, http.MethodGet, "/api/v1/auth/me", token, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /auth/me attempt %d: status = %d, body = %s", i+1, rec.Code, rec.Body.String())
		}
	}

	logoutRec := doJSON(t, router, http.MethodPost, "/api/v1/auth/logout", token, nil)
	if logoutRec.Code != http.StatusNoContent {
		t.Fatalf("logout: status = %d, body = %s", logoutRec.Code, logoutRec.Body.String())
	}

	rec := doJSON(t, router, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"identifier": "relogin@example.com",
		"password":   password,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("re-login after authenticated auth traffic: status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
