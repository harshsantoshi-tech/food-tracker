package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
)

// EchoHandler is a placeholder MessageHandler for Phase 2 — it proves
// the webhook → dedup → reply round trip works before the real
// LLM/nutrition pipeline exists 
type EchoHandler struct {
	logger *slog.Logger
	client *Client
}

func NewEchoHandler(logger *slog.Logger, client *Client) *EchoHandler {
	return &EchoHandler{logger: logger, client: client}
}

func (h *EchoHandler) HandleMessage(ctx context.Context, msg IncomingMessage) error {
	reply := fmt.Sprintf("Got your message: %q. Food understanding is coming soon! 🍽️", msg.Text)

	if err := h.client.SendTextMessage(ctx, msg.From, reply); err != nil {
		return fmt.Errorf("sending reply: %w", err)
	}

	h.logger.Info("replied to message", "to", msg.From, "message_id", msg.MessageID)
	return nil
}