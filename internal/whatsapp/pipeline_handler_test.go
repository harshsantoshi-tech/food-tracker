package whatsapp

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/harshsantoshi-tech/food-tracker/internal/config"
)

type fakeUserResolver struct {
	userID int64
	err    error
}

func (f fakeUserResolver) ResolveUserID(ctx context.Context, whatsappNumber string) (int64, error) {
	return f.userID, f.err
}

type fakeConversation struct {
	reply string
	err   error
}

func (f fakeConversation) HandleMessage(ctx context.Context, userID int64, messageText string) (string, error) {
	return f.reply, f.err
}

func newPipelineTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}

func TestPipelineHandler_HappyPath(t *testing.T) {
	var sentBody outgoingTextMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&sentBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(config.WhatsAppConfig{AccessToken: "t", PhoneNumberID: "1", APIVersion: "v20.0"})
	client.baseURL = server.URL

	handler := NewPipelineHandler(
		newPipelineTestLogger(),
		fakeUserResolver{userID: 42},
		fakeConversation{reply: "🍽️ Estimated Nutrition\n\n...620 kcal..."},
		client,
	)

	err := handler.HandleMessage(context.Background(), IncomingMessage{From: "16505551234", Text: "2 roti"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sentBody.To != "16505551234" {
		t.Errorf("expected reply sent to 16505551234, got %s", sentBody.To)
	}
	if sentBody.Text.Body == "" {
		t.Error("expected a non-empty reply body")
	}
}

func TestPipelineHandler_ConversationFailsSendsFallback(t *testing.T) {
	var sentBody outgoingTextMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&sentBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(config.WhatsAppConfig{AccessToken: "t", PhoneNumberID: "1", APIVersion: "v20.0"})
	client.baseURL = server.URL

	handler := NewPipelineHandler(
		newPipelineTestLogger(),
		fakeUserResolver{userID: 42},
		fakeConversation{err: errors.New("llm timeout")},
		client,
	)

	err := handler.HandleMessage(context.Background(), IncomingMessage{From: "16505551234", Text: "2 roti"})
	if err == nil {
		t.Fatal("expected an error to be returned for logging/observability, got nil")
	}
	if sentBody.Text.Body == "" {
		t.Error("expected a fallback message to still be sent to the user")
	}
}

func TestPipelineHandler_UserResolutionFails(t *testing.T) {
	handler := NewPipelineHandler(
		newPipelineTestLogger(),
		fakeUserResolver{err: errors.New("db unreachable")},
		fakeConversation{},
		NewClient(config.WhatsAppConfig{}),
	)

	err := handler.HandleMessage(context.Background(), IncomingMessage{From: "16505551234", Text: "2 roti"})
	if err == nil {
		t.Fatal("expected an error when user resolution fails, got nil")
	}
}