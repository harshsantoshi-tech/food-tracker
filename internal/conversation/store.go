package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Store persists and retrieves in-progress conversation state, keyed
// by user ID. Defined here (not tied to Redis directly) so tests can
// use an in-memory fake.
type Store interface {
	Get(ctx context.Context, userID int64) (*State, error) // nil, nil if no state exists
	Save(ctx context.Context, userID int64, state *State) error
	Clear(ctx context.Context, userID int64) error
}

// RedisCache is the minimal interface Store needs from Redis. Reusing
// this narrow shape (rather than a full Redis client) keeps this
// package testable and decoupled, same pattern as nutrition.Cache.
type RedisCache interface {
	Get(ctx context.Context, key string) (value string, found bool, err error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// RedisStore implements Store using Redis, so conversation state
// naturally expires if a user abandons a clarification flow.
type RedisStore struct {
	cache RedisCache
	ttl   time.Duration
}

// NewRedisStore creates a Store with a 30-minute TTL — long enough for
// a real person to read a clarification question and reply, short
// enough that abandoned conversations don't linger indefinitely.
func NewRedisStore(cache RedisCache) *RedisStore {
	return &RedisStore{cache: cache, ttl: 30 * time.Minute}
}

func (s *RedisStore) key(userID int64) string {
	return fmt.Sprintf("conversation:user:%d", userID)
}

func (s *RedisStore) Get(ctx context.Context, userID int64) (*State, error) {
	raw, found, err := s.cache.Get(ctx, s.key(userID))
	if err != nil {
		return nil, fmt.Errorf("getting conversation state: %w", err)
	}
	if !found {
		return nil, nil
	}

	var state State
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return nil, fmt.Errorf("decoding conversation state: %w", err)
	}
	return &state, nil
}

func (s *RedisStore) Save(ctx context.Context, userID int64, state *State) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encoding conversation state: %w", err)
	}
	if err := s.cache.Set(ctx, s.key(userID), string(raw), s.ttl); err != nil {
		return fmt.Errorf("saving conversation state: %w", err)
	}
	return nil
}

func (s *RedisStore) Clear(ctx context.Context, userID int64) error {
	if err := s.cache.Delete(ctx, s.key(userID)); err != nil {
		return fmt.Errorf("clearing conversation state: %w", err)
	}
	return nil
}