package keyprovider

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ProviderInfo exposes non-secret operational metadata for platform UI/health.
type ProviderInfo struct {
	Name      string `json:"name"`
	External  bool   `json:"external"`
	Endpoint  string `json:"endpoint,omitempty"`
	Algorithm string `json:"algorithm,omitempty"`
	ActiveKID string `json:"active_kid,omitempty"`
}

type InfoProvider interface{ ProviderInfo() ProviderInfo }

type ProviderHealth struct {
	Healthy   bool      `json:"healthy"`
	CheckedAt time.Time `json:"checked_at"`
	Error     string    `json:"error,omitempty"`
}

type HealthProvider interface {
	Health(context.Context) ProviderHealth
}

// RemoteProvider delegates signing to an HTTPS service. Private key bytes never
// enter the Lyrahcle backend process. A Vault/KMS/HSM adapter can implement the
// tiny HTTP contract documented in docs/key-management.md.
type RemoteProvider struct {
	endpoint  string
	token     string
	activeKID string
	algorithm string
	client    *http.Client
}

type remoteSigningKey struct {
	provider *RemoteProvider
	kid      string
}

func NewRemoteProvider(endpoint, token, activeKID, algorithm string, timeout time.Duration) (*RemoteProvider, error) {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	activeKID = strings.TrimSpace(activeKID)
	algorithm = strings.TrimSpace(algorithm)
	if endpoint == "" {
		return nil, errors.New("keyprovider: remote signer endpoint is required")
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return nil, errors.New("keyprovider: remote signer endpoint must be an absolute HTTPS URL")
	}
	if err := ValidateKID(activeKID); err != nil {
		return nil, fmt.Errorf("keyprovider: invalid remote signer active kid: %w", err)
	}
	if algorithm == "" {
		algorithm = "Ed25519"
	}
	if algorithm != "Ed25519" {
		return nil, fmt.Errorf("keyprovider: unsupported remote signer algorithm %q", algorithm)
	}
	if timeout <= 0 || timeout > 30*time.Second {
		timeout = 5 * time.Second
	}
	return &RemoteProvider{endpoint: endpoint, token: strings.TrimSpace(token), activeKID: activeKID, algorithm: algorithm, client: &http.Client{Timeout: timeout}}, nil
}

func (p *RemoteProvider) ActiveSigningKey(context.Context) (SigningKey, error) {
	if p == nil {
		return nil, ErrKeyNotFound
	}
	return &remoteSigningKey{provider: p, kid: p.activeKID}, nil
}
func (p *RemoteProvider) SigningKey(_ context.Context, kid string) (SigningKey, error) {
	if p == nil || strings.TrimSpace(kid) != p.activeKID {
		return nil, ErrKeyNotFound
	}
	return &remoteSigningKey{provider: p, kid: p.activeKID}, nil
}
func (p *RemoteProvider) ProviderInfo() ProviderInfo {
	if p == nil {
		return ProviderInfo{Name: "external-remote", External: true}
	}
	return ProviderInfo{Name: "external-remote", External: true, Endpoint: p.endpoint, Algorithm: p.algorithm, ActiveKID: p.activeKID}
}
func (p *RemoteProvider) Health(ctx context.Context) ProviderHealth {
	h := ProviderHealth{CheckedAt: time.Now().UTC()}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint+"/health", nil)
	if err != nil {
		h.Error = err.Error()
		return h
	}
	p.authorize(req)
	resp, err := p.client.Do(req)
	if err != nil {
		h.Error = "remote signer unavailable"
		return h
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		h.Error = fmt.Sprintf("remote signer health returned HTTP %d", resp.StatusCode)
		return h
	}
	h.Healthy = true
	return h
}
func (k *remoteSigningKey) KID() string       { return k.kid }
func (k *remoteSigningKey) Algorithm() string { return k.provider.algorithm }
func (k *remoteSigningKey) Sign(ctx context.Context, payload []byte) ([]byte, error) {
	body, _ := json.Marshal(map[string]string{"kid": k.kid, "algorithm": k.provider.algorithm, "payload_b64": base64.StdEncoding.EncodeToString(payload)})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, k.provider.endpoint+"/v1/sign", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	k.provider.authorize(req)
	resp, err := k.provider.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("keyprovider: remote sign: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("keyprovider: remote signer returned HTTP %d", resp.StatusCode)
	}
	var out struct {
		Signature string `json:"signature_b64"`
		KID       string `json:"kid"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, errors.New("keyprovider: invalid remote signer response")
	}
	if out.KID != "" && out.KID != k.kid {
		return nil, errors.New("keyprovider: remote signer returned unexpected kid")
	}
	sig, err := base64.StdEncoding.DecodeString(out.Signature)
	if err != nil || len(sig) == 0 {
		return nil, errors.New("keyprovider: invalid remote signature")
	}
	return sig, nil
}
func (p *RemoteProvider) authorize(req *http.Request) {
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}
}
