package validate

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func Email(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return fmt.Errorf("email is required")
	}
	if len(email) > 254 {
		return fmt.Errorf("email is too long")
	}
	if !emailPattern.MatchString(email) {
		return fmt.Errorf("email is not a valid address")
	}
	return nil
}

func Password(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	if len(password) > 128 {
		return fmt.Errorf("password must be at most 128 characters")
	}
	return nil
}

func AgentID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("agent_id is required")
	}
	if len(id) > 128 {
		return fmt.Errorf("agent_id is too long (max 128 characters)")
	}
	return nil
}

const (
	minFrequencyHz = 1_000
	maxFrequencyHz = 300_000_000_000
	minPowerDbm    = -200
	maxPowerDbm    = 50
)

func FrequencyHz(hz int64) error {
	if hz < minFrequencyHz || hz > maxFrequencyHz {
		return fmt.Errorf("frequency_hz must be between %d and %d Hz", minFrequencyHz, maxFrequencyHz)
	}
	return nil
}

func PowerDbm(dbm float64) error {
	if dbm < minPowerDbm || dbm > maxPowerDbm {
		return fmt.Errorf("power_dbm must be between %d and %d dBm", minPowerDbm, maxPowerDbm)
	}
	return nil
}

func CapturedAt(t time.Time) error {
	if t.IsZero() {
		return fmt.Errorf("captured_at is required")
	}
	if t.After(time.Now().Add(5 * time.Minute)) {
		return fmt.Errorf("captured_at cannot be in the future")
	}
	if t.Before(time.Now().Add(-24 * time.Hour)) {
		return fmt.Errorf("captured_at is more than 24h in the past")
	}
	return nil
}

func NoradID(id int) error {
	if id < 1 || id > 999999 {
		return fmt.Errorf("norad_id must be between 1 and 999999")
	}
	return nil
}

var assetKinds = map[string]bool{
	"domain": true, "ip": true, "satellite": true, "ground_station": true, "firmware": true,
}

func AssetKind(kind string) error {
	if !assetKinds[kind] {
		return fmt.Errorf("kind must be one of: domain, ip, satellite, ground_station, firmware")
	}
	return nil
}

func AssetValue(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("value is required")
	}
	if len(value) > 256 {
		return fmt.Errorf("value is too long (max 256 characters)")
	}
	return nil
}

func ProgramName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > 128 {
		return fmt.Errorf("name is too long (max 128 characters)")
	}
	return nil
}

func SatelliteName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > 128 {
		return fmt.Errorf("name is too long (max 128 characters)")
	}
	return nil
}

var tleLinePattern = regexp.MustCompile(`^[0-9 .+\-A-Z]+$`)

func TLELine(line string, expectedPrefix string) error {
	if len(line) < 68 || len(line) > 70 {
		return fmt.Errorf("TLE line must be 68-70 characters, got %d", len(line))
	}
	if !strings.HasPrefix(line, expectedPrefix) {
		return fmt.Errorf("TLE line must start with %q", expectedPrefix)
	}
	if !tleLinePattern.MatchString(line) {
		return fmt.Errorf("TLE line contains invalid characters")
	}
	return nil
}

const maxAX25FrameHexLen = 8192

func AX25FrameHex(frameHex string) error {
	return HexFrame(frameHex, maxAX25FrameHexLen)
}

const maxCCSDSPacketHexLen = 8192

func CCSDSPacketHex(packetHex string) error {
	return HexFrame(packetHex, maxCCSDSPacketHexLen)
}

const maxCSPPacketHexLen = 8192

func CSPPacketHex(packetHex string) error {
	return HexFrame(packetHex, maxCSPPacketHexLen)
}

func HexFrame(value string, maxLen int) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("hex frame is required")
	}
	if len(value) > maxLen {
		return fmt.Errorf("hex frame is too long (max %d hex characters)", maxLen)
	}
	if len(value)%2 != 0 {
		return fmt.Errorf("hex frame must have an even number of characters")
	}
	return nil
}

var incidentSeverities = map[string]bool{
	"critical": true, "high": true, "medium": true, "low": true, "info": true,
}

var incidentStatuses = map[string]bool{
	"new": true, "investigating": true, "contained": true, "resolved": true, "false_positive": true,
}

func IncidentTitle(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("title is required")
	}
	if len(value) > 160 {
		return fmt.Errorf("title is too long (max 160 characters)")
	}
	return nil
}

func IncidentDescription(value string) error {
	if len(value) > 4000 {
		return fmt.Errorf("description is too long (max 4000 characters)")
	}
	return nil
}

func IncidentSeverity(value string) error {
	if !incidentSeverities[value] {
		return fmt.Errorf("severity must be one of: critical, high, medium, low, info")
	}
	return nil
}

func IncidentStatus(value string) error {
	if !incidentStatuses[value] {
		return fmt.Errorf("status must be one of: new, investigating, contained, resolved, false_positive")
	}
	return nil
}

func IncidentNote(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("message is required")
	}
	if len(value) > 2000 {
		return fmt.Errorf("message is too long (max 2000 characters)")
	}
	return nil
}

func IncidentEvidenceType(value string) error {
	switch value {
	case "rf_anomaly", "detection_rule", "request", "cve", "agent", "satellite", "external":
		return nil
	default:
		return fmt.Errorf("evidence_type must be one of: rf_anomaly, detection_rule, request, cve, agent, satellite, external")
	}
}

func IncidentEvidenceRef(value string) error {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 256 {
		return fmt.Errorf("evidence_ref must be between 1 and 256 characters")
	}
	return nil
}

func IncidentEvidenceLabel(value string) error {
	if len(strings.TrimSpace(value)) > 160 {
		return fmt.Errorf("label must be at most 160 characters")
	}
	return nil
}

func DetectionRuleName(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("name is required")
	}
	if len(value) > 120 {
		return fmt.Errorf("name is too long (max 120 characters)")
	}
	return nil
}

func DetectionRuleDescription(value string) error {
	if len(value) > 1000 {
		return fmt.Errorf("description is too long (max 1000 characters)")
	}
	return nil
}

func DetectionRuleMetric(value string) error {
	switch value {
	case "any", "power_dbm", "frequency_hz", "power_dbm_trend", "frequency_hz_trend":
		return nil
	default:
		return fmt.Errorf("metric is invalid")
	}
}

func DetectionRuleOperator(value string) error {
	switch value {
	case "abs_z_gte", "z_gte", "z_lte":
		return nil
	default:
		return fmt.Errorf("operator is invalid")
	}
}

func DetectionRuleThreshold(value float64) error {
	if value <= 0 || value > 100 {
		return fmt.Errorf("threshold must be greater than 0 and at most 100")
	}
	return nil
}

func DetectionRuleCooldown(value int) error {
	if value < 0 || value > 86400 {
		return fmt.Errorf("cooldown_seconds must be between 0 and 86400")
	}
	return nil
}
