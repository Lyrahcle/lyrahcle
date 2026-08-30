package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStrictCORSAllowsExactOrigin(t *testing.T) {
	handler := strictCORS([]string{"https://console.example.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	req.Header.Set("Origin", "https://console.example.com")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "https://console.example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
}

func TestStrictCORSRejectsDisallowedPreflight(t *testing.T) {
	handler := strictCORS([]string{"https://console.example.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not receive denied preflight")
	}))
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/ping", nil)
	req.Header.Set("Origin", "https://evil.example")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", res.Code)
	}
	if res.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("denied origin must not receive CORS allow header")
	}
}

func TestStrictCORSPassesSameOriginRequestsWithoutHeaders(t *testing.T) {
	handler := strictCORS(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want 418", res.Code)
	}
	if res.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("same-origin request should not need CORS headers")
	}
}
