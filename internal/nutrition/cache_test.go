package nutrition

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// fakeCache is an in-memory stand-in for Redis.
type fakeCache struct {
	store map[string]string
}

func newFakeCache() *fakeCache {
	return &fakeCache{store: map[string]string{}}
}

func (f *fakeCache) Get(ctx context.Context, key string) (string, bool, error) {
	v, ok := f.store[key]
	return v, ok, nil
}

func (f *fakeCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	f.store[key] = value
	return nil
}

// countingProvider wraps a fixed response and counts how many times
// each method was actually called, so tests can prove the cache
// prevented a second call.
type countingProvider struct {
	searchCalls    int
	nutritionCalls int
	searchResult   []FoodMatch
	nutritionResult *NutritionData
}

func (c *countingProvider) SearchFood(ctx context.Context, query string) ([]FoodMatch, error) {
	c.searchCalls++
	return c.searchResult, nil
}

func (c *countingProvider) GetNutrition(ctx context.Context, foodID string) (*NutritionData, error) {
	c.nutritionCalls++
	return c.nutritionResult, nil
}

func TestCachingProvider_SearchFood_CachesResult(t *testing.T) {
	inner := &countingProvider{
		searchResult: []FoodMatch{{FoodID: "1", Description: "Chicken curry"}},
	}
	provider := NewCachingProvider(inner, newFakeCache())

	_, err := provider.SearchFood(context.Background(), "chicken curry")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_, err = provider.SearchFood(context.Background(), "chicken curry")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if inner.searchCalls != 1 {
		t.Errorf("expected underlying provider called once, got %d calls", inner.searchCalls)
	}
}

func TestCachingProvider_GetNutrition_CachesResult(t *testing.T) {
	inner := &countingProvider{
		nutritionResult: &NutritionData{FoodID: "1", CaloriesKcal: 180},
	}
	provider := NewCachingProvider(inner, newFakeCache())

	data1, err := provider.GetNutrition(context.Background(), "1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	data2, err := provider.GetNutrition(context.Background(), "1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if inner.nutritionCalls != 1 {
		t.Errorf("expected underlying provider called once, got %d calls", inner.nutritionCalls)
	}
	if data1.CaloriesKcal != data2.CaloriesKcal {
		t.Errorf("expected consistent cached results")
	}
}

func TestCachingProvider_DifferentKeysNotConflated(t *testing.T) {
	inner := &countingProvider{
		searchResult: []FoodMatch{{FoodID: "1", Description: "Chicken curry"}},
	}
	provider := NewCachingProvider(inner, newFakeCache())

	provider.SearchFood(context.Background(), "chicken curry")
	provider.SearchFood(context.Background(), "butter chicken") // different query

	if inner.searchCalls != 2 {
		t.Errorf("expected 2 calls for 2 distinct queries, got %d", inner.searchCalls)
	}
}

func TestCachingProvider_CorruptCacheFallsThrough(t *testing.T) {
	inner := &countingProvider{
		nutritionResult: &NutritionData{FoodID: "1", CaloriesKcal: 180},
	}
	cache := newFakeCache()
	cache.store["nutrition:food:1"] = "not valid json"

	provider := NewCachingProvider(inner, cache)

	data, err := provider.GetNutrition(context.Background(), "1")
	if err != nil {
		t.Fatalf("expected fallback to underlying provider, got error: %v", err)
	}
	if data.CaloriesKcal != 180 {
		t.Errorf("expected fallback data, got %v", data.CaloriesKcal)
	}
	if inner.nutritionCalls != 1 {
		t.Errorf("expected underlying provider called once as fallback, got %d", inner.nutritionCalls)
	}
}

var _ = json.Marshal // silence unused import if you trim tests