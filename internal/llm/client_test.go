package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harshsantoshi-tech/food-tracker/internal/config"
)

func TestComplete_Success(t *testing.T) {
	var capturedReq messagesRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected x-api-key header 'test-key', got %q", r.Header.Get("x-api-key"))
		}
		if err := json.NewDecoder(r.Body).Decode(&capturedReq); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(messagesResponse{
			Content: []contentBlock{{Type: "text", Text: `{"items":[],"needs_clarification":false}`}},
		})
	}))
	defer server.Close()

	client := NewClient(config.LLMConfig{APIKey: "test-key", Model: "claude-sonnet-4-6"})
	client.baseURL = server.URL

	text, err := client.Complete(context.Background(), "system prompt here", "2 roti and dal")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if text != `{"items":[],"needs_clarification":false}` {
		t.Errorf("unexpected response text: %s", text)
	}
	if capturedReq.Model != "claude-sonnet-4-6" {
		t.Errorf("expected model claude-sonnet-4-6, got %s", capturedReq.Model)
	}
	if capturedReq.Messages[0].Content != "2 roti and dal" {
		t.Errorf("expected user message '2 roti and dal', got %s", capturedReq.Messages[0].Content)
	}
}

func TestComplete_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	defer server.Close()

	client := NewClient(config.LLMConfig{APIKey: "test-key", Model: "claude-sonnet-4-6"})
	client.baseURL = server.URL

	_, err := client.Complete(context.Background(), "system", "user message")
	if err == nil {
		t.Fatal("expected an error for a 429 response, got nil")
	}
}

func TestComplete_NoTextContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(messagesResponse{Content: []contentBlock{}})
	}))
	defer server.Close()

	client := NewClient(config.LLMConfig{APIKey: "test-key", Model: "claude-sonnet-4-6"})
	client.baseURL = server.URL

	_, err := client.Complete(context.Background(), "system", "user message")
	if err == nil {
		t.Fatal("expected an error when response has no text content")
	}
}