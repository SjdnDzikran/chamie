package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dzikran/chamie/internal/whatsapp"
)

func TestHandleInboundReturnsProcessingError(t *testing.T) {
	wantErr := errors.New("model unavailable")
	processor := &fakeProcessor{err: wantErr}
	message := whatsapp.InboundMessage{EventID: "event-1", PhoneNumber: "628123", Body: "hello"}

	err := handleInbound(context.Background(), processor, message, time.Second)

	if !errors.Is(err, wantErr) {
		t.Fatalf("handleInbound() error = %v, want %v", err, wantErr)
	}
	if !processor.called {
		t.Fatal("processor was not called before handleInbound returned")
	}
}

type fakeProcessor struct {
	called bool
	err    error
}

func (f *fakeProcessor) Handle(context.Context, whatsapp.InboundMessage) error {
	f.called = true
	return f.err
}
