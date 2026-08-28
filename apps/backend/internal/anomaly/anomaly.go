package anomaly

import "math"

const (
	MinSamples      = 5
	ZScoreThreshold = 3.0
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

	mean := Mean(history)
	stddev := StdDev(history, mean)
	result.Mean = mean
	result.StdDev = stddev

	if stddev == 0 {
		return result
	}

	z := (value - mean) / stddev
	result.ZScore = z
	result.Anomaly = math.Abs(z) > ZScoreThreshold
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
