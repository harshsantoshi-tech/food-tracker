package storage

import "time"

// User is a WhatsApp user who has sent at least one message.
type User struct {
	ID             int64
	WhatsAppNumber string
	Name           string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// FoodLog is a single completed meal log — one user message that
// resolved (possibly after clarification) into a nutrition summary.
type FoodLog struct {
	ID              int64
	UserID          int64
	OriginalMessage string
	TotalCalories   float64
	TotalProtein    float64
	TotalCarbs      float64
	TotalFat        float64
	CreatedAt       time.Time
	Items           []FoodItem
}

// FoodItem is a single food item's contribution within a FoodLog.
type FoodItem struct {
	ID                int64
	FoodLogID         int64
	FoodName          string
	Quantity          float64
	Unit              string
	EstimatedWeightG  float64
	Calories          float64
	Protein           float64
	Carbs             float64
	Fat               float64
	Confidence        float64
}