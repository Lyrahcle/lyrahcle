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
	if result.ZScore <= ZScoreThreshold {
		t.Errorf("ZScore = %f, want > %f", result.ZScore, ZScoreThreshold)
	}
}

func TestDetectValueWithinRangeIsNotAnomaly(t *testing.T) {
	history := []float64{-60, -61, -59, -60.5, -59.5, -60.2, -60.1}
	result := Detect("power_dbm", history, -60.3)
	if result.Anomaly {
		t.Errorf("Detect flagged a value close to the mean as an anomaly; z=%f", result.ZScore)
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
	if result.ZScore >= -ZScoreThreshold {
		t.Errorf("ZScore = %f, want < %f", result.ZScore, -ZScoreThreshold)
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
