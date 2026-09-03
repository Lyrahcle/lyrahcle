package capture

import (
	"math/rand"
	"time"
)

type SpectralMetrics struct {
	PowerDbFS             float64
	NoiseFloorDbFS        float64
	PeakPowerDbFS         float64
	SNRDb                 float64
	CarrierOffsetHz       float64
	CarrierFrequencyHz    float64
	OccupiedBandwidthHz   float64
	ResolutionBandwidthHz float64
	BinWidthHz            float64
	SampleRateHz          float64
	FFTSize               int
	Segments              int
	EstimatorMethod       string
	CalibrationOffsetDb   float64
	Calibrated            bool
}

type Reading struct {
	FrequencyHz int64
	PowerDbm    float64
	CapturedAt  time.Time
	Spectral    *SpectralMetrics
}

type Source interface {
	Read(frequencyHz int64) (Reading, error)
}

type MockSource struct{}

func (MockSource) Read(frequencyHz int64) (Reading, error) {
	return Reading{
		FrequencyHz: frequencyHz,
		PowerDbm:    -90 + rand.Float64()*40,
		CapturedAt:  time.Now().UTC(),
	}, nil
}
