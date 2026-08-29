package anomaly

import (
	"math"
	"testing"
)

func TestDetectNotEnoughHistory(t *testing.T) {
	result := Detect("power_dbm", []float64{-60, -61, -59}, -20)
	if result.Anomaly {
		t.Error("Detect should not flag an anomaly with fewer than MinSamples history points")
	}
}

func TestDetectStableHistoryFlagsOutlier(t *testing.T) {
	history := []float64{-60, -61, -59, -60.5, -59.5, -60.2, -60.1}
	result := Detect("power_dbm", history, -20)
	if !result.Anomaly {
		t.Errorf("Detect should flag -20 as an anomaly against a stable ~-60 baseline; z=%f", result.ZScore)
	}
	if result.ZScore <= ModifiedZScoreThreshold {
		t.Errorf("ZScore = %f, want > %f", result.ZScore, ModifiedZScoreThreshold)
	}
}

func TestDetectValueWithinRangeIsNotAnomaly(t *testing.T) {
	history := []float64{-60, -61, -59, -60.5, -59.5, -60.2, -60.1}
	result := Detect("power_dbm", history, -60.3)
	if result.Anomaly {
		t.Errorf("Detect flagged a value close to the median as an anomaly; z=%f", result.ZScore)
	}
}

func TestDetectZeroVarianceHistoryNeverAnomalous(t *testing.T) {
	history := []float64{-60, -60, -60, -60, -60}
	result := Detect("power_dbm", history, -20)
	if result.Anomaly {
		t.Error("Detect should not divide by zero or flag an anomaly when history has zero variance")
	}
	if result.StdDev != 0 {
		t.Errorf("StdDev = %f, want 0", result.StdDev)
	}
}

func TestDetectNegativeOutlier(t *testing.T) {
	history := []float64{-60, -61, -59, -60.5, -59.5, -60.2, -60.1}
	result := Detect("power_dbm", history, -95)
	if !result.Anomaly {
		t.Errorf("Detect should flag -95 as an anomaly against a stable ~-60 baseline; z=%f", result.ZScore)
	}
	if result.ZScore >= -ModifiedZScoreThreshold {
		t.Errorf("ZScore = %f, want < %f", result.ZScore, -ModifiedZScoreThreshold)
	}
}

func TestDetectResistsAPriorOutlierInHistory(t *testing.T) {

	history := []float64{-60, -61, -59, -60.5, -59.5, -60.2, -15}
	result := Detect("power_dbm", history, -60.4)
	if result.Anomaly {
		t.Errorf("a normal -60ish reading was flagged, median-based baseline was skewed by the -15 outlier; median=%f mad-sigma=%f z=%f", result.Mean, result.StdDev, result.ZScore)
	}
	if math.Abs(result.Mean-(-60)) > 1.0 {
		t.Errorf("baseline median = %f, want close to -60 despite the -15 outlier in history", result.Mean)
	}
}

func TestDetectFallsBackToStdDevWhenMADIsZero(t *testing.T) {

	history := []float64{-60, -60, -60, -60, -55}
	result := Detect("power_dbm", history, -60)
	if result.StdDev == 0 {
		t.Error("expected a non-zero fallback stddev when MAD collapses to 0")
	}
}

func TestMedianOddAndEven(t *testing.T) {
	if got := Median([]float64{3, 1, 2}); got != 2 {
		t.Errorf("Median(odd) = %f, want 2", got)
	}
	if got := Median([]float64{1, 2, 3, 4}); got != 2.5 {
		t.Errorf("Median(even) = %f, want 2.5", got)
	}
	if got := Median(nil); got != 0 {
		t.Errorf("Median(nil) = %f, want 0", got)
	}
}

func TestMedianDoesNotMutateInput(t *testing.T) {
	values := []float64{5, 3, 1, 4, 2}
	original := append([]float64(nil), values...)
	Median(values)
	for i := range values {
		if values[i] != original[i] {
			t.Fatalf("Median mutated its input: got %v, want %v", values, original)
		}
	}
}

func TestMADKnownValues(t *testing.T) {

	values := []float64{1, 2, 2, 4, 3, 6, 5}
	median := Median(values)
	if median != 3 {
		t.Fatalf("Median = %f, want 3", median)
	}
	if got := MAD(values, median); got != 1 {
		t.Errorf("MAD = %f, want 1", got)
	}
}

func TestMeanEmpty(t *testing.T) {
	if Mean(nil) != 0 {
		t.Error("Mean of an empty slice should be 0")
	}
}

func TestStdDevSingleValue(t *testing.T) {
	if StdDev([]float64{5}, 5) != 0 {
		t.Error("StdDev of a single-value slice should be 0")
	}
}

func TestStdDevKnownValues(t *testing.T) {
	values := []float64{2, 4, 4, 4, 5, 5, 7, 9}
	mean := Mean(values)
	got := StdDev(values, mean)
	want := 2.138089935
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("StdDev = %f, want ~%f", got, want)
	}
}

func TestDetectTrendNotEnoughHistory(t *testing.T) {
	result := DetectTrend("power_dbm", []float64{-60, -61, -59}, -20)
	if result.Anomaly {
		t.Error("DetectTrend should not flag an anomaly with fewer than MinSamples history points")
	}
}

func TestDetectTrendZeroVarianceNeverAnomalous(t *testing.T) {
	history := []float64{-60, -60, -60, -60, -60}
	result := DetectTrend("power_dbm", history, -60)
	if result.Anomaly {
		t.Error("DetectTrend should not flag an anomaly when history has zero variance")
	}
}

func TestDetectTrendMetricNameHasSuffix(t *testing.T) {
	history := []float64{-60, -61, -59, -60.5, -59.5}
	result := DetectTrend("power_dbm", history, -60)
	if result.Metric != "power_dbm_trend" {
		t.Errorf("Metric = %q, want %q", result.Metric, "power_dbm_trend")
	}
}

func TestDetectTrendUsesChronologicalOrder(t *testing.T) {

	history := []float64{-61, -62, -63, -64, -65}
	result := DetectTrend("power_dbm", history, -61)

	want := -63.3616
	if math.Abs(result.Mean-want) > 1e-6 {
		t.Errorf("EWMA trend = %f, want %f (chronological oldest-to-newest replay); history may have been replayed in the wrong order", result.Mean, want)
	}
}

func TestDetectTrendFlagsSustainedShift(t *testing.T) {

	history := []float64{
		-60, -60.1, -59.9, -60, -60.05, -59.95, -60,
		-63, -63.1, -62.9, -63, -63.05,
	}
	result := DetectTrend("power_dbm", history, -63)
	if !result.Anomaly {
		t.Errorf("expected the sustained step to trip the EWMA control limit; ewma=%f sigma=%f z=%f", result.Mean, result.StdDev, result.ZScore)
	}
}
