package nutrition

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Cache is the minimal caching interface CachingProvider needs. Defining
// it here (not importing cache.Client directly) keeps this package
// decoupled from Redis specifics — the real implementation lives in
// internal/cache, tests use a fake.
type Cache interface {
	// Get returns the cached value, whether it was found, and any error.
	Get(ctx context.Context, key string) (value string, found bool, err error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
}

// CachingProvider wraps a Provider and caches its results, so repeated
// lookups for the same food don't repeatedly hit the external API
// (which is both slower and subject to USDA's rate limits).
type CachingProvider struct {
	inner Provider
	cache Cache
	ttl   time.Duration
}

// NewCachingProvider wraps inner with caching. A 7-day TTL is
// reasonable here — nutrition facts for a given food essentially never
// change, so long-lived caching is safe (unlike, say, conversation state).
func NewCachingProvider(inner Provider, cache Cache) *CachingProvider {
	return &CachingProvider{inner: inner, cache: cache, ttl: 7 * 24 * time.Hour}
}

func (c *CachingProvider) SearchFood(ctx context.Context, query string) ([]FoodMatch, error) {
	key := fmt.Sprintf("nutrition:search:%s", query)

	if cached, found, err := c.cache.Get(ctx, key); err == nil && found {
		var matches []FoodMatch
		if err := json.Unmarshal([]byte(cached), &matches); err == nil {
			return matches, nil
		}
		// If cached JSON is somehow corrupt, fall through and re-fetch
		// rather than failing the whole request.
	}

	matches, err := c.inner.SearchFood(ctx, query)
	if err != nil {
		return nil, err
	}

	if raw, err := json.Marshal(matches); err == nil {
		_ = c.cache.Set(ctx, key, string(raw), c.ttl)
	}

	return matches, nil
}

func (c *CachingProvider) GetNutrition(ctx context.Context, foodID string) (*NutritionData, error) {
	key := fmt.Sprintf("nutrition:food:%s", foodID)

	if cached, found, err := c.cache.Get(ctx, key); err == nil && found {
		var data NutritionData
		if err := json.Unmarshal([]byte(cached), &data); err == nil {
			return &data, nil
		}
	}

	data, err := c.inner.GetNutrition(ctx, foodID)
	if err != nil {
		return nil, err
	}

	if raw, err := json.Marshal(data); err == nil {
		_ = c.cache.Set(ctx, key, string(raw), c.ttl)
	}

	return data, nil
}