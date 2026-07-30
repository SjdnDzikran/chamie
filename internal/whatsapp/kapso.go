package whatsapp

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Sender interface {
	SendText(ctx context.Context, phoneNumber, text string) error
	MarkRead(ctx context.Context, messageID string) error
}

type KapsoSender struct {
	endpoint   string
	apiKey     string
	httpClient *http.Client
}

func NewKapsoSender(baseURL, apiKey, phoneNumberID string, httpClient *http.Client) *KapsoSender {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/" + phoneNumberID + "/messages"
	return &KapsoSender{endpoint: endpoint, apiKey: apiKey, httpClient: httpClient}
}

func (s *KapsoSender) SendText(ctx context.Context, phoneNumber, text string) error {
	return s.send(ctx, map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                phoneNumber,
		"type":              "text",
		"text": map[string]string{
			"body": text,
		},
	})
}

func (s *KapsoSender) MarkRead(ctx context.Context, messageID string) error {
	return s.send(ctx, map[string]any{
		"messaging_product": "whatsapp",
		"status":            "read",
		"message_id":        messageID,
	})
}

func (s *KapsoSender) send(ctx context.Context, payload map[string]any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode Kapso request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create Kapso request: %w", err)
	}
	req.Header.Set("X-API-Key", s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send Kapso request: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read Kapso response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Kapso API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	return nil
}

type InboundMessage struct {
	ID             string
	EventID        string
	PhoneNumber    string
	ConversationID string
	Body           string
}

type WebhookHandler struct {
	secret    string
	onMessage func(context.Context, InboundMessage) error
}

func NewWebhookHandler(secret string, onMessage func(context.Context, InboundMessage) error) *WebhookHandler {
	return &WebhookHandler{secret: secret, onMessage: onMessage}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !validSignature(body, r.Header.Get("X-Webhook-Signature"), h.secret) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	var payload struct {
		Message struct {
			ID   string `json:"id"`
			Type string `json:"type"`
			From string `json:"from"`
			Text *struct {
				Body string `json:"body"`
			} `json:"text"`
			Kapso struct {
				Direction string `json:"direction"`
			} `json:"kapso"`
		} `json:"message"`
		Conversation struct {
			ID string `json:"id"`
		} `json:"conversation"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if payload.Message.Kapso.Direction != "inbound" || payload.Message.Type != "text" || payload.Message.Text == nil || strings.TrimSpace(payload.Message.Text.Body) == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if h.onMessage != nil {
		eventID := strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
		if eventID == "" {
			eventID = payload.Message.ID
		}
		err := h.onMessage(r.Context(), InboundMessage{
			ID:             payload.Message.ID,
			EventID:        eventID,
			PhoneNumber:    payload.Message.From,
			ConversationID: payload.Conversation.ID,
			Body:           strings.TrimSpace(payload.Message.Text.Body),
		})
		if err != nil {
			http.Error(w, "message processing failed", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func validSignature(body []byte, signature, secret string) bool {
	if signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expected))
}
