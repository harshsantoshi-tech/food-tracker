package conversation

import (
	"fmt"
	"strings"

	"github.com/harshsantoshi-tech/food-tracker/internal/calculation"
)

// FormatSummary renders a NutritionSummary as the final WhatsApp reply
// text, matching the concise style from the product spec.
func FormatSummary(summary calculation.NutritionSummary) string {
	var itemNames []string
	for _, item := range summary.Items {
		itemNames = append(itemNames, formatItemLabel(item))
	}

	return fmt.Sprintf(
		"🍽️ Estimated Nutrition\n\n%s\n\n🔥 Calories: ~%.0f kcal\n🥩 Protein: ~%.0fg\n🍞 Carbs: ~%.0fg\n🥑 Fat: ~%.0fg\n\n⚠️ Estimate only. Actual nutrition depends on portion size, ingredients, oil/ghee and preparation method.",
		strings.Join(itemNames, " + "),
		summary.TotalCaloriesKcal,
		summary.TotalProteinG,
		summary.TotalCarbsG,
		summary.TotalFatG,
	)
}

func formatItemLabel(item calculation.LineItem) string {
	if item.Unit == "piece" {
		return fmt.Sprintf("%.0f %s", item.Quantity, item.FoodName)
	}
	return fmt.Sprintf("%.0fg %s", item.WeightG, item.FoodName)
}