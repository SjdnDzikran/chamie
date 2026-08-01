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
	chatFn func(ctx context.Context, sessionID, message string) (string, error)
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
