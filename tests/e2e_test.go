// Package tests contains end-to-end tests exercising the full pipeline:
// WhatsApp webhook → conversation manager → LLM parser → nutrition
// provider → calculation engine → PostgreSQL → WhatsApp reply.
// Only the LLM and WhatsApp Cloud APIs are faked (via httptest); every
// internal component is real.
package tests

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/harshsantoshi-tech/food-tracker/internal/cache"
	"github.com/harshsantoshi-tech/food-tracker/internal/calculation"
	"github.com/harshsantoshi-tech/food-tracker/internal/config"
	"github.com/harshsantoshi-tech/food-tracker/internal/conversation"
	"github.com/harshsantoshi-tech/food-tracker/internal/food"
	apphttp "github.com/harshsantoshi-tech/food-tracker/internal/http"
	"github.com/harshsantoshi-tech/food-tracker/internal/llm"
	"github.com/harshsantoshi-tech/food-tracker/internal/nutrition"
	"github.com/harshsantoshi-tech/food-tracker/internal/storage"
	"github.com/harshsantoshi-tech/food-tracker/internal/whatsapp"

	"log/slog"
)

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// fakeLLMServer returns an httptest server that mimics Anthropic's
// Messages API, always returning a fixed structured food JSON payload —
// standing in for the real LLM so this test is deterministic.
func fakeLLMServer(t *testing.T, responseJSON string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"content": []map[string]string{{"type": "text", "text": responseJSON}},
		})
	}))
}

// fakeUSDAServer mimics USDA FoodData Central's search + detail
// endpoints with fixed nutrition data for "roti".
func fakeUSDAServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/foods/search", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"foods": []map[string]interface{}{
				{"fdcId": 999, "description": "Bread, whole wheat (roti-like)", "dataType": "SR Legacy"},
			},
		})
	})
	mux.HandleFunc("/food/999", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"fdcId":       999,
			"description": "Bread, whole wheat",
			"foodNutrients": []map[string]interface{}{
				{"nutrient": map[string]string{"name": "Energy"}, "amount": 100},
				{"nutrient": map[string]string{"name": "Protein"}, "amount": 4},
				{"nutrient": map[string]string{"name": "Carbohydrate, by difference"}, "amount": 20},
				{"nutrient": map[string]string{"name": "Total lipid (fat)"}, "amount": 2},
			},
		})
	})
	return httptest.NewServer(mux)
}

func TestEndToEnd_SimpleMessageResolvesAndReplies(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// --- real Postgres (same local dev DB used throughout this project) ---
	pgCfg := config.PostgresConfig{
		Host: getEnvOrDefault("POSTGRES_HOST", "localhost"), Port: "5432",
		User: "foodtracker", Password: "foodtracker", DBName: "foodtracker", SSLMode: "disable",
	}
	db, err := storage.NewPostgres(ctx, pgCfg)
	if err != nil {
		t.Skipf("skipping e2e test: could not connect to postgres: %v", err)
	}
	defer func() {
		db.Exec("DELETE FROM food_items")
		db.Exec("DELETE FROM food_logs")
		db.Exec("DELETE FROM users")
		db.Close()
	}()

	// --- real Redis ---
	redisClient, err := cache.NewRedis(ctx, config.RedisConfig{Addr: getEnvOrDefault("REDIS_ADDR", "localhost:6379")})
	if err != nil {
		t.Skipf("skipping e2e test: could not connect to redis: %v", err)
	}
	defer redisClient.Close()

	// --- fake external services ---
	llmServer := fakeLLMServer(t, `{
		"items": [{"name": "roti", "quantity": 2, "unit": "piece", "estimated_weight_g": 40, "preparation": "typical", "confidence": 0.9}],
		"needs_clarification": false
	}`)
	defer llmServer.Close()

	usdaServer := fakeUSDAServer(t)
	defer usdaServer.Close()

	var whatsappSentText string
	whatsappServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if text, ok := body["text"].(map[string]interface{}); ok {
			whatsappSentText, _ = text["body"].(string)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer whatsappServer.Close()

	// --- wire the real pipeline, pointing external clients at fakes ---
	llmClient := llm.NewClient(config.LLMConfig{APIKey: "test", Model: "test-model"})
	llmClient.SetBaseURL(llmServer.URL)
	foodParser := food.NewParser(llmClient)

	usdaProvider := nutrition.NewUSDAProvider(config.NutritionConfig{APIKey: "test"})
	usdaProvider.SetBaseURL(usdaServer.URL)
	nutritionCache := cache.NewRedisNutritionCache(redisClient)
	nutritionProvider := nutrition.NewCachingProvider(usdaProvider, nutritionCache)

	calcEngine := calculation.NewEngine()

	convCache := cache.NewRedisConversationCache(redisClient)
	convStore := conversation.NewRedisStore(convCache)

	userRepo := storage.NewUserRepository(db)
	userResolver := storage.NewUserIDResolver(userRepo)
	foodLogRepo := storage.NewFoodLogRepository(db)

	convManager := conversation.NewManager(foodParser, nutritionProvider, calcEngine, convStore, foodLogRepo)

	waClient := whatsapp.NewClient(config.WhatsAppConfig{AccessToken: "test", PhoneNumberID: "1", APIVersion: "v20.0"})
	waClient.SetBaseURL(whatsappServer.URL)

	dedup := cache.NewRedisDeduplicator(redisClient)
	pipelineHandler := whatsapp.NewPipelineHandler(logger, userResolver, convManager, waClient)
	waHandler := whatsapp.NewHandler(logger, config.WhatsAppConfig{VerifyToken: "secret"}, dedup, pipelineHandler)

	httpServer := apphttp.NewServer(logger, db, redisClient, waHandler, 10*time.Second)

	// --- fire a real HTTP request at the real webhook handler ---
	payload := `{
		"object": "whatsapp_business_account",
		"entry": [{"id": "1", "changes": [{"field": "messages", "value": {
			"messaging_product": "whatsapp",
			"metadata": {"display_phone_number": "1", "phone_number_id": "1"},
			"messages": [{"from": "16505551234", "id": "wamid.E2E1", "timestamp": "1690000000", "type": "text", "text": {"body": "2 roti"}}]
		}}]}]
	}`

	req := httptest.NewRequest(http.MethodPost, "/webhook/whatsapp", strings.NewReader(payload))
	rec := httptest.NewRecorder()
	httpServer.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from webhook, got %d", rec.Code)
	}

	// Give the (synchronous, but let's be safe) handler a moment; in
	// this implementation processing is synchronous within the request,
	// so this should already be done by the time ServeHTTP returns.
	if whatsappSentText == "" {
		t.Fatal("expected a reply to have been sent via the (fake) WhatsApp API")
	}
	if !strings.Contains(whatsappSentText, "kcal") {
		t.Errorf("expected reply to mention kcal, got: %s", whatsappSentText)
	}

	// --- verify it was actually persisted to Postgres ---
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM food_logs").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query food_logs: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 food log persisted, got %d", count)
	}

	var totalCalories float64
	err = db.QueryRowContext(ctx, "SELECT total_calories FROM food_logs LIMIT 1").Scan(&totalCalories)
	if err != nil {
		t.Fatalf("failed to query total_calories: %v", err)
	}
	// 2 roti × 40g = 80g, at 100 kcal/100g → 80 kcal
	if totalCalories != 80 {
		t.Errorf("expected 80 total calories persisted, got %v", totalCalories)
	}
}