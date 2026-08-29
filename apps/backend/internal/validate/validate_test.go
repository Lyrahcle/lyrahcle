package validate

import (
	"strings"
	"testing"
	"time"
)

func TestEmail(t *testing.T) {
	cases := []struct {
		email   string
		wantErr bool
	}{
		{"user@example.com", false},
		{"user.name+tag@sub.example.co", false},
		{"", true},
		{"not-an-email", true},
		{"a@b", true},
		{"<script>alert(1)</script>@x.com", true},
		{strings.Repeat("a", 260) + "@example.com", true},
	}
	for _, c := range cases {
		err := Email(c.email)
		if (err != nil) != c.wantErr {
			t.Errorf("Email(%q) error = %v, wantErr %v", c.email, err, c.wantErr)
		}
	}
}

func TestPassword(t *testing.T) {
	cases := []struct {
		password string
		wantErr  bool
	}{
		{"Strong-Passphrase-9!", false},
		{"alllowercase-123!", true},
		{"NOLOWERCASE-123!", true},
		{"NoDigits-Allowed!", true},
		{"NoSymbolsAllowed123", true},
		{"short", true},
		{strings.Repeat("A1!a", 19), true},
	}
	for _, c := range cases {
		err := Password(c.password)
		if (err != nil) != c.wantErr {
			t.Errorf("Password(len=%d) error = %v, wantErr %v", len(c.password), err, c.wantErr)
		}
	}
}

func TestFrequencyHz(t *testing.T) {
	cases := []struct {
		hz      int64
		wantErr bool
	}{
		{437500000, false},
		{1000, false},
		{999, true},
		{-100, true},
		{300_000_000_001, true},
	}
	for _, c := range cases {
		err := FrequencyHz(c.hz)
		if (err != nil) != c.wantErr {
			t.Errorf("FrequencyHz(%d) error = %v, wantErr %v", c.hz, err, c.wantErr)
		}
	}
}

func TestPowerDbm(t *testing.T) {
	cases := []struct {
		dbm     float64
		wantErr bool
	}{
		{-42.3, false},
		{-200, false},
		{50, false},
		{-201, true},
		{51, true},
		{99999, true},
	}
	for _, c := range cases {
		err := PowerDbm(c.dbm)
		if (err != nil) != c.wantErr {
			t.Errorf("PowerDbm(%v) error = %v, wantErr %v", c.dbm, err, c.wantErr)
		}
	}
}

func TestCapturedAt(t *testing.T) {
	if err := CapturedAt(time.Now()); err != nil {
		t.Errorf("CapturedAt(now) should not error, got %v", err)
	}
	if err := CapturedAt(time.Time{}); err == nil {
		t.Error("CapturedAt(zero) should error")
	}
	if err := CapturedAt(time.Now().Add(48 * time.Hour)); err == nil {
		t.Error("CapturedAt(future) should error")
	}
	if err := CapturedAt(time.Now().Add(-48 * time.Hour)); err == nil {
		t.Error("CapturedAt(too far in the past) should error")
	}
}

func TestNoradID(t *testing.T) {
	cases := []struct {
		id      int
		wantErr bool
	}{
		{25544, false},
		{1, false},
		{999999, false},
		{0, true},
		{-5, true},
		{1000000, true},
	}
	for _, c := range cases {
		err := NoradID(c.id)
		if (err != nil) != c.wantErr {
			t.Errorf("NoradID(%d) error = %v, wantErr %v", c.id, err, c.wantErr)
		}
	}
}

func TestSatelliteName(t *testing.T) {
	if err := SatelliteName("ISS (ZARYA)"); err != nil {
		t.Errorf("SatelliteName should accept a normal name, got %v", err)
	}
	if err := SatelliteName(""); err == nil {
		t.Error("SatelliteName should reject an empty name")
	}
	if err := SatelliteName(strings.Repeat("a", 200)); err == nil {
		t.Error("SatelliteName should reject an overly long name")
	}
}

func TestTLELine(t *testing.T) {
	valid1 := "1 25544U 98067A   26239.14379413  .00008717  00000+0  16243-3 0  9995"
	valid2 := "2 25544  51.6326 308.8886 0007697  87.8986 272.2885 15.49649978582754"

	if err := TLELine(valid1, "1 "); err != nil {
		t.Errorf("TLELine(valid1) should pass, got %v", err)
	}
	if err := TLELine(valid2, "2 "); err != nil {
		t.Errorf("TLELine(valid2) should pass, got %v", err)
	}
	if err := TLELine(valid1, "2 "); err == nil {
		t.Error("TLELine should reject a line with the wrong prefix")
	}
	if err := TLELine("1 too short", "1 "); err == nil {
		t.Error("TLELine should reject a line that is too short")
	}
	if err := TLELine("1 25544U'; DROP TABLE users;--0001671700000+016243-30", "1 "); err == nil {
		t.Error("TLELine should reject a line with characters outside the TLE charset")
	}
}

func TestAX25FrameHex(t *testing.T) {
	if err := AX25FrameHex("82a0a4a64040609c60"); err != nil {
		t.Errorf("AX25FrameHex should accept a valid hex string, got %v", err)
	}
	if err := AX25FrameHex(""); err == nil {
		t.Error("AX25FrameHex should reject an empty string")
	}
	if err := AX25FrameHex("abc"); err == nil {
		t.Error("AX25FrameHex should reject an odd-length string")
	}
	if err := AX25FrameHex(strings.Repeat("ab", 10000)); err == nil {
		t.Error("AX25FrameHex should reject an oversized payload")
	}
}

func TestAgentID(t *testing.T) {
	if err := AgentID("sdr-agent-madrid"); err != nil {
		t.Errorf("AgentID should accept a normal id, got %v", err)
	}
	if err := AgentID(""); err == nil {
		t.Error("AgentID should reject an empty id")
	}
	if err := AgentID(strings.Repeat("a", 200)); err == nil {
		t.Error("AgentID should reject an overly long id")
	}
}

func TestDetectionRuleValidation(t *testing.T) {
	if err := DetectionRuleName("Critical frequency shift"); err != nil {
		t.Fatalf("valid name: %v", err)
	}
	if err := DetectionRuleName(""); err == nil {
		t.Fatal("empty name should fail")
	}
	if err := DetectionRuleMetric("frequency_hz"); err != nil {
		t.Fatalf("valid metric: %v", err)
	}
	if err := DetectionRuleMetric("shell"); err == nil {
		t.Fatal("unknown metric should fail")
	}
	if err := DetectionRuleOperator("abs_z_gte"); err != nil {
		t.Fatalf("valid operator: %v", err)
	}
	if err := DetectionRuleOperator("eval"); err == nil {
		t.Fatal("unknown operator should fail")
	}
	if err := DetectionRuleThreshold(6); err != nil {
		t.Fatalf("valid threshold: %v", err)
	}
	if err := DetectionRuleThreshold(0); err == nil {
		t.Fatal("zero threshold should fail")
	}
	if err := DetectionRuleCooldown(86401); err == nil {
		t.Fatal("oversized cooldown should fail")
	}
}
