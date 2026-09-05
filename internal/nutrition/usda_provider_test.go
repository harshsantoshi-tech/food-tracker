package nutrition

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harshsantoshi-tech/food-tracker/internal/config"
)

func TestSearchFood_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("query") != "chicken curry" {
			t.Errorf("expected query 'chicken curry', got %q", r.URL.Query().Get("query"))
		}
		json.NewEncoder(w).Encode(usdaSearchResponse{
			Foods: []usdaSearchFood{
				{FdcID: 123, Description: "Chicken curry, home-prepared", DataType: "Survey (FNDDS)"},
			},
		})
	}))
	defer server.Close()

	provider := NewUSDAProvider(config.NutritionConfig{APIKey: "test-key"})
	provider.baseURL = server.URL

	matches, err := provider.SearchFood(context.Background(), "chicken curry")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if matches[0].FoodID != "123" {
		t.Errorf("expected FoodID 123, got %s", matches[0].FoodID)
	}
}

func TestGetNutrition_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(usdaFoodDetail{
			FdcID:       123,
			Description: "Chicken curry, home-prepared",
			FoodNutrients: []usdaNutrient{
				{Nutrient: struct {
					Name string `json:"name"`
				}{Name: "Energy"}, Amount: 180},
				{Nutrient: struct {
					Name string `json:"name"`
				}{Name: "Protein"}, Amount: 12},
				{Nutrient: struct {
					Name string `json:"name"`
				}{Name: "Carbohydrate, by difference"}, Amount: 8},
				{Nutrient: struct {
					Name string `json:"name"`
				}{Name: "Total lipid (fat)"}, Amount: 11},
			},
		})
	}))
	defer server.Close()

	provider := NewUSDAProvider(config.NutritionConfig{APIKey: "test-key"})
	provider.baseURL = server.URL

	data, err := provider.GetNutrition(context.Background(), "123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if data.CaloriesKcal != 180 {
		t.Errorf("expected 180 kcal, got %v", data.CaloriesKcal)
	}
	if data.ProteinG != 12 {
		t.Errorf("expected 12g protein, got %v", data.ProteinG)
	}
}

func TestSearchFood_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	provider := NewUSDAProvider(config.NutritionConfig{APIKey: "test-key"})
	provider.baseURL = server.URL

	_, err := provider.SearchFood(context.Background(), "anything")
	if err == nil {
		t.Fatal("expected an error for a 429 response, got nil")
	}
}