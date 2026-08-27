package uplink

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/capture"
)

type Client struct {
	BackendURL string
	AgentID    string
	AuthToken  string
	http       *http.Client
}

func New(backendURL, agentID, authToken string, insecureSkipVerify bool) *Client {
	transport := &http.Transport{}
	if insecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return &Client{
		BackendURL: backendURL,
		AgentID:    agentID,
		AuthToken:  authToken,
		http:       &http.Client{Timeout: 5 * time.Second, Transport: transport},
	}
}

func (c *Client) Send(r capture.Reading) error {
	payload := map[string]any{
		"agent_id":     c.AgentID,
		"frequency_hz": r.FrequencyHz,
		"power_dbm":    r.PowerDbm,
		"captured_at":  r.CapturedAt.Format(time.RFC3339),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.BackendURL+"/api/v1/rf/ingest", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("uplink: backend returned status %d", resp.StatusCode)
	}
	return nil
}
