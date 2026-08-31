package secretprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ErrNotFound = errors.New("secretprovider: secret not found")

type Provider interface {
	Get(ctx context.Context, key string) (string, error)
	Info() Info
	Health(ctx context.Context) Health
}

type Info struct {
	Name     string `json:"name"`
	External bool   `json:"external"`
	Location string `json:"location,omitempty"`
}

type Health struct {
	Healthy bool   `json:"healthy"`
	Message string `json:"message,omitempty"`
}

type EnvProvider struct{}

func (EnvProvider) Get(_ context.Context, key string) (string, error) {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value, nil
	}
	return "", ErrNotFound
}
func (EnvProvider) Info() Info {
	return Info{Name: "env", External: false, Location: "process environment"}
}
func (EnvProvider) Health(context.Context) Health { return Health{Healthy: true} }

type FileProvider struct{ Dir string }

func (p FileProvider) Get(_ context.Context, key string) (string, error) {
	if !validKey(key) {
		return "", fmt.Errorf("secretprovider: invalid key")
	}
	path := filepath.Join(p.Dir, key)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("secretprovider: stat %s: %w", key, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("secretprovider: symlink secret files are not allowed")
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("secretprovider: secret must be a regular file")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("secretprovider: secret file %s must not be group/world accessible", key)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("secretprovider: read %s: %w", key, err)
	}
	value := strings.TrimSpace(string(b))
	if value == "" {
		return "", ErrNotFound
	}
	return value, nil
}
func (p FileProvider) Info() Info { return Info{Name: "file", External: false, Location: p.Dir} }
func (p FileProvider) Health(_ context.Context) Health {
	info, err := os.Stat(p.Dir)
	if err != nil || !info.IsDir() {
		return Health{Healthy: false, Message: "secret directory unavailable"}
	}
	return Health{Healthy: true}
}

type RemoteProvider struct {
	Endpoint string
	Token    string
	Client   *http.Client
}

func (p RemoteProvider) client() *http.Client {
	if p.Client != nil {
		return p.Client
	}
	return &http.Client{Timeout: 5 * time.Second}
}
func (p RemoteProvider) validate() error {
	u, err := url.Parse(p.Endpoint)
	if err != nil || !strings.EqualFold(u.Scheme, "https") || u.Host == "" {
		return errors.New("secretprovider: remote endpoint must be absolute HTTPS")
	}
	return nil
}
func (p RemoteProvider) Get(ctx context.Context, key string) (string, error) {
	if err := p.validate(); err != nil {
		return "", err
	}
	if !validKey(key) {
		return "", fmt.Errorf("secretprovider: invalid key")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(p.Endpoint, "/")+"/v1/secrets/"+url.PathEscape(key), nil)
	if err != nil {
		return "", err
	}
	if p.Token != "" {
		req.Header.Set("Authorization", "Bearer "+p.Token)
	}
	resp, err := p.client().Do(req)
	if err != nil {
		return "", fmt.Errorf("secretprovider: remote get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("secretprovider: remote get returned %s", resp.Status)
	}
	var payload struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&payload); err != nil {
		return "", fmt.Errorf("secretprovider: decode response: %w", err)
	}
	if payload.Value == "" {
		return "", ErrNotFound
	}
	return payload.Value, nil
}
func (p RemoteProvider) Info() Info {
	return Info{Name: "remote", External: true, Location: p.Endpoint}
}
func (p RemoteProvider) Health(ctx context.Context) Health {
	if err := p.validate(); err != nil {
		return Health{Healthy: false, Message: err.Error()}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(p.Endpoint, "/")+"/healthz", nil)
	if err != nil {
		return Health{Healthy: false, Message: "invalid health request"}
	}
	if p.Token != "" {
		req.Header.Set("Authorization", "Bearer "+p.Token)
	}
	resp, err := p.client().Do(req)
	if err != nil {
		return Health{Healthy: false, Message: "remote provider unreachable"}
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Health{Healthy: false, Message: "remote provider health check failed"}
	}
	return Health{Healthy: true}
}

func validKey(key string) bool {
	if key == "" {
		return false
	}
	for _, r := range key {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return false
	}
	return true
}
