package sparta

import (
	"sort"
	"time"
)

type TechniqueCoverage struct {
	TechniqueID     string    `json:"technique_id"`
	TechniqueName   string    `json:"technique_name"`
	Segment         string    `json:"segment"`
	Detectable      bool      `json:"detectable"`
	EvidenceCount   int       `json:"evidence_count"`
	ContextOnly     bool      `json:"context_only"`
	HighestConfid   int       `json:"highest_confidence"`
	EvidenceKinds   []string  `json:"evidence_kinds"`
	LastObservedAt  time.Time `json:"last_observed_at,omitempty"`
	FirstObservedAt time.Time `json:"first_observed_at,omitempty"`
}

type TacticCoverage struct {
	TacticID       string              `json:"tactic_id"`
	TacticName     string              `json:"tactic_name"`
	Order          int                 `json:"order"`
	Techniques     []TechniqueCoverage `json:"techniques"`
	ObservedCount  int                 `json:"observed_count"`
	TechniqueTotal int                 `json:"technique_total"`
}

type Matrix struct {
	Framework          string           `json:"framework"`
	FrameworkOwner     string           `json:"framework_owner"`
	FrameworkURL       string           `json:"framework_url"`
	CatalogVersion     string           `json:"catalog_version"`
	CatalogScopeNote   string           `json:"catalog_scope_note"`
	GeneratedAt        time.Time        `json:"generated_at"`
	Tactics            []TacticCoverage `json:"tactics"`
	TechniqueTotal     int              `json:"technique_total"`
	DetectableTotal    int              `json:"detectable_total"`
	ObservedTotal      int              `json:"observed_total"`
	ContextOnlyTotal   int              `json:"context_only_total"`
	EvidenceTotal      int              `json:"evidence_total"`
	ObservedPercentage int              `json:"observed_percentage"`
}

type ObservedAssessment struct {
	Assessment  Assessment
	ObservedAt  time.Time
	EvidenceRef string
}

func BuildMatrix(observations []ObservedAssessment, generatedAt time.Time) Matrix {
	detectable := detectableTechniqueIDs()
	byTechnique := make(map[string]*TechniqueCoverage)
	evidenceKinds := make(map[string]map[string]bool)
	evidenceRefs := make(map[string]map[string]bool)

	for _, technique := range Techniques() {
		byTechnique[technique.ID] = &TechniqueCoverage{
			TechniqueID:   technique.ID,
			TechniqueName: technique.Name,
			Segment:       technique.Segment,
			Detectable:    detectable[technique.ID],
			ContextOnly:   true,
		}
		evidenceKinds[technique.ID] = map[string]bool{}
		evidenceRefs[technique.ID] = map[string]bool{}
	}

	evidenceTotal := 0
	for _, observation := range observations {
		coverage, ok := byTechnique[observation.Assessment.TechniqueID]
		if !ok {
			continue
		}
		key := observation.Assessment.EvidenceKind + "|" + observation.EvidenceRef
		if observation.EvidenceRef != "" && evidenceRefs[coverage.TechniqueID][key] {
			continue
		}
		evidenceRefs[coverage.TechniqueID][key] = true
		coverage.EvidenceCount++
		evidenceTotal++
		if observation.Assessment.Confidence > coverage.HighestConfid {
			coverage.HighestConfid = observation.Assessment.Confidence
		}
		if !observation.Assessment.ContextOnly {
			coverage.ContextOnly = false
		}
		evidenceKinds[coverage.TechniqueID][observation.Assessment.EvidenceKind] = true
		if !observation.ObservedAt.IsZero() {
			observedAt := observation.ObservedAt.UTC()
			if coverage.LastObservedAt.IsZero() || observedAt.After(coverage.LastObservedAt) {
				coverage.LastObservedAt = observedAt
			}
			if coverage.FirstObservedAt.IsZero() || observedAt.Before(coverage.FirstObservedAt) {
				coverage.FirstObservedAt = observedAt
			}
		}
	}

	matrix := Matrix{
		Framework:        FrameworkName,
		FrameworkOwner:   FrameworkOwner,
		FrameworkURL:     FrameworkURL,
		CatalogVersion:   CatalogVersion,
		CatalogScopeNote: CatalogScopeNote,
		GeneratedAt:      generatedAt.UTC(),
		EvidenceTotal:    evidenceTotal,
	}

	for _, tactic := range Tactics() {
		tacticCoverage := TacticCoverage{TacticID: tactic.ID, TacticName: tactic.Name, Order: tactic.Order}
		for _, technique := range Techniques() {
			if technique.TacticID != tactic.ID {
				continue
			}
			coverage := byTechnique[technique.ID]
			kinds := make([]string, 0, len(evidenceKinds[technique.ID]))
			for kind := range evidenceKinds[technique.ID] {
				kinds = append(kinds, kind)
			}
			sort.Strings(kinds)
			coverage.EvidenceKinds = kinds
			if coverage.EvidenceCount == 0 {
				coverage.ContextOnly = false
			}
			tacticCoverage.Techniques = append(tacticCoverage.Techniques, *coverage)
			tacticCoverage.TechniqueTotal++
			matrix.TechniqueTotal++
			if coverage.Detectable {
				matrix.DetectableTotal++
			}
			if coverage.EvidenceCount > 0 {
				tacticCoverage.ObservedCount++
				matrix.ObservedTotal++
				if coverage.ContextOnly {
					matrix.ContextOnlyTotal++
				}
			}
		}
		matrix.Tactics = append(matrix.Tactics, tacticCoverage)
	}

	if matrix.TechniqueTotal > 0 {
		matrix.ObservedPercentage = matrix.ObservedTotal * 100 / matrix.TechniqueTotal
	}
	return matrix
}

func detectableTechniqueIDs() map[string]bool {
	out := map[string]bool{}
	for _, rule := range rules {
		for _, techniqueID := range rule.TechniqueIDs {
			out[techniqueID] = true
		}
	}
	return out
}
