package alerts

import (
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"strings"
)

type CPEAssessment struct {
	Applicable bool
	Method     string
	Confidence int
}

func AssessCPE(assetCPE string, matches []store.CPEMatch) CPEAssessment {
	asset, ok := parseCPE23(assetCPE)
	if !ok {
		return CPEAssessment{}
	}
	for _, candidate := range matches {
		if !candidate.Vulnerable {
			continue
		}
		pattern, ok := parseCPE23(candidate.Criteria)
		if !ok || !componentMatch(pattern[2], asset[2]) || !componentMatch(pattern[3], asset[3]) || !componentMatch(pattern[4], asset[4]) {
			continue
		}
		if pattern[5] != "*" && pattern[5] != "-" && !componentMatch(pattern[5], asset[5]) {
			continue
		}
		if !versionInRange(asset[5], candidate) {
			continue
		}
		confidence := 90
		if pattern[5] != "*" && pattern[5] != "-" {
			confidence = 100
		}
		return CPEAssessment{Applicable: true, Method: "cpe_2_3", Confidence: confidence}
	}
	return CPEAssessment{Method: "cpe_2_3", Confidence: 0}
}

func parseCPE23(raw string) ([]string, bool) {
	parts := strings.Split(strings.TrimSpace(raw), ":")
	if len(parts) < 6 || parts[0] != "cpe" || parts[1] != "2.3" {
		return nil, false
	}
	return parts, true
}

func componentMatch(pattern, actual string) bool {
	return pattern == "*" || pattern == "-" || strings.EqualFold(pattern, actual)
}

func versionInRange(version string, m store.CPEMatch) bool {
	if version == "" || version == "*" || version == "-" {
		return m.VersionStartIncluding == "" && m.VersionStartExcluding == "" && m.VersionEndIncluding == "" && m.VersionEndExcluding == ""
	}
	if m.VersionStartIncluding != "" && compareVersion(version, m.VersionStartIncluding) < 0 {
		return false
	}
	if m.VersionStartExcluding != "" && compareVersion(version, m.VersionStartExcluding) <= 0 {
		return false
	}
	if m.VersionEndIncluding != "" && compareVersion(version, m.VersionEndIncluding) > 0 {
		return false
	}
	if m.VersionEndExcluding != "" && compareVersion(version, m.VersionEndExcluding) >= 0 {
		return false
	}
	return true
}

func compareVersion(a, b string) int {
	as, bs := strings.FieldsFunc(a, func(r rune) bool { return r == '.' || r == '-' || r == '_' }), strings.FieldsFunc(b, func(r rune) bool { return r == '.' || r == '-' || r == '_' })
	for i := 0; i < len(as) || i < len(bs); i++ {
		av, bv := "0", "0"
		if i < len(as) {
			av = as[i]
		}
		if i < len(bs) {
			bv = bs[i]
		}
		if av == bv {
			continue
		}
		if len(av) != len(bv) {
			if len(av) < len(bv) {
				return -1
			}
			return 1
		}
		if av < bv {
			return -1
		}
		return 1
	}
	return 0
}
