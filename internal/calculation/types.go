// Package calculation performs deterministic nutrition math: scaling
// per-100g nutrition data by actual portion weight, and aggregating
// multiple food items into a single meal summary. Nothing in this
// package ever guesses or estimates — every number here is arithmetic
// on inputs supplied by the food parser and nutrition provider.
package calculation

// LineItem is the calculated nutrition contribution of a single food
// item, after scaling per-100g data by its actual estimated weight.
type LineItem struct {
	FoodName     string
	Quantity     float64
	Unit         string
	WeightG      float64
	CaloriesKcal float64
	ProteinG     float64
	CarbsG       float64
	FatG         float64
}

// NutritionSummary aggregates one or more LineItems into meal totals.
type NutritionSummary struct {
	Items             []LineItem
	TotalCaloriesKcal float64
	TotalProteinG     float64
	TotalCarbsG       float64
	TotalFatG         float64
}