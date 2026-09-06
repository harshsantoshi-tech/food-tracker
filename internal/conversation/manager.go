package conversation

import (
	"context"
	"fmt"
	"time"

	"github.com/harshsantoshi-tech/food-tracker/internal/calculation"
	"github.com/harshsantoshi-tech/food-tracker/internal/food"
	"github.com/harshsantoshi-tech/food-tracker/internal/nutrition"
	"github.com/harshsantoshi-tech/food-tracker/internal/storage"
)

// FoodParser is the minimal interface Manager needs from food.Parser.
type FoodParser interface {
	Parse(ctx context.Context, message string) (*food.ParseResult, error)
}

// NutritionProvider is the minimal interface Manager needs from
// nutrition.Provider (or nutrition.CachingProvider wrapping it).
type NutritionProvider interface {
	SearchFood(ctx context.Context, query string) ([]nutrition.FoodMatch, error)
	GetNutrition(ctx context.Context, foodID string) (*nutrition.NutritionData, error)
}

// CalculationEngine is the minimal interface Manager needs from
// calculation.Engine.
type CalculationEngine interface {
	CalculateItem(item food.Item, nutritionData *nutrition.NutritionData) (*calculation.LineItem, error)
	Aggregate(items []calculation.LineItem) calculation.NutritionSummary
}

// FoodLogSaver is the minimal interface Manager needs to persist a
// completed meal.
type FoodLogSaver interface {
	Save(ctx context.Context, log *storage.FoodLog) (int64, error)
}

// Manager orchestrates the full pipeline for one incoming WhatsApp
// message: parse → (maybe ask for clarification) → look up nutrition →
// calculate → persist → produce a reply.
type Manager struct {
	parser     FoodParser
	nutrition  NutritionProvider
	calc       CalculationEngine
	convStore  Store
	logSaver   FoodLogSaver
}

func NewManager(parser FoodParser, nutritionProvider NutritionProvider, calc CalculationEngine, convStore Store, logSaver FoodLogSaver) *Manager {
	return &Manager{
		parser:    parser,
		nutrition: nutritionProvider,
		calc:      calc,
		convStore: convStore,
		logSaver:  logSaver,
	}
}

// HandleMessage processes one incoming text message for a user and
// returns the text to reply with.
func (m *Manager) HandleMessage(ctx context.Context, userID int64, messageText string) (string, error) {
	existingState, err := m.convStore.Get(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("loading conversation state: %w", err)
	}

	var items []food.Item
	var originalMessage string

	if existingState != nil && existingState.Status == StatusAwaitingClarification {
		// The user is answering a clarification question. Re-parse
		// their answer combined with the original message so the LLM
		// has full context (e.g. original: "butter chicken", answer:
		// "one bowl" → combined message gives it everything it needs).
		combined := existingState.OriginalMessage + ". " + messageText
		result, err := m.parser.Parse(ctx, combined)
		if err != nil {
			return "", fmt.Errorf("parsing clarification response: %w", err)
		}
		if result.NeedsClarification {
			// Still ambiguous even after clarification — ask again
			// rather than looping forever silently.
			if err := m.saveClarificationState(ctx, userID, existingState.OriginalMessage, result); err != nil {
				return "", err
			}
			return result.ClarificationQuestion, nil
		}
		items = result.Items
		originalMessage = existingState.OriginalMessage
	} else {
		result, err := m.parser.Parse(ctx, messageText)
		if err != nil {
			return "", fmt.Errorf("parsing message: %w", err)
		}
		if result.NeedsClarification {
			if err := m.saveClarificationState(ctx, userID, messageText, result); err != nil {
				return "", err
			}
			return result.ClarificationQuestion, nil
		}
		items = result.Items
		originalMessage = messageText
	}

	// We have everything we need — resolve nutrition data and calculate.
	summary, storageItems, err := m.resolveAndCalculate(ctx, items)
	if err != nil {
		return "", err
	}

	if _, err := m.logSaver.Save(ctx, &storage.FoodLog{
		UserID:          userID,
		OriginalMessage: originalMessage,
		TotalCalories:   summary.TotalCaloriesKcal,
		TotalProtein:    summary.TotalProteinG,
		TotalCarbs:      summary.TotalCarbsG,
		TotalFat:        summary.TotalFatG,
		Items:           storageItems,
	}); err != nil {
		return "", fmt.Errorf("saving food log: %w", err)
	}

	if err := m.convStore.Clear(ctx, userID); err != nil {
		// Non-fatal: the conversation resolved successfully, a leftover
		// Redis key will just expire on its own via TTL.
		_ = err
	}

	return FormatSummary(summary), nil
}

func (m *Manager) saveClarificationState(ctx context.Context, userID int64, originalMessage string, result *food.ParseResult) error {
	return m.convStore.Save(ctx, userID, &State{
		Status:                StatusAwaitingClarification,
		OriginalMessage:       originalMessage,
		PendingItems:          result.Items,
		ClarificationQuestion: result.ClarificationQuestion,
		CreatedAt:             time.Now(),
	})
}

// resolveAndCalculate looks up nutrition data for each parsed item and
// runs it through the calculation engine, skipping (and logging,
// conceptually — see Known Limitations) any item whose nutrition data
// can't be found rather than failing the whole message.
func (m *Manager) resolveAndCalculate(ctx context.Context, items []food.Item) (calculation.NutritionSummary, []storage.FoodItem, error) {
	var lineItems []calculation.LineItem
	var storageItems []storage.FoodItem

	for _, item := range items {
		matches, err := m.nutrition.SearchFood(ctx, item.Name)
		if err != nil {
			return calculation.NutritionSummary{}, nil, fmt.Errorf("searching nutrition data for %q: %w", item.Name, err)
		}
		if len(matches) == 0 {
			return calculation.NutritionSummary{}, nil, fmt.Errorf("no nutrition data found for %q", item.Name)
		}

		data, err := m.nutrition.GetNutrition(ctx, matches[0].FoodID)
		if err != nil {
			return calculation.NutritionSummary{}, nil, fmt.Errorf("fetching nutrition data for %q: %w", item.Name, err)
		}

		line, err := m.calc.CalculateItem(item, data)
		if err != nil {
			return calculation.NutritionSummary{}, nil, fmt.Errorf("calculating %q: %w", item.Name, err)
		}

		lineItems = append(lineItems, *line)
		storageItems = append(storageItems, storage.FoodItem{
			FoodName:         item.Name,
			Quantity:         item.Quantity,
			Unit:             item.Unit,
			EstimatedWeightG: *item.EstimatedWeightG,
			Calories:         line.CaloriesKcal,
			Protein:          line.ProteinG,
			Carbs:            line.CarbsG,
			Fat:              line.FatG,
			Confidence:       item.Confidence,
		})
	}

	return m.calc.Aggregate(lineItems), storageItems, nil
}