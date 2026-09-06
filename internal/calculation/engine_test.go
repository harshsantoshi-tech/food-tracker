package calculation

import (
	"testing"

	"github.com/harshsantoshi-tech/food-tracker/internal/food"
	"github.com/harshsantoshi-tech/food-tracker/internal/nutrition"
)

func floatPtr(f float64) *float64 { return &f }

func TestCalculateItem_SingleFood(t *testing.T) {
	engine := NewEngine()

	item := food.Item{
		Name:             "aloo paratha",
		Quantity:         2,
		Unit:             "piece",
		EstimatedWeightG: floatPtr(100), // 100g per piece
	}
	nutritionData := &nutrition.NutritionData{
		CaloriesKcal: 250, // per 100g
		ProteinG:     6,
		CarbsG:       30,
		FatG:         10,
	}

	line, err := engine.CalculateItem(item, nutritionData)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// 2 pieces × 100g = 200g total → scale = 2.0
	if line.WeightG != 200 {
		t.Errorf("expected 200g total weight, got %v", line.WeightG)
	}
	if line.CaloriesKcal != 500 {
		t.Errorf("expected 500 kcal, got %v", line.CaloriesKcal)
	}
	if line.ProteinG != 12 {
		t.Errorf("expected 12g protein, got %v", line.ProteinG)
	}
	if line.CarbsG != 60 {
		t.Errorf("expected 60g carbs, got %v", line.CarbsG)
	}
	if line.FatG != 20 {
		t.Errorf("expected 20g fat, got %v", line.FatG)
	}
}

func TestCalculateItem_DecimalQuantity(t *testing.T) {
	engine := NewEngine()

	item := food.Item{
		Name:             "rice",
		Quantity:         1.5,
		Unit:             "serving",
		EstimatedWeightG: floatPtr(100),
	}
	nutritionData := &nutrition.NutritionData{CaloriesKcal: 130, ProteinG: 2.7, CarbsG: 28, FatG: 0.3}

	line, err := engine.CalculateItem(item, nutritionData)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// 1.5 × 100g = 150g → scale = 1.5
	if line.WeightG != 150 {
		t.Errorf("expected 150g total weight, got %v", line.WeightG)
	}
	if line.CaloriesKcal != 195 {
		t.Errorf("expected 195 kcal, got %v", line.CaloriesKcal)
	}
}

func TestCalculateItem_LargeQuantity(t *testing.T) {
	engine := NewEngine()

	item := food.Item{
		Name:             "bulk rice",
		Quantity:         50,
		Unit:             "serving",
		EstimatedWeightG: floatPtr(200),
	}
	nutritionData := &nutrition.NutritionData{CaloriesKcal: 130, ProteinG: 2.7, CarbsG: 28, FatG: 0.3}

	line, err := engine.CalculateItem(item, nutritionData)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// 50 × 200g = 10,000g → scale = 100
	if line.WeightG != 10000 {
		t.Errorf("expected 10000g total weight, got %v", line.WeightG)
	}
	if line.CaloriesKcal != 13000 {
		t.Errorf("expected 13000 kcal, got %v", line.CaloriesKcal)
	}
}

func TestCalculateItem_ZeroQuantity(t *testing.T) {
	engine := NewEngine()
	item := food.Item{Name: "roti", Quantity: 0, Unit: "piece", EstimatedWeightG: floatPtr(40)}
	nutritionData := &nutrition.NutritionData{CaloriesKcal: 100}

	_, err := engine.CalculateItem(item, nutritionData)
	if err == nil {
		t.Fatal("expected an error for zero quantity, got nil")
	}
}

func TestCalculateItem_NegativeQuantity(t *testing.T) {
	engine := NewEngine()
	item := food.Item{Name: "roti", Quantity: -2, Unit: "piece", EstimatedWeightG: floatPtr(40)}
	nutritionData := &nutrition.NutritionData{CaloriesKcal: 100}

	_, err := engine.CalculateItem(item, nutritionData)
	if err == nil {
		t.Fatal("expected an error for negative quantity, got nil")
	}
}

func TestCalculateItem_MissingWeight(t *testing.T) {
	engine := NewEngine()
	item := food.Item{Name: "butter chicken", Quantity: 1, Unit: "serving", EstimatedWeightG: nil}
	nutritionData := &nutrition.NutritionData{CaloriesKcal: 250}

	_, err := engine.CalculateItem(item, nutritionData)
	if err == nil {
		t.Fatal("expected an error when estimated weight is nil, got nil")
	}
}

func TestCalculateItem_ZeroWeight(t *testing.T) {
	engine := NewEngine()
	item := food.Item{Name: "roti", Quantity: 1, Unit: "piece", EstimatedWeightG: floatPtr(0)}
	nutritionData := &nutrition.NutritionData{CaloriesKcal: 100}

	_, err := engine.CalculateItem(item, nutritionData)
	if err == nil {
		t.Fatal("expected an error for zero weight, got nil")
	}
}

func TestCalculateItem_NegativeWeight(t *testing.T) {
	engine := NewEngine()
	item := food.Item{Name: "roti", Quantity: 1, Unit: "piece", EstimatedWeightG: floatPtr(-40)}
	nutritionData := &nutrition.NutritionData{CaloriesKcal: 100}

	_, err := engine.CalculateItem(item, nutritionData)
	if err == nil {
		t.Fatal("expected an error for negative weight, got nil")
	}
}

func TestCalculateItem_MissingNutritionData(t *testing.T) {
	engine := NewEngine()
	item := food.Item{Name: "mystery food", Quantity: 1, Unit: "piece", EstimatedWeightG: floatPtr(100)}

	_, err := engine.CalculateItem(item, nil)
	if err == nil {
		t.Fatal("expected an error when nutrition data is nil, got nil")
	}
}

func TestAggregate_MultipleFoods(t *testing.T) {
	engine := NewEngine()

	items := []LineItem{
		{FoodName: "aloo paratha", CaloriesKcal: 500, ProteinG: 12, CarbsG: 60, FatG: 20},
		{FoodName: "curd", CaloriesKcal: 120, ProteinG: 6, CarbsG: 9, FatG: 5},
	}

	summary := engine.Aggregate(items)

	if summary.TotalCaloriesKcal != 620 {
		t.Errorf("expected 620 total kcal, got %v", summary.TotalCaloriesKcal)
	}
	if summary.TotalProteinG != 18 {
		t.Errorf("expected 18g total protein, got %v", summary.TotalProteinG)
	}
	if summary.TotalCarbsG != 69 {
		t.Errorf("expected 69g total carbs, got %v", summary.TotalCarbsG)
	}
	if summary.TotalFatG != 25 {
		t.Errorf("expected 25g total fat, got %v", summary.TotalFatG)
	}
	if len(summary.Items) != 2 {
		t.Errorf("expected 2 line items retained in summary, got %d", len(summary.Items))
	}
}

func TestAggregate_SingleFood(t *testing.T) {
	engine := NewEngine()
	items := []LineItem{{FoodName: "roti", CaloriesKcal: 120, ProteinG: 4, CarbsG: 20, FatG: 3}}

	summary := engine.Aggregate(items)

	if summary.TotalCaloriesKcal != 120 {
		t.Errorf("expected 120 kcal, got %v", summary.TotalCaloriesKcal)
	}
}

func TestAggregate_EmptyList(t *testing.T) {
	engine := NewEngine()
	summary := engine.Aggregate([]LineItem{})

	if summary.TotalCaloriesKcal != 0 || summary.TotalProteinG != 0 || summary.TotalCarbsG != 0 || summary.TotalFatG != 0 {
		t.Error("expected all-zero totals for an empty item list")
	}
}