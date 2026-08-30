package detection

import (
	"math"
	"time"

	"github.com/google/uuid"
)

type Rule struct {
	ID              uuid.UUID
	OrganizationID  uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	Enabled         bool       `json:"enabled"`
	Metric          string     `json:"metric"`
	Operator        string     `json:"operator"`
	Threshold       float64    `json:"threshold"`
	Severity        string     `json:"severity"`
	CooldownSeconds int        `json:"cooldown_seconds"`
	AutoIncident    bool       `json:"auto_incident"`
	CreatedBy       *uuid.UUID `json:"-"`
	UpdatedBy       *uuid.UUID `json:"-"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type Signal struct {
	Metric string
	ZScore float64
}

func Matches(rule Rule, signal Signal) bool {
	if !rule.Enabled || (rule.Metric != "any" && rule.Metric != signal.Metric) {
		return false
	}
	switch rule.Operator {
	case "abs_z_gte":
		return math.Abs(signal.ZScore) >= rule.Threshold
	case "z_gte":
		return signal.ZScore >= rule.Threshold
	case "z_lte":
		return signal.ZScore <= -rule.Threshold
	default:
		return false
	}
}

func HigherSeverity(a, b string) string {
	if severityRank(b) > severityRank(a) {
		return b
	}
	return a
}

func severityRank(value string) int {
	switch value {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}
