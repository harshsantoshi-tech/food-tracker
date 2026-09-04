package whatsapp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/harshsantoshi-tech/food-tracker/internal/config"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}

// --- fakes ---

type fakeDedup struct {
	seen map[string]bool
	err  error
}

func newFakeDedup() *fakeDedup {
	return &fakeDedup{seen: map[string]bool{}}
}

func (f *fakeDedup) MarkProcessed(ctx context.Context, messageID string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	if f.seen[messageID] {
		return true, nil
	}
	f.seen[messageID] = true
	return false, nil
}

type fakeMessageHandler struct {
	handled []IncomingMessage
	err     error
}

func (f *fakeMessageHandler) HandleMessage(ctx context.Context, msg IncomingMessage) error {
	if f.err != nil {
		return f.err
	}
	f.handled = append(f.handled, msg)
	return nil
}

// --- tests ---

func TestHandleVerify_Success(t *testing.T) {
	h := NewHandler(newTestLogger(), config.WhatsAppConfig{VerifyToken: "secret"}, newFakeDedup(), &fakeMessageHandler{})

	req := httptest.NewRequest(http.MethodGet, "/webhook/whatsapp?hub.mode=subscribe&hub.verify_token=secret&hub.challenge=CHALLENGE123", nil)
	rec := httptest.NewRecorder()

	h.HandleVerify(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "CHALLENGE123" {
		t.Errorf("expected body 'CHALLENGE123', got %q", rec.Body.String())
	}
}

func TestHandleVerify_WrongToken(t *testing.T) {
	h := NewHandler(newTestLogger(), config.WhatsAppConfig{VerifyToken: "secret"}, newFakeDedup(), &fakeMessageHandler{})

	req := httptest.NewRequest(http.MethodGet, "/webhook/whatsapp?hub.mode=subscribe&hub.verify_token=wrong&hub.challenge=X", nil)
	rec := httptest.NewRecorder()

	h.HandleVerify(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

const samplePayload = `{
  "object": "whatsapp_business_account",
  "entry": [{
    "id": "1",
    "changes": [{
      "field": "messages",
      "value": {
        "messaging_product": "whatsapp",
        "metadata": {"display_phone_number": "15551234567", "phone_number_id": "1"},
        "messages": [{
          "from": "16505551234",
          "id": "wamid.ABC123",
          "timestamp": "1690000000",
          "type": "text",
          "text": {"body": "2 aloo paratha with curd"}
        }]
      }
    }]
  }]
}`

func TestHandleIncoming_ProcessesNewMessage(t *testing.T) {
	dedup := newFakeDedup()
	handler := &fakeMessageHandler{}
	h := NewHandler(newTestLogger(), config.WhatsAppConfig{}, dedup, handler)

	req := httptest.NewRequest(http.MethodPost, "/webhook/whatsapp", strings.NewReader(samplePayload))
	rec := httptest.NewRecorder()

	h.HandleIncoming(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if len(handler.handled) != 1 {
		t.Fatalf("expected 1 message handled, got %d", len(handler.handled))
	}
	if handler.handled[0].Text != "2 aloo paratha with curd" {
		t.Errorf("unexpected message text: %s", handler.handled[0].Text)
	}
	if handler.handled[0].From != "16505551234" {
		t.Errorf("unexpected sender: %s", handler.handled[0].From)
	}
}

func TestHandleIncoming_SkipsDuplicateMessage(t *testing.T) {
	dedup := newFakeDedup()
	handler := &fakeMessageHandler{}
	h := NewHandler(newTestLogger(), config.WhatsAppConfig{}, dedup, handler)

	// Simulate WhatsApp redelivering the same webhook twice.
	req1 := httptest.NewRequest(http.MethodPost, "/webhook/whatsapp", strings.NewReader(samplePayload))
	h.HandleIncoming(httptest.NewRecorder(), req1)

	req2 := httptest.NewRequest(http.MethodPost, "/webhook/whatsapp", strings.NewReader(samplePayload))
	h.HandleIncoming(httptest.NewRecorder(), req2)

	if len(handler.handled) != 1 {
		t.Fatalf("expected message to be handled exactly once, got %d", len(handler.handled))
	}
}

func TestHandleIncoming_MalformedJSON(t *testing.T) {
	h := NewHandler(newTestLogger(), config.WhatsAppConfig{}, newFakeDedup(), &fakeMessageHandler{})

	req := httptest.NewRequest(http.MethodPost, "/webhook/whatsapp", strings.NewReader("not json"))
	rec := httptest.NewRecorder()

	h.HandleIncoming(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleIncoming_ContinuesOnHandlerError(t *testing.T) {
	handler := &fakeMessageHandler{err: errors.New("downstream failure")}
	h := NewHandler(newTestLogger(), config.WhatsAppConfig{}, newFakeDedup(), handler)

	req := httptest.NewRequest(http.MethodPost, "/webhook/whatsapp", strings.NewReader(samplePayload))
	rec := httptest.NewRecorder()

	h.HandleIncoming(rec, req)

	// Even if the handler fails, we must still return 200 so WhatsApp
	// doesn't retry (we already marked the message processed).
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 even on handler error, got %d", rec.Code)
	}
}