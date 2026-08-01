package chat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type ChatService interface {
	Chat(ctx context.Context, sessionID, message string) (string, error)
}

type WebChatHandler struct {
	service ChatService
}

func NewWebChatHandler(service ChatService) *WebChatHandler {
	return &WebChatHandler{service: service}
}

func (h *WebChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var request struct {
		SessionID string `json:"session_id"`
		Message   string `json:"message"`
	}
	if err := json.Unmarshal(body, &request); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
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
