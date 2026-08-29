package webauthnflow

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Ceremony struct {
	UserID  uuid.UUID            `json:"user_id"`
	Purpose string               `json:"purpose"`
	Label   string               `json:"label,omitempty"`
	Session webauthn.SessionData `json:"session"`
}

type Store struct {
	client *redis.Client
	ttl    time.Duration
}

func New(client *redis.Client) *Store {
	return &Store{client: client, ttl: 5 * time.Minute}
}

func (s *Store) Create(ctx context.Context, ceremony Ceremony) (string, error) {
	if s.client == nil {
		return "", errors.New("webauthn ceremony storage unavailable")
	}
	id := uuid.NewString()
	raw, err := json.Marshal(ceremony)
	if err != nil {
		return "", err
	}
	if err := s.client.Set(ctx, "lyrahcle:webauthn:"+id, raw, s.ttl).Err(); err != nil {
		return "", err
	}
	return id, nil
}

func (s *Store) Consume(ctx context.Context, id, purpose string) (Ceremony, error) {
	if s.client == nil {
		return Ceremony{}, errors.New("webauthn ceremony storage unavailable")
	}
	if _, err := uuid.Parse(id); err != nil {
		return Ceremony{}, errors.New("invalid ceremony id")
	}
	raw, err := s.client.GetDel(ctx, "lyrahcle:webauthn:"+id).Bytes()
	if err != nil {
		return Ceremony{}, errors.New("webauthn ceremony expired or invalid")
	}
	var ceremony Ceremony
	if err := json.Unmarshal(raw, &ceremony); err != nil {
		return Ceremony{}, err
	}
	if ceremony.Purpose != purpose || ceremony.UserID == uuid.Nil {
		return Ceremony{}, errors.New("webauthn ceremony mismatch")
	}
	return ceremony, nil
}
