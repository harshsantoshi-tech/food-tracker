package storage

import (
	"context"
	"os"
	"testing"

	"github.com/harshsantoshi-tech/food-tracker/internal/config"
)

// setupTestDB connects to the local Postgres instance used in
// development. These tests require a real database — they're
// integration tests, not pure unit tests.
func setupTestDB(t *testing.T) *DB {
	t.Helper()

	cfg := config.PostgresConfig{
		Host:     getEnvOrDefault("POSTGRES_HOST", "localhost"),
		Port:     getEnvOrDefault("POSTGRES_PORT", "5432"),
		User:     getEnvOrDefault("POSTGRES_USER", "foodtracker"),
		Password: getEnvOrDefault("POSTGRES_PASSWORD", "foodtracker"),
		DBName:   getEnvOrDefault("POSTGRES_DB", "foodtracker"),
		SSLMode:  "disable",
	}

	db, err := NewPostgres(context.Background(), cfg)
	if err != nil {
		t.Skipf("skipping: could not connect to test postgres: %v", err)
	}

	t.Cleanup(func() {
		db.Exec("DELETE FROM food_items")
		db.Exec("DELETE FROM food_logs")
		db.Exec("DELETE FROM users")
		db.Close()
	})

	return db
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func TestUserRepository_GetOrCreateByWhatsAppNumber_CreatesNewUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	user, err := repo.GetOrCreateByWhatsAppNumber(context.Background(), "16505551234")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.WhatsAppNumber != "16505551234" {
		t.Errorf("expected whatsapp number 16505551234, got %s", user.WhatsAppNumber)
	}
	if user.ID == 0 {
		t.Error("expected a non-zero user ID")
	}
}

func TestUserRepository_GetOrCreateByWhatsAppNumber_ReturnsExistingUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	first, err := repo.GetOrCreateByWhatsAppNumber(context.Background(), "16505559999")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	second, err := repo.GetOrCreateByWhatsAppNumber(context.Background(), "16505559999")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if first.ID != second.ID {
		t.Errorf("expected same user ID on repeat lookup, got %d and %d", first.ID, second.ID)
	}
}

func TestFoodLogRepository_Save(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	logRepo := NewFoodLogRepository(db)

	user, err := userRepo.GetOrCreateByWhatsAppNumber(context.Background(), "16505550000")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	log := &FoodLog{
		UserID:          user.ID,
		OriginalMessage: "2 aloo paratha with curd",
		TotalCalories:   620,
		TotalProtein:    18,
		TotalCarbs:      82,
		TotalFat:        25,
		Items: []FoodItem{
			{FoodName: "aloo paratha", Quantity: 2, Unit: "piece", EstimatedWeightG: 100, Calories: 500, Protein: 12, Carbs: 60, Fat: 20, Confidence: 0.85},
			{FoodName: "curd", Quantity: 1, Unit: "serving", EstimatedWeightG: 150, Calories: 120, Protein: 6, Carbs: 9, Fat: 5, Confidence: 0.9},
		},
	}

	logID, err := logRepo.Save(context.Background(), log)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if logID == 0 {
		t.Error("expected a non-zero food log ID")
	}

	var itemCount int
	err = db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM food_items WHERE food_log_id = $1", logID).Scan(&itemCount)
	if err != nil {
		t.Fatalf("failed to count food items: %v", err)
	}
	if itemCount != 2 {
		t.Errorf("expected 2 food items saved, got %d", itemCount)
	}
}