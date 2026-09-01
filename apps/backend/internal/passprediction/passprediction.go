package passprediction

import (
	"fmt"
	"math"
	"time"

	"github.com/mlko01/lyrahcle/apps/backend/internal/orbitalcontext"
)

const (
	defaultStep      = 30 * time.Second
	refineResolution = time.Second
)

type Request struct {
	Station         orbitalcontext.Station
	From            time.Time
	Horizon         time.Duration
	MinElevationDeg float64
	Limit           int
}

type Pass struct {
	AOS             time.Time `json:"aos"`
	LOS             time.Time `json:"los"`
	PeakAt          time.Time `json:"peak_at"`
	MaxElevationDeg float64   `json:"max_elevation_deg"`
	DurationSeconds int64     `json:"duration_seconds"`
	AOSTruncated    bool      `json:"aos_truncated,omitempty"`
	LOSTruncated    bool      `json:"los_truncated,omitempty"`
}

type Result struct {
	From            time.Time `json:"from"`
	Until           time.Time `json:"until"`
	MinElevationDeg float64   `json:"min_elevation_deg"`
	Passes          []Pass    `json:"passes"`
}

func Predict(tleLine1, tleLine2 string, req Request) (Result, error) {
	if req.Horizon <= 0 {
		return Result{}, fmt.Errorf("horizon must be positive")
	}
	if req.Limit <= 0 {
		return Result{}, fmt.Errorf("limit must be positive")
	}
	if req.MinElevationDeg < -90 || req.MinElevationDeg > 90 {
		return Result{}, fmt.Errorf("min elevation must be between -90 and 90 degrees")
	}

	from := req.From.UTC()
	until := from.Add(req.Horizon)
	out := Result{From: from, Until: until, MinElevationDeg: req.MinElevationDeg, Passes: []Pass{}}

	firstElevation, err := elevationAt(req.Station, tleLine1, tleLine2, from)
	if err != nil {
		return Result{}, err
	}
	above := firstElevation >= req.MinElevationDeg
	var current *Pass
	if above {
		current = &Pass{AOS: from, AOSTruncated: true}
	}

	prevAt := from
	for at := from.Add(defaultStep); !at.After(until); at = at.Add(defaultStep) {
		elevation, err := elevationAt(req.Station, tleLine1, tleLine2, at)
		if err != nil {
			return Result{}, err
		}
		nowAbove := elevation >= req.MinElevationDeg

		if !above && nowAbove {
			aos, err := refineCrossing(req.Station, tleLine1, tleLine2, prevAt, at, req.MinElevationDeg, true)
			if err != nil {
				return Result{}, err
			}
			current = &Pass{AOS: aos}
		}
		if above && !nowAbove && current != nil {
			los, err := refineCrossing(req.Station, tleLine1, tleLine2, prevAt, at, req.MinElevationDeg, false)
			if err != nil {
				return Result{}, err
			}
			current.LOS = los
			if err := completePass(current, req.Station, tleLine1, tleLine2); err != nil {
				return Result{}, err
			}
			out.Passes = append(out.Passes, *current)
			current = nil
			if len(out.Passes) >= req.Limit {
				return out, nil
			}
		}

		above = nowAbove
		prevAt = at
	}

	if current != nil && len(out.Passes) < req.Limit {
		current.LOS = until
		current.LOSTruncated = true
		if err := completePass(current, req.Station, tleLine1, tleLine2); err != nil {
			return Result{}, err
		}
		out.Passes = append(out.Passes, *current)
	}

	return out, nil
}

func elevationAt(station orbitalcontext.Station, line1, line2 string, at time.Time) (float64, error) {
	result, err := orbitalcontext.Evaluate(station, line1, line2, at, 0)
	if err != nil {
		return 0, err
	}
	return result.ElevationDeg, nil
}

func refineCrossing(station orbitalcontext.Station, line1, line2 string, low, high time.Time, threshold float64, rising bool) (time.Time, error) {
	low = low.UTC()
	high = high.UTC()
	for high.Sub(low) > refineResolution {
		mid := low.Add(high.Sub(low) / 2)
		elevation, err := elevationAt(station, line1, line2, mid)
		if err != nil {
			return time.Time{}, err
		}
		above := elevation >= threshold
		if rising {
			if above {
				high = mid
			} else {
				low = mid
			}
		} else {
			if above {
				low = mid
			} else {
				high = mid
			}
		}
	}
	return high, nil
}

func completePass(pass *Pass, station orbitalcontext.Station, line1, line2 string) error {
	if pass.LOS.Before(pass.AOS) {
		return fmt.Errorf("invalid pass interval")
	}
	pass.DurationSeconds = int64(pass.LOS.Sub(pass.AOS).Seconds())
	peakAt, maxElevation, err := findPeak(station, line1, line2, pass.AOS, pass.LOS)
	if err != nil {
		return err
	}
	pass.PeakAt = peakAt
	pass.MaxElevationDeg = math.Round(maxElevation*100) / 100
	return nil
}

func findPeak(station orbitalcontext.Station, line1, line2 string, start, end time.Time) (time.Time, float64, error) {
	if !end.After(start) {
		elevation, err := elevationAt(station, line1, line2, start)
		return start, elevation, err
	}

	low, high := start, end
	for high.Sub(low) > 2*time.Second {
		span := high.Sub(low)
		left := low.Add(span / 3)
		right := high.Add(-span / 3)
		leftElevation, err := elevationAt(station, line1, line2, left)
		if err != nil {
			return time.Time{}, 0, err
		}
		rightElevation, err := elevationAt(station, line1, line2, right)
		if err != nil {
			return time.Time{}, 0, err
		}
		if leftElevation < rightElevation {
			low = left
		} else {
			high = right
		}
	}
	peak := low.Add(high.Sub(low) / 2)
	elevation, err := elevationAt(station, line1, line2, peak)
	return peak, elevation, err
}
