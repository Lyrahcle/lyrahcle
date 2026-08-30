package orbitalcontext

import (
	"math"
	"time"

	"github.com/mlko01/lyrahcle/apps/backend/internal/orbit"
)

const (
	earthRadiusM  = 6371000.0
	lightSpeedMPS = 299792458.0
)

type Station struct {
	LatitudeDeg  float64 `json:"latitude_deg"`
	LongitudeDeg float64 `json:"longitude_deg"`
	AltitudeM    float64 `json:"altitude_m"`
}

type Result struct {
	Visible           bool    `json:"visible"`
	ElevationDeg      float64 `json:"elevation_deg"`
	RangeKm           float64 `json:"range_km"`
	RangeRateMPS      float64 `json:"range_rate_mps"`
	ExpectedDopplerHz float64 `json:"expected_doppler_hz"`
}

func Evaluate(station Station, tleLine1, tleLine2 string, at time.Time, carrierHz float64) (Result, error) {
	current, err := orbit.PositionAt(tleLine1, tleLine2, at)
	if err != nil {
		return Result{}, err
	}
	next, err := orbit.PositionAt(tleLine1, tleLine2, at.Add(time.Second))
	if err != nil {
		return Result{}, err
	}
	stationECEF := geodeticToECEF(station.LatitudeDeg, station.LongitudeDeg, station.AltitudeM)
	currentECEF := geodeticToECEF(current.LatitudeDeg, current.LongitudeDeg, current.AltitudeKm*1000)
	nextECEF := geodeticToECEF(next.LatitudeDeg, next.LongitudeDeg, next.AltitudeKm*1000)
	rangeNow := distance(stationECEF, currentECEF)
	rangeNext := distance(stationECEF, nextECEF)
	rangeRate := rangeNext - rangeNow
	elevation := elevationDeg(station, current)
	doppler := 0.0
	if carrierHz > 0 {
		doppler = -carrierHz * rangeRate / lightSpeedMPS
	}
	return Result{
		Visible:           elevation > 0,
		ElevationDeg:      elevation,
		RangeKm:           rangeNow / 1000,
		RangeRateMPS:      rangeRate,
		ExpectedDopplerHz: doppler,
	}, nil
}

type vector struct{ x, y, z float64 }

func geodeticToECEF(latDeg, lonDeg, altitudeM float64) vector {
	lat := latDeg * math.Pi / 180
	lon := lonDeg * math.Pi / 180
	r := earthRadiusM + altitudeM
	cosLat := math.Cos(lat)
	return vector{x: r * cosLat * math.Cos(lon), y: r * cosLat * math.Sin(lon), z: r * math.Sin(lat)}
}

func distance(a, b vector) float64 {
	dx, dy, dz := b.x-a.x, b.y-a.y, b.z-a.z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func elevationDeg(station Station, satellite orbit.Position) float64 {
	lat := station.LatitudeDeg * math.Pi / 180
	lon := station.LongitudeDeg * math.Pi / 180
	stationECEF := geodeticToECEF(station.LatitudeDeg, station.LongitudeDeg, station.AltitudeM)
	satelliteECEF := geodeticToECEF(satellite.LatitudeDeg, satellite.LongitudeDeg, satellite.AltitudeKm*1000)
	dx, dy, dz := satelliteECEF.x-stationECEF.x, satelliteECEF.y-stationECEF.y, satelliteECEF.z-stationECEF.z
	east := -math.Sin(lon)*dx + math.Cos(lon)*dy
	north := -math.Sin(lat)*math.Cos(lon)*dx - math.Sin(lat)*math.Sin(lon)*dy + math.Cos(lat)*dz
	up := math.Cos(lat)*math.Cos(lon)*dx + math.Cos(lat)*math.Sin(lon)*dy + math.Sin(lat)*dz
	return math.Atan2(up, math.Sqrt(east*east+north*north)) * 180 / math.Pi
}
