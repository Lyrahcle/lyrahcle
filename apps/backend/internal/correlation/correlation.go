package correlation

import "math"

type Input struct {
	DetectionRuleCount       int
	OrbitalCandidateSeverity string
	OrbitalConfidence        float64
	AgentStatus              string
	RecentAgentAuditEvents   int
	ContextAssetAlerts       int
	ManeuverEvents           int
	ConjunctionEvents        int
	OrbitalOpsConfidence     float64
	OrbitalOpsSeverity       string
}

type Result struct {
	Confidence           float64  `json:"confidence"`
	CandidateSeverity    string   `json:"candidate_severity,omitempty"`
	CorroboratingDomains []string `json:"corroborating_domains"`
	ContextOnlyDomains   []string `json:"context_only_domains"`
	Reasons              []string `json:"reasons"`
}

func Evaluate(in Input) Result {
	confidence := 0.35
	domains := []string{"rf"}
	reasons := []string{"rf anomaly matched one or more enabled detection rules"}
	candidate := ""

	if in.DetectionRuleCount > 0 {
		confidence += math.Min(0.20, float64(in.DetectionRuleCount)*0.05)
		domains = append(domains, "detection_rules")
	}
	if in.OrbitalCandidateSeverity != "" {
		confidence += 0.20 * clamp01(in.OrbitalConfidence)
		domains = append(domains, "orbital")
		reasons = append(reasons, "orbital geometry or Doppler context corroborates the RF anomaly")
		candidate = higherSeverity(candidate, in.OrbitalCandidateSeverity)
	}
	if in.AgentStatus != "" && in.AgentStatus != "active" {
		confidence += 0.15
		domains = append(domains, "agent_posture")
		reasons = append(reasons, "agent identity posture is not active")
		candidate = higherSeverity(candidate, "critical")
	}

	if in.ManeuverEvents > 0 || in.ConjunctionEvents > 0 {
		confidence += 0.15 * clamp01(in.OrbitalOpsConfidence)
		domains = append(domains, "orbital_operations")
		if in.ManeuverEvents > 0 {
			reasons = append(reasons, "recent maneuver context is available for the affected satellite")
		}
		if in.ConjunctionEvents > 0 {
			reasons = append(reasons, "recent or upcoming conjunction context is available for the affected satellite")
		}
		candidate = higherSeverity(candidate, in.OrbitalOpsSeverity)
	}

	if in.RecentAgentAuditEvents > 0 {
		confidence += math.Min(0.10, float64(in.RecentAgentAuditEvents)*0.02)
		domains = append(domains, "audit")
		reasons = append(reasons, "recent agent audit activity is available for operator review")
	}

	contextOnly := []string{}
	if in.ContextAssetAlerts > 0 {
		contextOnly = append(contextOnly, "asset_alerts")
		reasons = append(reasons, "unacknowledged asset CVE alerts are attached as context only and do not affect severity")
	}

	return Result{
		Confidence:           math.Round(math.Min(0.99, confidence)*100) / 100,
		CandidateSeverity:    candidate,
		CorroboratingDomains: unique(domains),
		ContextOnlyDomains:   unique(contextOnly),
		Reasons:              reasons,
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func higherSeverity(a, b string) string {
	rank := map[string]int{"": 0, "low": 1, "medium": 2, "high": 3, "critical": 4}
	if rank[b] > rank[a] {
		return b
	}
	return a
}

func unique(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, item := range in {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}
