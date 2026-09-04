// Package cache manages the Redis client used for conversation state,
// webhook deduplication, nutrition response caching, and rate limiting.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/harshsantoshi-tech/food-tracker/internal/config"
)

// Client wraps a Redis client.
type Client struct {
	*redis.Client
}

// NewRedis creates a Redis client and verifies connectivity with a PING.
func NewRedis(ctx context.Context, cfg config.RedisConfig) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("pinging redis: %w", err)
	}

	return &Client{rdb}, nil
}

// HealthCheck verifies Redis is reachable within the given context.
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return c.Ping(ctx).Err()
}