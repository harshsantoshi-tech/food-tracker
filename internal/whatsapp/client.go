package whatsapp

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

// Client sends messages via the WhatsApp Cloud API.
type Client struct {
	httpClient    *http.Client
	baseURL       string // overridable in tests
	accessToken   string
	phoneNumberID string
	apiVersion    string
}

// NewClient builds a WhatsApp API client from config.
func NewClient(cfg config.WhatsAppConfig) *Client {
	return &Client{
		httpClient:    &http.Client{Timeout: 10 * time.Second},
		baseURL:       "https://graph.facebook.com",
		accessToken:   cfg.AccessToken,
		phoneNumberID: cfg.PhoneNumberID,
		apiVersion:    cfg.APIVersion,
	}
}

// SendTextMessage sends a plain-text WhatsApp message to the given
// recipient (a WhatsApp phone number / wa_id).
func (c *Client) SendTextMessage(ctx context.Context, to, body string) error {
	payload := outgoingTextMessage{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "text",
		Text:             outgoingText{Body: body, PreviewURL: false},
	}

	rawBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling outgoing message: %w", err)
	}

	url := fmt.Sprintf("%s/%s/%s/messages", c.baseURL, c.apiVersion, c.phoneNumberID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rawBody))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request to whatsapp api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("whatsapp api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}