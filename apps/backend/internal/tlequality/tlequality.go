package tlequality

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Status string

const (
	StatusFresh    Status = "fresh"
	StatusAging    Status = "aging"
	StatusStale    Status = "stale"
	StatusUnusable Status = "unusable"
)

type Policy struct {
	CorrelationMaxAge time.Duration
	StaleMaxAge       time.Duration
}

type Assessment struct {
	Status            Status
	Age               time.Duration
	Confidence        string
	EscalationAllowed bool
}

func Epoch(line1 string) (time.Time, error) {
	if len(line1) < 32 {
		return time.Time{}, fmt.Errorf("TLE line 1 is too short")
	}
	yearValue, err := strconv.Atoi(strings.TrimSpace(line1[18:20]))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid TLE epoch year: %w", err)
	}
	dayValue, err := strconv.ParseFloat(strings.TrimSpace(line1[20:32]), 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid TLE epoch day: %w", err)
	}
	if dayValue < 1 || dayValue >= 367 {
		return time.Time{}, fmt.Errorf("TLE epoch day is out of range")
	}
	year := 1900 + yearValue
	if yearValue < 57 {
		year = 2000 + yearValue
	}
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	return start.Add(time.Duration((dayValue - 1) * float64(24*time.Hour))), nil
}

func Assess(epoch *time.Time, at time.Time, policy Policy) Assessment {
	if epoch == nil || epoch.IsZero() || policy.CorrelationMaxAge <= 0 || policy.StaleMaxAge <= policy.CorrelationMaxAge {
		return Assessment{Status: StatusUnusable, Confidence: "unavailable"}
	}
	at = at.UTC()
	epochUTC := epoch.UTC()
	if epochUTC.After(at.Add(6 * time.Hour)) {
		return Assessment{Status: StatusUnusable, Confidence: "unavailable"}
	}
	age := at.Sub(epochUTC)
	if age < 0 {
		age = 0
	}
	if age <= policy.CorrelationMaxAge {
		return Assessment{Status: StatusFresh, Age: age, Confidence: "high", EscalationAllowed: true}
	}
	if age <= policy.StaleMaxAge {
		return Assessment{Status: StatusAging, Age: age, Confidence: "degraded"}
	}
	return Assessment{Status: StatusStale, Age: age, Confidence: "unavailable"}
}
