package food

import (
	"context"
	"errors"
	"testing"
)

type fakeLLMClient struct {
	response string
	err      error
}

func (f fakeLLMClient) Complete(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.response, nil
}

func TestParse_SimpleTwoItemMessage(t *testing.T) {
	fake := fakeLLMClient{response: `{
		"items": [
			{"name": "aloo paratha", "quantity": 2, "unit": "piece", "estimated_weight_g": 100, "preparation": "typical", "confidence": 0.85},
			{"name": "curd", "quantity": 1, "unit": "serving", "estimated_weight_g": 150, "preparation": "plain", "confidence": 0.9}
		],
		"needs_clarification": false
	}`}

	parser := NewParser(fake)
	result, err := parser.Parse(context.Background(), "2 aloo paratha with curd")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0].Name != "aloo paratha" {
		t.Errorf("expected first item 'aloo paratha', got %s", result.Items[0].Name)
	}
	if result.NeedsClarification {
		t.Error("expected needs_clarification to be false")
	}
}

func TestParse_ClarificationNeeded(t *testing.T) {
	fake := fakeLLMClient{response: `{
		"items": [{"name": "butter chicken", "quantity": 1, "unit": "serving", "estimated_weight_g": null, "preparation": "restaurant-style", "confidence": 0.55}],
		"needs_clarification": true,
		"clarification_question": "Approximately how much butter chicken did you eat?"
	}`}

	parser := NewParser(fake)
	result, err := parser.Parse(context.Background(), "I ate butter chicken")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.NeedsClarification {
		t.Error("expected needs_clarification to be true")
	}
	if result.ClarificationQuestion == "" {
		t.Error("expected a non-empty clarification_question")
	}
	if result.Items[0].EstimatedWeightG != nil {
		t.Error("expected estimated_weight_g to be nil when portion is unknown")
	}
}

func TestParse_StripsMarkdownFence(t *testing.T) {
	fake := fakeLLMClient{response: "```json\n{\"items\":[{\"name\":\"roti\",\"quantity\":2,\"unit\":\"piece\",\"confidence\":0.8}],\"needs_clarification\":false}\n```"}

	parser := NewParser(fake)
	result, err := parser.Parse(context.Background(), "2 roti")
	if err != nil {
		t.Fatalf("expected fenced JSON to be handled, got error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	fake := fakeLLMClient{response: "this is not json at all"}

	parser := NewParser(fake)
	_, err := parser.Parse(context.Background(), "2 roti")
	if err == nil {
		t.Fatal("expected an error for invalid JSON, got nil")
	}
}

func TestParse_FailsValidation(t *testing.T) {
	// unit "kg" is not in our accepted unit list — this should be
	// rejected by Validate() even though the JSON itself is well-formed.
	fake := fakeLLMClient{response: `{
		"items": [{"name": "rice", "quantity": 1, "unit": "kg", "confidence": 0.7}],
		"needs_clarification": false
	}`}

	parser := NewParser(fake)
	_, err := parser.Parse(context.Background(), "1kg rice")
	if err == nil {
		t.Fatal("expected validation error for invalid unit, got nil")
	}
}

func TestParse_LLMCallFails(t *testing.T) {
	fake := fakeLLMClient{err: errors.New("connection timeout")}

	parser := NewParser(fake)
	_, err := parser.Parse(context.Background(), "2 roti")
	if err == nil {
		t.Fatal("expected an error when the LLM call fails, got nil")
	}
}