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

var consumeCeremonyScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`)

func New(client *redis.Client) *Store {
	return &Store{client: client, ttl: 5 * time.Minute}
}

func ceremonyKey(id string) string {
	return "lyrahcle:webauthn:" + id
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
	if err := s.client.Set(ctx, ceremonyKey(id), raw, s.ttl).Err(); err != nil {
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
	key := ceremonyKey(id)
	raw, err := s.client.Get(ctx, key).Bytes()
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
	deleted, err := consumeCeremonyScript.Run(ctx, s.client, []string{key}, raw).Int64()
	if err != nil || deleted != 1 {
		return Ceremony{}, errors.New("webauthn ceremony expired or invalid")
	}
	return ceremony, nil
}
