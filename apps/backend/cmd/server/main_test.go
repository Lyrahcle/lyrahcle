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

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
	"github.com/mlko01/lyrahcle/apps/backend/internal/config"
	"github.com/mlko01/lyrahcle/apps/backend/internal/detection"
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
		JWTSecret:       []byte("test-jwt-secret-for-http-tests-only"),
		JWTAccessTTL:    time.Hour,
		WebAuthnRPID:    "localhost",
		WebAuthnOrigins: []string{"http://localhost:5173"},
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
	incidents := store.NewIncidentStore(pool)
	rbac := store.NewRBACStore(pool)

	router := newRouter(cfg, pool, assets, programs, rf, rfAnomalies, satellites, users, agentKeys, auditLog, spaceCVEs, assetAlerts, incidents, rbac, nil)
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

func TestReadyzChecksDatabase(t *testing.T) {
	router, _, _ := testRouter(t)
	rec := doJSON(t, router, http.MethodGet, "/readyz", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /readyz: status = %d, body = %s", rec.Code, rec.Body.String())
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

	adminToken := seedUserAndLogin(t, router, users, "first@example.com", "Correct-Horse-Battery-9!", "admin")

	rec := doJSON(t, router, http.MethodPost, "/api/v1/users/", adminToken, map[string]string{
		"email":    "operator@example.com",
		"password": "Correct-Horse-Battery-9!",
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
		"password": "Correct-Horse-Battery-9!",
	})
	if rec.Code == http.StatusCreated {
		t.Fatalf("POST /auth/register: status = %d, want anything but 201 (public registration must be disabled)", rec.Code)
	}
}

func TestViewerCannotAccessAdminEndpoints(t *testing.T) {
	router, _, users := testRouter(t)

	adminToken := seedUserAndLogin(t, router, users, "admin@example.com", "Correct-Horse-Battery-9!", "admin")

	rec := doJSON(t, router, http.MethodPost, "/api/v1/users/", adminToken, map[string]string{
		"email":    "viewer@example.com",
		"password": "Correct-Horse-Battery-9!",
		"role":     "viewer",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("admin create viewer: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	viewerToken := registerAndLogin_loginOnly(t, router, "viewer@example.com", "Correct-Horse-Battery-9!")

	rec = doJSON(t, router, http.MethodGet, "/api/v1/users/", viewerToken, nil)
	if rec.Code != http.StatusForbidden {
		t.Errorf("GET /users as viewer: status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	rec = doJSON(t, router, http.MethodPost, "/api/v1/users/", viewerToken, map[string]string{
		"email":    "other@example.com",
		"password": "Correct-Horse-Battery-9!",
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
	token := registerAndLogin(t, router, users, "operator@example.com", "Correct-Horse-Battery-9!")

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
	if list[0]["tle_status"] == nil || list[0]["orbital_confidence"] == nil {
		t.Errorf("satellite response missing TLE quality fields: %v", list[0])
	}
}

func TestDecodeCSPEndpoint(t *testing.T) {
	router, _, users := testRouter(t)
	token := registerAndLogin(t, router, users, "decoder@example.com", "Correct-Horse-Battery-9!")

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
	token := registerAndLogin(t, router, users, "rf@example.com", "Correct-Horse-Battery-9!")

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

	rec = doJSON(t, router, http.MethodGet, "/api/v1/incidents/?agent_id=sdr-http-test", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /incidents after RF anomaly: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var incidentPage store.Page[store.Incident]
	if err := json.Unmarshal(rec.Body.Bytes(), &incidentPage); err != nil {
		t.Fatalf("failed to unmarshal incident response: %v", err)
	}
	foundAutoIncident := false
	for _, incident := range incidentPage.Items {
		if incident.SourceType == "detection_rule" && incident.AgentID == "sdr-http-test" {
			foundAutoIncident = true
			break
		}
	}
	if !foundAutoIncident {
		t.Fatal("expected an RF detection rule to create a correlated incident")
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

	token := seedUserAndLogin(t, router, users, "totp@example.com", "Correct-Horse-Battery-9!", "viewer")

	rec := doJSON(t, router, http.MethodPost, "/api/v1/auth/totp/setup", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /auth/totp/setup: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var setupResp struct {
		Provider   string `json:"provider"`
		Issuer     string `json:"issuer"`
		Secret     string `json:"secret"`
		OtpauthURL string `json:"otpauth_url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &setupResp); err != nil {
		t.Fatalf("failed to unmarshal setup response: %v", err)
	}
	if setupResp.Provider != "google_authenticator" || setupResp.Issuer != "Lyrahcle" || setupResp.Secret == "" || setupResp.OtpauthURL == "" {
		t.Fatalf("setup response is invalid: %+v", setupResp)
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
		"password":   "Correct-Horse-Battery-9!",
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
		"password":   "Correct-Horse-Battery-9!",
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

	rec = doJSON(t, router, http.MethodPost, "/api/v1/auth/totp/disable", loginResp.Token, map[string]string{"password": "Correct-Horse-Battery-9!"})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("POST /auth/totp/disable: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, router, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"identifier": "totp@example.com",
		"password":   "Correct-Horse-Battery-9!",
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
	token := seedUserAndLogin(t, router, users, "me@example.com", "Correct-Horse-Battery-9!", "viewer")

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
	token := seedUserAndLogin(t, router, users, "sysinfo@example.com", "Correct-Horse-Battery-9!", "viewer")

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
	operatorToken := seedUserAndLogin(t, router, users, "stream-operator@example.com", "Correct-Horse-Battery-9!", "operator")
	viewerToken := seedUserAndLogin(t, router, users, "stream-viewer@example.com", "Correct-Horse-Battery-9!", "viewer")

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
	adminToken := seedUserAndLogin(t, router, users, "agentadmin@example.com", "Correct-Horse-Battery-9!", "admin")

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
	token := seedUserAndLogin(t, router, users, "cvefilter@example.com", "Correct-Horse-Battery-9!", "viewer")

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
	password := "Correct-Horse-Battery-9!"
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

func TestIncidentRBACAndTimeline(t *testing.T) {
	router, _, users := testRouter(t)
	operatorToken := seedUserAndLogin(t, router, users, "incident-operator@example.com", "operator-pass", string(auth.RoleOperator))
	viewerToken := seedUserAndLogin(t, router, users, "incident-viewer@example.com", "viewer-pass", string(auth.RoleViewer))

	viewerCreate := doJSON(t, router, http.MethodPost, "/api/v1/incidents/", viewerToken, map[string]string{
		"title":    "Unauthorized incident",
		"severity": "high",
	})
	if viewerCreate.Code != http.StatusForbidden {
		t.Fatalf("viewer create: status = %d, body = %s", viewerCreate.Code, viewerCreate.Body.String())
	}

	viewerUser, err := users.GetByEmail(context.Background(), "incident-viewer@example.com")
	if err != nil {
		t.Fatalf("get viewer: %v", err)
	}
	operatorAssignedCreate := doJSON(t, router, http.MethodPost, "/api/v1/incidents/", operatorToken, map[string]string{
		"title":       "Capability-based assignment",
		"severity":    "medium",
		"assigned_to": viewerUser.ID.String(),
	})
	if operatorAssignedCreate.Code != http.StatusCreated {
		t.Fatalf("operator with incidents.manage assignment on create: status = %d, body = %s", operatorAssignedCreate.Code, operatorAssignedCreate.Body.String())
	}
	var assignedIncident store.Incident
	if err := json.Unmarshal(operatorAssignedCreate.Body.Bytes(), &assignedIncident); err != nil {
		t.Fatalf("decode assigned incident: %v", err)
	}
	if assignedIncident.AssignedTo == nil || *assignedIncident.AssignedTo != viewerUser.ID {
		t.Fatalf("assigned_to = %v, want %s", assignedIncident.AssignedTo, viewerUser.ID)
	}

	created := doJSON(t, router, http.MethodPost, "/api/v1/incidents/", operatorToken, map[string]string{
		"title":       "Unexpected RF behavior",
		"description": "Frequency behavior requires investigation.",
		"severity":    "high",
		"agent_id":    "sdr-incident-01",
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("operator create: status = %d, body = %s", created.Code, created.Body.String())
	}
	var incident store.Incident
	if err := json.Unmarshal(created.Body.Bytes(), &incident); err != nil {
		t.Fatalf("decode incident: %v", err)
	}
	if incident.ID == uuid.Nil || incident.Reference == "" {
		t.Fatalf("created incident missing identity: %+v", incident)
	}

	listed := doJSON(t, router, http.MethodGet, "/api/v1/incidents/?severity=high", viewerToken, nil)
	if listed.Code != http.StatusOK {
		t.Fatalf("viewer list: status = %d, body = %s", listed.Code, listed.Body.String())
	}

	updated := doJSON(t, router, http.MethodPatch, "/api/v1/incidents/"+incident.ID.String(), operatorToken, map[string]string{
		"status": "investigating",
	})
	if updated.Code != http.StatusOK {
		t.Fatalf("operator update: status = %d, body = %s", updated.Code, updated.Body.String())
	}

	note := doJSON(t, router, http.MethodPost, "/api/v1/incidents/"+incident.ID.String()+"/notes", operatorToken, map[string]string{
		"message": "Signal source is under review.",
	})
	if note.Code != http.StatusNoContent {
		t.Fatalf("operator note: status = %d, body = %s", note.Code, note.Body.String())
	}

	detail := doJSON(t, router, http.MethodGet, "/api/v1/incidents/"+incident.ID.String(), viewerToken, nil)
	if detail.Code != http.StatusOK {
		t.Fatalf("viewer detail: status = %d, body = %s", detail.Code, detail.Body.String())
	}
	var decoded store.IncidentDetail
	if err := json.Unmarshal(detail.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if decoded.Incident.Status != "investigating" {
		t.Fatalf("status = %q, want investigating", decoded.Incident.Status)
	}
	if len(decoded.Events) < 3 {
		t.Fatalf("events = %d, want at least 3", len(decoded.Events))
	}
}

func TestIncidentInputValidation(t *testing.T) {
	router, _, users := testRouter(t)
	operatorToken := seedUserAndLogin(t, router, users, "incident-validator@example.com", "operator-pass", string(auth.RoleOperator))

	cases := []struct {
		name string
		body map[string]string
	}{
		{name: "empty title", body: map[string]string{"title": "", "severity": "high"}},
		{name: "invalid severity", body: map[string]string{"title": "Test", "severity": "root"}},
		{name: "invalid agent", body: map[string]string{"title": "Test", "severity": "high", "agent_id": strings.Repeat("x", 129)}},
		{name: "invalid satellite", body: map[string]string{"title": "Test", "severity": "high", "satellite_id": "not-a-uuid"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doJSON(t, router, http.MethodPost, "/api/v1/incidents/", operatorToken, tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestIncidentStateMachineRejectsTerminalTransition(t *testing.T) {
	router, _, users := testRouter(t)
	operatorToken := seedUserAndLogin(t, router, users, "state-operator@example.com", "operator-pass", string(auth.RoleOperator))
	created := doJSON(t, router, http.MethodPost, "/api/v1/incidents/", operatorToken, map[string]string{"title": "State machine", "severity": "high"})
	if created.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, body = %s", created.Code, created.Body.String())
	}
	var incident store.Incident
	if err := json.Unmarshal(created.Body.Bytes(), &incident); err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{"investigating", "resolved"} {
		rec := doJSON(t, router, http.MethodPatch, "/api/v1/incidents/"+incident.ID.String(), operatorToken, map[string]string{"status": status})
		if rec.Code != http.StatusOK {
			t.Fatalf("transition to %s: status = %d, body = %s", status, rec.Code, rec.Body.String())
		}
	}
	rec := doJSON(t, router, http.MethodPatch, "/api/v1/incidents/"+incident.ID.String(), operatorToken, map[string]string{"status": "new"})
	if rec.Code != http.StatusConflict {
		t.Fatalf("resolved -> new: status = %d, want %d, body = %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestIncidentCreateIsIdempotent(t *testing.T) {
	router, _, users := testRouter(t)
	operatorToken := seedUserAndLogin(t, router, users, "idempotent-operator@example.com", "operator-pass", string(auth.RoleOperator))
	body := map[string]string{"title": "Idempotent incident", "severity": "medium"}
	request := func() *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+operatorToken)
		req.Header.Set("Idempotency-Key", "incident-test-key-001")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	first := request()
	second := request()
	if first.Code != http.StatusCreated || second.Code != http.StatusCreated {
		t.Fatalf("statuses = %d, %d", first.Code, second.Code)
	}
	var a, b store.Incident
	_ = json.Unmarshal(first.Body.Bytes(), &a)
	_ = json.Unmarshal(second.Body.Bytes(), &b)
	if a.ID == uuid.Nil || a.ID != b.ID {
		t.Fatalf("idempotent create returned different incidents: %s != %s", a.ID, b.ID)
	}
}

func TestIncidentEvidenceRequiresOperator(t *testing.T) {
	router, _, users := testRouter(t)
	operatorToken := seedUserAndLogin(t, router, users, "evidence-operator@example.com", "operator-pass", string(auth.RoleOperator))
	viewerToken := seedUserAndLogin(t, router, users, "evidence-viewer@example.com", "viewer-pass", string(auth.RoleViewer))
	created := doJSON(t, router, http.MethodPost, "/api/v1/incidents/", operatorToken, map[string]string{"title": "Evidence incident", "severity": "medium"})
	var incident store.Incident
	_ = json.Unmarshal(created.Body.Bytes(), &incident)
	viewer := doJSON(t, router, http.MethodPost, "/api/v1/incidents/"+incident.ID.String()+"/evidence", viewerToken, map[string]interface{}{"evidence_type": "request", "evidence_ref": "req-001"})
	if viewer.Code != http.StatusForbidden {
		t.Fatalf("viewer evidence: status = %d", viewer.Code)
	}
	operator := doJSON(t, router, http.MethodPost, "/api/v1/incidents/"+incident.ID.String()+"/evidence", operatorToken, map[string]interface{}{"evidence_type": "request", "evidence_ref": "req-001", "label": "Detection request"})
	if operator.Code != http.StatusCreated {
		t.Fatalf("operator evidence: status = %d, body = %s", operator.Code, operator.Body.String())
	}
	duplicate := doJSON(t, router, http.MethodPost, "/api/v1/incidents/"+incident.ID.String()+"/evidence", operatorToken, map[string]interface{}{"evidence_type": "request", "evidence_ref": "req-001", "label": "Changed label"})
	if duplicate.Code != http.StatusCreated {
		t.Fatalf("duplicate evidence: status = %d, body = %s", duplicate.Code, duplicate.Body.String())
	}
	detail := doJSON(t, router, http.MethodGet, "/api/v1/incidents/"+incident.ID.String(), viewerToken, nil)
	var decoded store.IncidentDetail
	_ = json.Unmarshal(detail.Body.Bytes(), &decoded)
	if len(decoded.Evidence) != 1 || decoded.Evidence[0].EvidenceRef != "req-001" {
		t.Fatalf("unexpected evidence: %+v", decoded.Evidence)
	}
	if decoded.Evidence[0].Label != "Detection request" {
		t.Fatalf("evidence label was mutated: %q", decoded.Evidence[0].Label)
	}
}

func TestDetectionRuleRBACAndValidation(t *testing.T) {
	router, _, users := testRouter(t)
	viewerToken := seedUserAndLogin(t, router, users, "rule-viewer@example.com", "viewer-pass", string(auth.RoleViewer))
	adminToken := seedUserAndLogin(t, router, users, "rule-admin@example.com", "admin-pass", string(auth.RoleAdmin))

	listed := doJSON(t, router, http.MethodGet, "/api/v1/detection-rules/", viewerToken, nil)
	if listed.Code != http.StatusOK {
		t.Fatalf("viewer list: status = %d, body = %s", listed.Code, listed.Body.String())
	}
	var page store.Page[detection.Rule]
	if err := json.Unmarshal(listed.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode rules: %v", err)
	}
	if page.Total < 2 {
		t.Fatalf("default rules = %d, want at least 2", page.Total)
	}

	viewerCreate := doJSON(t, router, http.MethodPost, "/api/v1/detection-rules/", viewerToken, map[string]interface{}{
		"name": "Viewer rule", "metric": "any", "operator": "abs_z_gte", "threshold": 5, "severity": "high", "cooldown_seconds": 60,
	})
	if viewerCreate.Code != http.StatusForbidden {
		t.Fatalf("viewer create: status = %d", viewerCreate.Code)
	}

	invalid := doJSON(t, router, http.MethodPost, "/api/v1/detection-rules/", adminToken, map[string]interface{}{
		"name": "Invalid rule", "metric": "any", "operator": "abs_z_gte", "threshold": 0, "severity": "high", "cooldown_seconds": 60,
	})
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid rule: status = %d, body = %s", invalid.Code, invalid.Body.String())
	}

	created := doJSON(t, router, http.MethodPost, "/api/v1/detection-rules/", adminToken, map[string]interface{}{
		"name": "Negative frequency shift", "description": "Detect negative frequency deviations", "metric": "frequency_hz", "operator": "z_lte", "threshold": 7, "severity": "critical", "cooldown_seconds": 300, "auto_incident": true,
	})
	if created.Code != http.StatusCreated {
		t.Fatalf("admin create: status = %d, body = %s", created.Code, created.Body.String())
	}
	var rule detection.Rule
	if err := json.Unmarshal(created.Body.Bytes(), &rule); err != nil {
		t.Fatalf("decode rule: %v", err)
	}
	if rule.ID == uuid.Nil || !rule.Enabled {
		t.Fatalf("unexpected rule: %+v", rule)
	}

	viewerUpdate := doJSON(t, router, http.MethodPatch, "/api/v1/detection-rules/"+rule.ID.String(), viewerToken, map[string]interface{}{"enabled": false})
	if viewerUpdate.Code != http.StatusForbidden {
		t.Fatalf("viewer update: status = %d", viewerUpdate.Code)
	}
	adminUpdate := doJSON(t, router, http.MethodPatch, "/api/v1/detection-rules/"+rule.ID.String(), adminToken, map[string]interface{}{"enabled": false})
	if adminUpdate.Code != http.StatusOK {
		t.Fatalf("admin update: status = %d, body = %s", adminUpdate.Code, adminUpdate.Body.String())
	}
	if err := json.Unmarshal(adminUpdate.Body.Bytes(), &rule); err != nil {
		t.Fatalf("decode updated rule: %v", err)
	}
	if rule.Enabled {
		t.Fatal("rule should be disabled")
	}
}

func TestProvisionedAdminMustChangePasswordBeforeUsingProtectedAPI(t *testing.T) {
	router, _, users := testRouter(t)
	hash, err := auth.HashPassword("admin")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	user, err := users.CreateProvisioned(context.Background(), "admin", hash, string(auth.RoleAdmin))
	if err != nil {
		t.Fatalf("CreateProvisioned: %v", err)
	}

	rec := doJSON(t, router, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"identifier": "admin", "password": "admin"})
	if rec.Code != http.StatusOK {
		t.Fatalf("login: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var loginResp struct {
		Token              string `json:"token"`
		MustChangePassword bool   `json:"must_change_password"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if !loginResp.MustChangePassword {
		t.Fatal("must_change_password = false, want true")
	}

	rec = doJSON(t, router, http.MethodGet, "/api/v1/satellites/", loginResp.Token, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("protected API before password change: status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	newPassword := "Stronger-Admin-Pass-9!"
	rec = doJSON(t, router, http.MethodPost, "/api/v1/auth/change-password", loginResp.Token, map[string]string{"current_password": "admin", "new_password": newPassword})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("change password: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, router, http.MethodGet, "/api/v1/satellites/", loginResp.Token, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("old token after password change: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	rec = doJSON(t, router, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"identifier": user.Username, "password": newPassword})
	if rec.Code != http.StatusOK {
		t.Fatalf("relogin: status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestAdminCanDisableAndSoftDeleteUser(t *testing.T) {
	router, _, users := testRouter(t)
	adminToken := seedUserAndLogin(t, router, users, "admin-lifecycle@example.com", "Correct-Horse-Battery-9!", "admin")
	user, err := users.Create(context.Background(), "target@example.com", "hash", "viewer")
	if err != nil {
		t.Fatalf("create target: %v", err)
	}

	rec := doJSON(t, router, http.MethodPatch, "/api/v1/users/"+user.ID.String()+"/status", adminToken, map[string]bool{"enabled": false})
	if rec.Code != http.StatusOK {
		t.Fatalf("disable user: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	updated, err := users.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetByID after disable: %v", err)
	}
	if updated.Enabled {
		t.Fatal("user remained enabled")
	}

	rec = doJSON(t, router, http.MethodDelete, "/api/v1/users/"+user.ID.String(), adminToken, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete user: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if _, err := users.GetByID(context.Background(), user.ID); err != store.ErrUserNotFound {
		t.Fatalf("GetByID after delete: err = %v, want ErrUserNotFound", err)
	}
}

func TestAuditorCanReadAuditLogButCannotManageUsers(t *testing.T) {
	router, _, users := testRouter(t)
	auditorToken := seedUserAndLogin(t, router, users, "auditor@example.com", "Correct-Horse-Battery-9!", "auditor")

	rec := doJSON(t, router, http.MethodGet, "/api/v1/audit-log", auditorToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("auditor GET /audit-log: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, router, http.MethodGet, "/api/v1/users/", auditorToken, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("auditor GET /users: status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestDisablingAndReenablingUserDoesNotRestoreOldToken(t *testing.T) {
	router, _, users := testRouter(t)
	adminToken := seedUserAndLogin(t, router, users, "admin-auth-version@example.com", "Correct-Horse-Battery-9!", "admin")
	viewer, err := users.Create(context.Background(), "viewer-auth-version@example.com", "", "viewer")
	if err != nil {
		t.Fatalf("create viewer: %v", err)
	}
	hash, err := auth.HashPassword("Correct-Horse-Battery-9!")
	if err != nil {
		t.Fatalf("hash viewer password: %v", err)
	}
	if err := users.UpdatePassword(context.Background(), viewer.ID, hash); err != nil {
		t.Fatalf("set viewer password: %v", err)
	}
	viewerToken := registerAndLogin_loginOnly(t, router, "viewer-auth-version@example.com", "Correct-Horse-Battery-9!")

	rec := doJSON(t, router, http.MethodGet, "/api/v1/satellites/", viewerToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("viewer before disable: status = %d", rec.Code)
	}

	rec = doJSON(t, router, http.MethodPatch, "/api/v1/users/"+viewer.ID.String()+"/status", adminToken, map[string]bool{"enabled": false})
	if rec.Code != http.StatusOK {
		t.Fatalf("disable viewer: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, router, http.MethodGet, "/api/v1/satellites/", viewerToken, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("disabled viewer old token: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	rec = doJSON(t, router, http.MethodPatch, "/api/v1/users/"+viewer.ID.String()+"/status", adminToken, map[string]bool{"enabled": true})
	if rec.Code != http.StatusOK {
		t.Fatalf("reenable viewer: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, router, http.MethodGet, "/api/v1/satellites/", viewerToken, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("reenabled viewer old token: status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAssetProgramCRUDUsesCapabilitiesAndAudit(t *testing.T) {
	router, pool, users := testRouter(t)
	adminToken := registerAndLogin(t, router, users, "asset-admin@example.com", "Correct-Horse-Battery-9!")

	createProgram := doJSON(t, router, http.MethodPost, "/api/v1/programs/", adminToken, map[string]string{"name": "Mission Alpha"})
	if createProgram.Code != http.StatusCreated {
		t.Fatalf("POST /programs: status=%d body=%s", createProgram.Code, createProgram.Body.String())
	}
	var program store.Program
	if err := json.Unmarshal(createProgram.Body.Bytes(), &program); err != nil {
		t.Fatal(err)
	}

	createAsset := doJSON(t, router, http.MethodPost, "/api/v1/assets/", adminToken, map[string]string{
		"program_id": program.ID.String(), "kind": "domain", "value": "ops.example.com",
	})
	if createAsset.Code != http.StatusCreated {
		t.Fatalf("POST /assets: status=%d body=%s", createAsset.Code, createAsset.Body.String())
	}

	listAssets := doJSON(t, router, http.MethodGet, "/api/v1/assets/?limit=50&offset=0", adminToken, nil)
	if listAssets.Code != http.StatusOK {
		t.Fatalf("GET /assets: status=%d body=%s", listAssets.Code, listAssets.Body.String())
	}
	var page store.Page[store.Asset]
	if err := json.Unmarshal(listAssets.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("asset count=%d want 1", len(page.Items))
	}
	assetID := page.Items[0].ID

	updateProgram := doJSON(t, router, http.MethodPut, "/api/v1/programs/"+program.ID.String(), adminToken, map[string]string{"name": "Mission Beta"})
	if updateProgram.Code != http.StatusOK {
		t.Fatalf("PUT /programs/{id}: status=%d body=%s", updateProgram.Code, updateProgram.Body.String())
	}

	updateAsset := doJSON(t, router, http.MethodPut, "/api/v1/assets/"+assetID.String(), adminToken, map[string]string{
		"program_id": program.ID.String(), "kind": "ip", "value": "198.51.100.42",
	})
	if updateAsset.Code != http.StatusOK {
		t.Fatalf("PUT /assets/{id}: status=%d body=%s", updateAsset.Code, updateAsset.Body.String())
	}
	var updated store.Asset
	if err := json.Unmarshal(updateAsset.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Kind != "ip" || updated.Value != "198.51.100.42" {
		t.Fatalf("updated asset=%#v", updated)
	}

	viewerHash, err := auth.HashPassword("Correct-Horse-Battery-9!")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := users.Create(context.Background(), "asset-viewer@example.com", viewerHash, string(auth.RoleViewer)); err != nil {
		t.Fatal(err)
	}
	viewerToken := registerAndLogin_loginOnly(t, router, "asset-viewer@example.com", "Correct-Horse-Battery-9!")
	denied := doJSON(t, router, http.MethodDelete, "/api/v1/assets/"+assetID.String(), viewerToken, nil)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("viewer DELETE /assets/{id}: status=%d want 403", denied.Code)
	}

	deleteAsset := doJSON(t, router, http.MethodDelete, "/api/v1/assets/"+assetID.String(), adminToken, nil)
	if deleteAsset.Code != http.StatusNoContent {
		t.Fatalf("DELETE /assets/{id}: status=%d body=%s", deleteAsset.Code, deleteAsset.Body.String())
	}
	deleteProgram := doJSON(t, router, http.MethodDelete, "/api/v1/programs/"+program.ID.String(), adminToken, nil)
	if deleteProgram.Code != http.StatusNoContent {
		t.Fatalf("DELETE /programs/{id}: status=%d body=%s", deleteProgram.Code, deleteProgram.Body.String())
	}

	var auditCount int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM audit_log
		WHERE action = ANY($1::text[])
	`, []string{
		"program.created", "asset.upserted", "program.updated", "asset.updated", "asset.deleted", "program.deleted",
	}).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 6 {
		t.Fatalf("asset/program mutation audit count=%d want 6", auditCount)
	}
}

func TestPlatformOrganizationUpdateUsesPlatformBoundaryAndAudits(t *testing.T) {
	router, pool, users := testRouter(t)
	platformToken := seedUserAndLogin(t, router, users, "platform-org-admin@example.com", "Correct-Horse-Battery-9!", string(auth.RoleAdmin))
	platformUser, err := users.GetByEmail(context.Background(), "platform-org-admin@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if err := users.GrantPlatformAdministrator(context.Background(), platformUser.ID); err != nil {
		t.Fatalf("grant platform admin: %v", err)
	}

	create := doJSON(t, router, http.MethodPost, "/api/v1/organizations/", platformToken, map[string]string{
		"slug": "mission-alpha", "name": "Mission Alpha",
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("POST /organizations: status=%d body=%s", create.Code, create.Body.String())
	}
	var organization store.Organization
	if err := json.Unmarshal(create.Body.Bytes(), &organization); err != nil {
		t.Fatal(err)
	}

	update := doJSON(t, router, http.MethodPut, "/api/v1/organizations/"+organization.ID.String(), platformToken, map[string]string{
		"slug": "mission-beta", "name": "Mission Beta",
	})
	if update.Code != http.StatusOK {
		t.Fatalf("PUT /organizations/{id}: status=%d body=%s", update.Code, update.Body.String())
	}
	var updated store.Organization
	if err := json.Unmarshal(update.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Slug != "mission-beta" || updated.Name != "Mission Beta" || updated.ID != organization.ID {
		t.Fatalf("updated organization=%#v", updated)
	}

	secondCreate := doJSON(t, router, http.MethodPost, "/api/v1/organizations/", platformToken, map[string]string{
		"slug": "mission-gamma", "name": "Mission Gamma",
	})
	if secondCreate.Code != http.StatusCreated {
		t.Fatalf("create second organization: status=%d body=%s", secondCreate.Code, secondCreate.Body.String())
	}
	var second store.Organization
	if err := json.Unmarshal(secondCreate.Body.Bytes(), &second); err != nil {
		t.Fatal(err)
	}
	conflict := doJSON(t, router, http.MethodPut, "/api/v1/organizations/"+second.ID.String(), platformToken, map[string]string{
		"slug": "mission-beta", "name": "Conflicting Mission",
	})
	if conflict.Code != http.StatusConflict {
		t.Fatalf("duplicate organization slug update: status=%d want 409 body=%s", conflict.Code, conflict.Body.String())
	}

	tenantAdminToken := seedUserAndLogin(t, router, users, "tenant-only-admin@example.com", "Correct-Horse-Battery-9!", string(auth.RoleAdmin))
	denied := doJSON(t, router, http.MethodPut, "/api/v1/organizations/"+organization.ID.String(), tenantAdminToken, map[string]string{
		"slug": "forbidden-change", "name": "Forbidden Change",
	})
	if denied.Code != http.StatusForbidden {
		t.Fatalf("tenant admin PUT /organizations/{id}: status=%d want 403", denied.Code)
	}

	var auditCount int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_log WHERE organization_id=$1 AND action='organization.updated'`, organization.ID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 {
		t.Fatalf("organization.updated audit count=%d want 1", auditCount)
	}
}

func TestSatelliteUpdateUsesOrbitalManageAndAudits(t *testing.T) {
	router, pool, users := testRouter(t)
	operatorToken := seedUserAndLogin(t, router, users, "sat-update-operator@example.com", "Correct-Horse-Battery-9!", "operator")
	viewerToken := seedUserAndLogin(t, router, users, "sat-update-viewer@example.com", "Correct-Horse-Battery-9!", "viewer")

	line1 := "1 25544U 98067A   26239.14379413  .00008717  00000+0  16243-3 0  9995"
	line2 := "2 25544  51.6326 308.8886 0007697  87.8986 272.2885 15.49649978582754"
	rec := doJSON(t, router, http.MethodPost, "/api/v1/satellites/", operatorToken, map[string]interface{}{
		"norad_id": 25544, "name": "ISS", "tle_line1": line1, "tle_line2": line2,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("seed satellite: status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, router, http.MethodGet, "/api/v1/satellites/", operatorToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list satellites: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var list []struct {
		ID uuid.UUID `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list) != 1 {
		t.Fatalf("satellite list = %+v, err = %v", list, err)
	}
	satelliteID := list[0].ID
	updatedLine1 := "1 25544U 98067A   26240.00000000  .00008717  00000+0  16243-3 0  9996"
	body := map[string]interface{}{
		"norad_id": 25544, "name": "ISS (ZARYA)", "tle_line1": updatedLine1, "tle_line2": line2,
	}

	rec = doJSON(t, router, http.MethodPut, "/api/v1/satellites/"+satelliteID.String(), viewerToken, body)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("viewer PUT /satellites/{id}: status = %d, want %d; body=%s", rec.Code, http.StatusForbidden, rec.Body.String())
	}

	rec = doJSON(t, router, http.MethodPut, "/api/v1/satellites/"+satelliteID.String(), operatorToken, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("operator PUT /satellites/{id}: status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var updated store.Satellite
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated satellite: %v", err)
	}
	if updated.ID != satelliteID || updated.Name != "ISS (ZARYA)" || updated.TLELine1 != updatedLine1 {
		t.Fatalf("updated satellite = %+v", updated)
	}

	var auditCount int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM audit_log WHERE action='satellite.updated' AND details->>'satellite_id'=$1`, satelliteID.String()).Scan(&auditCount); err != nil {
		t.Fatalf("query satellite.updated audit: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("satellite.updated audit count = %d, want 1", auditCount)
	}

	rec = doJSON(t, router, http.MethodPut, "/api/v1/satellites/00000000-0000-0000-0000-000000000000", operatorToken, body)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("PUT unknown satellite: status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestSatelliteUpdateRejectsDuplicateNoradID(t *testing.T) {
	router, _, users := testRouter(t)
	token := seedUserAndLogin(t, router, users, "sat-conflict@example.com", "Correct-Horse-Battery-9!", "operator")
	line1a := "1 25544U 98067A   26239.14379413  .00008717  00000+0  16243-3 0  9995"
	line2a := "2 25544  51.6326 308.8886 0007697  87.8986 272.2885 15.49649978582754"
	line1b := "1 40967U 15058A   26239.00000000  .00000000  00000+0  00000+0 0  9991"
	line2b := "2 40967   0.0170  88.0000 0001000  90.0000 270.0000  1.00270000    01"
	for _, body := range []map[string]interface{}{
		{"norad_id": 25544, "name": "ISS", "tle_line1": line1a, "tle_line2": line2a},
		{"norad_id": 40967, "name": "SAT-B", "tle_line1": line1b, "tle_line2": line2b},
	} {
		rec := doJSON(t, router, http.MethodPost, "/api/v1/satellites/", token, body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("seed satellite: status = %d, body = %s", rec.Code, rec.Body.String())
		}
	}

	rec := doJSON(t, router, http.MethodGet, "/api/v1/satellites/", token, nil)
	var list []store.Satellite
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list) != 2 {
		t.Fatalf("satellite list = %+v, err=%v", list, err)
	}
	var target store.Satellite
	for _, sat := range list {
		if sat.NoradID == 40967 {
			target = sat
		}
	}
	rec = doJSON(t, router, http.MethodPut, "/api/v1/satellites/"+target.ID.String(), token, map[string]interface{}{
		"norad_id": 25544, "name": "Conflict", "tle_line1": line1b, "tle_line2": line2b,
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate NORAD update: status = %d, want %d; body=%s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}
