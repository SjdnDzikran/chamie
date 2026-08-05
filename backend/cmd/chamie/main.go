package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/dzikran/chamie/internal/ai"
	"github.com/dzikran/chamie/internal/chat"
	"github.com/dzikran/chamie/internal/config"
	"github.com/dzikran/chamie/internal/conversation"
	"github.com/dzikran/chamie/internal/cors"
	"github.com/dzikran/chamie/internal/db"
	"github.com/dzikran/chamie/internal/whatsapp"
)

func main() {
	if err := run(); err != nil {
		slog.Error("chamie stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	systemPrompt, err := os.ReadFile(cfg.SystemPromptPath)
	if err != nil {
		return fmt.Errorf("read system prompt: %w", err)
	}
	if strings.TrimSpace(string(systemPrompt)) == "" {
		return fmt.Errorf("system prompt is empty: %s", cfg.SystemPromptPath)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	store := conversation.NewPostgresStore(pool)
	model := ai.NewClient(cfg.AIBaseURL, cfg.AIAPIKey, cfg.AIModel, nil)

	var sender whatsapp.Sender
	if cfg.KapsoAPIKey != "" && cfg.KapsoPhoneNumberID != "" && cfg.KapsoWebhookSecret != "" {
		sender = whatsapp.NewKapsoSender(cfg.KapsoBaseURL, cfg.KapsoAPIKey, cfg.KapsoPhoneNumberID, nil)
	}
	tutor := chat.NewService(store, model, sender, strings.TrimSpace(string(systemPrompt)), cfg.HistoryLimit)

	mux := http.NewServeMux()
	if sender != nil {
		mux.Handle("POST /webhooks/whatsapp", whatsapp.NewWebhookHandler(cfg.KapsoWebhookSecret,
			func(requestCtx context.Context, message whatsapp.InboundMessage) error {
				err := handleInbound(requestCtx, tutor, message, 80*time.Second)
				if err != nil {
					slog.Error("process WhatsApp message", "message_id", message.ID, "phone", message.PhoneNumber, "error", err)
				}
				return err
			},
		))
	}
	mux.Handle("POST /api/chat", chat.NewWebChatHandler(tutor))
	mux.Handle("POST /api/chat/stream", chat.NewWebChatStreamHandler(tutor))
	mux.Handle("GET /api/chat/history", chat.NewWebChatHistoryHandler(tutor))
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	handler := cors.Middleware(cfg.CORSAllowedOrigins)(mux)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("Chamie listening", "address", cfg.HTTPAddr, "model", cfg.AIModel)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	return nil
}

type messageProcessor interface {
	Handle(ctx context.Context, message whatsapp.InboundMessage) error
}

func handleInbound(ctx context.Context, processor messageProcessor, message whatsapp.InboundMessage, timeout time.Duration) error {
	messageCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return processor.Handle(messageCtx, message)
}
