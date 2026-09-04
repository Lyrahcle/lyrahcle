package satnogs

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	SourceNetwork          = "satnogs_network"
	EvidenceKindExternal   = "external_community"
	ApplicabilityReception = "reception_context_only"

	MaxImportRecords     = 200
	maxImportConfidence  = 60
	unknownConfidence    = 30
	unsuccessfulConfid   = 10
	maxObservationWindow = 90 * 24 * time.Hour
)

type ObservationRecord struct {
	ExternalID         int64      `json:"external_id"`
	Source             string     `json:"source"`
	StationID          int64      `json:"station_id"`
	StationName        string     `json:"station_name"`
	Observer           string     `json:"observer"`
	TransmitterUUID    string     `json:"transmitter_uuid"`
	Mode               string     `json:"mode"`
	DownlinkHz         *int64     `json:"downlink_hz,omitempty"`
	StationLatitude    *float64   `json:"station_latitude,omitempty"`
	StationLongitude   *float64   `json:"station_longitude,omitempty"`
	StartedAt          time.Time  `json:"started_at"`
	EndedAt            time.Time  `json:"ended_at"`
	Status             string     `json:"status"`
	VettedStatus       string     `json:"vetted_status"`
	ReceptionConfirmed bool       `json:"reception_confirmed"`
	EvidenceKind       string     `json:"evidence_kind"`
	Applicability      string     `json:"applicability"`
	Confidence         int        `json:"confidence"`
	ObservedAt         time.Time  `json:"observed_at"`
	WaterfallURL       string     `json:"waterfall_url,omitempty"`
	LastReception      *time.Time `json:"-"`
}

type ImportResult struct {
	NORADID     int                 `json:"norad_id"`
	Source      string              `json:"source"`
	SourceState SourceState         `json:"source_state"`
	Records     []ObservationRecord `json:"records"`
	Skipped     int                 `json:"skipped"`
}

func (c *Client) FetchObservations(ctx context.Context, noradID int, since time.Time, limit int) ([]Observation, SourceState) {
	now := c.now()
	if limit <= 0 || limit > MaxImportRecords {
		limit = MaxImportRecords
	}
	if since.IsZero() || now.Sub(since) > maxObservationWindow {
		since = now.Add(-maxObservationWindow)
	}
	params := map[string]string{
		"norad_cat_id": strconv.Itoa(noradID),
		"start":        since.UTC().Format(time.RFC3339),
		"end":          now.Format(time.RFC3339),
		"ordering":     "-end",
		"page_size":    strconv.Itoa(minInt(limit, 100)),
		"format":       "json",
	}
	var raw json.RawMessage
	state := c.fetchJSON(ctx, "SatNOGS Network observations", endpoint(c.networkURL+"/observations/", params), &raw, now)
	if !state.Available {
		return nil, state
	}
	observations := decodeObservations(raw)
	sort.Slice(observations, func(i, j int) bool { return observations[i].End.After(observations[j].End) })
	if len(observations) > limit {
		observations = observations[:limit]
	}
	state.RecordCount = len(observations)
	return observations, state
}

func (c *Client) Import(ctx context.Context, noradID int, since time.Time, limit int) ImportResult {
	observations, state := c.FetchObservations(ctx, noradID, since, limit)
	records, skipped := BuildObservationRecords(observations, c.now())
	result := ImportResult{NORADID: noradID, Source: SourceNetwork, SourceState: state, Records: records, Skipped: skipped}
	if result.Records == nil {
		result.Records = []ObservationRecord{}
	}
	return result
}

func BuildObservationRecords(observations []Observation, now time.Time) ([]ObservationRecord, int) {
	seen := map[int64]bool{}
	records := make([]ObservationRecord, 0, len(observations))
	skipped := 0
	for _, observation := range observations {
		if observation.ID <= 0 || seen[observation.ID] {
			skipped++
			continue
		}
		if observation.Start.IsZero() || observation.End.IsZero() || !observation.End.After(observation.Start) {
			skipped++
			continue
		}
		if observation.End.After(now) {
			skipped++
			continue
		}
		if now.Sub(observation.End) > maxObservationWindow {
			skipped++
			continue
		}
		seen[observation.ID] = true

		status := strings.ToLower(strings.TrimSpace(observation.Status))
		vetted := strings.ToLower(strings.TrimSpace(observation.VettedStatus))
		confirmed := isConfirmedReception(status, vetted)
		records = append(records, ObservationRecord{
			ExternalID:         observation.ID,
			Source:             SourceNetwork,
			StationID:          observation.StationID,
			StationName:        trimField(observation.Station, 200),
			Observer:           trimField(observation.Observer, 200),
			TransmitterUUID:    trimField(observation.Transmitter, 100),
			Mode:               trimField(observation.TransmitterMode, 50),
			DownlinkHz:         sanitizeFrequency(observation.TransmitterDownlinkLow),
			StationLatitude:    sanitizeCoordinate(observation.StationLatitude, 90),
			StationLongitude:   sanitizeCoordinate(observation.StationLongitude, 180),
			StartedAt:          observation.Start.UTC(),
			EndedAt:            observation.End.UTC(),
			Status:             trimField(status, 50),
			VettedStatus:       trimField(vetted, 50),
			ReceptionConfirmed: confirmed,
			EvidenceKind:       EvidenceKindExternal,
			Applicability:      ApplicabilityReception,
			Confidence:         receptionConfidence(status, vetted, confirmed),
			ObservedAt:         observation.End.UTC(),
			WaterfallURL:       httpsOnly(observation.Waterfall),
		})
	}
	sort.Slice(records, func(i, j int) bool { return records[i].ObservedAt.After(records[j].ObservedAt) })
	if len(records) > MaxImportRecords {
		skipped += len(records) - MaxImportRecords
		records = records[:MaxImportRecords]
	}
	return records, skipped
}

func LatestConfirmedReception(records []ObservationRecord) *time.Time {
	for _, record := range records {
		if !record.ReceptionConfirmed {
			continue
		}
		observed := record.ObservedAt
		return &observed
	}
	return nil
}

func isConfirmedReception(status, vetted string) bool {
	switch {
	case strings.Contains(status, "good"), strings.Contains(vetted, "good"):
		return true
	case strings.Contains(status, "bad"), strings.Contains(vetted, "bad"):
		return false
	case strings.Contains(status, "failed"), strings.Contains(vetted, "failed"):
		return false
	default:
		return false
	}
}

func receptionConfidence(status, vetted string, confirmed bool) int {
	if confirmed {
		return maxImportConfidence
	}
	if strings.Contains(status, "bad") || strings.Contains(status, "failed") || strings.Contains(vetted, "bad") || strings.Contains(vetted, "failed") {
		return unsuccessfulConfid
	}
	return unknownConfidence
}

func sanitizeFrequency(value *int64) *int64 {
	if value == nil || *value <= 0 || *value > 1_000_000_000_000 {
		return nil
	}
	frequency := *value
	return &frequency
}

func sanitizeCoordinate(value *float64, bound float64) *float64 {
	if value == nil {
		return nil
	}
	if *value != *value || *value < -bound || *value > bound {
		return nil
	}
	coordinate := *value
	return &coordinate
}

func trimField(value string, max int) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) > max {
		return trimmed[:max]
	}
	return trimmed
}

func httpsOnly(value string) string {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(strings.ToLower(trimmed), "https://") {
		return ""
	}
	if len(trimmed) > 500 {
		return ""
	}
	return trimmed
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
