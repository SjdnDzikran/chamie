package chat

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/dzikran/chamie/internal/conversation"
	"github.com/dzikran/chamie/internal/whatsapp"
)

type Completer interface {
	Complete(ctx context.Context, systemPrompt string, history []conversation.Message) (string, error)
}

var (
	ErrMessageRequired = errors.New("message is required")
	ErrSessionRequired = errors.New("session ID is required")
)

type Service struct {
	store        conversation.Store
	completer    Completer
	sender       whatsapp.Sender
	systemPrompt string
	historyLimit int
	phoneLocks   sync.Map
}

func NewService(store conversation.Store, completer Completer, sender whatsapp.Sender, systemPrompt string, historyLimit int) *Service {
	return &Service{
		store:        store,
		completer:    completer,
		sender:       sender,
		systemPrompt: systemPrompt,
		historyLimit: historyLimit,
	}
}

func (s *Service) Handle(ctx context.Context, inbound whatsapp.InboundMessage) error {
	body := strings.TrimSpace(inbound.Body)
	if body == "" {
		return nil
	}
	if strings.TrimSpace(inbound.EventID) == "" {
		return fmt.Errorf("webhook event ID is required")
	}

	lock := s.phoneLock(inbound.PhoneNumber)
	lock.Lock()
	defer lock.Unlock()

	process, err := s.store.BeginEvent(ctx, inbound.EventID)
	if err != nil {
		return fmt.Errorf("begin webhook event: %w", err)
	}
	if !process {
		return nil
	}

	if err := s.store.Append(ctx, inbound.PhoneNumber, conversation.Message{
		ID:        inbound.EventID + ":user",
		Role:      "user",
		Content:   body,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("store inbound message: %w", err)
	}

	if s.sender != nil {
		if err := s.sender.MarkRead(ctx, inbound.ID); err != nil {
			slog.Warn("failed to mark WhatsApp message as read", "message_id", inbound.ID, "error", err)
		}
	}

	history, err := s.store.Recent(ctx, inbound.PhoneNumber, s.historyLimit)
	if err != nil {
		return fmt.Errorf("load conversation history: %w", err)
	}
	response, err := s.completer.Complete(ctx, s.systemPrompt, history)
	if err != nil {
		return fmt.Errorf("generate tutor response: %w", err)
	}
	response = strings.TrimSpace(response)
	if response == "" {
		return fmt.Errorf("generate tutor response: empty response")
	}

	if s.sender != nil {
		if err := s.sender.SendText(ctx, inbound.PhoneNumber, response); err != nil {
			return fmt.Errorf("send tutor response: %w", err)
		}
	}
	if err := s.store.Append(ctx, inbound.PhoneNumber, conversation.Message{
		ID:        inbound.EventID + ":assistant",
		Role:      "assistant",
		Content:   response,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("store assistant message: %w", err)
	}
	if err := s.store.CompleteEvent(ctx, inbound.EventID); err != nil {
		return fmt.Errorf("complete webhook event: %w", err)
	}
	return nil
}

func (s *Service) Chat(ctx context.Context, sessionID, message string) (string, error) {
	body := strings.TrimSpace(message)
	if body == "" {
		return "", ErrMessageRequired
	}
	if strings.TrimSpace(sessionID) == "" {
		return "", ErrSessionRequired
	}

	participant := "web:" + sessionID
	lock := s.phoneLock(participant)
	lock.Lock()
	defer lock.Unlock()

	userMessageID := participant + ":" + fmt.Sprintf("%d", time.Now().UTC().UnixNano()) + ":user"
	if err := s.store.Append(ctx, participant, conversation.Message{
		ID:        userMessageID,
		Role:      "user",
		Content:   body,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return "", fmt.Errorf("store user message: %w", err)
	}

	history, err := s.store.Recent(ctx, participant, s.historyLimit)
	if err != nil {
		return "", fmt.Errorf("load conversation history: %w", err)
	}
	response, err := s.completer.Complete(ctx, s.systemPrompt, history)
	if err != nil {
		return "", fmt.Errorf("generate tutor response: %w", err)
	}
	response = strings.TrimSpace(response)
	if response == "" {
		return "", fmt.Errorf("generate tutor response: empty response")
	}

	if err := s.store.Append(ctx, participant, conversation.Message{
		ID:        participant + ":" + fmt.Sprintf("%d", time.Now().UTC().UnixNano()) + ":assistant",
		Role:      "assistant",
		Content:   response,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return "", fmt.Errorf("store assistant message: %w", err)
	}
	return response, nil
}

func (s *Service) phoneLock(phoneNumber string) *sync.Mutex {
	lock, _ := s.phoneLocks.LoadOrStore(phoneNumber, &sync.Mutex{})
	return lock.(*sync.Mutex)
}
