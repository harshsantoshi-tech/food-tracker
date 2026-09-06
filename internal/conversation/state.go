// Package conversation manages multi-turn clarification flows: when a
// message is ambiguous (e.g. "butter chicken" with no stated portion),
// the bot asks a follow-up question and must remember what was being
// discussed when the user's answer arrives as a separate message.
package conversation

import (
	"time"

	"github.com/harshsantoshi-tech/food-tracker/internal/food"
)

// Status describes where a user's conversation currently stands.
type Status string

const (
	// StatusAwaitingClarification means we asked the user a follow-up
	// question and are waiting for their reply to complete the
	// pending food item(s).
	StatusAwaitingClarification Status = "awaiting_clarification"
)

// State is the data we remember between messages for a single user
// while a clarification flow is in progress. There is deliberately no
// "idle" state stored — if there's no State in Redis for a user, that
// IS the idle state (nothing to remember).
type State struct {
	Status                Status      `json:"status"`
	OriginalMessage       string      `json:"original_message"`
	PendingItems          []food.Item `json:"pending_items"`
	ClarificationQuestion string      `json:"clarification_question"`
	CreatedAt             time.Time   `json:"created_at"`
}