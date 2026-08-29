package identity

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	BackendURL      string
	AgentID         string
	EnrollmentToken string
	ServerCAPath    string
	CertificatePath string
	PrivateKeyPath  string
	AgentVersion    string
	FirmwareVersion string
	RotateBefore    time.Duration
}

type enrollmentResponse struct {
	AgentID        string    `json:"agent_id"`
	CertificatePEM string    `json:"certificate_pem"`
	ExpiresAt      time.Time `json:"expires_at"`
}

type rotateResponse struct {
	CertificatePEM string    `json:"certificate_pem"`
	ExpiresAt      time.Time `json:"expires_at"`
}

func Ensure(cfg Config) error {
	_, err := Maintain(cfg)
	return err
}

func Maintain(cfg Config) (bool, error) {
	if cfg.CertificatePath == "" || cfg.PrivateKeyPath == "" {
		return false, nil
	}
	certExists := fileExists(cfg.CertificatePath)
	keyExists := fileExists(cfg.PrivateKeyPath)
	if certExists != keyExists {
		return false, errors.New("identity: certificate and private key must exist together")
	}
	if !certExists {
		if cfg.EnrollmentToken == "" {
			return false, nil
		}
		if err := enroll(cfg); err != nil {
			return false, err
		}
		return true, nil
	}
	if cfg.RotateBefore <= 0 {
		cfg.RotateBefore = 24 * time.Hour
	}
	notAfter, err := certificateExpiry(cfg.CertificatePath)
	if err != nil {
		return false, err
	}
	if time.Until(notAfter) > cfg.RotateBefore {
		return false, nil
	}
	if err := rotate(cfg); err != nil {
		return false, err
	}
	return true, nil
}

func enroll(cfg Config) error {
	if err := requireHTTPS(cfg.BackendURL); err != nil {
		return err
	}
	keyPEM, csrPEM, err := generateKeyAndCSR(cfg.AgentID)
	if err != nil {
		return err
	}
	client, err := newHTTPClient(cfg.ServerCAPath, "", "")
	if err != nil {
		return err
	}
	payload := map[string]string{
		"token":            cfg.EnrollmentToken,
		"csr_pem":          csrPEM,
		"agent_version":    cfg.AgentVersion,
		"firmware_version": cfg.FirmwareVersion,
	}
	var response enrollmentResponse
	if err := postJSON(client, cfg.BackendURL+"/api/v1/agents/enroll", payload, &response); err != nil {
		return err
	}
	if response.AgentID != cfg.AgentID {
		return errors.New("identity: enrollment response agent_id mismatch")
	}
	return persistIdentity(cfg.PrivateKeyPath, cfg.CertificatePath, keyPEM, response.CertificatePEM)
}

func rotate(cfg Config) error {
	if err := requireHTTPS(cfg.BackendURL); err != nil {
		return err
	}
	keyPEM, csrPEM, err := generateKeyAndCSR(cfg.AgentID)
	if err != nil {
		return err
	}
	client, err := newHTTPClient(cfg.ServerCAPath, cfg.CertificatePath, cfg.PrivateKeyPath)
	if err != nil {
		return err
	}
	var response rotateResponse
	if err := postJSON(client, cfg.BackendURL+"/api/v1/agents/me/certificate/rotate", map[string]string{"csr_pem": csrPEM}, &response); err != nil {
		return err
	}
	return persistIdentity(cfg.PrivateKeyPath, cfg.CertificatePath, keyPEM, response.CertificatePEM)
}

func requireHTTPS(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" {
		return errors.New("identity: mTLS enrollment and rotation require an https backend URL")
	}
	return nil
}

func generateKeyAndCSR(agentID string) ([]byte, string, error) {
	agentID = strings.TrimSpace(agentID)
	if agentID == "" {
		return nil, "", errors.New("identity: agent_id is required")
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, "", err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, "", err
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject: pkix.Name{CommonName: agentID, Organization: []string{"Lyrahcle SDR Agent"}},
	}, key)
	if err != nil {
		return nil, "", err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	csrPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER}))
	return keyPEM, csrPEM, nil
}

func newHTTPClient(serverCAPath, certPath, keyPath string) (*http.Client, error) {
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	if serverCAPath != "" {
		pemBytes, err := os.ReadFile(serverCAPath)
		if err != nil {
			return nil, fmt.Errorf("identity: read server CA: %w", err)
		}
		roots, err := x509.SystemCertPool()
		if err != nil || roots == nil {
			roots = x509.NewCertPool()
		}
		if !roots.AppendCertsFromPEM(pemBytes) {
			return nil, errors.New("identity: server CA is not valid PEM")
		}
		tlsConfig.RootCAs = roots
	}
	if certPath != "" || keyPath != "" {
		if certPath == "" || keyPath == "" {
			return nil, errors.New("identity: client certificate and key must be configured together")
		}
		certificate, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, fmt.Errorf("identity: load client identity: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{certificate}
	}
	return &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{TLSClientConfig: tlsConfig}}, nil
}

func postJSON(client *http.Client, endpoint string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("identity: backend returned status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func persistIdentity(keyPath, certPath string, keyPEM []byte, certPEM string) error {
	if _, err := parseCertificate([]byte(certPEM)); err != nil {
		return err
	}
	if _, err := tls.X509KeyPair([]byte(certPEM), keyPEM); err != nil {
		return fmt.Errorf("identity: certificate and private key do not match: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(certPath), 0o700); err != nil {
		return err
	}

	keyNew := keyPath + ".new"
	certNew := certPath + ".new"
	keyBackup := keyPath + ".bak"
	certBackup := certPath + ".bak"
	cleanup := func() {
		_ = os.Remove(keyNew)
		_ = os.Remove(certNew)
	}
	defer cleanup()

	if err := os.WriteFile(keyNew, keyPEM, 0o600); err != nil {
		return err
	}
	if err := os.Chmod(keyNew, 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(certNew, []byte(certPEM), 0o644); err != nil {
		return err
	}
	if err := os.Chmod(certNew, 0o644); err != nil {
		return err
	}

	hadKey := fileExists(keyPath)
	hadCert := fileExists(certPath)
	_ = os.Remove(keyBackup)
	_ = os.Remove(certBackup)
	if hadKey {
		if err := os.Rename(keyPath, keyBackup); err != nil {
			return err
		}
	}
	if hadCert {
		if err := os.Rename(certPath, certBackup); err != nil {
			if hadKey {
				_ = os.Rename(keyBackup, keyPath)
			}
			return err
		}
	}

	rollback := func() {
		_ = os.Remove(keyPath)
		_ = os.Remove(certPath)
		if hadKey {
			_ = os.Rename(keyBackup, keyPath)
		}
		if hadCert {
			_ = os.Rename(certBackup, certPath)
		}
	}
	if err := os.Rename(keyNew, keyPath); err != nil {
		rollback()
		return err
	}
	if err := os.Rename(certNew, certPath); err != nil {
		rollback()
		return err
	}
	_ = os.Remove(keyBackup)
	_ = os.Remove(certBackup)
	return nil
}

func certificateExpiry(path string) (time.Time, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}, err
	}
	cert, err := parseCertificate(pemBytes)
	if err != nil {
		return time.Time{}, err
	}
	return cert.NotAfter, nil
}

func parseCertificate(pemBytes []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("identity: invalid certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	if !time.Now().Before(cert.NotAfter) {
		return nil, errors.New("identity: certificate is expired")
	}
	return cert, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
