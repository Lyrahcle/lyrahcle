package satnogs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	databaseAPI = "https://db.satnogs.org/api"
	networkAPI  = "https://network.satnogs.org/api"
	maxBodySize = 4 << 20
)

type SourceState struct {
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Available   bool      `json:"available"`
	FetchedAt   time.Time `json:"fetched_at"`
	RecordCount int       `json:"record_count"`
	Freshness   string    `json:"freshness"`
	Message     string    `json:"message,omitempty"`
}

type SatelliteMetadata struct {
	SatNOGSID string `json:"sat_id,omitempty"`
	NORADID   int    `json:"norad_cat_id"`
	Name      string `json:"name,omitempty"`
	Names     string `json:"names,omitempty"`
	Status    string `json:"status,omitempty"`
	Image     string `json:"image,omitempty"`
	Website   string `json:"website,omitempty"`
	Countries string `json:"countries,omitempty"`
}

type Transmitter struct {
	UUID           string `json:"uuid"`
	Description    string `json:"description,omitempty"`
	Mode           string `json:"mode,omitempty"`
	Status         string `json:"status,omitempty"`
	Alive          bool   `json:"alive"`
	DownlinkLowHz  *int64 `json:"downlink_low_hz,omitempty"`
	DownlinkHighHz *int64 `json:"downlink_high_hz,omitempty"`
	UplinkLowHz    *int64 `json:"uplink_low_hz,omitempty"`
	UplinkHighHz   *int64 `json:"uplink_high_hz,omitempty"`
}

func (t *Transmitter) UnmarshalJSON(data []byte) error {
	type plain Transmitter
	var raw struct {
		plain
		DownlinkLow *int64 `json:"downlink_low"`
		DownlinkHigh *int64 `json:"downlink_high"`
		UplinkLow *int64 `json:"uplink_low"`
		UplinkHigh *int64 `json:"uplink_high"`
	}
	if err := json.Unmarshal(data, &raw); err != nil { return err }
	*t = Transmitter(raw.plain)
	t.DownlinkLowHz, t.DownlinkHighHz = raw.DownlinkLow, raw.DownlinkHigh
	t.UplinkLowHz, t.UplinkHighHz = raw.UplinkLow, raw.UplinkHigh
	return nil
}

type Observation struct {
	ID                    int64     `json:"id"`
	Start                 time.Time `json:"start"`
	End                   time.Time `json:"end"`
	Status                string    `json:"status,omitempty"`
	StationID             int64     `json:"ground_station,omitempty"`
	Station               string    `json:"station_name,omitempty"`
	Transmitter           string    `json:"transmitter,omitempty"`
	VettedStatus          string    `json:"vetted_status,omitempty"`
	Observer              string    `json:"observer,omitempty"`
	TransmitterMode       string    `json:"transmitter_mode,omitempty"`
	TransmitterDownlinkLow *int64   `json:"transmitter_downlink_low,omitempty"`
	StationLatitude       *float64  `json:"station_lat,omitempty"`
	StationLongitude      *float64  `json:"station_lng,omitempty"`
	Waterfall             string    `json:"waterfall,omitempty"`
}

type Intelligence struct {
	NORADID        int               `json:"norad_id"`
	EvidenceKind   string            `json:"evidence_kind"`
	Applicability  string            `json:"applicability"`
	Metadata       *SatelliteMetadata `json:"metadata,omitempty"`
	Transmitters   []Transmitter     `json:"transmitters"`
	Observations   []Observation     `json:"observations"`
	LastReception  *time.Time        `json:"last_reception,omitempty"`
	OverallQuality string            `json:"overall_quality"`
	Sources        []SourceState     `json:"sources"`
}

type Client struct {
	httpClient *http.Client
	databaseURL string
	networkURL string
	now func() time.Time
}

func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 || timeout > 30*time.Second { timeout = 10 * time.Second }
	allowed := map[string]bool{"db.satnogs.org": true, "network.satnogs.org": true}
	return &Client{
		httpClient: &http.Client{Timeout: timeout, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 || req.URL.Scheme != "https" || !allowed[strings.ToLower(req.URL.Hostname())] { return http.ErrUseLastResponse }
			return nil
		}},
		databaseURL: databaseAPI,
		networkURL: networkAPI,
		now: func() time.Time { return time.Now().UTC() },
	}
}

func (c *Client) Fetch(ctx context.Context, noradID int) Intelligence {
	now := c.now()
	result := Intelligence{NORADID: noradID, EvidenceKind: "external_community", Applicability: "radio_activity_context_only", Transmitters: []Transmitter{}, Observations: []Observation{}}

	metadataURL := endpoint(c.databaseURL+"/satellites/", map[string]string{"norad_cat_id": strconv.Itoa(noradID), "format": "json"})
	var metadata []SatelliteMetadata
	state := c.fetchJSON(ctx, "SatNOGS DB metadata", metadataURL, &metadata, now)
	state.RecordCount = len(metadata)
	if len(metadata) > 0 { result.Metadata = &metadata[0] }
	result.Sources = append(result.Sources, state)

	transmitterURL := endpoint(c.databaseURL+"/transmitters/", map[string]string{"satellite__norad_cat_id": strconv.Itoa(noradID), "format": "json"})
	state = c.fetchJSON(ctx, "SatNOGS DB transmitters", transmitterURL, &result.Transmitters, now)
	state.RecordCount = len(result.Transmitters)
	result.Sources = append(result.Sources, state)

	observationURL := endpoint(c.networkURL+"/observations/", map[string]string{"norad_cat_id": strconv.Itoa(noradID), "end": now.Format(time.RFC3339), "ordering": "-end", "page_size": "10", "format": "json"})
	var raw json.RawMessage
	state = c.fetchJSON(ctx, "SatNOGS Network observations", observationURL, &raw, now)
	if state.Available {
		result.Observations = decodeObservations(raw)
		sort.Slice(result.Observations, func(i, j int) bool { return result.Observations[i].End.After(result.Observations[j].End) })
		if len(result.Observations) > 10 { result.Observations = result.Observations[:10] }
		for _, observation := range result.Observations {
			if !observation.End.IsZero() && !observation.End.After(now) && !strings.EqualFold(observation.Status, "future") { last := observation.End; result.LastReception = &last; break }
		}
	}
	state.RecordCount = len(result.Observations)
	state.Freshness = receptionFreshness(result.LastReception, now)
	result.Sources = append(result.Sources, state)
	result.OverallQuality = overallQuality(result.Sources, result.LastReception, now)
	return result
}

func (c *Client) fetchJSON(ctx context.Context, name, target string, destination interface{}, now time.Time) SourceState {
	state := SourceState{Name: name, URL: target, FetchedAt: now, Freshness: "current"}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil { state.Message = "request could not be created"; state.Freshness = "unavailable"; return state }
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Lyrahcle/1.0 satellite-source-sync")
	response, err := c.httpClient.Do(req)
	if err != nil { state.Message = "source unavailable"; state.Freshness = "unavailable"; return state }
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK { state.Message = fmt.Sprintf("source returned HTTP %d", response.StatusCode); state.Freshness = "unavailable"; return state }
	if response.ContentLength > maxBodySize { state.Message = "source response too large"; state.Freshness = "unavailable"; return state }
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBodySize+1))
	if err != nil || len(body) > maxBodySize { state.Message = "source response could not be read"; state.Freshness = "unavailable"; return state }
	if raw, ok := destination.(*json.RawMessage); ok { *raw = append((*raw)[:0], body...); state.Available = true; return state }
	if err := json.Unmarshal(body, destination); err != nil { state.Message = "source returned invalid data"; state.Freshness = "unavailable"; return state }
	state.Available = true
	return state
}

func endpoint(base string, values map[string]string) string {
	u, _ := url.Parse(base)
	query := u.Query()
	for key, value := range values { query.Set(key, value) }
	u.RawQuery = query.Encode()
	return u.String()
}

func decodeObservations(raw json.RawMessage) []Observation {
	var direct []Observation
	if json.Unmarshal(raw, &direct) == nil { return direct }
	var page struct { Results []Observation `json:"results"` }
	if json.Unmarshal(raw, &page) == nil { return page.Results }
	return []Observation{}
}

func receptionFreshness(last *time.Time, now time.Time) string {
	if last == nil || last.IsZero() { return "unknown" }
	age := now.Sub(*last)
	if age < 0 { return "unknown" }
	if age <= 24*time.Hour { return "fresh" }
	if age <= 7*24*time.Hour { return "aging" }
	return "stale"
}

func overallQuality(sources []SourceState, last *time.Time, now time.Time) string {
	available := 0
	for _, source := range sources { if source.Available { available++ } }
	if available == 0 { return "unavailable" }
	if available < len(sources) { return "degraded" }
	switch receptionFreshness(last, now) { case "fresh": return "good"; case "aging", "unknown": return "degraded"; default: return "stale" }
}
