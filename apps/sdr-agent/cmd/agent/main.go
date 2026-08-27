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
	freqHz, _ := strconv.ParseInt(getEnv("FREQUENCY_HZ", "437500000"), 10, 64)
	intervalSec, _ := strconv.Atoi(getEnv("CAPTURE_INTERVAL_SECONDS", "10"))

	var source capture.Source = capture.MockSource{}
	client := uplink.New(backendURL, agentID)

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
