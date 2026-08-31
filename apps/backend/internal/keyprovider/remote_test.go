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
