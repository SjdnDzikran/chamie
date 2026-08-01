package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dzikran/chamie/internal/conversation"
)

func TestClientSendsOpenAICompatibleChatCompletion(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"  Hello!  "}}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL+"/v1/", "secret", "deepseek-chat", server.Client())
	response, err := client.Complete(context.Background(), "Tutor instructions", []conversation.Message{
		{Role: "user", Content: "Hi"},
		{Role: "assistant", Content: "Hello"},
		{Role: "user", Content: "Help me"},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if response != "Hello!" {
		t.Errorf("Complete() = %q", response)
	}
	if gotPath != "/v1/chat/completions" {
		t.Errorf("request path = %q", gotPath)
	}
	if gotAuth != "Bearer secret" {
		t.Errorf("Authorization = %q", gotAuth)
	}
	if gotBody.Model != "deepseek-chat" {
		t.Errorf("model = %q", gotBody.Model)
	}
	if len(gotBody.Messages) != 4 {
		t.Fatalf("len(messages) = %d", len(gotBody.Messages))
	}
	if gotBody.Messages[0].Role != "system" || gotBody.Messages[0].Content != "Tutor instructions" {
		t.Errorf("system message = %#v", gotBody.Messages[0])
	}
	if gotBody.Messages[3].Role != "user" || gotBody.Messages[3].Content != "Help me" {
		t.Errorf("last message = %#v", gotBody.Messages[3])
	}
}

func TestClientReturnsAPIErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":{"message":"rate limited"}}`, http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", "deepseek-chat", server.Client())
	_, err := client.Complete(context.Background(), "prompt", []conversation.Message{{Role: "user", Content: "Hi"}})
	if err == nil || !strings.Contains(err.Error(), "429") || !strings.Contains(err.Error(), "rate limited") {
		t.Fatalf("Complete() error = %v", err)
	}
}

func TestClientRejectsEmptyCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", "deepseek-chat", server.Client())
	_, err := client.Complete(context.Background(), "prompt", []conversation.Message{{Role: "user", Content: "Hi"}})
	if err == nil || !strings.Contains(err.Error(), "empty completion") {
		t.Fatalf("Complete() error = %v", err)
	}
}
