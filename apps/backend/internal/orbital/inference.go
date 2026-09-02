package orbital

import (
	"math"
	"time"
)

type StateVector struct {
	ObservedAt          time.Time
	XKm, YKm, ZKm       float64
	VXKmS, VYKmS, VZKmS float64
}

type ManeuverInference struct {
	Candidate          bool    `json:"candidate"`
	DeltaVMS           float64 `json:"delta_v_m_s"`
	PositionResidualKm float64 `json:"position_residual_km"`
	Reason             string  `json:"reason"`
}

func InferManeuverCandidate(previous, observed StateVector) ManeuverInference {
	dt := observed.ObservedAt.Sub(previous.ObservedAt).Seconds()
	if dt <= 0 {
		return ManeuverInference{Reason: "non-forward observation interval"}
	}
	px := previous.XKm + previous.VXKmS*dt
	py := previous.YKm + previous.VYKmS*dt
	pz := previous.ZKm + previous.VZKmS*dt
	residual := math.Sqrt(sq(observed.XKm-px) + sq(observed.YKm-py) + sq(observed.ZKm-pz))
	dv := math.Sqrt(sq(observed.VXKmS-previous.VXKmS)+sq(observed.VYKmS-previous.VYKmS)+sq(observed.VZKmS-previous.VZKmS)) * 1000
	candidate := dv >= 2.0 && residual >= 1.0
	reason := "within maneuver-candidate thresholds"
	if candidate {
		reason = "velocity and propagated-position residuals exceed maneuver-candidate thresholds"
	}
	return ManeuverInference{Candidate: candidate, DeltaVMS: dv, PositionResidualKm: residual, Reason: reason}
}

func sq(v float64) float64 { return v * v }

type ConjunctionAssessment struct {
	Candidate         bool    `json:"candidate"`
	ClosestApproachKm float64 `json:"closest_approach_km"`
	RelativeSpeedKmS  float64 `json:"relative_speed_km_s"`
	Reason            string  `json:"reason"`
}

func AssessConjunctionCandidate(a, b StateVector, thresholdKm float64) ConjunctionAssessment {
	if thresholdKm <= 0 {
		thresholdKm = 10
	}
	d := math.Sqrt(sq(a.XKm-b.XKm) + sq(a.YKm-b.YKm) + sq(a.ZKm-b.ZKm))
	rv := math.Sqrt(sq(a.VXKmS-b.VXKmS) + sq(a.VYKmS-b.VYKmS) + sq(a.VZKmS-b.VZKmS))
	candidate := d <= thresholdKm
	reason := "separation above screening threshold"
	if candidate {
		reason = "separation within conjunction screening threshold"
	}
	return ConjunctionAssessment{Candidate: candidate, ClosestApproachKm: d, RelativeSpeedKmS: rv, Reason: reason}
}
