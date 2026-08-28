package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/capture"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/forensics"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/replay"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/soapy"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/streaming"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/uplink"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v, err := strconv.Atoi(getEnv(key, ""))
	if err != nil {
		return fallback
	}
	return v
}

func main() {
	backendURL := getEnv("BACKEND_URL", "http://backend:8080")
	agentID := getEnv("AGENT_ID", "sdr-agent-dev")
	apiKey := getEnv("AGENT_API_KEY", "")
	authToken := getEnv("AGENT_TOKEN", "")
	insecureSkipVerify := getEnv("AGENT_INSECURE_SKIP_VERIFY", "false") == "true"
	freqHz, _ := strconv.ParseInt(getEnv("FREQUENCY_HZ", "437500000"), 10, 64)
	intervalSec := getEnvInt("CAPTURE_INTERVAL_SECONDS", 10)

	if apiKey == "" && authToken == "" {
		log.Println("WARNING: neither AGENT_API_KEY nor AGENT_TOKEN is set; the backend will reject RF ingestion. AGENT_API_KEY (from POST /api/v1/agent-keys) is the recommended, agent-scoped credential.")
	} else if apiKey == "" {
		log.Println("NOTICE: using AGENT_TOKEN (a shared operator JWT). AGENT_API_KEY is preferred: it's scoped to this agent_id alone and can be revoked independently.")
	}
	if insecureSkipVerify {
		log.Println("WARNING: AGENT_INSECURE_SKIP_VERIFY is enabled; TLS certificate verification is OFF. Use only against known self-signed dev certs, never in production.")
	}

	client := uplink.New(backendURL, agentID, apiKey, authToken, insecureSkipVerify)

	// Forensic replay mode: load a previously saved IQ capture and
	// re-ingest it as historical RF readings, then exit. Mutually
	// exclusive with the live capture loop below.
	if replayFile := getEnv("REPLAY_FILE", ""); replayFile != "" {
		runReplay(replayFile, client)
		return
	}

	// Optional background IQ-to-disk recording, independent of the RF
	// power readings sent every CAPTURE_INTERVAL_SECONDS below. Runs
	// against the mock device until real SoapySDR hardware is wired in;
	// see docs/forensics.md.
	if captureDir := getEnv("IQ_CAPTURE_DIR", ""); captureDir != "" {
		go runIQCaptureLoop(captureDir, agentID, freqHz)
	}

	var source capture.Source = capture.MockSource{}

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

// runIQCaptureLoop periodically snapshots raw IQ samples and saves them to
// captureDir as forensic captures (see internal/forensics), independent of
// the RF power readings the live loop sends every capture cycle.
func runIQCaptureLoop(captureDir, agentID string, freqHz int64) {
	intervalSec := getEnvInt("IQ_CAPTURE_INTERVAL_SECONDS", 60)
	numSamples := getEnvInt("IQ_CAPTURE_SAMPLES", 8192)

	device := soapy.NewMockDevice()
	if err := device.SetFrequency(uint64(freqHz)); err != nil {
		log.Printf("iq-capture: set frequency: %v", err)
		return
	}

	sm := streaming.NewStreamManager(device, streaming.StreamConfig{NumSamples: numSamples, BufferSize: numSamples})
	if err := sm.StartCapture(); err != nil {
		log.Printf("iq-capture: start: %v", err)
		return
	}
	defer sm.StopCapture()

	log.Printf("iq-capture: recording %d samples every %ds to %s", numSamples, intervalSec, captureDir)

	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		data, err := sm.ReadStream(time.Duration(intervalSec) * time.Second)
		if err != nil {
			log.Printf("iq-capture: read: %v", err)
			continue
		}
		dataPath, _, err := forensics.Save(captureDir, agentID, data)
		if err != nil {
			log.Printf("iq-capture: save: %v", err)
			continue
		}
		log.Printf("iq-capture: saved %d samples to %s", len(data.Samples), dataPath)
	}
}

// runReplay loads a saved IQ capture and re-ingests it as historical RF
// power readings, preserving the capture's original timeline.
func runReplay(replayFile string, client *uplink.Client) {
	windowSamples := getEnvInt("REPLAY_WINDOW_SAMPLES", 65536)
	delayMs := getEnvInt("REPLAY_DELAY_MS", 300)

	log.Printf("replay: loading capture %s", replayFile)
	meta, samples, err := forensics.Load(replayFile)
	if err != nil {
		log.Fatalf("replay: %v", err)
	}

	readings := replay.PowerWindows(meta, samples, windowSamples)
	log.Printf(
		"replay: capture from %s (agent=%s, freq=%d Hz, %d samples) -> %d readings",
		meta.CapturedAt.Format(time.RFC3339), meta.AgentID, meta.FrequencyHz, len(samples), len(readings),
	)

	sent := 0
	for _, r := range readings {
		if err := client.Send(r); err != nil {
			log.Printf("replay: uplink error at %s: %v", r.CapturedAt.Format(time.RFC3339), err)
			continue
		}
		sent++
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}

	log.Printf("replay: done — sent %d/%d readings", sent, len(readings))
}
