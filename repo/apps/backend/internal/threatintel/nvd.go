package threatintel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type CVE struct {
	ID          string
	Description string
	Published   time.Time
	Severity    string
	Score       float64
	HasScore    bool
}

type nvdResponse struct {
	Vulnerabilities []struct {
		CVE struct {
			ID           string `json:"id"`
			Published    string `json:"published"`
			Descriptions []struct {
				Lang  string `json:"lang"`
				Value string `json:"value"`
			} `json:"descriptions"`
			Metrics struct {
				CvssMetricV31 []cvssMetric `json:"cvssMetricV31"`
				CvssMetricV30 []cvssMetric `json:"cvssMetricV30"`
				CvssMetricV2  []cvssMetric `json:"cvssMetricV2"`
			} `json:"metrics"`
		} `json:"cve"`
	} `json:"vulnerabilities"`
}

type cvssMetric struct {
	CvssData struct {
		BaseScore    float64 `json:"baseScore"`
		BaseSeverity string  `json:"baseSeverity"`
	} `json:"cvssData"`
	BaseSeverity string `json:"baseSeverity"`
}

var nvdBaseURLForTest = nvdBaseURL

const nvdBaseURL = "https://services.nvd.nist.gov/rest/json/cves/2.0"

func FetchByKeyword(keyword string, resultsPerPage int) ([]CVE, error) {
	return fetchByKeywordFrom(nvdBaseURLForTest, keyword, resultsPerPage)
}

func fetchByKeywordFrom(baseURL, keyword string, resultsPerPage int) ([]CVE, error) {
	url := fmt.Sprintf("%s?keywordSearch=%s&resultsPerPage=%d", baseURL, keyword, resultsPerPage)

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("threatintel: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("threatintel: unexpected status %d", resp.StatusCode)
	}

	var parsed nvdResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("threatintel: decode response: %w", err)
	}

	out := make([]CVE, 0, len(parsed.Vulnerabilities))
	for _, v := range parsed.Vulnerabilities {
		cve := CVE{ID: v.CVE.ID, Severity: "unknown"}

		for _, d := range v.CVE.Descriptions {
			if d.Lang == "en" {
				cve.Description = d.Value
				break
			}
		}

		if t, err := time.Parse("2006-01-02T15:04:05.000", v.CVE.Published); err == nil {
			cve.Published = t
		}

		if m := firstMetric(v.CVE.Metrics.CvssMetricV31); m != nil {
			cve.Score, cve.Severity, cve.HasScore = m.CvssData.BaseScore, m.CvssData.BaseSeverity, true
		} else if m := firstMetric(v.CVE.Metrics.CvssMetricV30); m != nil {
			cve.Score, cve.Severity, cve.HasScore = m.CvssData.BaseScore, m.CvssData.BaseSeverity, true
		} else if m := firstMetric(v.CVE.Metrics.CvssMetricV2); m != nil {
			cve.Score, cve.Severity, cve.HasScore = m.CvssData.BaseScore, m.BaseSeverity, true
		}

		out = append(out, cve)
	}

	return out, nil
}

func firstMetric(metrics []cvssMetric) *cvssMetric {
	if len(metrics) == 0 {
		return nil
	}
	return &metrics[0]
}
