// Package whatsapp implements the WhatsApp Cloud API client and webhook
// handling: verifying webhook subscriptions, parsing incoming messages,
// and sending outgoing text replies.
package whatsapp

//Incoming webhook payload (what WhatsApp POSTs to us)

// WebhookPayload is the top-level JSON body WhatsApp sends to our
// webhook for every event (messages, status updates, etc).
type WebhookPayload struct {
	Object string  `json:"object"`
	Entry  []Entry `json:"entry"`
}

type Entry struct {
	ID      string   `json:"id"`
	Changes []Change `json:"changes"`
}

type Change struct {
	Value Value  `json:"value"`
	Field string `json:"field"`
}

type Value struct {
	MessagingProduct string           `json:"messaging_product"`
	Metadata         Metadata         `json:"metadata"`
	Contacts         []Contact        `json:"contacts"`
	Messages         []RawMessage     `json:"messages"`
}

type Metadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type Contact struct {
	WaID    string `json:"wa_id"`
	Profile struct {
		Name string `json:"name"`
	} `json:"profile"`
}

// RawMessage is a single message exactly as WhatsApp sends it. We only
// support "text" messages in the MVP (per PROJECT_PLAN: no images/audio yet).
type RawMessage struct {
	From      string `json:"from"`
	ID        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Text      *struct {
		Body string `json:"body"`
	} `json:"text,omitempty"`
}

// --- Our simplified internal representation ---

// IncomingMessage is the clean, flattened shape the rest of our app
// works with — callers shouldn't need to know WhatsApp's nested JSON.
type IncomingMessage struct {
	MessageID string
	From      string // sender's WhatsApp number (wa_id)
	Text      string
}

// ExtractMessages flattens a WebhookPayload into a slice of
// IncomingMessage, skipping anything that isn't a text message
// (status callbacks, unsupported media types, etc).
func ExtractMessages(payload WebhookPayload) []IncomingMessage {
	var messages []IncomingMessage

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			for _, raw := range change.Value.Messages {
				if raw.Type != "text" || raw.Text == nil {
					continue
				}
				messages = append(messages, IncomingMessage{
					MessageID: raw.ID,
					From:      raw.From,
					Text:      raw.Text.Body,
				})
			}
		}
	}

	return messages
}

// --- Outgoing message payload (what we POST to WhatsApp) ---

type outgoingTextMessage struct {
	MessagingProduct string      `json:"messaging_product"`
	To               string      `json:"to"`
	Type             string      `json:"type"`
	Text             outgoingText `json:"text"`
}

type outgoingText struct {
	Body       string `json:"body"`
	PreviewURL bool   `json:"preview_url"`
}