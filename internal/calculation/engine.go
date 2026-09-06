package calculation

import (
	"fmt"

	"github.com/harshsantoshi-tech/food-tracker/internal/food"
	"github.com/harshsantoshi-tech/food-tracker/internal/nutrition"
)

// Engine performs deterministic nutrition calculations. It has no
// external dependencies and no I/O — every method is a pure function
// of its inputs, which is what makes it trivially and exhaustively
// testable.
type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

// CalculateItem scales nutritionData (expressed per 100g) by the
// item's actual estimated total weight (quantity × weight-per-unit),
// producing the real nutrition contribution of that food item.
//
// Formula, per the product spec:
//
//	totalWeightG = quantity × estimated_weight_g_per_unit
//	value        = nutrition_per_100g × totalWeightG / 100
func (e *Engine) CalculateItem(item food.Item, nutritionData *nutrition.NutritionData) (*LineItem, error) {
	if nutritionData == nil {
		return nil, fmt.Errorf("missing nutrition data for %q", item.Name)
	}
	if item.Quantity <= 0 {
		return nil, fmt.Errorf("invalid quantity %v for %q: must be positive", item.Quantity, item.Name)
	}
	if item.EstimatedWeightG == nil {
		return nil, fmt.Errorf("missing estimated weight for %q: cannot calculate without a portion size", item.Name)
	}
	if *item.EstimatedWeightG <= 0 {
		return nil, fmt.Errorf("invalid estimated weight %v for %q: must be positive", *item.EstimatedWeightG, item.Name)
	}

	totalWeightG := item.Quantity * (*item.EstimatedWeightG)
	scale := totalWeightG / 100.0

	return &LineItem{
		FoodName:     item.Name,
		Quantity:     item.Quantity,
		Unit:         item.Unit,
		WeightG:      totalWeightG,
		CaloriesKcal: nutritionData.CaloriesKcal * scale,
		ProteinG:     nutritionData.ProteinG * scale,
		CarbsG:       nutritionData.CarbsG * scale,
		FatG:         nutritionData.FatG * scale,
	}, nil
}

// Aggregate sums a set of LineItems into a single meal-level total.
// An empty slice produces a zero-valued summary, not an error — a
// message that resolved to zero valid items is a caller-level concern,
// not a math error.
func (e *Engine) Aggregate(items []LineItem) NutritionSummary {
	summary := NutritionSummary{Items: items}

	for _, item := range items {
		summary.TotalCaloriesKcal += item.CaloriesKcal
		summary.TotalProteinG += item.ProteinG
		summary.TotalCarbsG += item.CarbsG
		summary.TotalFatG += item.FatG
	}

	return summary
}