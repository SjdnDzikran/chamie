package conversation

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Message struct {
	ID        string
	Role      string
	Content   string
	CreatedAt time.Time
}

type Store interface {
	BeginEvent(ctx context.Context, eventID string) (bool, error)
	Append(ctx context.Context, phoneNumber string, message Message) error
	Recent(ctx context.Context, phoneNumber string, limit int) ([]Message, error)
	CompleteEvent(ctx context.Context, eventID string) error
}

type MemoryStore struct {
	mu         sync.RWMutex
	messages   map[string][]Message
	messageIDs map[string]struct{}
	events     map[string]bool
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		messages:   make(map[string][]Message),
		messageIDs: make(map[string]struct{}),
		events:     make(map[string]bool),
	}
}

func (s *MemoryStore) BeginEvent(_ context.Context, eventID string) (bool, error) {
	if strings.TrimSpace(eventID) == "" {
		return false, fmt.Errorf("event ID is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	completed, exists := s.events[eventID]
	if !exists {
		s.events[eventID] = false
	}
	return !completed, nil
}

func (s *MemoryStore) Append(_ context.Context, phoneNumber string, message Message) error {
	if err := validateMessage(phoneNumber, message); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.messageIDs[message.ID]; exists {
		return nil
	}
	s.messages[phoneNumber] = append(s.messages[phoneNumber], message)
	s.messageIDs[message.ID] = struct{}{}
	return nil
}

func (s *MemoryStore) CompleteEvent(_ context.Context, eventID string) error {
	if strings.TrimSpace(eventID) == "" {
		return fmt.Errorf("event ID is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.events[eventID]; !exists {
		return fmt.Errorf("webhook event not found: %s", eventID)
	}
	s.events[eventID] = true
	return nil
}

func (s *MemoryStore) Recent(_ context.Context, phoneNumber string, limit int) ([]Message, error) {
	if strings.TrimSpace(phoneNumber) == "" {
		return nil, fmt.Errorf("phone number is required")
	}
	if limit < 1 {
		return nil, fmt.Errorf("limit must be positive")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	messages := s.messages[phoneNumber]
	start := max(0, len(messages)-limit)
	result := make([]Message, len(messages)-start)
	copy(result, messages[start:])
	return result, nil
}

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) BeginEvent(ctx context.Context, eventID string) (bool, error) {
	if strings.TrimSpace(eventID) == "" {
		return false, fmt.Errorf("event ID is required")
	}
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO webhook_events (event_id) VALUES ($1)
		ON CONFLICT (event_id) DO NOTHING
	`, eventID); err != nil {
		return false, fmt.Errorf("begin webhook event: %w", err)
	}

	var pending bool
	if err := s.pool.QueryRow(ctx, `
		SELECT processed_at IS NULL FROM webhook_events WHERE event_id = $1
	`, eventID).Scan(&pending); err != nil {
		return false, fmt.Errorf("read webhook event: %w", err)
	}
	return pending, nil
}

func (s *PostgresStore) Append(ctx context.Context, phoneNumber string, message Message) error {
	if err := validateMessage(phoneNumber, message); err != nil {
		return err
	}
	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now().UTC()
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO conversation_messages (message_id, phone_number, role, content, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (message_id) DO NOTHING
	`, message.ID, phoneNumber, message.Role, message.Content, message.CreatedAt)
	if err != nil {
		return fmt.Errorf("append conversation message: %w", err)
	}
	return nil
}

func (s *PostgresStore) Recent(ctx context.Context, phoneNumber string, limit int) ([]Message, error) {
	if strings.TrimSpace(phoneNumber) == "" {
		return nil, fmt.Errorf("phone number is required")
	}
	if limit < 1 {
		return nil, fmt.Errorf("limit must be positive")
	}

	rows, err := s.pool.Query(ctx, `
		SELECT message_id, role, content, created_at
		FROM (
			SELECT id, message_id, role, content, created_at
			FROM conversation_messages
			WHERE phone_number = $1
			ORDER BY id DESC
			LIMIT $2
		) recent
		ORDER BY id ASC
	`, phoneNumber, limit)
	if err != nil {
		return nil, fmt.Errorf("query recent conversation messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var message Message
		if err := rows.Scan(&message.ID, &message.Role, &message.Content, &message.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan conversation message: %w", err)
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversation messages: %w", err)
	}
	return messages, nil
}

func (s *PostgresStore) CompleteEvent(ctx context.Context, eventID string) error {
	result, err := s.pool.Exec(ctx, `
		UPDATE webhook_events SET processed_at = NOW() WHERE event_id = $1
	`, eventID)
	if err != nil {
		return fmt.Errorf("complete webhook event: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("webhook event not found: %s", eventID)
	}
	return nil
}

func validateMessage(phoneNumber string, message Message) error {
	if strings.TrimSpace(phoneNumber) == "" {
		return fmt.Errorf("phone number is required")
	}
	if strings.TrimSpace(message.ID) == "" {
		return fmt.Errorf("message ID is required")
	}
	if message.Role != "user" && message.Role != "assistant" {
		return fmt.Errorf("role must be user or assistant")
	}
	if strings.TrimSpace(message.Content) == "" {
		return fmt.Errorf("message content is required")
	}
	return nil
}
