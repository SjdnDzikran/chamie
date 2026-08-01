package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dzikran/chamie/internal/conversation"
)

type Client struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey, model string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		model:      model,
		httpClient: httpClient,
	}
}

func (c *Client) Complete(ctx context.Context, systemPrompt string, history []conversation.Message) (string, error) {
	type chatMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	messages := make([]chatMessage, 0, len(history)+1)
	if strings.TrimSpace(systemPrompt) != "" {
		messages = append(messages, chatMessage{Role: "system", Content: systemPrompt})
	}
	for _, message := range history {
		messages = append(messages, chatMessage{Role: message.Role, Content: message.Content})
	}

	payload := struct {
		Model    string        `json:"model"`
		Messages []chatMessage `json:"messages"`
	}{Model: c.model, Messages: messages}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode chat completion request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create chat completion request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send chat completion request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read chat completion response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("chat completion API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return "", fmt.Errorf("decode chat completion response: %w", err)
	}
	if len(result.Choices) == 0 || strings.TrimSpace(result.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("chat completion API returned an empty completion")
	}
	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}

func (c *Client) CompleteStream(ctx context.Context, systemPrompt string, history []conversation.Message, onDelta func(string)) (string, error) {
	type chatMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	messages := make([]chatMessage, 0, len(history)+1)
	if strings.TrimSpace(systemPrompt) != "" {
		messages = append(messages, chatMessage{Role: "system", Content: systemPrompt})
	}
	for _, message := range history {
		messages = append(messages, chatMessage{Role: message.Role, Content: message.Content})
	}

	payload := struct {
		Model    string        `json:"model"`
		Messages []chatMessage `json:"messages"`
		Stream   bool          `json:"stream"`
	}{Model: c.model, Messages: messages, Stream: true}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode chat completion request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create chat completion request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send chat completion request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if readErr != nil {
			return "", fmt.Errorf("chat completion API returned %d", resp.StatusCode)
		}
		return "", fmt.Errorf("chat completion API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var builder strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return builder.String(), fmt.Errorf("decode chat completion stream chunk: %w", err)
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		builder.WriteString(delta)
		if onDelta != nil {
			onDelta(delta)
		}
	}
	if err := scanner.Err(); err != nil {
		return builder.String(), fmt.Errorf("read chat completion stream: %w", err)
	}

	result := strings.TrimSpace(builder.String())
	if result == "" {
		return "", fmt.Errorf("chat completion API returned an empty completion")
	}
	return result, nil
}
