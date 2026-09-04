package whatsapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harshsantoshi-tech/food-tracker/internal/config"
)

func TestSendTextMessage_Success(t *testing.T) {
	var capturedAuth string
	var capturedBody outgoingTextMessage

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"messaging_product":"whatsapp","messages":[{"id":"wamid.123"}]}`))
	}))
	defer server.Close()

	client := NewClient(config.WhatsAppConfig{
		AccessToken:   "test-token",
		PhoneNumberID: "12345",
		APIVersion:    "v20.0",
	})
	client.baseURL = server.URL

	err := client.SendTextMessage(context.Background(), "16505551234", "hello there")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if capturedAuth != "Bearer test-token" {
		t.Errorf("expected Authorization header 'Bearer test-token', got %q", capturedAuth)
	}
	if capturedBody.To != "16505551234" {
		t.Errorf("expected To=16505551234, got %s", capturedBody.To)
	}
	if capturedBody.Text.Body != "hello there" {
		t.Errorf("expected text body 'hello there', got %s", capturedBody.Text.Body)
	}
}

func TestSendTextMessage_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"invalid token"}}`))
	}))
	defer server.Close()

	client := NewClient(config.WhatsAppConfig{
		AccessToken:   "bad-token",
		PhoneNumberID: "12345",
		APIVersion:    "v20.0",
	})
	client.baseURL = server.URL

	err := client.SendTextMessage(context.Background(), "16505551234", "hello")
	if err == nil {
		t.Fatal("expected an error for a 401 response, got nil")
	}
}