package whatsapp

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/harshsantoshi-tech/food-tracker/internal/config"
)

// Deduplicator marks a WhatsApp message ID as processed and reports
// whether it had already been processed before (i.e. this is a
// duplicate delivery). WhatsApp can and does redeliver webhooks, so
// this check must happen before any side effects (like replying).
type Deduplicator interface {
	MarkProcessed(ctx context.Context, messageID string) (alreadyProcessed bool, err error)
}

// MessageHandler processes a single incoming message. 
// this will run the LLM parser + nutrition lookup + calculation
// pipeline.
type MessageHandler interface {
	HandleMessage(ctx context.Context, msg IncomingMessage) error
}

// Handler implements the WhatsApp webhook HTTP endpoints.
type Handler struct {
	logger      *slog.Logger
	verifyToken string
	dedup       Deduplicator
	messages    MessageHandler
}

func NewHandler(logger *slog.Logger, cfg config.WhatsAppConfig, dedup Deduplicator, messages MessageHandler) *Handler {
	return &Handler{
		logger:      logger,
		verifyToken: cfg.VerifyToken,
		dedup:       dedup,
		messages:    messages,
	}
}

// HandleVerify implements WhatsApp's webhook verification handshake.
// When you configure the webhook URL in Meta's dashboard, WhatsApp
// sends a GET request with these query params; we must echo back
// hub.challenge if hub.verify_token matches what we configured.
func (h *Handler) HandleVerify(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	if mode != "subscribe" || token != h.verifyToken {
		h.logger.Warn("webhook verification failed", "mode", mode)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(challenge))
}

// HandleIncoming receives WhatsApp message events, deduplicates them,
// and dispatches each new message to the configured MessageHandler.
func (h *Handler) HandleIncoming(w http.ResponseWriter, r *http.Request) {
	var payload WebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.logger.Error("failed to decode webhook payload", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Always return 200 quickly — WhatsApp retries aggressively on
	// non-2xx responses, which would cause duplicate processing.
	w.WriteHeader(http.StatusOK)

	ctx := r.Context()
	for _, msg := range ExtractMessages(payload) {
		alreadyProcessed, err := h.dedup.MarkProcessed(ctx, msg.MessageID)
		if err != nil {
			h.logger.Error("dedup check failed", "message_id", msg.MessageID, "error", err)
			continue
		}
		if alreadyProcessed {
			h.logger.Info("skipping duplicate message", "message_id", msg.MessageID)
			continue
		}

		if err := h.messages.HandleMessage(ctx, msg); err != nil {
			h.logger.Error("failed to handle message", "message_id", msg.MessageID, "error", err)
		}
	}
}