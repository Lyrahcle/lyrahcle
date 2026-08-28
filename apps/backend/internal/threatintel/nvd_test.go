package threatintel

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const sampleNVDResponse = `{
	"vulnerabilities": [
		{
			"cve": {
				"id": "CVE-2025-29909",
				"published": "2025-03-17T18:15:00.000",
				"descriptions": [
					{"lang": "en", "value": "CryptoLib CCSDS Space Data Link Security vulnerability"}
				],
				"metrics": {
					"cvssMetricV31": [
						{"cvssData": {"baseScore": 9.8, "baseSeverity": "CRITICAL"}}
					]
				}
			}
		},
		{
			"cve": {
				"id": "CVE-2020-0001",
				"published": "2020-01-01T00:00:00.000",
				"descriptions": [
					{"lang": "en", "value": "An old vulnerability with only CVSSv2 scoring"}
				],
				"metrics": {
					"cvssMetricV2": [
						{"cvssData": {"baseScore": 5.0}, "baseSeverity": "MEDIUM"}
					]
				}
			}
		},
		{
			"cve": {
				"id": "CVE-2099-0001",
				"published": "2099-01-01T00:00:00.000",
				"descriptions": [
					{"lang": "en", "value": "Unscored entry"}
				],
				"metrics": {}
			}
		}
	]
}`

func withMockNVD(t *testing.T, body string) func() {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))
	original := nvdBaseURLForTest
	nvdBaseURLForTest = server.URL
	return func() {
		nvdBaseURLForTest = original
		server.Close()
	}
}

func TestFetchByKeywordParsesV31Score(t *testing.T) {
	cleanup := withMockNVD(t, sampleNVDResponse)
	defer cleanup()

	cves, err := fetchByKeywordFrom(nvdBaseURLForTest, "CCSDS", 10)
	if err != nil {
		t.Fatalf("fetchByKeywordFrom returned error: %v", err)
	}
	if len(cves) != 3 {
		t.Fatalf("got %d CVEs, want 3", len(cves))
	}

	first := cves[0]
	if first.ID != "CVE-2025-29909" {
		t.Errorf("ID = %q, want %q", first.ID, "CVE-2025-29909")
	}
	if !first.HasScore || first.Score != 9.8 || first.Severity != "CRITICAL" {
		t.Errorf("first CVE score/severity = %v/%q, want 9.8/CRITICAL", first.Score, first.Severity)
	}
	if !strings.Contains(first.Description, "CryptoLib") {
		t.Errorf("Description = %q, want it to mention CryptoLib", first.Description)
	}
}

func TestFetchByKeywordFallsBackToV2(t *testing.T) {
	cleanup := withMockNVD(t, sampleNVDResponse)
	defer cleanup()

	cves, err := fetchByKeywordFrom(nvdBaseURLForTest, "CCSDS", 10)
	if err != nil {
		t.Fatalf("fetchByKeywordFrom returned error: %v", err)
	}

	second := cves[1]
	if !second.HasScore || second.Score != 5.0 || second.Severity != "MEDIUM" {
		t.Errorf("second CVE (v2 fallback) score/severity = %v/%q, want 5.0/MEDIUM", second.Score, second.Severity)
	}
}

func TestFetchByKeywordHandlesMissingScore(t *testing.T) {
	cleanup := withMockNVD(t, sampleNVDResponse)
	defer cleanup()

	cves, err := fetchByKeywordFrom(nvdBaseURLForTest, "CCSDS", 10)
	if err != nil {
		t.Fatalf("fetchByKeywordFrom returned error: %v", err)
	}

	third := cves[2]
	if third.HasScore {
		t.Error("third CVE has no metrics, HasScore should be false")
	}
}

func TestFetchByKeywordHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	if _, err := fetchByKeywordFrom(server.URL, "CCSDS", 10); err == nil {
		t.Error("fetchByKeywordFrom should return an error on a non-200 response")
	}
}

func TestFetchByKeywordMalformedJSON(t *testing.T) {
	cleanup := withMockNVD(t, "not json")
	defer cleanup()

	if _, err := fetchByKeywordFrom(nvdBaseURLForTest, "CCSDS", 10); err == nil {
		t.Error("fetchByKeywordFrom should return an error on malformed JSON")
	}
}
