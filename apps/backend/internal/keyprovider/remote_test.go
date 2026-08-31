package keyprovider

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRemoteProviderSignsWithoutLocalPrivateKey(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	kid := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	token := "service-secret"
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", 401)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/v1/sign", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", 401)
			return
		}
		var in struct {
			KID     string `json:"kid"`
			Payload string `json:"payload_b64"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad", 400)
			return
		}
		payload, _ := base64.StdEncoding.DecodeString(in.Payload)
		json.NewEncoder(w).Encode(map[string]string{"kid": in.KID, "signature_b64": base64.StdEncoding.EncodeToString(ed25519.Sign(priv, payload))})
	})
	ts := httptest.NewTLSServer(mux)
	defer ts.Close()
	p, err := NewRemoteProvider(ts.URL, token, kid, "Ed25519", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	p.client = ts.Client()
	key, err := p.ActiveSigningKey(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("signed remotely")
	sig, err := key.Sign(context.Background(), payload)
	if err != nil {
		t.Fatal(err)
	}
	if !ed25519.Verify(pub, payload, sig) {
		t.Fatal("signature did not verify")
	}
	if health := p.Health(context.Background()); !health.Healthy {
		t.Fatalf("expected healthy remote provider: %+v", health)
	}
	info := p.ProviderInfo()
	if !info.External || info.ActiveKID != kid {
		t.Fatalf("unexpected provider info: %+v", info)
	}
}

func TestRemoteProviderRequiresHTTPS(t *testing.T) {
	kid := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if _, err := NewRemoteProvider("http://signer.internal", "", kid, "Ed25519", time.Second); err == nil {
		t.Fatal("expected HTTP endpoint rejection")
	}
}

func TestRemoteProviderRejectsInvalidConfiguration(t *testing.T) {
	validKID := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	cases := []struct {
		name, endpoint, kid, algorithm string
	}{
		{"missing endpoint", "", validKID, "Ed25519"},
		{"relative endpoint", "/signer", validKID, "Ed25519"},
		{"invalid kid", "https://signer.internal", "../bad", "Ed25519"},
		{"unsupported algorithm", "https://signer.internal", validKID, "RSA"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewRemoteProvider(tc.endpoint, "", tc.kid, tc.algorithm, time.Second); err == nil {
				t.Fatal("expected invalid remote provider configuration to be rejected")
			}
		})
	}
}

func TestRemoteProviderRejectsSignerErrorsAndUnexpectedKID(t *testing.T) {
	kid := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	otherKID := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"http error", func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", http.StatusBadGateway) }},
		{"invalid json", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("{")) }},
		{"unexpected kid", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]string{"kid": otherKID, "signature_b64": base64.StdEncoding.EncodeToString([]byte("sig"))})
		}},
		{"invalid signature", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]string{"kid": kid, "signature_b64": "%%%"})
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewTLSServer(tc.handler)
			defer ts.Close()
			p, err := NewRemoteProvider(ts.URL, "", kid, "Ed25519", time.Second)
			if err != nil {
				t.Fatal(err)
			}
			p.client = ts.Client()
			key, err := p.ActiveSigningKey(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if _, err := key.Sign(context.Background(), []byte("payload")); err == nil {
				t.Fatal("expected remote signer response to be rejected")
			}
		})
	}
}

func TestRemoteProviderHealthReportsNonSuccess(t *testing.T) {
	kid := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "degraded", http.StatusServiceUnavailable)
	}))
	defer ts.Close()
	p, err := NewRemoteProvider(ts.URL, "", kid, "Ed25519", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	p.client = ts.Client()
	health := p.Health(context.Background())
	if health.Healthy || health.Error == "" {
		t.Fatalf("health=%+v want unhealthy with error", health)
	}
}
