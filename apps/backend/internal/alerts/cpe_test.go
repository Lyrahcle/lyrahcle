package alerts

import (
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"testing"
)

func TestAssessCPEExactAndRange(t *testing.T) {
	matches := []store.CPEMatch{{Criteria: "cpe:2.3:a:acme:flight_stack:*:*:*:*:*:*:*:*", Vulnerable: true, VersionStartIncluding: "2.0", VersionEndExcluding: "2.5"}}
	got := AssessCPE("cpe:2.3:a:acme:flight_stack:2.4:*:*:*:*:*:*:*", matches)
	if !got.Applicable || got.Method != "cpe_2_3" || got.Confidence != 90 {
		t.Fatalf("unexpected assessment: %+v", got)
	}
	if AssessCPE("cpe:2.3:a:acme:flight_stack:2.5:*:*:*:*:*:*:*", matches).Applicable {
		t.Fatal("2.5 must be outside exclusive end")
	}
}

func TestAssessCPEDoesNotMatchDifferentProduct(t *testing.T) {
	matches := []store.CPEMatch{{Criteria: "cpe:2.3:a:acme:flight_stack:*:*:*:*:*:*:*:*", Vulnerable: true}}
	if AssessCPE("cpe:2.3:a:acme:ground_stack:2.4:*:*:*:*:*:*:*", matches).Applicable {
		t.Fatal("different product matched")
	}
}
