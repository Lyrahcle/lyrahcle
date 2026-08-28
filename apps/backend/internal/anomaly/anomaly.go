// Package anomaly flags outliers and sustained drift in a metric's recent
// history using two complementary statistical baselines:
//
//   - Detect: a spot check against a robust "modified z-score" (median +
//     median absolute deviation, MAD). Unlike a plain mean/stddev z-score,
//     this resists being skewed by a handful of prior anomalies already
//     sitting in the history window — one wild reading can't drag the
//     baseline far enough to mask the next one.
//   - DetectTrend: an EWMA (exponentially weighted moving average) control
//     chart, which catches a sustained drift — e.g. a slowly failing
//     amplifier or a drifting oscillator — that a single-point comparison
//     against the window's center can miss, because each individual step
//     is too small to trip a spot check on its own.
//
// Both are standard statistical process control techniques, not a trained
// model: there's no external ML dependency, and every score is exactly
// reproducible from the input history.
package anomaly

import (
	"math"
	"sort"
)

const (
	MinSamples = 5

	// ModifiedZScoreThreshold is the standard robust-outlier threshold
	// for the modified z-score, per Iglewicz & Hoya (1993).
	ModifiedZScoreThreshold = 3.5

	// madConsistencyConstant scales MAD to be a consistent estimator of
	// the standard deviation under a normal distribution: 1/Φ⁻¹(3/4).
	madConsistencyConstant = 1.4826

	// EWMALambda is the smoothing constant for the trend baseline: how
	// much weight each new sample gets versus the accumulated trend.
	// Lower values react slower but catch smaller sustained shifts.
	EWMALambda = 0.2

	// EWMAControlLimit is the number of EWMA-scaled sigmas a reading must
	// deviate from the trend to be flagged — the "L" in a standard EWMA
	// control chart.
	EWMAControlLimit = 3.0
)

// Result mirrors the shape callers already persist as store.RFAnomaly:
// Mean/StdDev describe the baseline that Value was compared against, and
// ZScore is how many baseline-sigmas away it landed. What those fields
// represent depends on which detector produced the Result — see Detect
// and DetectTrend.
type Result struct {
	Metric  string
	Value   float64
	Mean    float64
	StdDev  float64
	ZScore  float64
	Anomaly bool
}

// Detect flags value as a spot outlier against history using the modified
// z-score: 0.6745 * (value - median) / MAD. Mean and StdDev in the result
// are the median and MAD-derived sigma, not the arithmetic mean/stddev.
//
// If MAD collapses to zero (every history value equal but one, or a
// perfectly flat baseline), Detect falls back to the classic mean/stddev
// z-score rather than reporting no spread at all.
func Detect(metric string, history []float64, value float64) Result {
	result := Result{Metric: metric, Value: value}
	if len(history) < MinSamples {
		return result
	}

	median := Median(history)
	mad := MAD(history, median)

	if mad == 0 {
		mean := Mean(history)
		stddev := StdDev(history, mean)
		result.Mean = mean
		result.StdDev = stddev
		if stddev == 0 {
			return result
		}
		z := (value - mean) / stddev
		result.ZScore = z
		result.Anomaly = math.Abs(z) > 3.0
		return result
	}

	result.Mean = median
	result.StdDev = mad * madConsistencyConstant
	z := 0.6745 * (value - median) / mad
	result.ZScore = z
	result.Anomaly = math.Abs(z) > ModifiedZScoreThreshold
	return result
}

// DetectTrend flags value as a deviation from the EWMA-smoothed trend of
// history, using a standard EWMA control chart. history is expected most-
// recent-first (as store.RFStore.RecentForAgent returns it); DetectTrend
// replays it in chronological order internally so the trend represents
// "up to just before now", not reversed.
//
// Mean in the result is the EWMA-projected value the new reading was
// compared against; StdDev is the EWMA's steady-state sigma
// (stddev * sqrt(λ/(2-λ))).
func DetectTrend(metric string, history []float64, value float64) Result {
	result := Result{Metric: metric + "_trend", Value: value}
	if len(history) < MinSamples {
		return result
	}

	mean := Mean(history)
	stddev := StdDev(history, mean)
	if stddev == 0 {
		return result
	}

	ewma := history[len(history)-1] // oldest point seeds the trend
	for i := len(history) - 2; i >= 0; i-- {
		ewma = EWMALambda*history[i] + (1-EWMALambda)*ewma
	}
	result.Mean = ewma

	ewmaSigma := stddev * math.Sqrt(EWMALambda/(2-EWMALambda))
	result.StdDev = ewmaSigma
	if ewmaSigma == 0 {
		return result
	}

	z := (value - ewma) / ewmaSigma
	result.ZScore = z
	result.Anomaly = math.Abs(z) > EWMAControlLimit
	return result
}

func Mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func StdDev(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	var sumSq float64
	for _, v := range values {
		d := v - mean
		sumSq += d * d
	}
	return math.Sqrt(sumSq / float64(len(values)-1))
}

// Median returns the median of values without modifying the input slice.
func Median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

// MAD returns the median absolute deviation of values from center,
// unscaled (multiply by madConsistencyConstant for a sigma-equivalent).
func MAD(values []float64, center float64) float64 {
	if len(values) == 0 {
		return 0
	}
	deviations := make([]float64, len(values))
	for i, v := range values {
		deviations[i] = math.Abs(v - center)
	}
	return Median(deviations)
}
