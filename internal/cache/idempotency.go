package cache

import (
	"context"
	"fmt"
	"time"
)

// RedisDeduplicator implements idempotency tracking for webhook message
// IDs using Redis. It satisfies whatsapp.Deduplicator structurally
// (Go interfaces are implicit — no import cycle needed).
type RedisDeduplicator struct {
	client *Client
	ttl    time.Duration
}

// NewRedisDeduplicator creates a deduplicator that remembers processed
// message IDs for the given TTL. WhatsApp typically stops retrying
// well within 24h, so that's a safe default.
func NewRedisDeduplicator(client *Client) *RedisDeduplicator {
	return &RedisDeduplicator{client: client, ttl: 24 * time.Hour}
}

// MarkProcessed atomically checks whether messageID has been seen
// before and marks it as seen, in a single round trip. Redis's SETNX
// ("set if not exists") is atomic, which avoids the race condition of
// doing a separate GET then SET.
func (r *RedisDeduplicator) MarkProcessed(ctx context.Context, messageID string) (bool, error) {
	key := fmt.Sprintf("whatsapp:dedup:%s", messageID)

	// SetNX returns true if the key was newly set (i.e. NOT a duplicate).
	wasSet, err := r.client.SetNX(ctx, key, "1", r.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("checking message dedup key: %w", err)
	}

	alreadyProcessed := !wasSet
	return alreadyProcessed, nil
}