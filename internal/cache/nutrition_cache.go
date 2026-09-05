package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisNutritionCache implements nutrition.Cache using Redis. It
// satisfies that interface structurally — no import needed in either
// direction.
type RedisNutritionCache struct {
	client *Client
}

func NewRedisNutritionCache(client *Client) *RedisNutritionCache {
	return &RedisNutritionCache{client: client}
}

// Get returns (value, found=true, nil) on a cache hit, ("", false, nil)
// on a clean miss, and ("", false, err) only for real Redis failures.
func (r *RedisNutritionCache) Get(ctx context.Context, key string) (string, bool, error) {
	val, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return val, true, nil
}

func (r *RedisNutritionCache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}