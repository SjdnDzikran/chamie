package chat

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeChatService struct {
	chatFn       func(ctx context.Context, sessionID, message string) (string, error)
	chatStreamFn func(ctx context.Context, sessionID, message string, onDelta func(string)) (string, error)
}

func (f *fakeChatService) Chat(ctx context.Context, sessionID, message string) (string, error) {
	if f.chatFn != nil {
		return f.chatFn(ctx, sessionID, message)
	}
	if sessionID == "" {
		return "", ErrSessionRequired
	}
	if strings.TrimSpace(message) == "" {
		return "", ErrMessageRequired
	}
	return "hi", nil
}

func (f *fakeChatService) ChatStream(ctx context.Context, sessionID, message string, onDelta func(string)) (string, error) {
	if f.chatStreamFn != nil {
		return f.chatStreamFn(ctx, sessionID, message, onDelta)
	}
	return f.Chat(ctx, sessionID, message)
}

func TestWebChatHandlerReturnsReply(t *testing.T) {
	service := &fakeChatService{chatFn: func(_ context.Context, sessionID, message string) (string, error) {
		if sessionID != "session-1" {
			t.Errorf("sessionID = %q", sessionID)
		}
		if message != "hello" {
			t.Errorf("message = %q", message)
		}
		return "hi there", nil
	}}

	handler := NewWebChatHandler(service)
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"session_id":"session-1","message":"hello"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"reply":"hi there"`) {
		t.Errorf("body = %q", body)
	}
}

func TestWebChatHandlerRejectsEmptySession(t *testing.T) {
	handler := NewWebChatHandler(&fakeChatService{})
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"message":"hello"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestWebChatHandlerRejectsEmptyMessage(t *testing.T) {
	handler := NewWebChatHandler(&fakeChatService{})
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"session_id":"session-1","message":"  "}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestWebChatHandlerRejectsBadJSON(t *testing.T) {
	handler := NewWebChatHandler(&fakeChatService{})
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"broken`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestWebChatHandlerReturnsServerErrorOnFailure(t *testing.T) {
	service := &fakeChatService{chatFn: func(context.Context, string, string) (string, error) {
		return "", errors.New("model down")
	}}
	handler := NewWebChatHandler(service)
	req := httptest.NewRequest(http.MethodPost, "/api/chat", strings.NewReader(`{"session_id":"session-1","message":"hello"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestWebChatHandlerRejectsNonPost(t *testing.T) {
	handler := NewWebChatHandler(&fakeChatService{})
	req := httptest.NewRequest(http.MethodGet, "/api/chat", bytes.NewReader(nil))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestWebChatStreamHandlerEmitsDeltasAndDone(t *testing.T) {
	service := &fakeChatService{chatStreamFn: func(_ context.Context, sessionID, message string, onDelta func(string)) (string, error) {
		if sessionID != "session-1" {
			t.Errorf("sessionID = %q", sessionID)
		}
		if message != "hello" {
			t.Errorf("message = %q", message)
		}
		onDelta("Hel")
		onDelta("lo")
		return "Hello", nil
	}}

	handler := NewWebChatStreamHandler(service)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/stream", strings.NewReader(`{"session_id":"session-1","message":"hello"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("Content-Type = %q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `data: {"delta":"Hel"}`) || !strings.Contains(body, `data: {"delta":"lo"}`) {
		t.Errorf("body = %q", body)
	}
	if !strings.Contains(body, `data: {"done":true}`) {
		t.Errorf("body missing done event: %q", body)
	}
}

func TestWebChatStreamHandlerStreamsErrorEventOnFailure(t *testing.T) {
	service := &fakeChatService{chatStreamFn: func(context.Context, string, string, func(string)) (string, error) {
		return "", errors.New("model down")
	}}
	handler := NewWebChatStreamHandler(service)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/stream", strings.NewReader(`{"session_id":"session-1","message":"hello"}`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (SSE error is in-body)", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"error"`) {
		t.Errorf("body missing error event: %q", body)
	}
}

func TestWebChatStreamHandlerRejectsBadRequest(t *testing.T) {
	handler := NewWebChatStreamHandler(&fakeChatService{})
	req := httptest.NewRequest(http.MethodPost, "/api/chat/stream", strings.NewReader(`{"broken`))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
