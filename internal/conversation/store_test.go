package conversation

import (
	"context"
	"testing"
	"time"
)

func TestMemoryStoreReturnsOnlyMostRecentMessagesInOrder(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	for i, content := range []string{"one", "two", "three", "four"} {
		err := store.Append(ctx, "628123", Message{
			ID:        content,
			Role:      "user",
			Content:   content,
			CreatedAt: time.Unix(int64(i+1), 0),
		})
		if err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	got, err := store.Recent(ctx, "628123", 3)
	if err != nil {
		t.Fatalf("Recent() error = %v", err)
	}

	want := []string{"two", "three", "four"}
	if len(got) != len(want) {
		t.Fatalf("len(Recent()) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Content != want[i] {
			t.Errorf("Recent()[%d].Content = %q, want %q", i, got[i].Content, want[i])
		}
	}
}

func TestMemoryStoreSeparatesPhoneNumbers(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	if err := store.Append(ctx, "first", Message{ID: "first-1", Role: "user", Content: "hello"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(ctx, "second", Message{ID: "second-1", Role: "user", Content: "halo"}); err != nil {
		t.Fatal(err)
	}

	got, err := store.Recent(ctx, "first", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Content != "hello" {
		t.Fatalf("Recent() = %#v", got)
	}
}

func TestMemoryStoreRejectsInvalidMessages(t *testing.T) {
	store := NewMemoryStore()

	for _, msg := range []Message{
		{ID: "1", Role: "", Content: "hello"},
		{ID: "2", Role: "user", Content: ""},
		{ID: "3", Role: "tool", Content: "hello"},
		{ID: "", Role: "user", Content: "hello"},
	} {
		if err := store.Append(context.Background(), "628123", msg); err == nil {
			t.Errorf("Append(%#v) error = nil", msg)
		}
	}
}

func TestMemoryStoreDoesNotAppendDuplicateMessageIDs(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	message := Message{ID: "wamid.1:user", Role: "user", Content: "hello"}

	if err := store.Append(ctx, "628123", message); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(ctx, "628123", message); err != nil {
		t.Fatal(err)
	}
	history, err := store.Recent(ctx, "628123", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 {
		t.Fatalf("len(history) = %d, want 1", len(history))
	}
}

func TestMemoryStoreTracksCompletedWebhookEvents(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	process, err := store.BeginEvent(ctx, "event-1")
	if err != nil || !process {
		t.Fatalf("BeginEvent() = %v, %v; want true, nil", process, err)
	}
	process, err = store.BeginEvent(ctx, "event-1")
	if err != nil || !process {
		t.Fatalf("pending BeginEvent() = %v, %v; want true, nil", process, err)
	}
	if err := store.CompleteEvent(ctx, "event-1"); err != nil {
		t.Fatal(err)
	}
	process, err = store.BeginEvent(ctx, "event-1")
	if err != nil || process {
		t.Fatalf("completed BeginEvent() = %v, %v; want false, nil", process, err)
	}
}
