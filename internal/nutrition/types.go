// Package nutrition provides food nutrition lookups behind a provider
// interface, so the underlying data source (USDA FoodData Central,
// or something else later) can be swapped without touching callers.
package nutrition

import "context"

// FoodMatch is a single search result — a candidate food the caller
// can then fetch full nutrition data for.
type FoodMatch struct {
	FoodID      string
	Description string
	DataType    string // e.g. "Foundation", "SR Legacy", "Branded"
}

// NutritionData holds nutrition values normalized to a common basis:
// per 100 grams. This normalization is what makes the calculation
// engine's math simple later (nutrition_per_100g × grams / 100).
type NutritionData struct {
	FoodID      string
	Description string
	CaloriesKcal float64
	ProteinG    float64
	CarbsG      float64
	FatG        float64
}

// Provider looks up food nutrition data from some external or internal
// source. Implementations must never fabricate data — if nutrition
// data cannot be found, return an error rather than a guess.
type Provider interface {
	SearchFood(ctx context.Context, query string) ([]FoodMatch, error)
	GetNutrition(ctx context.Context, foodID string) (*NutritionData, error)
}