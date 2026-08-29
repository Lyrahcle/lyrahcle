package uplink

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/capture"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/soapy"
)

type Client struct {
	BackendURL string
	AgentID    string
	APIKey     string
	AuthToken  string
	http       *http.Client
}

type deviceReport struct {
	Driver       string `json:"driver"`
	Hardware     string `json:"hardware"`
	Manufacturer string `json:"manufacturer"`
	Product      string `json:"product"`
	Serial       string `json:"serial"`
	RX           bool   `json:"rx"`
	TX           bool   `json:"tx"`
}

func New(backendURL, agentID, apiKey, authToken string, insecureSkipVerify bool) *Client {
	transport := &http.Transport{}
	if insecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &Client{
		BackendURL: backendURL,
		AgentID:    agentID,
		APIKey:     apiKey,
		AuthToken:  authToken,
		http:       &http.Client{Timeout: 5 * time.Second, Transport: transport},
	}
}

func (c *Client) Send(reading capture.Reading) error {
	payload := struct {
		AgentID     string  `json:"agent_id"`
		FrequencyHz int64   `json:"frequency_hz"`
		PowerDbm    float64 `json:"power_dbm"`
		CapturedAt  string  `json:"captured_at"`
	}{
		AgentID:     c.AgentID,
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
	if c.APIKey != "" {
		req.Header.Set("X-Agent-Key", c.APIKey)
		return
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}
}
