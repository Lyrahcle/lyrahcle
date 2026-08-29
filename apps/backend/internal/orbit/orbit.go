package orbit

import (
	"time"

	satellite "github.com/joshuaferrara/go-satellite"
)

type Position struct {
	LatitudeDeg  float64 `json:"latitude_deg"`
	LongitudeDeg float64 `json:"longitude_deg"`
	AltitudeKm   float64 `json:"altitude_km"`
}

func CurrentPosition(tleLine1, tleLine2 string) (Position, error) {
	sat := satellite.TLEToSat(tleLine1, tleLine2, satellite.GravityWGS84)

	now := time.Now().UTC()
	pos, _ := satellite.Propagate(sat, now.Year(), int(now.Month()), now.Day(), now.Hour(), now.Minute(), now.Second())

	gmst := satellite.GSTimeFromDate(now.Year(), int(now.Month()), now.Day(), now.Hour(), now.Minute(), now.Second())
	altitude, _, latLong := satellite.ECIToLLA(pos, gmst)
	deg := satellite.LatLongDeg(latLong)

	return Position{
		LatitudeDeg:  deg.Latitude,
		LongitudeDeg: deg.Longitude,
		AltitudeKm:   altitude,
	}, nil
}
