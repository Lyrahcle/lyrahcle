package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrIncidentNotFound = errors.New("store: incident not found")

type Incident struct {
	ID               uuid.UUID  `json:"id"`
	Reference        string     `json:"reference"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	Severity         string     `json:"severity"`
	Status           string     `json:"status"`
	SourceType       string     `json:"source_type"`
	SatelliteID      *uuid.UUID `json:"satellite_id,omitempty"`
	SatelliteName    string     `json:"satellite_name,omitempty"`
	AgentID          string     `json:"agent_id,omitempty"`
	SourceAnomalyID  *uuid.UUID `json:"source_anomaly_id,omitempty"`
	AssignedTo       *uuid.UUID `json:"assigned_to,omitempty"`
	AssignedUsername string     `json:"assigned_username,omitempty"`
	CreatedBy        *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	ResolvedAt       *time.Time `json:"resolved_at,omitempty"`
}

type IncidentEvent struct {
	ID            uuid.UUID              `json:"id"`
	IncidentID    uuid.UUID              `json:"incident_id"`
	EventType     string                 `json:"event_type"`
	ActorID       *uuid.UUID             `json:"actor_id,omitempty"`
	ActorUsername string                 `json:"actor_username,omitempty"`
	Message       string                 `json:"message"`
	Details       map[string]interface{} `json:"details"`
	CreatedAt     time.Time              `json:"created_at"`
}

type IncidentDetail struct {
	Incident Incident        `json:"incident"`
	Events   []IncidentEvent `json:"events"`
}

type IncidentFilter struct {
	Status      string
	Severity    string
	AgentID     string
	AssignedTo  *uuid.UUID
	SatelliteID *uuid.UUID
}

type CreateIncidentInput struct {
	Title           string
	Description     string
	Severity        string
	SourceType      string
	SatelliteID     *uuid.UUID
	AgentID         string
	SourceAnomalyID *uuid.UUID
	AssignedTo      *uuid.UUID
	CreatedBy       *uuid.UUID
}

type UpdateIncidentInput struct {
	Status           string
	Severity         string
	AssignedTo       *uuid.UUID
	UpdateAssignment bool
	ActorID          uuid.UUID
}

type IncidentStore struct {
	pool *pgxpool.Pool
}

func NewIncidentStore(pool *pgxpool.Pool) *IncidentStore {
	return &IncidentStore{pool: pool}
}

func (s *IncidentStore) List(ctx context.Context, limit, offset int, filter IncidentFilter) (Page[Incident], error) {
	limit, offset = ClampPageParams(limit, offset)
	assigned := uuid.Nil
	if filter.AssignedTo != nil {
		assigned = *filter.AssignedTo
	}
	satellite := uuid.Nil
	if filter.SatelliteID != nil {
		satellite = *filter.SatelliteID
	}

	var total int
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM incidents i
		WHERE ($1 = '' OR i.status = $1)
		  AND ($2 = '' OR i.severity = $2)
		  AND ($3 = '' OR i.agent_id = $3)
		  AND ($4::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR i.assigned_to = $4)
		  AND ($5::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR i.satellite_id = $5)
	`, filter.Status, filter.Severity, filter.AgentID, assigned, satellite).Scan(&total); err != nil {
		return Page[Incident]{}, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT i.id, i.reference, i.title, i.description, i.severity, i.status, i.source_type,
		       i.satellite_id, coalesce(s.name, ''), coalesce(i.agent_id, ''), i.source_anomaly_id,
		       i.assigned_to, coalesce(u.username, ''), i.created_by, i.created_at, i.updated_at, i.resolved_at
		FROM incidents i
		LEFT JOIN satellites s ON s.id = i.satellite_id
		LEFT JOIN users u ON u.id = i.assigned_to
		WHERE ($1 = '' OR i.status = $1)
		  AND ($2 = '' OR i.severity = $2)
		  AND ($3 = '' OR i.agent_id = $3)
		  AND ($4::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR i.assigned_to = $4)
		  AND ($5::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR i.satellite_id = $5)
		ORDER BY i.created_at DESC
		LIMIT $6 OFFSET $7
	`, filter.Status, filter.Severity, filter.AgentID, assigned, satellite, limit, offset)
	if err != nil {
		return Page[Incident]{}, err
	}
	defer rows.Close()

	items := []Incident{}
	for rows.Next() {
		incident, err := scanIncident(rows)
		if err != nil {
			return Page[Incident]{}, err
		}
		items = append(items, incident)
	}
	if err := rows.Err(); err != nil {
		return Page[Incident]{}, err
	}

	return Page[Incident]{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *IncidentStore) Create(ctx context.Context, input CreateIncidentInput) (Incident, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Incident{}, err
	}
	defer tx.Rollback(ctx)

	reference := fmt.Sprintf("INC-%d-%s", time.Now().UTC().Year(), uuid.NewString()[:8])
	var incident Incident
	err = tx.QueryRow(ctx, `
		INSERT INTO incidents (
			reference, title, description, severity, source_type, satellite_id, agent_id,
			source_anomaly_id, assigned_to, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, nullif($7, ''), $8, $9, $10)
		RETURNING id, reference, title, description, severity, status, source_type, satellite_id,
		          coalesce(agent_id, ''), source_anomaly_id, assigned_to, created_by, created_at, updated_at, resolved_at
	`, reference, input.Title, input.Description, input.Severity, input.SourceType, input.SatelliteID,
		input.AgentID, input.SourceAnomalyID, input.AssignedTo, input.CreatedBy).Scan(
		&incident.ID, &incident.Reference, &incident.Title, &incident.Description, &incident.Severity,
		&incident.Status, &incident.SourceType, &incident.SatelliteID, &incident.AgentID,
		&incident.SourceAnomalyID, &incident.AssignedTo, &incident.CreatedBy, &incident.CreatedAt,
		&incident.UpdatedAt, &incident.ResolvedAt,
	)
	if err != nil {
		return Incident{}, err
	}

	eventType := "created"
	if input.SourceType == "rf_anomaly" {
		eventType = "auto_detected"
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO incident_events (incident_id, event_type, actor_id, details)
		VALUES ($1, $2, $3, $4)
	`, incident.ID, eventType, input.CreatedBy, map[string]interface{}{
		"severity":    incident.Severity,
		"source_type": incident.SourceType,
	}); err != nil {
		return Incident{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Incident{}, err
	}
	return incident, nil
}

func (s *IncidentStore) Get(ctx context.Context, id uuid.UUID) (IncidentDetail, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT i.id, i.reference, i.title, i.description, i.severity, i.status, i.source_type,
		       i.satellite_id, coalesce(s.name, ''), coalesce(i.agent_id, ''), i.source_anomaly_id,
		       i.assigned_to, coalesce(u.username, ''), i.created_by, i.created_at, i.updated_at, i.resolved_at
		FROM incidents i
		LEFT JOIN satellites s ON s.id = i.satellite_id
		LEFT JOIN users u ON u.id = i.assigned_to
		WHERE i.id = $1
	`, id)
	incident, err := scanIncident(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return IncidentDetail{}, ErrIncidentNotFound
		}
		return IncidentDetail{}, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT e.id, e.incident_id, e.event_type, e.actor_id, coalesce(u.username, ''), e.message, e.details, e.created_at
		FROM incident_events e
		LEFT JOIN users u ON u.id = e.actor_id
		WHERE e.incident_id = $1
		ORDER BY e.created_at ASC, e.id ASC
	`, id)
	if err != nil {
		return IncidentDetail{}, err
	}
	defer rows.Close()

	events := []IncidentEvent{}
	for rows.Next() {
		var event IncidentEvent
		if err := rows.Scan(&event.ID, &event.IncidentID, &event.EventType, &event.ActorID, &event.ActorUsername, &event.Message, &event.Details, &event.CreatedAt); err != nil {
			return IncidentDetail{}, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return IncidentDetail{}, err
	}
	return IncidentDetail{Incident: incident, Events: events}, nil
}

func (s *IncidentStore) Update(ctx context.Context, id uuid.UUID, input UpdateIncidentInput) (Incident, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Incident{}, err
	}
	defer tx.Rollback(ctx)

	var before Incident
	err = tx.QueryRow(ctx, `
		SELECT id, reference, title, description, severity, status, source_type, satellite_id,
		       coalesce(agent_id, ''), source_anomaly_id, assigned_to, created_by, created_at, updated_at, resolved_at
		FROM incidents WHERE id = $1 FOR UPDATE
	`, id).Scan(
		&before.ID, &before.Reference, &before.Title, &before.Description, &before.Severity, &before.Status,
		&before.SourceType, &before.SatelliteID, &before.AgentID, &before.SourceAnomalyID, &before.AssignedTo,
		&before.CreatedBy, &before.CreatedAt, &before.UpdatedAt, &before.ResolvedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Incident{}, ErrIncidentNotFound
		}
		return Incident{}, err
	}

	var updated Incident
	err = tx.QueryRow(ctx, `
		UPDATE incidents
		SET status = CASE WHEN $2 = '' THEN status ELSE $2 END,
		    severity = CASE WHEN $3 = '' THEN severity ELSE $3 END,
		    assigned_to = CASE WHEN $5 THEN $4 ELSE assigned_to END,
		    resolved_at = CASE
		        WHEN $2 IN ('resolved', 'false_positive') AND status NOT IN ('resolved', 'false_positive') THEN now()
		        WHEN $2 <> '' AND $2 NOT IN ('resolved', 'false_positive') THEN NULL
		        ELSE resolved_at
		    END,
		    updated_at = now()
		WHERE id = $1
		RETURNING id, reference, title, description, severity, status, source_type, satellite_id,
		          coalesce(agent_id, ''), source_anomaly_id, assigned_to, created_by, created_at, updated_at, resolved_at
	`, id, input.Status, input.Severity, input.AssignedTo, input.UpdateAssignment).Scan(
		&updated.ID, &updated.Reference, &updated.Title, &updated.Description, &updated.Severity, &updated.Status,
		&updated.SourceType, &updated.SatelliteID, &updated.AgentID, &updated.SourceAnomalyID, &updated.AssignedTo,
		&updated.CreatedBy, &updated.CreatedAt, &updated.UpdatedAt, &updated.ResolvedAt,
	)
	if err != nil {
		return Incident{}, err
	}

	if before.Status != updated.Status {
		if err := insertIncidentEvent(ctx, tx, id, "status_changed", &input.ActorID, "", map[string]interface{}{"from": before.Status, "to": updated.Status}); err != nil {
			return Incident{}, err
		}
	}
	if before.Severity != updated.Severity {
		if err := insertIncidentEvent(ctx, tx, id, "severity_changed", &input.ActorID, "", map[string]interface{}{"from": before.Severity, "to": updated.Severity}); err != nil {
			return Incident{}, err
		}
	}
	if input.UpdateAssignment && !uuidPtrEqual(before.AssignedTo, updated.AssignedTo) {
		if err := insertIncidentEvent(ctx, tx, id, "assignment_changed", &input.ActorID, "", map[string]interface{}{"from": uuidString(before.AssignedTo), "to": uuidString(updated.AssignedTo)}); err != nil {
			return Incident{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Incident{}, err
	}
	return updated, nil
}

func (s *IncidentStore) AddNote(ctx context.Context, id, actorID uuid.UUID, message string) error {
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM incidents WHERE id = $1)`, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrIncidentNotFound
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO incident_events (incident_id, event_type, actor_id, message)
		VALUES ($1, 'note', $2, $3)
	`, id, actorID, message)
	return err
}

type incidentScanner interface {
	Scan(dest ...interface{}) error
}

func scanIncident(row incidentScanner) (Incident, error) {
	var incident Incident
	err := row.Scan(
		&incident.ID, &incident.Reference, &incident.Title, &incident.Description, &incident.Severity,
		&incident.Status, &incident.SourceType, &incident.SatelliteID, &incident.SatelliteName, &incident.AgentID,
		&incident.SourceAnomalyID, &incident.AssignedTo, &incident.AssignedUsername, &incident.CreatedBy,
		&incident.CreatedAt, &incident.UpdatedAt, &incident.ResolvedAt,
	)
	return incident, err
}

func insertIncidentEvent(ctx context.Context, tx pgx.Tx, incidentID uuid.UUID, eventType string, actorID *uuid.UUID, message string, details map[string]interface{}) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO incident_events (incident_id, event_type, actor_id, message, details)
		VALUES ($1, $2, $3, $4, $5)
	`, incidentID, eventType, actorID, message, details)
	return err
}

func uuidPtrEqual(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func uuidString(value *uuid.UUID) string {
	if value == nil {
		return ""
	}
	return value.String()
}
