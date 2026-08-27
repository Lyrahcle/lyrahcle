package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/capture"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/uplink"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	backendURL := getEnv("BACKEND_URL", "http://backend:8080")
	agentID := getEnv("AGENT_ID", "sdr-agent-dev")
	apiKey := getEnv("AGENT_API_KEY", "")
	authToken := getEnv("AGENT_TOKEN", "")
	insecureSkipVerify := getEnv("AGENT_INSECURE_SKIP_VERIFY", "false") == "true"
	freqHz, _ := strconv.ParseInt(getEnv("FREQUENCY_HZ", "437500000"), 10, 64)
	intervalSec, _ := strconv.Atoi(getEnv("CAPTURE_INTERVAL_SECONDS", "10"))

	if apiKey == "" && authToken == "" {
		log.Println("WARNING: neither AGENT_API_KEY nor AGENT_TOKEN is set; the backend will reject RF ingestion. AGENT_API_KEY (from POST /api/v1/agent-keys) is the recommended, agent-scoped credential.")
	} else if apiKey == "" {
		log.Println("NOTICE: using AGENT_TOKEN (a shared operator JWT). AGENT_API_KEY is preferred: it's scoped to this agent_id alone and can be revoked independently.")
	}
	if insecureSkipVerify {
		log.Println("WARNING: AGENT_INSECURE_SKIP_VERIFY is enabled; TLS certificate verification is OFF. Use only against known self-signed dev certs, never in production.")
	}

	var source capture.Source = capture.MockSource{}
	client := uplink.New(backendURL, agentID, apiKey, authToken, insecureSkipVerify)

	log.Printf("lyrahcle sdr-agent %q starting, target=%s, freq=%d Hz, interval=%ds", agentID, backendURL, freqHz, intervalSec)

	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		reading, err := source.Read(freqHz)
		if err != nil {
			log.Printf("capture error: %v", err)
			continue
		}
		if err := client.Send(reading); err != nil {
			log.Printf("uplink error: %v", err)
			continue
		}
		log.Printf("sent reading: %.2f dBm @ %d Hz", reading.PowerDbm, reading.FrequencyHz)
	}
}
