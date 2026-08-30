package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mlko01/lyrahcle/apps/backend/internal/orbitalcontext"
	"github.com/mlko01/lyrahcle/apps/backend/internal/passprediction"
	"github.com/mlko01/lyrahcle/apps/backend/internal/store"
	"github.com/mlko01/lyrahcle/apps/backend/internal/tlequality"
)

const (
	defaultPassHorizon = 24 * time.Hour
	maxPassHorizon     = 72 * time.Hour
	defaultPassLimit   = 10
	maxPassLimit       = 50
)

type satellitePassPredictionResponse struct {
	SatelliteID       uuid.UUID              `json:"satellite_id"`
	NoradID           int                    `json:"norad_id"`
	SatelliteName     string                 `json:"satellite_name"`
	TLEStatus         string                 `json:"tle_status"`
	TLEAgeSeconds     int64                  `json:"tle_age_seconds"`
	OrbitalConfidence string                 `json:"orbital_confidence"`
	Station           orbitalcontext.Station `json:"station"`
	passprediction.Result
}

func handleSatellitePassPrediction(satellites *store.SatelliteStore, policy tlequality.Policy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		satelliteID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "satelliteID")))
		if err != nil {
			http.Error(w, "invalid satellite ID", http.StatusBadRequest)
			return
		}
		sat, err := satellites.Get(r.Context(), satelliteID)
		if err != nil {
			if err == pgx.ErrNoRows {
				http.Error(w, "satellite not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		station, err := parsePassStation(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		from, horizon, minElevation, limit, err := parsePassOptions(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		assessment := tlequality.Assess(sat.TLEEpoch, from, policy)
		recordTLEMetrics(sat.NoradID, assessment)
		if assessment.Status == tlequality.StatusStale || assessment.Status == tlequality.StatusUnusable {
			http.Error(w, "satellite TLE is too stale for pass prediction", http.StatusUnprocessableEntity)
			return
		}

		prediction, err := passprediction.Predict(sat.TLELine1, sat.TLELine2, passprediction.Request{
			Station:         station,
			From:            from,
			Horizon:         horizon,
			MinElevationDeg: minElevation,
			Limit:           limit,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}

		response := satellitePassPredictionResponse{
			SatelliteID:       sat.ID,
			NoradID:           sat.NoradID,
			SatelliteName:     sat.Name,
			TLEStatus:         string(assessment.Status),
			TLEAgeSeconds:     int64(assessment.Age.Seconds()),
			OrbitalConfidence: assessment.Confidence,
			Station:           station,
			Result:            prediction,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func parsePassStation(r *http.Request) (orbitalcontext.Station, error) {
	latitude, err := requiredFloatQuery(r, "latitude_deg")
	if err != nil {
		return orbitalcontext.Station{}, err
	}
	longitude, err := requiredFloatQuery(r, "longitude_deg")
	if err != nil {
		return orbitalcontext.Station{}, err
	}
	altitude := 0.0
	if value := strings.TrimSpace(r.URL.Query().Get("altitude_m")); value != "" {
		altitude, err = strconv.ParseFloat(value, 64)
		if err != nil {
			return orbitalcontext.Station{}, queryError("altitude_m", "must be a number")
		}
	}
	if latitude < -90 || latitude > 90 {
		return orbitalcontext.Station{}, queryError("latitude_deg", "must be between -90 and 90")
	}
	if longitude < -180 || longitude > 180 {
		return orbitalcontext.Station{}, queryError("longitude_deg", "must be between -180 and 180")
	}
	if altitude < -500 || altitude > 10000 {
		return orbitalcontext.Station{}, queryError("altitude_m", "must be between -500 and 10000")
	}
	return orbitalcontext.Station{LatitudeDeg: latitude, LongitudeDeg: longitude, AltitudeM: altitude}, nil
}

func parsePassOptions(r *http.Request) (time.Time, time.Duration, float64, int, error) {
	from := time.Now().UTC()
	if value := strings.TrimSpace(r.URL.Query().Get("from")); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return time.Time{}, 0, 0, 0, queryError("from", "must be RFC3339")
		}
		from = parsed.UTC()
	}

	horizon := defaultPassHorizon
	if value := strings.TrimSpace(r.URL.Query().Get("hours")); value != "" {
		hours, err := strconv.ParseFloat(value, 64)
		if err != nil || hours <= 0 || hours > maxPassHorizon.Hours() {
			return time.Time{}, 0, 0, 0, queryError("hours", "must be greater than 0 and at most 72")
		}
		horizon = time.Duration(hours * float64(time.Hour))
	}

	minElevation := 0.0
	if value := strings.TrimSpace(r.URL.Query().Get("min_elevation_deg")); value != "" {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil || parsed < 0 || parsed > 90 {
			return time.Time{}, 0, 0, 0, queryError("min_elevation_deg", "must be between 0 and 90")
		}
		minElevation = parsed
	}

	limit := defaultPassLimit
	if value := strings.TrimSpace(r.URL.Query().Get("limit")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > maxPassLimit {
			return time.Time{}, 0, 0, 0, queryError("limit", "must be between 1 and 50")
		}
		limit = parsed
	}
	return from, horizon, minElevation, limit, nil
}

func requiredFloatQuery(r *http.Request, name string) (float64, error) {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if value == "" {
		return 0, queryError(name, "is required")
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, queryError(name, "must be a number")
	}
	return parsed, nil
}

type queryParameterError struct {
	parameter string
	message   string
}

func (e queryParameterError) Error() string { return e.parameter + ": " + e.message }
func queryError(parameter, message string) error {
	return queryParameterError{parameter: parameter, message: message}
}
