package satnogs

import (
	"math"
	"time"

	"github.com/mlko01/lyrahcle/apps/backend/internal/orbitalcontext"
)

const (
	ConsistencyConsistent   = "consistent"
	ConsistencyInconsistent = "inconsistent"
	ConsistencyUnknown      = "unknown"

	geometrySamples  = 9
	speedOfLightMps  = 299_792_458.0
	minVisibleDegree = 0.0
)

type GeometryCheck struct {
	ExternalID        int64    `json:"external_id"`
	Evaluated         bool     `json:"evaluated"`
	Visible           bool     `json:"visible"`
	MaxElevationDeg   float64  `json:"max_elevation_deg"`
	RangeRateMps      *float64 `json:"range_rate_mps,omitempty"`
	ExpectedDopplerHz *float64 `json:"expected_doppler_hz,omitempty"`
	Consistency       string   `json:"consistency"`
	Reason            string   `json:"reason"`
}

func CheckGeometry(records []ObservationRecord, tleLine1, tleLine2 string) []GeometryCheck {
	out := make([]GeometryCheck, 0, len(records))
	for _, record := range records {
		out = append(out, checkRecord(record, tleLine1, tleLine2))
	}
	return out
}

func checkRecord(record ObservationRecord, tleLine1, tleLine2 string) GeometryCheck {
	check := GeometryCheck{ExternalID: record.ExternalID, Consistency: ConsistencyUnknown}
	if tleLine1 == "" || tleLine2 == "" {
		check.Reason = "no orbital elements available for this satellite"
		return check
	}
	if record.StationLatitude == nil || record.StationLongitude == nil {
		check.Reason = "the observing station did not publish its coordinates"
		return check
	}
	if record.EndedAt.Before(record.StartedAt) {
		check.Reason = "the observation window is not usable"
		return check
	}

	station := orbitalcontext.Station{LatitudeDeg: *record.StationLatitude, LongitudeDeg: *record.StationLongitude}
	window := record.EndedAt.Sub(record.StartedAt)
	step := window / time.Duration(geometrySamples-1)
	if step <= 0 {
		step = time.Second
	}

	maxElevation := math.Inf(-1)
	var peakRangeRate float64
	evaluated := false
	for index := 0; index < geometrySamples; index++ {
		at := record.StartedAt.Add(time.Duration(index) * step)
		if at.After(record.EndedAt) {
			break
		}
		result, err := orbitalcontext.Evaluate(station, tleLine1, tleLine2, at, minVisibleDegree)
		if err != nil {
			continue
		}
		evaluated = true
		if result.ElevationDeg > maxElevation {
			maxElevation = result.ElevationDeg
			peakRangeRate = result.RangeRateMPS
		}
	}
	if !evaluated || math.IsInf(maxElevation, -1) {
		check.Reason = "the orbital model could not be evaluated for this window"
		return check
	}

	check.Evaluated = true
	check.MaxElevationDeg = maxElevation
	check.Visible = maxElevation >= minVisibleDegree
	rangeRate := peakRangeRate
	check.RangeRateMps = &rangeRate

	if record.DownlinkHz != nil && *record.DownlinkHz > 0 {
		doppler := -rangeRate / speedOfLightMps * float64(*record.DownlinkHz)
		if !math.IsNaN(doppler) && !math.IsInf(doppler, 0) {
			check.ExpectedDopplerHz = &doppler
		}
	}

	switch {
	case record.ReceptionConfirmed && check.Visible:
		check.Consistency = ConsistencyConsistent
		check.Reason = "a vetted reception coincides with a modelled visibility window"
	case record.ReceptionConfirmed && !check.Visible:
		check.Consistency = ConsistencyInconsistent
		check.Reason = "a vetted reception falls outside the modelled visibility window, which usually means stale orbital elements or a mismatched satellite"
	case !record.ReceptionConfirmed && check.Visible:
		check.Consistency = ConsistencyUnknown
		check.Reason = "the satellite was modelled as visible but the upstream network did not vet a reception"
	default:
		check.Consistency = ConsistencyUnknown
		check.Reason = "no vetted reception and no modelled visibility window"
	}
	return check
}
