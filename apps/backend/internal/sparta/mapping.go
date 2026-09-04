package sparta

import (
	"math"
	"sort"
	"strings"
)

const (
	EvidenceRFAnomaly     = "rf_anomaly"
	EvidenceIncident      = "incident"
	EvidenceTMTCFinding   = "tmtc_finding"
	EvidenceAgentPosture  = "agent_posture"
	EvidenceCVEContext    = "cve_context"
	EvidenceExternalRecep = "external_reception"
)

type Assessment struct {
	TechniqueID   string `json:"technique_id"`
	TechniqueName string `json:"technique_name"`
	TacticID      string `json:"tactic_id"`
	TacticName    string `json:"tactic_name"`
	EvidenceKind  string `json:"evidence_kind"`
	EvidenceRef   string `json:"evidence_ref,omitempty"`
	Confidence    int    `json:"confidence"`
	ContextOnly   bool   `json:"context_only"`
	Rationale     string `json:"rationale"`
}

type Rule struct {
	EvidenceKind string   `json:"evidence_kind"`
	Match        string   `json:"match"`
	TechniqueIDs []string `json:"technique_ids"`
	Confidence   int      `json:"confidence"`
	ContextOnly  bool     `json:"context_only"`
	Rationale    string   `json:"rationale"`
}

var rules = []Rule{
	{EvidenceKind: EvidenceRFAnomaly, Match: "power_dbm", TechniqueIDs: []string{"EX-0016", "DE-0002", "IMP-0004"}, Confidence: 45, ContextOnly: true, Rationale: "A sustained power deviation on a mission link is consistent with jamming or link denial, but power alone does not identify an emitter."},
	{EvidenceKind: EvidenceRFAnomaly, Match: "frequency_hz", TechniqueIDs: []string{"EX-0014", "IA-0008"}, Confidence: 40, ContextOnly: true, Rationale: "An unexpected carrier offset can indicate an unauthorised emitter, and needs Doppler comparison against the orbital model before escalation."},
	{EvidenceKind: EvidenceRFAnomaly, Match: "snr_db", TechniqueIDs: []string{"EX-0016", "IMP-0004"}, Confidence: 35, ContextOnly: true, Rationale: "Degraded link margin is consistent with interference but also with weather, geometry and receiver configuration."},
	{EvidenceKind: EvidenceRFAnomaly, Match: "occupied_bandwidth_hz", TechniqueIDs: []string{"EX-0013", "EX-0014"}, Confidence: 35, ContextOnly: true, Rationale: "An occupied bandwidth outside the declared transmitter profile suggests foreign traffic on the channel."},
	{EvidenceKind: EvidenceIncident, Match: "rf_anomaly", TechniqueIDs: []string{"EX-0016", "DE-0002"}, Confidence: 50, ContextOnly: false, Rationale: "An RF-sourced incident has been triaged by a detection rule rather than being a raw reading."},
	{EvidenceKind: EvidenceIncident, Match: "orbital_event", TechniqueIDs: []string{"REC-0009", "IMP-0002"}, Confidence: 40, ContextOnly: true, Rationale: "Manoeuvre and conjunction context describes mission state, not a proven adversary action."},
	{EvidenceKind: EvidenceIncident, Match: "agent_posture", TechniqueIDs: []string{"IA-0002", "PER-0003"}, Confidence: 55, ContextOnly: false, Rationale: "A degraded or unauthorised ground station agent is a ground-segment compromise indicator."},
	{EvidenceKind: EvidenceIncident, Match: "manual", TechniqueIDs: []string{"IA-0007"}, Confidence: 30, ContextOnly: true, Rationale: "Operator-created incidents carry no automatic technique attribution; the mapping is a starting point for the analyst."},
	{EvidenceKind: EvidenceTMTCFinding, Match: "sequence_gap", TechniqueIDs: []string{"DE-0002", "IMP-0002"}, Confidence: 45, ContextOnly: true, Rationale: "Missing frame counters are consistent with downlink denial and with ordinary link loss."},
	{EvidenceKind: EvidenceTMTCFinding, Match: "sequence_repeat", TechniqueIDs: []string{"EX-0001", "EX-0014"}, Confidence: 60, ContextOnly: false, Rationale: "Repeated counters on a command path are a direct replay indicator."},
	{EvidenceKind: EvidenceTMTCFinding, Match: "unauthenticated_command", TechniqueIDs: []string{"EX-0012", "IA-0008"}, Confidence: 65, ContextOnly: false, Rationale: "Commanding without the expected authentication layer is an unauthorised commanding indicator."},
	{EvidenceKind: EvidenceTMTCFinding, Match: "malformed_frame", TechniqueIDs: []string{"EX-0013", "EX-0009"}, Confidence: 40, ContextOnly: true, Rationale: "Malformed frames can indicate probing of the decoder or simply a poor link."},
	{EvidenceKind: EvidenceAgentPosture, Match: "certificate_auth_failed", TechniqueIDs: []string{"IA-0008", "DE-0004"}, Confidence: 55, ContextOnly: false, Rationale: "Failed agent certificate authentication indicates something attempting to present itself as a mission station."},
	{EvidenceKind: EvidenceAgentPosture, Match: "config_signature_rejected", TechniqueIDs: []string{"PER-0004", "DE-0004"}, Confidence: 60, ContextOnly: false, Rationale: "A rejected configuration signature indicates tampering with the remote configuration path."},
	{EvidenceKind: EvidenceAgentPosture, Match: "offline", TechniqueIDs: []string{"DE-0002", "IMP-0002"}, Confidence: 25, ContextOnly: true, Rationale: "A station going offline is an availability signal that is usually operational rather than adversarial."},
	{EvidenceKind: EvidenceCVEContext, Match: "ground_software", TechniqueIDs: []string{"REC-0008", "EX-0009"}, Confidence: 20, ContextOnly: true, Rationale: "A contextual CVE match describes potential exposure of ground software, not confirmed exploitation."},
	{EvidenceKind: EvidenceCVEContext, Match: "flight_software", TechniqueIDs: []string{"REC-0008", "EX-0009"}, Confidence: 20, ContextOnly: true, Rationale: "A contextual CVE match describes potential exposure of flight software, not confirmed exploitation."},
	{EvidenceKind: EvidenceExternalRecep, Match: "community_observation", TechniqueIDs: []string{"REC-0005", "REC-0003"}, Confidence: 15, ContextOnly: true, Rationale: "Public reception of a downlink shows the link is observable by third parties; it is context, never an incident."},
}

func Rules() []Rule {
	out := make([]Rule, len(rules))
	copy(out, rules)
	return out
}

func Assess(evidenceKind, match, evidenceRef string) []Assessment {
	normalizedKind := strings.ToLower(strings.TrimSpace(evidenceKind))
	normalizedMatch := strings.ToLower(strings.TrimSpace(match))
	var out []Assessment
	for _, rule := range rules {
		if rule.EvidenceKind != normalizedKind || rule.Match != normalizedMatch {
			continue
		}
		for _, techniqueID := range rule.TechniqueIDs {
			technique, ok := TechniqueByID(techniqueID)
			if !ok {
				continue
			}
			tactic, _ := TacticByID(technique.TacticID)
			out = append(out, Assessment{
				TechniqueID:   technique.ID,
				TechniqueName: technique.Name,
				TacticID:      technique.TacticID,
				TacticName:    tactic.Name,
				EvidenceKind:  normalizedKind,
				EvidenceRef:   evidenceRef,
				Confidence:    rule.Confidence,
				ContextOnly:   rule.ContextOnly,
				Rationale:     rule.Rationale,
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].TechniqueID < out[j].TechniqueID })
	return out
}

func AssessAnomaly(metric string, zScore float64, evidenceRef string) []Assessment {
	assessments := Assess(EvidenceRFAnomaly, metric, evidenceRef)
	magnitude := math.Abs(zScore)
	for index := range assessments {
		assessments[index].Confidence = boundedConfidence(assessments[index].Confidence + confidenceBonus(magnitude))
	}
	return assessments
}

func AssessIncident(sourceType, evidenceRef string) []Assessment {
	return Assess(EvidenceIncident, sourceType, evidenceRef)
}

func confidenceBonus(magnitude float64) int {
	switch {
	case magnitude >= 8:
		return 15
	case magnitude >= 5:
		return 10
	case magnitude >= 3:
		return 5
	default:
		return 0
	}
}

func boundedConfidence(value int) int {
	if value < 0 {
		return 0
	}
	if value > 75 {
		return 75
	}
	return value
}
