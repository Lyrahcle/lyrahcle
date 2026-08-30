package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrIncidentNotFound          = errors.New("store: incident not found")
	ErrInvalidIncidentTransition = errors.New("store: invalid incident status transition")
)

type Incident struct {
	ID               uuid.UUID  `json:"id"`
	OrganizationID   uuid.UUID  `json:"organization_id"`
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

type IncidentEvidence struct {
	ID           uuid.UUID              `json:"id"`
	IncidentID   uuid.UUID              `json:"incident_id"`
	EvidenceType string                 `json:"evidence_type"`
	EvidenceRef  string                 `json:"evidence_ref"`
	Label        string                 `json:"label"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedBy    *uuid.UUID             `json:"created_by,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

type IncidentDetail struct {
	Incident Incident           `json:"incident"`
	Events   []IncidentEvent    `json:"events"`
	Evidence []IncidentEvidence `json:"evidence"`
}

type IncidentFilter struct {
	Status      string
	Severity    string
	AgentID     string
	AssignedTo  *uuid.UUID
	SatelliteID *uuid.UUID
}

type CreateIncidentInput struct {
	OrganizationID  uuid.UUID
	Title           string
	Description     string
	Severity        string
	SourceType      string
	SatelliteID     *uuid.UUID
	AgentID         string
	SourceAnomalyID *uuid.UUID
	AssignedTo      *uuid.UUID
	CreatedBy       *uuid.UUID
	IdempotencyKey  string
}

type CreateIncidentEvidenceInput struct {
	EvidenceType string
	EvidenceRef  string
	Label        string
	Metadata     map[string]interface{}
	CreatedBy    *uuid.UUID
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
			organization_id, reference, title, description, severity, source_type, satellite_id, agent_id,
			source_anomaly_id, assigned_to, created_by, idempotency_key
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, nullif($8, ''), $9, $10, $11, nullif($12, ''))
		RETURNING id, organization_id, reference, title, description, severity, status, source_type, satellite_id,
		          coalesce(agent_id, ''), source_anomaly_id, assigned_to, created_by, created_at, updated_at, resolved_at
	`, incidentOrganizationID(input.OrganizationID), reference, input.Title, input.Description, input.Severity, input.SourceType, input.SatelliteID,
		input.AgentID, input.SourceAnomalyID, input.AssignedTo, input.CreatedBy, input.IdempotencyKey).Scan(
		&incident.ID, &incident.OrganizationID, &incident.Reference, &incident.Title, &incident.Description, &incident.Severity,
		&incident.Status, &incident.SourceType, &incident.SatelliteID, &incident.AgentID,
		&incident.SourceAnomalyID, &incident.AssignedTo, &incident.CreatedBy, &incident.CreatedAt,
		&incident.UpdatedAt, &incident.ResolvedAt,
	)
	if err != nil {
		_ = tx.Rollback(ctx)
		if existing, ok := s.existingForDuplicate(ctx, input, err); ok {
			return existing, nil
		}
		return Incident{}, err
	}

	eventType := "created"
	if input.SourceType == "rf_anomaly" || input.SourceType == "detection_rule" {
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

	evidenceRows, err := s.pool.Query(ctx, `
		SELECT id, incident_id, evidence_type, evidence_ref, label, metadata, created_by, created_at
		FROM incident_evidence
		WHERE incident_id = $1
		ORDER BY created_at ASC, id ASC
	`, id)
	if err != nil {
		return IncidentDetail{}, err
	}
	defer evidenceRows.Close()
	evidence := []IncidentEvidence{}
	for evidenceRows.Next() {
		var item IncidentEvidence
		if err := evidenceRows.Scan(&item.ID, &item.IncidentID, &item.EvidenceType, &item.EvidenceRef, &item.Label, &item.Metadata, &item.CreatedBy, &item.CreatedAt); err != nil {
			return IncidentDetail{}, err
		}
		evidence = append(evidence, item)
	}
	if err := evidenceRows.Err(); err != nil {
		return IncidentDetail{}, err
	}
	return IncidentDetail{Incident: incident, Events: events, Evidence: evidence}, nil
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
	if input.Status != "" && input.Status != before.Status && !allowedIncidentTransition(before.Status, input.Status) {
		return Incident{}, fmt.Errorf("%w: %s -> %s", ErrInvalidIncidentTransition, before.Status, input.Status)
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

func (s *IncidentStore) AddEvidence(ctx context.Context, incidentID uuid.UUID, input CreateIncidentEvidenceInput) (IncidentEvidence, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return IncidentEvidence{}, err
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM incidents WHERE id = $1)`, incidentID).Scan(&exists); err != nil {
		return IncidentEvidence{}, err
	}
	if !exists {
		return IncidentEvidence{}, ErrIncidentNotFound
	}

	if input.Metadata == nil {
		input.Metadata = map[string]interface{}{}
	}
	var evidence IncidentEvidence
	err = tx.QueryRow(ctx, `
		INSERT INTO incident_evidence (incident_id, evidence_type, evidence_ref, label, metadata, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (incident_id, evidence_type, evidence_ref) DO NOTHING
		RETURNING id, incident_id, evidence_type, evidence_ref, label, metadata, created_by, created_at
	`, incidentID, input.EvidenceType, input.EvidenceRef, input.Label, input.Metadata, input.CreatedBy).Scan(
		&evidence.ID, &evidence.IncidentID, &evidence.EvidenceType, &evidence.EvidenceRef, &evidence.Label, &evidence.Metadata, &evidence.CreatedBy, &evidence.CreatedAt,
	)
	created := err == nil
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			SELECT id, incident_id, evidence_type, evidence_ref, label, metadata, created_by, created_at
			FROM incident_evidence
			WHERE incident_id = $1 AND evidence_type = $2 AND evidence_ref = $3
		`, incidentID, input.EvidenceType, input.EvidenceRef).Scan(
			&evidence.ID, &evidence.IncidentID, &evidence.EvidenceType, &evidence.EvidenceRef, &evidence.Label, &evidence.Metadata, &evidence.CreatedBy, &evidence.CreatedAt,
		)
	}
	if err != nil {
		return IncidentEvidence{}, err
	}
	if created {
		if err := insertIncidentEvent(ctx, tx, incidentID, "evidence_added", input.CreatedBy, "", map[string]interface{}{
			"evidence_id": evidence.ID.String(),
			"type":        evidence.EvidenceType,
		}); err != nil {
			return IncidentEvidence{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return IncidentEvidence{}, err
	}
	return evidence, nil
}

func (s *IncidentStore) existingForDuplicate(ctx context.Context, input CreateIncidentInput, createErr error) (Incident, bool) {
	var pgErr *pgconn.PgError
	if !errors.As(createErr, &pgErr) || pgErr.Code != "23505" {
		return Incident{}, false
	}
	switch pgErr.ConstraintName {
	case "idx_incidents_idempotency_key":
		if input.IdempotencyKey == "" {
			return Incident{}, false
		}
		incident, err := s.getByIdempotencyKey(ctx, input.IdempotencyKey)
		return incident, err == nil
	case "incidents_source_anomaly_id_key":
		if input.SourceAnomalyID == nil {
			return Incident{}, false
		}
		incident, err := s.getBySourceAnomalyID(ctx, *input.SourceAnomalyID)
		return incident, err == nil
	default:
		return Incident{}, false
	}
}

func (s *IncidentStore) getByIdempotencyKey(ctx context.Context, key string) (Incident, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT i.id, i.reference, i.title, i.description, i.severity, i.status, i.source_type,
		       i.satellite_id, coalesce(s.name, ''), coalesce(i.agent_id, ''), i.source_anomaly_id,
		       i.assigned_to, coalesce(u.username, ''), i.created_by, i.created_at, i.updated_at, i.resolved_at
		FROM incidents i
		LEFT JOIN satellites s ON s.id = i.satellite_id
		LEFT JOIN users u ON u.id = i.assigned_to
		WHERE i.idempotency_key = $1
	`, key)
	return scanIncident(row)
}

func (s *IncidentStore) getBySourceAnomalyID(ctx context.Context, anomalyID uuid.UUID) (Incident, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT i.id, i.reference, i.title, i.description, i.severity, i.status, i.source_type,
		       i.satellite_id, coalesce(s.name, ''), coalesce(i.agent_id, ''), i.source_anomaly_id,
		       i.assigned_to, coalesce(u.username, ''), i.created_by, i.created_at, i.updated_at, i.resolved_at
		FROM incidents i
		LEFT JOIN satellites s ON s.id = i.satellite_id
		LEFT JOIN users u ON u.id = i.assigned_to
		WHERE i.source_anomaly_id = $1
	`, anomalyID)
	return scanIncident(row)
}

func allowedIncidentTransition(from, to string) bool {
	transitions := map[string]map[string]bool{
		"new":            {"investigating": true, "false_positive": true},
		"investigating":  {"contained": true, "resolved": true, "false_positive": true},
		"contained":      {"investigating": true, "resolved": true},
		"resolved":       {},
		"false_positive": {},
	}
	return transitions[from][to]
}

type incidentScanner interface {
	Scan(dest ...interface{}) error
}

func incidentOrganizationID(id uuid.UUID) uuid.UUID {
	if id == uuid.Nil {
		return DefaultOrganizationID
	}
	return id
}

func (s *IncidentStore) ExistsInOrganization(ctx context.Context, organizationID, id uuid.UUID) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM incidents WHERE id=$1 AND organization_id=$2)`, id, organizationID).Scan(&ok)
	return ok, err
}

func (s *IncidentStore) ListByOrganization(ctx context.Context, organizationID uuid.UUID, limit, offset int, filter IncidentFilter) (Page[Incident], error) {
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
		SELECT count(*) FROM incidents i
		WHERE i.organization_id=$1
		  AND ($2='' OR i.status=$2)
		  AND ($3='' OR i.severity=$3)
		  AND ($4='' OR i.agent_id=$4)
		  AND ($5::uuid='00000000-0000-0000-0000-000000000000'::uuid OR i.assigned_to=$5)
		  AND ($6::uuid='00000000-0000-0000-0000-000000000000'::uuid OR i.satellite_id=$6)
	`, organizationID, filter.Status, filter.Severity, filter.AgentID, assigned, satellite).Scan(&total); err != nil {
		return Page[Incident]{}, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT i.id, i.reference, i.title, i.description, i.severity, i.status, i.source_type,
		       i.satellite_id, coalesce(s.name,''), coalesce(i.agent_id,''), i.source_anomaly_id,
		       i.assigned_to, coalesce(u.username,''), i.created_by, i.created_at, i.updated_at, i.resolved_at
		FROM incidents i
		LEFT JOIN satellites s ON s.id=i.satellite_id
		LEFT JOIN users u ON u.id=i.assigned_to AND u.organization_id=i.organization_id
		WHERE i.organization_id=$1
		  AND ($2='' OR i.status=$2)
		  AND ($3='' OR i.severity=$3)
		  AND ($4='' OR i.agent_id=$4)
		  AND ($5::uuid='00000000-0000-0000-0000-000000000000'::uuid OR i.assigned_to=$5)
		  AND ($6::uuid='00000000-0000-0000-0000-000000000000'::uuid OR i.satellite_id=$6)
		ORDER BY i.created_at DESC LIMIT $7 OFFSET $8
	`, organizationID, filter.Status, filter.Severity, filter.AgentID, assigned, satellite, limit, offset)
	if err != nil {
		return Page[Incident]{}, err
	}
	defer rows.Close()

	items := []Incident{}
	for rows.Next() {
		item, err := scanIncident(rows)
		if err != nil {
			return Page[Incident]{}, err
		}
		item.OrganizationID = organizationID
		items = append(items, item)
	}
	return Page[Incident]{Items: items, Total: total, Limit: limit, Offset: offset}, rows.Err()
}

func (s *IncidentStore) GetByOrganization(ctx context.Context, organizationID, id uuid.UUID) (IncidentDetail, error) {
	ok, err := s.ExistsInOrganization(ctx, organizationID, id)
	if err != nil {
		return IncidentDetail{}, err
	}
	if !ok {
		return IncidentDetail{}, ErrIncidentNotFound
	}
	out, err := s.Get(ctx, id)
	if err == nil {
		out.Incident.OrganizationID = organizationID
	}
	return out, err
}

func (s *IncidentStore) UpdateByOrganization(ctx context.Context, organizationID, id uuid.UUID, input UpdateIncidentInput) (Incident, error) {
	ok, err := s.ExistsInOrganization(ctx, organizationID, id)
	if err != nil {
		return Incident{}, err
	}
	if !ok {
		return Incident{}, ErrIncidentNotFound
	}
	out, err := s.Update(ctx, id, input)
	if err == nil {
		out.OrganizationID = organizationID
	}
	return out, err
}

func (s *IncidentStore) AddNoteByOrganization(ctx context.Context, organizationID, id, actorID uuid.UUID, message string) error {
	ok, err := s.ExistsInOrganization(ctx, organizationID, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrIncidentNotFound
	}
	return s.AddNote(ctx, id, actorID, message)
}

func (s *IncidentStore) AddEvidenceByOrganization(ctx context.Context, organizationID, incidentID uuid.UUID, input CreateIncidentEvidenceInput) (IncidentEvidence, error) {
	ok, err := s.ExistsInOrganization(ctx, organizationID, incidentID)
	if err != nil {
		return IncidentEvidence{}, err
	}
	if !ok {
		return IncidentEvidence{}, ErrIncidentNotFound
	}
	return s.AddEvidence(ctx, incidentID, input)
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
