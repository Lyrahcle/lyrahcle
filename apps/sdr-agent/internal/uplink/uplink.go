package uplink

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/capture"
)

type Client struct {
	BackendURL string
	AgentID    string
	http       *http.Client
}

func New(backendURL, agentID string) *Client {
	return &Client{
		BackendURL: backendURL,
		AgentID:    agentID,
		http:       &http.Client{Timeout: 5 * time.Second},
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

	resp, err := c.http.Post(c.BackendURL+"/api/v1/rf/ingest", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("uplink: backend returned status %d", resp.StatusCode)
	}
	return nil
}
