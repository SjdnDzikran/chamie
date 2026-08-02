package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dzikran/chamie/internal/conversation"
)

type ChatService interface {
	Chat(ctx context.Context, sessionID, message string) (string, error)
}

type ChatStreamService interface {
	ChatStream(ctx context.Context, sessionID, message string, onDelta func(string)) (string, error)
}

type ChatHistoryService interface {
	History(ctx context.Context, sessionID string) ([]conversation.Message, error)
}

type chatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

type chatHistoryMessage struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func decodeChatRequest(w http.ResponseWriter, r *http.Request) (*chatRequest, bool) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return nil, false
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return nil, false
	}

	var request chatRequest
	if err := json.Unmarshal(body, &request); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return nil, false
	}
	return &request, true
}

type WebChatHandler struct {
	service ChatService
}

func NewWebChatHandler(service ChatService) *WebChatHandler {
	return &WebChatHandler{service: service}
}

func (h *WebChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeChatRequest(w, r)
	if !ok {
		return
	}

	reply, err := h.service.Chat(r.Context(), request.SessionID, request.Message)
	if err != nil {
		switch {
		case errors.Is(err, ErrMessageRequired), errors.Is(err, ErrSessionRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			http.Error(w, "tutor request failed", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"reply": reply})
}

type WebChatStreamHandler struct {
	service ChatStreamService
}

func NewWebChatStreamHandler(service ChatStreamService) *WebChatStreamHandler {
	return &WebChatStreamHandler{service: service}
}

func (h *WebChatStreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	request, ok := decodeChatRequest(w, r)
	if !ok {
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	writeEvent := func(event map[string]any) error {
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("encode SSE event: %w", err)
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	_, err := h.service.ChatStream(r.Context(), request.SessionID, request.Message, func(delta string) {
		_ = writeEvent(map[string]any{"delta": delta})
	})
	if err != nil {
		_ = writeEvent(map[string]any{"error": "tutor request failed"})
		return
	}
	_ = writeEvent(map[string]any{"done": true})
}

type WebChatHistoryHandler struct {
	service ChatHistoryService
}

func NewWebChatHistoryHandler(service ChatHistoryService) *WebChatHistoryHandler {
	return &WebChatHistoryHandler{service: service}
}

func (h *WebChatHistoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	history, err := h.service.History(r.Context(), r.URL.Query().Get("session_id"))
	if err != nil {
		if errors.Is(err, ErrSessionRequired) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "load chat history failed", http.StatusInternalServerError)
		return
	}

	messages := make([]chatHistoryMessage, len(history))
	for i, message := range history {
		messages[i] = chatHistoryMessage{
			ID:        message.ID,
			Role:      message.Role,
			Content:   message.Content,
			CreatedAt: message.CreatedAt,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"messages": messages})
}
