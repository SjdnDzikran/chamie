package chat

import (
	"context"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dzikran/chamie/internal/conversation"
	"github.com/dzikran/chamie/internal/whatsapp"
)

func TestServiceProcessesMessageInOrder(t *testing.T) {
	var operations []string
	store := &fakeStore{
		beginEventFn: func(_ context.Context, eventID string) (bool, error) {
			operations = append(operations, "begin:"+eventID)
			return true, nil
		},
		appendFn: func(_ context.Context, phone string, message conversation.Message) error {
			operations = append(operations, "append:"+message.ID+":"+message.Role+":"+message.Content)
			return nil
		},
		recentFn: func(_ context.Context, phone string, limit int) ([]conversation.Message, error) {
			operations = append(operations, "recent")
			if phone != "628123" || limit != 30 {
				t.Errorf("Recent(%q, %d)", phone, limit)
			}
			return []conversation.Message{{Role: "user", Content: "How do I use has?"}}, nil
		},
		completeEventFn: func(_ context.Context, eventID string) error {
			operations = append(operations, "complete-event:"+eventID)
			return nil
		},
	}
	completer := &fakeCompleter{completeFn: func(_ context.Context, prompt string, history []conversation.Message) (string, error) {
		operations = append(operations, "complete")
		if prompt != "system prompt" {
			t.Errorf("prompt = %q", prompt)
		}
		if len(history) != 1 || history[0].Content != "How do I use has?" {
			t.Errorf("history = %#v", history)
		}
		return "Use 'has' with he, she, or it.", nil
	}}
	sender := &fakeSender{
		markReadFn: func(_ context.Context, id string) error {
			operations = append(operations, "read:"+id)
			return nil
		},
		sendTextFn: func(_ context.Context, phone, text string) error {
			operations = append(operations, "send:"+phone+":"+text)
			return nil
		},
	}

	service := NewService(store, completer, sender, "system prompt", 30)
	err := service.Handle(context.Background(), whatsapp.InboundMessage{
		ID:          "wamid.1",
		EventID:     "event-1",
		PhoneNumber: "628123",
		Body:        "How do I use has?",
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	want := []string{
		"begin:event-1",
		"append:event-1:user:user:How do I use has?",
		"read:wamid.1",
		"recent",
		"complete",
		"send:628123:Use 'has' with he, she, or it.",
		"append:event-1:assistant:assistant:Use 'has' with he, she, or it.",
		"complete-event:event-1",
	}
	if !reflect.DeepEqual(operations, want) {
		t.Errorf("operations = %#v, want %#v", operations, want)
	}
}

func TestServiceSkipsCompletedWebhookEvent(t *testing.T) {
	called := false
	store := &fakeStore{
		beginEventFn: func(context.Context, string) (bool, error) { return false, nil },
		appendFn: func(context.Context, string, conversation.Message) error {
			called = true
			return nil
		},
	}
	service := NewService(store, &fakeCompleter{}, &fakeSender{}, "prompt", 30)

	err := service.Handle(context.Background(), whatsapp.InboundMessage{EventID: "event-1", PhoneNumber: "628123", Body: "hello"})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if called {
		t.Fatal("completed event was processed again")
	}
}

func TestServiceSerializesMessagesForSamePhoneNumber(t *testing.T) {
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var calls atomic.Int32

	store := &fakeStore{
		beginEventFn: func(context.Context, string) (bool, error) { return true, nil },
		appendFn:     func(context.Context, string, conversation.Message) error { return nil },
		recentFn: func(context.Context, string, int) ([]conversation.Message, error) {
			return []conversation.Message{{Role: "user", Content: "hello"}}, nil
		},
		completeEventFn: func(context.Context, string) error { return nil },
	}
	completer := &fakeCompleter{completeFn: func(context.Context, string, []conversation.Message) (string, error) {
		if calls.Add(1) == 1 {
			close(firstStarted)
			<-releaseFirst
		} else {
			close(secondStarted)
		}
		return "reply", nil
	}}
	sender := &fakeSender{
		markReadFn: func(context.Context, string) error { return nil },
		sendTextFn: func(context.Context, string, string) error { return nil },
	}
	service := NewService(store, completer, sender, "prompt", 30)
	errors := make(chan error, 2)

	go func() {
		errors <- service.Handle(context.Background(), whatsapp.InboundMessage{ID: "1", EventID: "1", PhoneNumber: "628123", Body: "first"})
	}()
	<-firstStarted
	go func() {
		errors <- service.Handle(context.Background(), whatsapp.InboundMessage{ID: "2", EventID: "2", PhoneNumber: "628123", Body: "second"})
	}()

	select {
	case <-secondStarted:
		t.Fatal("second message started before first completed")
	case <-time.After(50 * time.Millisecond):
	}
	close(releaseFirst)
	for range 2 {
		if err := <-errors; err != nil {
			t.Fatal(err)
		}
	}
}

func TestServiceIgnoresEmptyMessages(t *testing.T) {
	called := false
	store := &fakeStore{appendFn: func(context.Context, string, conversation.Message) error {
		called = true
		return nil
	}}
	service := NewService(store, &fakeCompleter{}, &fakeSender{}, "prompt", 30)

	if err := service.Handle(context.Background(), whatsapp.InboundMessage{PhoneNumber: "628123", Body: "   "}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if called {
		t.Fatal("empty message was stored")
	}
}

type fakeStore struct {
	beginEventFn    func(context.Context, string) (bool, error)
	appendFn        func(context.Context, string, conversation.Message) error
	recentFn        func(context.Context, string, int) ([]conversation.Message, error)
	completeEventFn func(context.Context, string) error
}

func (f *fakeStore) BeginEvent(ctx context.Context, eventID string) (bool, error) {
	return f.beginEventFn(ctx, eventID)
}

func (f *fakeStore) Append(ctx context.Context, phone string, message conversation.Message) error {
	return f.appendFn(ctx, phone, message)
}

func (f *fakeStore) Recent(ctx context.Context, phone string, limit int) ([]conversation.Message, error) {
	return f.recentFn(ctx, phone, limit)
}

func (f *fakeStore) CompleteEvent(ctx context.Context, eventID string) error {
	return f.completeEventFn(ctx, eventID)
}

type fakeCompleter struct {
	completeFn func(context.Context, string, []conversation.Message) (string, error)
}

func (f *fakeCompleter) Complete(ctx context.Context, prompt string, history []conversation.Message) (string, error) {
	return f.completeFn(ctx, prompt, history)
}

type fakeSender struct {
	sendTextFn func(context.Context, string, string) error
	markReadFn func(context.Context, string) error
}

func (f *fakeSender) SendText(ctx context.Context, phone, text string) error {
	return f.sendTextFn(ctx, phone, text)
}

func (f *fakeSender) MarkRead(ctx context.Context, id string) error {
	return f.markReadFn(ctx, id)
}
