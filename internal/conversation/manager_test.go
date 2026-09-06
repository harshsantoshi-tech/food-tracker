package conversation

import (
	"context"
	"errors"
	"testing"

	"github.com/harshsantoshi-tech/food-tracker/internal/calculation"
	"github.com/harshsantoshi-tech/food-tracker/internal/food"
	"github.com/harshsantoshi-tech/food-tracker/internal/nutrition"
	"github.com/harshsantoshi-tech/food-tracker/internal/storage"
)


func floatPtr(f float64) *float64 { return &f }

// --- fakes ---

type fakeParser struct {
	result *food.ParseResult
	err    error
}

func (f *fakeParser) Parse(ctx context.Context, message string) (*food.ParseResult, error) {
	return f.result, f.err
}

type fakeNutrition struct{}

func (f *fakeNutrition) SearchFood(ctx context.Context, query string) ([]nutrition.FoodMatch, error) {
	return []nutrition.FoodMatch{{FoodID: "1", Description: query}}, nil
}

func (f *fakeNutrition) GetNutrition(ctx context.Context, foodID string) (*nutrition.NutritionData, error) {
	return &nutrition.NutritionData{CaloriesKcal: 200, ProteinG: 10, CarbsG: 20, FatG: 8}, nil
}

type fakeStore struct {
	states map[int64]*State
}

func newFakeStore() *fakeStore { return &fakeStore{states: map[int64]*State{}} }

func (f *fakeStore) Get(ctx context.Context, userID int64) (*State, error) {
	return f.states[userID], nil
}
func (f *fakeStore) Save(ctx context.Context, userID int64, state *State) error {
	f.states[userID] = state
	return nil
}
func (f *fakeStore) Clear(ctx context.Context, userID int64) error {
	delete(f.states, userID)
	return nil
}

type fakeLogSaver struct {
	saved []*storage.FoodLog
}

func (f *fakeLogSaver) Save(ctx context.Context, log *storage.FoodLog) (int64, error) {
	f.saved = append(f.saved, log)
	return int64(len(f.saved)), nil
}

// --- tests ---

func TestHandleMessage_ResolvesImmediately(t *testing.T) {
	parser := &fakeParser{result: &food.ParseResult{
		Items: []food.Item{
			{Name: "aloo paratha", Quantity: 2, Unit: "piece", EstimatedWeightG: floatPtr(100), Confidence: 0.85},
		},
	}}
	store := newFakeStore()
	logSaver := &fakeLogSaver{}

	manager := NewManager(parser, &fakeNutrition{}, calculation.NewEngine(), store, logSaver)

	reply, err := manager.HandleMessage(context.Background(), 1, "2 aloo paratha")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(logSaver.saved) != 1 {
		t.Fatalf("expected 1 food log saved, got %d", len(logSaver.saved))
	}
	if logSaver.saved[0].TotalCalories != 400 { // 2 × 100g @ 200kcal/100g
		t.Errorf("expected 400 total calories, got %v", logSaver.saved[0].TotalCalories)
	}
	if reply == "" {
		t.Error("expected a non-empty reply")
	}
	if _, exists := store.states[1]; exists {
		t.Error("expected conversation state to be cleared after resolving")
	}
}

func TestHandleMessage_AsksForClarification(t *testing.T) {
	parser := &fakeParser{result: &food.ParseResult{
		Items:                  []food.Item{{Name: "butter chicken", Quantity: 1, Unit: "serving", Confidence: 0.5}},
		NeedsClarification:     true,
		ClarificationQuestion:  "How much butter chicken did you eat?",
	}}
	store := newFakeStore()
	logSaver := &fakeLogSaver{}

	manager := NewManager(parser, &fakeNutrition{}, calculation.NewEngine(), store, logSaver)

	reply, err := manager.HandleMessage(context.Background(), 1, "I ate butter chicken")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if reply != "How much butter chicken did you eat?" {
		t.Errorf("expected clarification question as reply, got %q", reply)
	}
	if len(logSaver.saved) != 0 {
		t.Errorf("expected no food log saved yet, got %d", len(logSaver.saved))
	}
	if store.states[1] == nil || store.states[1].Status != StatusAwaitingClarification {
		t.Error("expected conversation state to be saved as awaiting clarification")
	}
}

func TestHandleMessage_CompletesAfterClarification(t *testing.T) {
	store := newFakeStore()
	store.states[1] = &State{
		Status:                StatusAwaitingClarification,
		OriginalMessage:       "I ate butter chicken",
		ClarificationQuestion: "How much butter chicken did you eat?",
	}

	// Second call: user answered "1 bowl", and this time the parser
	// resolves fully (no more clarification needed).
	parser := &fakeParser{result: &food.ParseResult{
		Items: []food.Item{
			{Name: "butter chicken", Quantity: 1, Unit: "bowl", EstimatedWeightG: floatPtr(250), Confidence: 0.8},
		},
	}}
	logSaver := &fakeLogSaver{}

	manager := NewManager(parser, &fakeNutrition{}, calculation.NewEngine(), store, logSaver)

	reply, err := manager.HandleMessage(context.Background(), 1, "1 bowl")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(logSaver.saved) != 1 {
		t.Fatalf("expected the food log to be saved after clarification, got %d", len(logSaver.saved))
	}
	if logSaver.saved[0].OriginalMessage != "I ate butter chicken" {
		t.Errorf("expected original message to be preserved, got %q", logSaver.saved[0].OriginalMessage)
	}
	if reply == "" {
		t.Error("expected a non-empty final reply")
	}
}

func TestHandleMessage_ParserFails(t *testing.T) {
	parser := &fakeParser{err: errors.New("llm timeout")}
	manager := NewManager(parser, &fakeNutrition{}, calculation.NewEngine(), newFakeStore(), &fakeLogSaver{})

	_, err := manager.HandleMessage(context.Background(), 1, "2 roti")
	if err == nil {
		t.Fatal("expected an error when the parser fails, got nil")
	}
}