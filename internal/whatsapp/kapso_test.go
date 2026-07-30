package whatsapp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKapsoSenderSendsText(t *testing.T) {
	var gotPath, gotAPIKey string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAPIKey = r.Header.Get("X-API-Key")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	sender := NewKapsoSender(server.URL+"/v24.0/", "kapso-key", "phone-id", server.Client())
	if err := sender.SendText(context.Background(), "628123", "Hello"); err != nil {
		t.Fatalf("SendText() error = %v", err)
	}

	if gotPath != "/v24.0/phone-id/messages" {
		t.Errorf("path = %q", gotPath)
	}
	if gotAPIKey != "kapso-key" {
		t.Errorf("X-API-Key = %q", gotAPIKey)
	}
	if gotBody["to"] != "628123" || gotBody["type"] != "text" {
		t.Errorf("payload = %#v", gotBody)
	}
	text, ok := gotBody["text"].(map[string]any)
	if !ok || text["body"] != "Hello" {
		t.Errorf("text payload = %#v", gotBody["text"])
	}
}

func TestKapsoSenderReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "invalid recipient", http.StatusBadRequest)
	}))
	defer server.Close()

	sender := NewKapsoSender(server.URL, "kapso-key", "phone-id", server.Client())
	err := sender.SendText(context.Background(), "bad", "Hello")
	if err == nil || !strings.Contains(err.Error(), "400") || !strings.Contains(err.Error(), "invalid recipient") {
		t.Fatalf("SendText() error = %v", err)
	}
}

func TestKapsoSenderMarksMessageRead(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	sender := NewKapsoSender(server.URL, "kapso-key", "phone-id", server.Client())
	if err := sender.MarkRead(context.Background(), "wamid.1"); err != nil {
		t.Fatalf("MarkRead() error = %v", err)
	}
	if gotBody["status"] != "read" || gotBody["message_id"] != "wamid.1" {
		t.Errorf("payload = %#v", gotBody)
	}
}

func TestWebhookAcceptsSignedInboundText(t *testing.T) {
	const body = `{
		"message":{"id":"wamid.1","timestamp":"123","type":"text","from":"628123","text":{"body":"Hello"},"kapso":{"direction":"inbound"}},
		"conversation":{"id":"conversation-1"}
	}`

	var got InboundMessage
	handler := NewWebhookHandler("webhook-secret", func(_ context.Context, message InboundMessage) error {
		got = message
		return nil
	})
	req := httptest.NewRequest(http.MethodPost, "/webhooks/whatsapp", strings.NewReader(body))
	req.Header.Set("X-Webhook-Signature", sign(body, "webhook-secret"))
	req.Header.Set("X-Idempotency-Key", "event-1")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if got.ID != "wamid.1" || got.PhoneNumber != "628123" || got.ConversationID != "conversation-1" || got.Body != "Hello" {
		t.Errorf("inbound message = %#v", got)
	}
	if got.EventID != "event-1" {
		t.Errorf("EventID = %q", got.EventID)
	}
}

func TestWebhookRejectsInvalidSignature(t *testing.T) {
	handler := NewWebhookHandler("webhook-secret", nil)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/whatsapp", strings.NewReader(`{}`))
	req.Header.Set("X-Webhook-Signature", "invalid")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestWebhookIgnoresOutboundAndNonTextMessages(t *testing.T) {
	called := 0
	handler := NewWebhookHandler("webhook-secret", func(_ context.Context, _ InboundMessage) error {
		called++
		return nil
	})

	for _, body := range []string{
		`{"message":{"id":"1","type":"text","kapso":{"direction":"outbound"}}}`,
		`{"message":{"id":"2","type":"image","kapso":{"direction":"inbound"}}}`,
	} {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("X-Webhook-Signature", sign(body, "webhook-secret"))
		handler.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Errorf("status = %d for %s", recorder.Code, body)
		}
	}
	if called != 0 {
		t.Fatalf("callback called %d times", called)
	}
}

func sign(body, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}
