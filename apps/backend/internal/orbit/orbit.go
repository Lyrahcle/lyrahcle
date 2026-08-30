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
	return PositionAt(tleLine1, tleLine2, time.Now().UTC())
}

func PositionAt(tleLine1, tleLine2 string, at time.Time) (Position, error) {
	sat := satellite.TLEToSat(tleLine1, tleLine2, satellite.GravityWGS84)
	at = at.UTC()
	pos, _ := satellite.Propagate(sat, at.Year(), int(at.Month()), at.Day(), at.Hour(), at.Minute(), at.Second())
	gmst := satellite.GSTimeFromDate(at.Year(), int(at.Month()), at.Day(), at.Hour(), at.Minute(), at.Second())
	altitude, _, latLong := satellite.ECIToLLA(pos, gmst)
	deg := satellite.LatLongDeg(latLong)
	return Position{LatitudeDeg: deg.Latitude, LongitudeDeg: deg.Longitude, AltitudeKm: altitude}, nil
}
