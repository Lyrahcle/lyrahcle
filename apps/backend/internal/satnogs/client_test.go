package satnogs

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)
func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }

func TestFetchNormalizesSatNOGSSources(t *testing.T) {
	now := time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)
	client := NewClient(time.Second)
	client.now = func() time.Time { return now }
	client.httpClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var body string
		switch {
		case strings.Contains(req.URL.Path, "/satellites/"): body = `[{"sat_id":"ISS","norad_cat_id":25544,"name":"ISS","status":"alive"}]`
		case strings.Contains(req.URL.Path, "/transmitters/"): body = `[{"uuid":"tx-1","description":"Packet","downlink_low":145800000,"mode":"FM","alive":true,"status":"active"}]`
		default: body = `{"count":1,"results":[{"id":42,"start":"2026-09-03T08:00:00Z","end":"2026-09-03T08:10:00Z","status":"good","ground_station":7}]}`
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	})
	got := client.Fetch(context.Background(), 25544)
	if got.Metadata == nil || got.Metadata.Name != "ISS" || len(got.Transmitters) != 1 || len(got.Observations) != 1 { t.Fatalf("unexpected intelligence: %+v", got) }
	if got.OverallQuality != "good" || got.LastReception == nil || got.Sources[2].Freshness != "fresh" { t.Fatalf("unexpected quality: %+v", got) }
}

func TestReceptionFreshnessBoundaries(t *testing.T) {
	now := time.Now().UTC()
	fresh := now.Add(-time.Hour)
	aging := now.Add(-48*time.Hour)
	stale := now.Add(-8*24*time.Hour)
	if receptionFreshness(&fresh, now) != "fresh" || receptionFreshness(&aging, now) != "aging" || receptionFreshness(&stale, now) != "stale" || receptionFreshness(nil, now) != "unknown" { t.Fatal("unexpected freshness classification") }
}

func TestEndpointEncodesNORADFilter(t *testing.T) {
	got := endpoint(databaseAPI+"/satellites/", map[string]string{"norad_cat_id": "25544"})
	if !strings.Contains(got, "norad_cat_id=25544") { t.Fatalf("unexpected endpoint: %s", got) }
}
