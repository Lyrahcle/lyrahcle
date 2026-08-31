package uplink

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/capture"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/soapy"
)

type Client struct {
	BackendURL  string
	AgentID     string
	APIKey      string
	AuthToken   string
	mu          sync.RWMutex
	satelliteID string
	http        *http.Client
	transport   *http.Transport
	mtls        bool
}

type RemoteConfigEnvelope struct {
	KID       string `json:"kid,omitempty"`
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
	Checksum  string `json:"checksum_sha256"`
}

type RemoteConfigAck struct {
	Version  int64  `json:"version"`
	Status   string `json:"status"`
	Message  string `json:"message"`
	Checksum string `json:"checksum_sha256"`
}

type deviceCapabilitiesReport struct {
	Architecture            string   `json:"architecture,omitempty"`
	OS                      string   `json:"os,omitempty"`
	DriverVersion           string   `json:"driver_version,omitempty"`
	FirmwareVersion         string   `json:"firmware_version,omitempty"`
	FrequencyMinHz          int64    `json:"frequency_min_hz,omitempty"`
	FrequencyMaxHz          int64    `json:"frequency_max_hz,omitempty"`
	SampleRatesHz           []int64  `json:"sample_rates_hz,omitempty"`
	BandwidthsHz            []int64  `json:"bandwidths_hz,omitempty"`
	GainElements            []string `json:"gain_elements,omitempty"`
	Channels                int      `json:"channels,omitempty"`
	GPS                     bool     `json:"gps"`
	PPS                     bool     `json:"pps"`
	TemperatureC            *float64 `json:"temperature_c,omitempty"`
	SampleDrops             int64    `json:"sample_drops"`
	IntegrityManifestSHA256 string   `json:"integrity_manifest_sha256,omitempty"`
	IntegrityMachineIDHash  string   `json:"integrity_machine_id_hash,omitempty"`
	IntegrityBootID         string   `json:"integrity_boot_id,omitempty"`
	IntegrityFileCount      int      `json:"integrity_file_count,omitempty"`
}

type deviceReport struct {
	Driver       string                   `json:"driver"`
	Hardware     string                   `json:"hardware"`
	Manufacturer string                   `json:"manufacturer"`
	Product      string                   `json:"product"`
	Serial       string                   `json:"serial"`
	RX           bool                     `json:"rx"`
	TX           bool                     `json:"tx"`
	Capabilities deviceCapabilitiesReport `json:"capabilities"`
}

func New(backendURL, agentID, apiKey, authToken, caCertPath string) (*Client, error) {
	return NewWithIdentity(backendURL, agentID, apiKey, authToken, caCertPath, "", "")
}

func NewWithIdentity(backendURL, agentID, apiKey, authToken, caCertPath, clientCertPath, clientKeyPath string) (*Client, error) {
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	if caCertPath != "" {
		certPEM, err := os.ReadFile(caCertPath)
		if err != nil {
			return nil, fmt.Errorf("uplink: read CA certificate: %w", err)
		}
		roots, err := x509.SystemCertPool()
		if err != nil || roots == nil {
			roots = x509.NewCertPool()
		}
		if !roots.AppendCertsFromPEM(certPEM) {
			return nil, fmt.Errorf("uplink: CA certificate is not valid PEM")
		}
		tlsConfig.RootCAs = roots
	}

	mtls := false
	if clientCertPath != "" || clientKeyPath != "" {
		if clientCertPath == "" || clientKeyPath == "" {
			return nil, fmt.Errorf("uplink: client certificate and key must be configured together")
		}
		if _, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath); err != nil {
			return nil, fmt.Errorf("uplink: load client identity: %w", err)
		}
		tlsConfig.GetClientCertificate = func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			certificate, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
			if err != nil {
				return nil, err
			}
			return &certificate, nil
		}
		mtls = true
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}
	return &Client{
		BackendURL: backendURL,
		AgentID:    agentID,
		APIKey:     apiKey,
		AuthToken:  authToken,
		http:       &http.Client{Timeout: 5 * time.Second, Transport: transport},
		transport:  transport,
		mtls:       mtls,
	}, nil
}

func (c *Client) Send(reading capture.Reading) error {
	payload := struct {
		AgentID     string  `json:"agent_id"`
		SatelliteID string  `json:"satellite_id,omitempty"`
		FrequencyHz int64   `json:"frequency_hz"`
		PowerDbm    float64 `json:"power_dbm"`
		CapturedAt  string  `json:"captured_at"`
	}{
		AgentID:     c.AgentID,
		SatelliteID: c.currentSatelliteID(),
		FrequencyHz: reading.FrequencyHz,
		PowerDbm:    reading.PowerDbm,
		CapturedAt:  reading.CapturedAt.Format(time.RFC3339),
	}
	return c.postJSON("/api/v1/rf/ingest", payload)
}

func (c *Client) ReportDevice(info *soapy.DeviceInfo) error {
	if info == nil {
		return nil
	}

	payload := deviceReport{
		Driver:       info.Driver,
		Hardware:     info.Hardware,
		Manufacturer: info.Manufacturer,
		Product:      info.Product,
		Serial:       info.Serial,
		RX:           info.RX,
		TX:           info.TX,
		Capabilities: deviceCapabilitiesReport{
			Architecture: info.Capabilities.Architecture, OS: info.Capabilities.OS, DriverVersion: info.Capabilities.DriverVersion,
			FirmwareVersion: info.Capabilities.FirmwareVersion, FrequencyMinHz: info.Capabilities.FrequencyMinHz,
			FrequencyMaxHz: info.Capabilities.FrequencyMaxHz, SampleRatesHz: info.Capabilities.SampleRatesHz,
			BandwidthsHz: info.Capabilities.BandwidthsHz, GainElements: info.Capabilities.GainElements, Channels: info.Capabilities.Channels,
			GPS: info.Capabilities.GPS, PPS: info.Capabilities.PPS, TemperatureC: info.Capabilities.TemperatureC, SampleDrops: info.Capabilities.SampleDrops,
			IntegrityManifestSHA256: info.Capabilities.IntegrityManifestSHA256, IntegrityMachineIDHash: info.Capabilities.IntegrityMachineIDHash, IntegrityBootID: info.Capabilities.IntegrityBootID, IntegrityFileCount: info.Capabilities.IntegrityFileCount,
		},
	}
	return c.postJSON("/api/v1/agents/me/device", payload)
}

func (c *Client) postJSON(path string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.BackendURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyAuth(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("uplink: backend returned status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) applyAuth(req *http.Request) {
	if c.mtls {
		return
	}
	if c.APIKey != "" {
		req.Header.Set("X-Agent-Key", c.APIKey)
		return
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}
}

func (c *Client) ReloadIdentity() {
	if c.transport != nil {
		c.transport.CloseIdleConnections()
	}
}

func (c *Client) FetchRemoteConfig(afterVersion int64) (RemoteConfigEnvelope, bool, error) {
	if !c.mtls {
		return RemoteConfigEnvelope{}, false, fmt.Errorf("uplink: remote configuration requires mTLS")
	}
	endpoint, err := url.Parse(c.BackendURL + "/api/v1/agents/me/config")
	if err != nil {
		return RemoteConfigEnvelope{}, false, err
	}
	query := endpoint.Query()
	query.Set("after_version", strconv.FormatInt(afterVersion, 10))
	endpoint.RawQuery = query.Encode()
	req, err := http.NewRequest(http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return RemoteConfigEnvelope{}, false, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return RemoteConfigEnvelope{}, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return RemoteConfigEnvelope{}, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return RemoteConfigEnvelope{}, false, fmt.Errorf("uplink: remote config returned status %d", resp.StatusCode)
	}
	var envelope RemoteConfigEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return RemoteConfigEnvelope{}, false, err
	}
	return envelope, true, nil
}

func (c *Client) AckRemoteConfig(ack RemoteConfigAck) error {
	if !c.mtls {
		return fmt.Errorf("uplink: remote configuration requires mTLS")
	}
	return c.postJSON("/api/v1/agents/me/config/ack", ack)
}

func (c *Client) SetSatelliteID(satelliteID string) {
	c.mu.Lock()
	c.satelliteID = satelliteID
	c.mu.Unlock()
}

func (c *Client) currentSatelliteID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.satelliteID
}
