package anomaly

import (
	"math"
	"sort"
)

const (
	MinSamples = 5

	ModifiedZScoreThreshold = 3.5

	madConsistencyConstant = 1.4826

	EWMALambda = 0.2

	EWMAControlLimit = 3.0
)

type Result struct {
	Metric  string
	Value   float64
	Mean    float64
	StdDev  float64
	ZScore  float64
	Anomaly bool
}

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

	ewma := history[len(history)-1]
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
