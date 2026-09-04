package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/capture"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/forensics"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/identity"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/integrity"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/remoteconfig"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/replay"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/soapy"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/streaming"
	"github.com/mlko01/lyrahcle/apps/sdr-agent/internal/uplink"
)

type runtimeAgentConfig struct {
	FrequencyHz            int64
	CaptureIntervalSeconds int
}

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

func getEnvFloat(key string, fallback float64) float64 {
	v, err := strconv.ParseFloat(getEnv(key, ""), 64)
	if err != nil {
		return fallback
	}
	return v
}

func getEnvBool(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func deviceSpecFromEnv(freqHz int64) soapy.DeviceSpec {
	return soapy.DeviceSpec{
		URI:            getEnv("SDR_DEVICE", "mock://"),
		FrequencyHz:    uint64(freqHz),
		SampleRateHz:   uint64(getEnvInt("SDR_SAMPLE_RATE_HZ", 2048000)),
		GainDb:         getEnvFloat("SDR_GAIN_DB", 30),
		AutoGain:       getEnvBool("SDR_AUTO_GAIN"),
		FreqCorrection: int32(getEnvInt("SDR_FREQ_CORRECTION_PPM", 0)),
		BiasTee:        getEnvBool("SDR_BIAS_TEE"),
	}
}

func spectralConfigFromEnv() capture.SpectralConfig {
	return capture.SpectralConfig{
		FFTSize:            getEnvInt("SDR_FFT_SIZE", 4096),
		Window:             getEnv("SDR_FFT_WINDOW", "hann"),
		OverlapRatio:       getEnvFloat("SDR_FFT_OVERLAP", 0.5),
		SampleCount:        getEnvInt("SDR_SAMPLES_PER_READING", 32768),
		CalibrationOffsetD: getEnvFloat("SDR_CALIBRATION_OFFSET_DB", 0),
		ReconnectAttempts:  getEnvInt("SDR_RECONNECT_ATTEMPTS", 3),
	}
}

func buildCaptureSource(freqHz int64) (capture.Source, soapy.Device, func()) {
	spec := deviceSpecFromEnv(freqHz)
	device, err := soapy.NewDeviceFromSpec(spec)
	if err != nil {
		log.Fatalf("sdr device: %v", err)
	}
	if device.GetDeviceInfo() != nil && device.GetDeviceInfo().Driver == "mock" {
		log.Println("WARNING: SDR_DEVICE is not configured; running on the mock device. Spectral output is synthetic and must not be used as detection evidence.")
		return capture.MockSource{}, device, func() { device.Close() }
	}
	if spec.SampleRateHz > 0 {
		if err := device.SetSampleRate(spec.SampleRateHz); err != nil {
			log.Fatalf("sdr device: set sample rate: %v", err)
		}
	}
	source, err := capture.NewSpectralSource(device, spectralConfigFromEnv())
	if err != nil {
		log.Fatalf("spectral source: %v", err)
	}
	log.Printf("sdr device: %s (%s), sample rate %d Hz, FFT %d", device.GetDeviceInfo().Hardware, spec.URI, spec.SampleRateHz, getEnvInt("SDR_FFT_SIZE", 4096))
	return source, device, func() { source.Close() }
}

func main() {
	backendURL := getEnv("BACKEND_URL", "http://backend:8080")
	agentID := getEnv("AGENT_ID", "sdr-agent-dev")
	apiKey := getEnv("AGENT_API_KEY", "")
	authToken := getEnv("AGENT_TOKEN", "")
	caCertPath := getEnv("AGENT_CA_CERT", "")
	clientCertPath := getEnv("AGENT_CERT_FILE", "")
	clientKeyPath := getEnv("AGENT_KEY_FILE", "")
	enrollmentToken := getEnv("AGENT_ENROLLMENT_TOKEN", "")
	agentVersion := getEnv("AGENT_VERSION", "v9.1")
	firmwareVersion := getEnv("AGENT_FIRMWARE_VERSION", "")
	configVerifyKeyPath := getEnv("AGENT_CONFIG_VERIFY_KEY", "")
	configVerifyKeyDir := getEnv("AGENT_CONFIG_VERIFY_KEY_DIR", "")
	rotateBefore, err := time.ParseDuration(getEnv("AGENT_ROTATE_BEFORE", "24h"))
	if err != nil || rotateBefore < time.Hour {
		log.Fatal("AGENT_ROTATE_BEFORE must be a duration of at least 1h")
	}
	if enrollmentToken != "" && (clientCertPath == "" || clientKeyPath == "") {
		log.Fatal("AGENT_CERT_FILE and AGENT_KEY_FILE are required when AGENT_ENROLLMENT_TOKEN is set")
	}
	identityCfg := identity.Config{
		BackendURL: backendURL, AgentID: agentID, EnrollmentToken: enrollmentToken,
		ServerCAPath: caCertPath, CertificatePath: clientCertPath, PrivateKeyPath: clientKeyPath,
		AgentVersion: agentVersion, FirmwareVersion: firmwareVersion, RotateBefore: rotateBefore,
	}
	if clientCertPath != "" || clientKeyPath != "" {
		if clientCertPath == "" || clientKeyPath == "" {
			log.Fatal("AGENT_CERT_FILE and AGENT_KEY_FILE must be configured together")
		}
		if err := identity.Ensure(identityCfg); err != nil {
			log.Fatalf("agent identity: %v", err)
		}
	}
	freqHz, err := strconv.ParseInt(getEnv("FREQUENCY_HZ", "437500000"), 10, 64)
	if err != nil || freqHz < 100000 || freqHz > 6000000000 {
		log.Fatal("FREQUENCY_HZ must be between 100000 and 6000000000")
	}
	intervalSec := getEnvInt("CAPTURE_INTERVAL_SECONDS", 10)
	if intervalSec < 1 || intervalSec > 3600 {
		log.Fatal("CAPTURE_INTERVAL_SECONDS must be between 1 and 3600")
	}
	if (configVerifyKeyPath != "" || configVerifyKeyDir != "") && clientCertPath == "" {
		log.Fatal("remote config verification keys require mTLS agent identity")
	}

	if clientCertPath != "" {
		log.Println("agent authentication: mTLS client certificate")
	} else if apiKey == "" && authToken == "" {
		log.Println("WARNING: no agent credential is configured; the backend will reject authenticated agent requests")
	} else if apiKey == "" {
		log.Println("NOTICE: using a shared operator JWT; migrate the agent to mTLS enrollment")
	} else {
		log.Println("NOTICE: using a legacy agent API key; migrate the agent to mTLS enrollment")
	}
	client, err := uplink.NewWithIdentity(backendURL, agentID, apiKey, authToken, caCertPath, clientCertPath, clientKeyPath)
	if err != nil {
		log.Fatalf("uplink client: %v", err)
	}
	if clientCertPath != "" {
		go runIdentityMaintenance(identityCfg, client)
	}

	var liveConfig atomic.Value
	liveConfig.Store(runtimeAgentConfig{FrequencyHz: freqHz, CaptureIntervalSeconds: intervalSec})
	configChanged := make(chan struct{}, 1)
	if configVerifyKeyPath != "" || configVerifyKeyDir != "" {
		verifier, err := remoteconfig.LoadVerifierWithKeyDirectory(configVerifyKeyPath, configVerifyKeyDir, agentID)
		if err != nil {
			log.Fatalf("remote configuration: %v", err)
		}
		manager := remoteconfig.NewManager(client, verifier, func(settings remoteconfig.Settings) error {
			liveConfig.Store(runtimeAgentConfig{FrequencyHz: settings.FrequencyHz, CaptureIntervalSeconds: settings.CaptureIntervalSeconds})
			client.SetSatelliteID(settings.SatelliteID)
			select {
			case configChanged <- struct{}{}:
			default:
			}
			return nil
		})
		go runRemoteConfig(manager)
	}

	if replayFile := getEnv("REPLAY_FILE", ""); replayFile != "" {
		replayRoot := getEnv("REPLAY_ROOT", "")
		if filepath.IsAbs(replayFile) && replayRoot == "" {
			log.Fatal("REPLAY_ROOT is required when REPLAY_FILE is absolute")
		}
		if replayRoot == "" {
			replayRoot = "."
		}
		runReplay(replayRoot, replayFile, client)
		return
	}

	source, device, closeSource := buildCaptureSource(freqHz)
	defer closeSource()

	if captureDir := getEnv("IQ_CAPTURE_DIR", ""); captureDir != "" {
		go runIQCaptureLoop(captureDir, agentID, freqHz)
	}

	if info := device.GetDeviceInfo(); info != nil {
		report, err := integrity.Measure(integrity.PathsFromCSV(getEnv("AGENT_INTEGRITY_PATHS", "")))
		if err != nil {
			log.Printf("edge integrity measurement failed (non-fatal): %v", err)
		} else {
			info.Capabilities.IntegrityManifestSHA256 = report.ManifestSHA256
			info.Capabilities.IntegrityMachineIDHash = report.MachineIDHash
			info.Capabilities.IntegrityBootID = report.BootID
			info.Capabilities.IntegrityFileCount = report.FileCount
		}
		if err := client.ReportDevice(info); err != nil {
			log.Printf("device self-report failed (non-fatal): %v", err)
		}
	}

	log.Printf("lyrahcle sdr-agent %q starting, target=%s, freq=%d Hz, interval=%ds", agentID, backendURL, freqHz, intervalSec)

	for {
		current := liveConfig.Load().(runtimeAgentConfig)
		timer := time.NewTimer(time.Duration(current.CaptureIntervalSeconds) * time.Second)
		select {
		case <-configChanged:
			if !timer.Stop() {
				<-timer.C
			}
			continue
		case <-timer.C:
		}
		current = liveConfig.Load().(runtimeAgentConfig)
		reading, err := source.Read(current.FrequencyHz)
		if err != nil {
			log.Printf("capture error: %v", err)
			continue
		}
		if err := client.Send(reading); err != nil {
			log.Printf("uplink error: %v", err)
			continue
		}
		if reading.Spectral != nil {
			log.Printf(
				"sent reading: %.2f dBFS @ %d Hz (noise %.2f dBFS, SNR %.1f dB, carrier offset %+.1f Hz via %s, OBW %.0f Hz)",
				reading.Spectral.PowerDbFS, reading.FrequencyHz, reading.Spectral.NoiseFloorDbFS,
				reading.Spectral.SNRDb, reading.Spectral.CarrierOffsetHz, reading.Spectral.EstimatorMethod,
				reading.Spectral.OccupiedBandwidthHz,
			)
		} else {
			log.Printf("sent reading: %.2f dBm @ %d Hz", reading.PowerDbm, reading.FrequencyHz)
		}
	}
}

func runIQCaptureLoop(captureDir, agentID string, freqHz int64) {
	intervalSec := getEnvInt("IQ_CAPTURE_INTERVAL_SECONDS", 60)
	numSamples := getEnvInt("IQ_CAPTURE_SAMPLES", 8192)
	sigmfCaptures := getEnvBool("IQ_CAPTURE_SIGMF")

	device, err := soapy.NewDeviceFromSpec(deviceSpecFromEnv(freqHz))
	if err != nil {
		log.Printf("iq-capture: open device: %v", err)
		return
	}
	defer device.Close()
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
		save := forensics.Save
		format := "cf32"
		if sigmfCaptures {
			save = forensics.SaveSigMF
			format = "sigmf"
		}
		dataPath, _, err := save(captureDir, agentID, data)
		if err != nil {
			log.Printf("iq-capture: save: %v", err)
			continue
		}
		log.Printf("iq-capture: saved %d samples to %s (%s)", len(data.Samples), dataPath, format)
	}
}

func runReplay(replayRoot, replayFile string, client *uplink.Client) {
	windowSamples := getEnvInt("REPLAY_WINDOW_SAMPLES", 65536)
	delayMs := getEnvInt("REPLAY_DELAY_MS", 300)

	replayName := replayFile
	var err error
	if filepath.IsAbs(replayFile) {
		replayName, err = filepath.Rel(replayRoot, replayFile)
		if err != nil {
			log.Fatalf("replay: resolve capture path: %v", err)
		}
	}
	log.Printf("replay: loading capture %s within %s", replayName, replayRoot)
	meta, samples, err := forensics.LoadAnyWithin(replayRoot, replayName)
	if err != nil {
		log.Fatalf("replay: %v", err)
	}

	var readings []capture.Reading
	if getEnvBool("REPLAY_SPECTRAL") {
		readings, err = replay.SpectralWindows(meta, samples, replay.SpectralOptions{
			FFTSize:             getEnvInt("SDR_FFT_SIZE", 4096),
			Window:              getEnv("SDR_FFT_WINDOW", "hann"),
			OverlapRatio:        getEnvFloat("SDR_FFT_OVERLAP", 0.5),
			WindowSamples:       windowSamples,
			CalibrationOffsetDb: getEnvFloat("SDR_CALIBRATION_OFFSET_DB", 0),
		})
		if err != nil {
			log.Fatalf("replay: spectral analysis: %v", err)
		}
	} else {
		readings = replay.PowerWindows(meta, samples, windowSamples)
	}
	log.Printf(
		"replay: capture from %s (agent=%s, freq=%d Hz, %d samples, format=%s, provenance=%s/%s) -> %d readings",
		meta.CapturedAt.Format(time.RFC3339), meta.AgentID, meta.FrequencyHz, len(samples), meta.Format,
		meta.Extra["evidence_kind"], meta.Extra["timestamp_from"], len(readings),
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

func runIdentityMaintenance(cfg identity.Config, client *uplink.Client) {
	interval, err := time.ParseDuration(getEnv("AGENT_IDENTITY_CHECK_INTERVAL", "10m"))
	if err != nil || interval < time.Minute {
		interval = 10 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		changed, err := identity.Maintain(cfg)
		if err != nil {
			log.Printf("agent identity maintenance failed: %v", err)
			continue
		}
		if changed {
			client.ReloadIdentity()
			log.Println("agent identity rotated and TLS connections reloaded")
		}
	}
}
func runRemoteConfig(manager *remoteconfig.Manager) {
	interval, err := time.ParseDuration(getEnv("AGENT_CONFIG_POLL_INTERVAL", "30s"))
	if err != nil || interval < 5*time.Second || interval > 10*time.Minute {
		interval = 30 * time.Second
	}
	poll := func() {
		changed, err := manager.Poll()
		if err != nil {
			log.Printf("remote configuration poll failed: %v", err)
			return
		}
		if changed {
			log.Printf("remote configuration applied: version=%d", manager.Version())
		}
	}
	poll()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		poll()
	}
}
