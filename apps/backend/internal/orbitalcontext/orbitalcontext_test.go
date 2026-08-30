package orbitalcontext

import (
	"math"
	"testing"
	"time"
)

func TestGeodeticToECEFAtEquator(t *testing.T) {
	v := geodeticToECEF(0, 0, 0)
	if math.Abs(v.x-earthRadiusM) > 0.001 || math.Abs(v.y) > 0.001 || math.Abs(v.z) > 0.001 {
		t.Fatalf("unexpected ECEF: %+v", v)
	}
}

func TestEvaluateRejectsNoErrorForValidTLEShape(t *testing.T) {
	line1 := "1 25544U 98067A   24183.55387731  .00016717  00000+0  30288-3 0  9995"
	line2 := "2 25544  51.6405 213.1661 0004575  78.9109  45.4666 15.49758034460586"
	result, err := Evaluate(Station{LatitudeDeg: 40.4168, LongitudeDeg: -3.7038, AltitudeM: 667}, line1, line2, time.Date(2024, 7, 1, 12, 0, 0, 0, time.UTC), 437500000)
	if err != nil {
		t.Fatal(err)
	}
	if math.IsNaN(result.ElevationDeg) || math.IsNaN(result.RangeKm) || result.RangeKm <= 0 {
		t.Fatalf("invalid result: %+v", result)
	}
}
