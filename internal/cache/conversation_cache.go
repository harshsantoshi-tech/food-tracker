package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConversationCache implements conversation.RedisCache. Kept
// separate from RedisNutritionCache (Phase 4) even though the Get/Set
// logic is nearly identical, because conversation state additionally
// needs deletion (clearing state once a conversation resolves) — a
// small enough difference that a shared abstraction isn't worth the
// added indirection yet.
type RedisConversationCache struct {
	client *Client
}

func NewRedisConversationCache(client *Client) *RedisConversationCache {
	return &RedisConversationCache{client: client}
}

func (r *RedisConversationCache) Get(ctx context.Context, key string) (string, bool, error) {
	val, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return val, true, nil
}

func (r *RedisConversationCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisConversationCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}