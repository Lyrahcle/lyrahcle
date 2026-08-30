package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrbitalEvent struct {
	ID          uuid.UUID              `json:"id"`
	SatelliteID uuid.UUID              `json:"satellite_id"`
	EventType   string                 `json:"event_type"`
	EventTime   time.Time              `json:"event_time"`
	Source      string                 `json:"source"`
	Confidence  float64                `json:"confidence"`
	Severity    string                 `json:"severity"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedBy   *uuid.UUID             `json:"created_by,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

type CreateOrbitalEventInput struct {
	SatelliteID uuid.UUID
	EventType   string
	EventTime   time.Time
	Source      string
	Confidence  float64
	Severity    string
	Metadata    map[string]interface{}
	CreatedBy   *uuid.UUID
}

type OrbitalEventStore struct{ pool *pgxpool.Pool }

func NewOrbitalEventStore(pool *pgxpool.Pool) *OrbitalEventStore {
	return &OrbitalEventStore{pool: pool}
}

func (s *OrbitalEventStore) Create(ctx context.Context, in CreateOrbitalEventInput) (OrbitalEvent, error) {
	if in.Metadata == nil {
		in.Metadata = map[string]interface{}{}
	}
	var out OrbitalEvent
	err := s.pool.QueryRow(ctx, `
        INSERT INTO orbital_events (satellite_id, event_type, event_time, source, confidence, severity, metadata, created_by)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
        RETURNING id, satellite_id, event_type, event_time, source, confidence, severity, metadata, created_by, created_at
    `, in.SatelliteID, in.EventType, in.EventTime, in.Source, in.Confidence, in.Severity, in.Metadata, in.CreatedBy).Scan(
		&out.ID, &out.SatelliteID, &out.EventType, &out.EventTime, &out.Source, &out.Confidence,
		&out.Severity, &out.Metadata, &out.CreatedBy, &out.CreatedAt,
	)
	return out, err
}

func (s *OrbitalEventStore) RecentForSatellite(ctx context.Context, satelliteID uuid.UUID, since time.Time, limit int) ([]OrbitalEvent, error) {
	rows, err := s.pool.Query(ctx, `
        SELECT id, satellite_id, event_type, event_time, source, confidence, severity, metadata, created_by, created_at
        FROM orbital_events
        WHERE satellite_id = $1 AND event_time >= $2
        ORDER BY event_time DESC, id DESC
        LIMIT $3
    `, satelliteID, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OrbitalEvent{}
	for rows.Next() {
		var item OrbitalEvent
		if err := rows.Scan(&item.ID, &item.SatelliteID, &item.EventType, &item.EventTime, &item.Source,
			&item.Confidence, &item.Severity, &item.Metadata, &item.CreatedBy, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
