// Package llm provides a client for calling an LLM with structured
// JSON output requirements. It knows nothing about food or nutrition —
// it only knows how to send a prompt and get text back.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/harshsantoshi-tech/food-tracker/internal/config"
)

// Client calls the Anthropic Messages API.
type Client struct {
	httpClient *http.Client
	baseURL    string // overridable in tests
	apiKey     string
	model      string
}

func NewClient(cfg config.LLMConfig) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://api.anthropic.com",
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
	}
}

type messagesRequest struct {
	Model     string           `json:"model"`
	MaxTokens int              `json:"max_tokens"`
	System    string           `json:"system"`
	Messages  []requestMessage `json:"messages"`
}

type requestMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messagesResponse struct {
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Complete sends a system prompt + a single user message to the LLM and
// returns the raw text of its response. Callers are responsible for
// parsing/validating that text (see food.Parser) — this package only
// handles the wire protocol.
func (c *Client) Complete(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	reqBody := messagesRequest{
		Model:     c.model,
		MaxTokens: 1024,
		System:    systemPrompt,
		Messages: []requestMessage{
			{Role: "user", Content: userMessage},
		},
	}

	rawBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling llm request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(rawBody))
	if err != nil {
		return "", fmt.Errorf("building llm request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling llm api: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading llm response body: %w", err)
	}

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("llm api returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var parsed messagesResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", fmt.Errorf("parsing llm response envelope: %w", err)
	}

	for _, block := range parsed.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}

	return "", fmt.Errorf("llm response contained no text content")
}