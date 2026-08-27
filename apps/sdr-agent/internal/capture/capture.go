package capture

import (
	"math/rand"
	"time"
)

type Reading struct {
	FrequencyHz int64
	PowerDbm    float64
	CapturedAt  time.Time
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
