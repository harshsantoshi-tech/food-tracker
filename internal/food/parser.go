package food

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// LLMClient is the minimal interface the parser needs from an LLM
// client. Defining it here (rather than importing llm.Client directly)
// means this package doesn't depend on how the LLM is actually called —
// tests can supply a fake, and the real llm.Client satisfies this
// interface automatically since Go interfaces are structural.
type LLMClient interface {
	Complete(ctx context.Context, systemPrompt, userMessage string) (string, error)
}

// Parser turns a natural language food message into a validated
// ParseResult using an LLM, without ever trusting the LLM's output
// blindly.
type Parser struct {
	llm LLMClient
}

func NewParser(llm LLMClient) *Parser {
	return &Parser{llm: llm}
}

// Parse sends the user's message to the LLM and returns a validated
// ParseResult. If the LLM's output is malformed or fails validation,
// an error is returned rather than trusting partial/bad data.
func (p *Parser) Parse(ctx context.Context, message string) (*ParseResult, error) {
	raw, err := p.llm.Complete(ctx, SystemPrompt, message)
	if err != nil {
		return nil, fmt.Errorf("llm completion failed: %w", err)
	}

	cleaned := stripMarkdownFence(raw)

	var result ParseResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, fmt.Errorf("llm returned invalid JSON: %w (raw: %s)", err, raw)
	}

	if err := result.Validate(); err != nil {
		return nil, fmt.Errorf("llm output failed validation: %w", err)
	}

	return &result, nil
}

// stripMarkdownFence removes ```json ... ``` wrapping if the LLM added
// it despite instructions not to — a defensive measure since LLMs
// don't always follow formatting instructions perfectly.
func stripMarkdownFence(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}