// Package food defines the structured representation of parsed food
// items and validates output produced by the LLM parser.
package food

// Item is a single food item extracted from a user's message.
type Item struct {
	Name              string   `json:"name"`
	Quantity          float64  `json:"quantity"`
	Unit              string   `json:"unit"`
	EstimatedWeightG  *float64 `json:"estimated_weight_g"`
	Preparation       string   `json:"preparation"`
	Confidence        float64  `json:"confidence"`
}

// ParseResult is the full structured output of the LLM food parser for
// a single user message. It may contain zero or more items, and may
// ask the user a clarifying question when portion size is ambiguous.
type ParseResult struct {
	Items                  []Item `json:"items"`
	NeedsClarification     bool   `json:"needs_clarification"`
	ClarificationQuestion  string `json:"clarification_question,omitempty"`
}