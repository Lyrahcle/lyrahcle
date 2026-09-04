package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const MaxExternalObservationsPerImport = 200

type ExternalObservation struct {
	ID                 uuid.UUID `json:"id"`
	OrganizationID     uuid.UUID `json:"organization_id"`
	SatelliteID        uuid.UUID `json:"satellite_id"`
	Source             string    `json:"source"`
	ExternalID         int64     `json:"external_id"`
	StationID          int64     `json:"station_id"`
	StationName        string    `json:"station_name"`
	Observer           string    `json:"observer"`
	TransmitterUUID    string    `json:"transmitter_uuid"`
	Mode               string    `json:"mode"`
	DownlinkHz         *int64    `json:"downlink_hz,omitempty"`
	StationLatitude    *float64  `json:"station_latitude,omitempty"`
	StationLongitude   *float64  `json:"station_longitude,omitempty"`
	StartedAt          time.Time `json:"started_at"`
	EndedAt            time.Time `json:"ended_at"`
	Status             string    `json:"status"`
	VettedStatus       string    `json:"vetted_status"`
	ReceptionConfirmed bool      `json:"reception_confirmed"`
	EvidenceKind       string    `json:"evidence_kind"`
	Applicability      string    `json:"applicability"`
	Confidence         int       `json:"confidence"`
	WaterfallURL       string    `json:"waterfall_url,omitempty"`
	ImportedAt         time.Time `json:"imported_at"`
}

type ExternalObservationInput struct {
	Source             string
	ExternalID         int64
	StationID          int64
	StationName        string
	Observer           string
	TransmitterUUID    string
	Mode               string
	DownlinkHz         *int64
	StationLatitude    *float64
	StationLongitude   *float64
	StartedAt          time.Time
	EndedAt            time.Time
	Status             string
	VettedStatus       string
	ReceptionConfirmed bool
	EvidenceKind       string
	Applicability      string
	Confidence         int
	WaterfallURL       string
}

type ExternalObservationStore struct {
	pool *pgxpool.Pool
}

func NewExternalObservationStore(pool *pgxpool.Pool) *ExternalObservationStore {
	return &ExternalObservationStore{pool: pool}
}

func (s *ExternalObservationStore) UpsertBatch(ctx context.Context, organizationID, satelliteID uuid.UUID, inputs []ExternalObservationInput) (int, error) {
	if len(inputs) == 0 {
		return 0, nil
	}
	if len(inputs) > MaxExternalObservationsPerImport {
		inputs = inputs[:MaxExternalObservationsPerImport]
	}
	stored := 0
	for _, input := range inputs {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO external_rf_observations (
				organization_id, satellite_id, source, external_id, station_id, station_name, observer,
				transmitter_uuid, mode, downlink_hz, station_latitude, station_longitude,
				started_at, ended_at, status, vetted_status, reception_confirmed,
				evidence_kind, applicability, confidence, waterfall_url
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
			ON CONFLICT (organization_id, source, external_id) DO UPDATE SET
				satellite_id = EXCLUDED.satellite_id,
				station_id = EXCLUDED.station_id,
				station_name = EXCLUDED.station_name,
				observer = EXCLUDED.observer,
				transmitter_uuid = EXCLUDED.transmitter_uuid,
				mode = EXCLUDED.mode,
				downlink_hz = EXCLUDED.downlink_hz,
				station_latitude = EXCLUDED.station_latitude,
				station_longitude = EXCLUDED.station_longitude,
				started_at = EXCLUDED.started_at,
				ended_at = EXCLUDED.ended_at,
				status = EXCLUDED.status,
				vetted_status = EXCLUDED.vetted_status,
				reception_confirmed = EXCLUDED.reception_confirmed,
				confidence = EXCLUDED.confidence,
				waterfall_url = EXCLUDED.waterfall_url,
				updated_at = now()
		`,
			organizationID, satelliteID, input.Source, input.ExternalID, input.StationID, input.StationName, input.Observer,
			input.TransmitterUUID, input.Mode, input.DownlinkHz, input.StationLatitude, input.StationLongitude,
			input.StartedAt, input.EndedAt, input.Status, input.VettedStatus, input.ReceptionConfirmed,
			input.EvidenceKind, input.Applicability, input.Confidence, input.WaterfallURL,
		)
		if err != nil {
			return stored, err
		}
		stored++
	}
	return stored, nil
}

func (s *ExternalObservationStore) ListBySatellite(ctx context.Context, organizationID, satelliteID uuid.UUID, limit int) ([]ExternalObservation, error) {
	if limit <= 0 || limit > MaxPageLimit {
		limit = MaxPageLimit
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, satellite_id, source, external_id, station_id, station_name, observer,
		       transmitter_uuid, mode, downlink_hz, station_latitude, station_longitude,
		       started_at, ended_at, status, vetted_status, reception_confirmed,
		       evidence_kind, applicability, confidence, waterfall_url, imported_at
		FROM external_rf_observations
		WHERE organization_id = $1 AND satellite_id = $2
		ORDER BY ended_at DESC
		LIMIT $3
	`, organizationID, satelliteID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ExternalObservation, 0, limit)
	for rows.Next() {
		var observation ExternalObservation
		if err := rows.Scan(
			&observation.ID, &observation.OrganizationID, &observation.SatelliteID, &observation.Source, &observation.ExternalID,
			&observation.StationID, &observation.StationName, &observation.Observer, &observation.TransmitterUUID, &observation.Mode,
			&observation.DownlinkHz, &observation.StationLatitude, &observation.StationLongitude,
			&observation.StartedAt, &observation.EndedAt, &observation.Status, &observation.VettedStatus, &observation.ReceptionConfirmed,
			&observation.EvidenceKind, &observation.Applicability, &observation.Confidence, &observation.WaterfallURL, &observation.ImportedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, observation)
	}
	return out, rows.Err()
}

func (s *ExternalObservationStore) RecentByOrganization(ctx context.Context, organizationID uuid.UUID, limit int) ([]ExternalObservation, error) {
	if limit <= 0 || limit > MaxPageLimit {
		limit = MaxPageLimit
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, satellite_id, source, external_id, station_id, station_name, observer,
		       transmitter_uuid, mode, downlink_hz, station_latitude, station_longitude,
		       started_at, ended_at, status, vetted_status, reception_confirmed,
		       evidence_kind, applicability, confidence, waterfall_url, imported_at
		FROM external_rf_observations
		WHERE organization_id = $1
		ORDER BY ended_at DESC
		LIMIT $2
	`, organizationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ExternalObservation, 0, limit)
	for rows.Next() {
		var observation ExternalObservation
		if err := rows.Scan(
			&observation.ID, &observation.OrganizationID, &observation.SatelliteID, &observation.Source, &observation.ExternalID,
			&observation.StationID, &observation.StationName, &observation.Observer, &observation.TransmitterUUID, &observation.Mode,
			&observation.DownlinkHz, &observation.StationLatitude, &observation.StationLongitude,
			&observation.StartedAt, &observation.EndedAt, &observation.Status, &observation.VettedStatus, &observation.ReceptionConfirmed,
			&observation.EvidenceKind, &observation.Applicability, &observation.Confidence, &observation.WaterfallURL, &observation.ImportedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, observation)
	}
	return out, rows.Err()
}
