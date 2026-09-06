package storage

import (
	"context"
	"fmt"
)

// FoodLogRepository manages persistence of completed food logs and
// their line items.
type FoodLogRepository struct {
	db *DB
}

func NewFoodLogRepository(db *DB) *FoodLogRepository {
	return &FoodLogRepository{db: db}
}

// Save persists a food log and all of its items in a single
// transaction — either the whole meal is recorded, or none of it is.
// A meal log with only some of its items saved would silently under-
// report nutrition, which is worse than failing the whole write.
func (r *FoodLogRepository) Save(ctx context.Context, log *FoodLog) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback() // no-op if Commit succeeds first

	const insertLog = `
		INSERT INTO food_logs (user_id, original_message, total_calories, total_protein, total_carbs, total_fat)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	var logID int64
	err = tx.QueryRowContext(ctx, insertLog,
		log.UserID, log.OriginalMessage, log.TotalCalories, log.TotalProtein, log.TotalCarbs, log.TotalFat,
	).Scan(&logID)
	if err != nil {
		return 0, fmt.Errorf("inserting food log: %w", err)
	}

	const insertItem = `
		INSERT INTO food_items (food_log_id, food_name, quantity, unit, estimated_weight_g, calories, protein, carbs, fat, confidence)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	for _, item := range log.Items {
		_, err := tx.ExecContext(ctx, insertItem,
			logID, item.FoodName, item.Quantity, item.Unit, item.EstimatedWeightG,
			item.Calories, item.Protein, item.Carbs, item.Fat, item.Confidence,
		)
		if err != nil {
			return 0, fmt.Errorf("inserting food item %q: %w", item.FoodName, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("committing transaction: %w", err)
	}

	return logID, nil
}