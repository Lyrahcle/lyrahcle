package store

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Program struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type ProgramStore struct {
	pool *pgxpool.Pool
}

func NewProgramStore(pool *pgxpool.Pool) *ProgramStore {
	return &ProgramStore{pool: pool}
}

func (s *ProgramStore) List(ctx context.Context) ([]Program, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name FROM programs ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Program
	for rows.Next() {
		var p Program
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *ProgramStore) Create(ctx context.Context, name string) (Program, error) {
	var p Program
	p.Name = name
	err := s.pool.QueryRow(ctx, `INSERT INTO programs (name) VALUES ($1) RETURNING id`, name).Scan(&p.ID)
	return p, err
}
