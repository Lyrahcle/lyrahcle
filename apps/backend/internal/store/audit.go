package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditEntry struct {
	ID             uuid.UUID              `json:"id"`
	OrganizationID uuid.UUID              `json:"organization_id"`
	OccurredAt     time.Time              `json:"occurred_at"`
	ActorType      string                 `json:"actor_type"`
	ActorID        string                 `json:"actor_id"`
	Action         string                 `json:"action"`
	IP             string                 `json:"ip"`
	Details        map[string]interface{} `json:"details"`
}

type AuditStore struct {
	pool *pgxpool.Pool
}

func NewAuditStore(pool *pgxpool.Pool) *AuditStore {
	return &AuditStore{pool: pool}
}

func (s *AuditStore) Insert(ctx context.Context, actorType, actorID, action, ip string, details map[string]interface{}) error {
	return s.InsertInOrganization(ctx, DefaultOrganizationID, actorType, actorID, action, ip, details)
}

func (s *AuditStore) List(ctx context.Context, limit, offset int) (Page[AuditEntry], error) {
	limit, offset = ClampPageParams(limit, offset)

	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM audit_log`).Scan(&total); err != nil {
		return Page[AuditEntry]{}, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, occurred_at, actor_type, actor_id, action, ip, details
		FROM audit_log
		ORDER BY occurred_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return Page[AuditEntry]{}, err
	}
	defer rows.Close()

	items := []AuditEntry{}
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.OccurredAt, &e.ActorType, &e.ActorID, &e.Action, &e.IP, &e.Details); err != nil {
			return Page[AuditEntry]{}, err
		}
		items = append(items, e)
	}
	if err := rows.Err(); err != nil {
		return Page[AuditEntry]{}, err
	}

	return Page[AuditEntry]{Items: items, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *AuditStore) RecentByActor(ctx context.Context, actorType, actorID string, since time.Time, limit int) ([]AuditEntry, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, occurred_at, actor_type, actor_id, action, ip, details
		FROM audit_log
		WHERE actor_type = $1 AND actor_id = $2 AND occurred_at >= $3
		ORDER BY occurred_at DESC
		LIMIT $4
	`, actorType, actorID, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []AuditEntry{}
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.OccurredAt, &e.ActorType, &e.ActorID, &e.Action, &e.IP, &e.Details); err != nil {
			return nil, err
		}
		items = append(items, e)
	}
	return items, rows.Err()
}

func (s *AuditStore) InsertInOrganization(ctx context.Context, organizationID uuid.UUID, actorType, actorID, action, ip string, details map[string]interface{}) error {
	if details == nil {
		details = map[string]interface{}{}
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO audit_log (organization_id, actor_type, actor_id, action, ip, details)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, organizationID, actorType, actorID, action, ip, details)
	return err
}

// InsertIdempotentInOrganization records a system side effect exactly once per
// tenant/dedupe key. It is intended for at-least-once background processing,
// where a retry must be able to repair a missing audit entry without creating
// duplicates if the previous attempt already committed it.
func (s *AuditStore) InsertIdempotentInOrganization(ctx context.Context, organizationID uuid.UUID, dedupeKey, actorType, actorID, action, ip string, details map[string]interface{}) (bool, error) {
	if details == nil {
		details = map[string]interface{}{}
	}
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO audit_log (organization_id, dedupe_key, actor_type, actor_id, action, ip, details)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (organization_id, dedupe_key) WHERE dedupe_key IS NOT NULL DO NOTHING
	`, organizationID, dedupeKey, actorType, actorID, action, ip, details)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *AuditStore) ListByOrganization(ctx context.Context, organizationID uuid.UUID, limit, offset int) (Page[AuditEntry], error) {
	limit, offset = ClampPageParams(limit, offset)
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE organization_id=$1`, organizationID).Scan(&total); err != nil {
		return Page[AuditEntry]{}, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, organization_id, occurred_at, actor_type, actor_id, action, ip, details
		FROM audit_log
		WHERE organization_id=$1
		ORDER BY occurred_at DESC
		LIMIT $2 OFFSET $3
	`, organizationID, limit, offset)
	if err != nil {
		return Page[AuditEntry]{}, err
	}
	defer rows.Close()
	items := []AuditEntry{}
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.OrganizationID, &e.OccurredAt, &e.ActorType, &e.ActorID, &e.Action, &e.IP, &e.Details); err != nil {
			return Page[AuditEntry]{}, err
		}
		items = append(items, e)
	}
	return Page[AuditEntry]{Items: items, Total: total, Limit: limit, Offset: offset}, rows.Err()
}
