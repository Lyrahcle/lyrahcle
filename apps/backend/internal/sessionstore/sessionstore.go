package sessionstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrUnavailable = errors.New("sessionstore: unavailable")
var ErrNotFound = errors.New("sessionstore: session not found")

type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
}

type Store struct {
	client *redis.Client
}

func New(client *redis.Client) *Store {
	return &Store{client: client}
}

func sessionKey(id string) string {
	return "lyrahcle:session:" + id
}

func userSessionsKey(userID string) string {
	return "lyrahcle:user-sessions:" + userID
}

func normalizeUserAgent(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 160 {
		value = value[:160]
	}
	return value
}

func (s *Store) Register(ctx context.Context, session Session) error {
	if s == nil || s.client == nil {
		return ErrUnavailable
	}
	if session.ID == "" || session.UserID == "" || session.ExpiresAt.IsZero() || !session.ExpiresAt.After(time.Now()) {
		return errors.New("sessionstore: invalid session")
	}
	session.IP = strings.TrimSpace(session.IP)
	session.UserAgent = normalizeUserAgent(session.UserAgent)
	payload, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("sessionstore: encode: %w", err)
	}
	ttl := time.Until(session.ExpiresAt)
	pipe := s.client.TxPipeline()
	pipe.Set(ctx, sessionKey(session.ID), payload, ttl)
	pipe.ZAdd(ctx, userSessionsKey(session.UserID), redis.Z{Score: float64(session.ExpiresAt.Unix()), Member: session.ID})
	pipe.Expire(ctx, userSessionsKey(session.UserID), ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("sessionstore: register: %w", err)
	}
	return nil
}

func (s *Store) Get(ctx context.Context, id string) (Session, error) {
	if s == nil || s.client == nil {
		return Session{}, ErrUnavailable
	}
	payload, err := s.client.Get(ctx, sessionKey(id)).Bytes()
	if errors.Is(err, redis.Nil) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("sessionstore: get: %w", err)
	}
	var session Session
	if err := json.Unmarshal(payload, &session); err != nil {
		return Session{}, fmt.Errorf("sessionstore: decode: %w", err)
	}
	return session, nil
}

func (s *Store) List(ctx context.Context, userID string) ([]Session, error) {
	if s == nil || s.client == nil {
		return nil, ErrUnavailable
	}
	key := userSessionsKey(userID)
	now := time.Now().Unix()
	pipe := s.client.TxPipeline()
	pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", now))
	idsCmd := pipe.ZRange(ctx, key, 0, -1)
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("sessionstore: list ids: %w", err)
	}
	ids, err := idsCmd.Result()
	if err != nil {
		return nil, fmt.Errorf("sessionstore: list ids result: %w", err)
	}
	if len(ids) == 0 {
		return []Session{}, nil
	}
	getPipe := s.client.Pipeline()
	commands := make([]*redis.StringCmd, 0, len(ids))
	for _, id := range ids {
		commands = append(commands, getPipe.Get(ctx, sessionKey(id)))
	}
	if _, err := getPipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("sessionstore: list sessions: %w", err)
	}
	out := make([]Session, 0, len(commands))
	nowTime := time.Now()
	for _, cmd := range commands {
		payload, err := cmd.Bytes()
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("sessionstore: list session: %w", err)
		}
		var session Session
		if err := json.Unmarshal(payload, &session); err != nil {
			return nil, fmt.Errorf("sessionstore: decode session: %w", err)
		}
		if session.UserID == userID && session.ExpiresAt.After(nowTime) {
			out = append(out, session)
		}
	}
	return out, nil
}

func (s *Store) Delete(ctx context.Context, userID, id string) error {
	if s == nil || s.client == nil {
		return ErrUnavailable
	}
	pipe := s.client.TxPipeline()
	pipe.Del(ctx, sessionKey(id))
	pipe.ZRem(ctx, userSessionsKey(userID), id)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("sessionstore: delete: %w", err)
	}
	return nil
}

func (s *Store) DeleteUser(ctx context.Context, userID string) error {
	if s == nil || s.client == nil {
		return ErrUnavailable
	}
	ids, err := s.client.ZRange(ctx, userSessionsKey(userID), 0, -1).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("sessionstore: list user sessions: %w", err)
	}
	pipe := s.client.TxPipeline()
	for _, id := range ids {
		pipe.Del(ctx, sessionKey(id))
	}
	pipe.Del(ctx, userSessionsKey(userID))
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("sessionstore: delete user: %w", err)
	}
	return nil
}
