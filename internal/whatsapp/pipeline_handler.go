package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
)

// ConversationHandler is the minimal interface PipelineHandler needs
// from conversation.Manager.
type ConversationHandler interface {
	HandleMessage(ctx context.Context, userID int64, messageText string) (string, error)
}

// UserResolver is the minimal interface PipelineHandler needs to turn
// a WhatsApp phone number into an internal user ID.
type UserResolver interface {
	ResolveUserID(ctx context.Context, whatsappNumber string) (int64, error)
}

// PipelineHandler wires an incoming WhatsApp message through the real
// food-tracking pipeline (user resolution → conversation manager →
// LLM/nutrition/calculation) and sends the result back over WhatsApp.
type PipelineHandler struct {
	logger       *slog.Logger
	users        UserResolver
	conversation ConversationHandler
	client       *Client
}

func NewPipelineHandler(logger *slog.Logger, users UserResolver, conversation ConversationHandler, client *Client) *PipelineHandler {
	return &PipelineHandler{logger: logger, users: users, conversation: conversation, client: client}
}

func (h *PipelineHandler) HandleMessage(ctx context.Context, msg IncomingMessage) error {
	userID, err := h.users.ResolveUserID(ctx, msg.From)
	if err != nil {
		return fmt.Errorf("resolving user for %s: %w", msg.From, err)
	}

	reply, err := h.conversation.HandleMessage(ctx, userID, msg.Text)
	if err != nil {
		h.logger.Error("conversation pipeline failed", "user_id", userID, "error", err)

		// Per the spec's error-handling rules: never fabricate a
		// nutrition answer on failure — send a friendly retry message
		// instead, and still report the underlying error so it's
		// observable in logs/metrics.
		fallback := "Sorry, I couldn't process that message right now. Please try again in a moment. 🙏"
		if sendErr := h.client.SendTextMessage(ctx, msg.From, fallback); sendErr != nil {
			h.logger.Error("failed to send fallback message", "error", sendErr)
		}
		return fmt.Errorf("handling message via conversation manager: %w", err)
	}

	if err := h.client.SendTextMessage(ctx, msg.From, reply); err != nil {
		return fmt.Errorf("sending reply to %s: %w", msg.From, err)
	}

	return nil
}