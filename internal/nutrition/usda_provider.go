package nutrition

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/harshsantoshi-tech/food-tracker/internal/config"
)

// USDAProvider implements Provider using the USDA FoodData Central API.
type USDAProvider struct {
	httpClient *http.Client
	baseURL    string // overridable in tests
	apiKey     string
}

func NewUSDAProvider(cfg config.NutritionConfig) *USDAProvider {
	apiKey := cfg.APIKey
	if apiKey == "" {
		// USDA provides a shared demo key for exploration, heavily
		// rate-limited (30 req/hour). A real key should be used in
		// production — see .env.example.
		apiKey = "DEMO_KEY"
	}
	return &USDAProvider{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    "https://api.nal.usda.gov/fdc/v1",
		apiKey:     apiKey,
	}
}

// --- USDA API response shapes (only the fields we need) ---

type usdaSearchResponse struct {
	Foods []usdaSearchFood `json:"foods"`
}

type usdaSearchFood struct {
	FdcID       int    `json:"fdcId"`
	Description string `json:"description"`
	DataType    string `json:"dataType"`
}

type usdaFoodDetail struct {
	FdcID       int              `json:"fdcId"`
	Description string           `json:"description"`
	FoodNutrients []usdaNutrient `json:"foodNutrients"`
}

type usdaNutrient struct {
	Nutrient struct {
		Name string `json:"name"`
	} `json:"nutrient"`
	Amount float64 `json:"amount"`
}

// SearchFood queries USDA FoodData Central for foods matching query.
func (p *USDAProvider) SearchFood(ctx context.Context, query string) ([]FoodMatch, error) {
	reqURL := fmt.Sprintf("%s/foods/search?api_key=%s&query=%s&pageSize=5",
		p.baseURL, p.apiKey, url.QueryEscape(query))

	var result usdaSearchResponse
	if err := p.getJSON(ctx, reqURL, &result); err != nil {
		return nil, fmt.Errorf("searching usda for %q: %w", query, err)
	}

	matches := make([]FoodMatch, 0, len(result.Foods))
	for _, f := range result.Foods {
		matches = append(matches, FoodMatch{
			FoodID:      fmt.Sprintf("%d", f.FdcID),
			Description: f.Description,
			DataType:    f.DataType,
		})
	}
	return matches, nil
}

// GetNutrition fetches full nutrition detail for a food and normalizes
// it to per-100g values.
func (p *USDAProvider) GetNutrition(ctx context.Context, foodID string) (*NutritionData, error) {
	reqURL := fmt.Sprintf("%s/food/%s?api_key=%s", p.baseURL, foodID, p.apiKey)

	var detail usdaFoodDetail
	if err := p.getJSON(ctx, reqURL, &detail); err != nil {
		return nil, fmt.Errorf("fetching usda food %s: %w", foodID, err)
	}

	data := &NutritionData{
		FoodID:      fmt.Sprintf("%d", detail.FdcID),
		Description: detail.Description,
	}

	// USDA reports each nutrient by name; SR Legacy/Foundation data is
	// already expressed per 100g, which is exactly the basis we want.
	for _, n := range detail.FoodNutrients {
		switch n.Nutrient.Name {
		case "Energy":
			data.CaloriesKcal = n.Amount
		case "Protein":
			data.ProteinG = n.Amount
		case "Carbohydrate, by difference":
			data.CarbsG = n.Amount
		case "Total lipid (fat)":
			data.FatG = n.Amount
		}
	}

	return data, nil
}

func (p *USDAProvider) getJSON(ctx context.Context, reqURL string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("calling usda api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("usda api returned status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding usda response: %w", err)
	}
	return nil
}