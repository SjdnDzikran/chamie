package config

import (
	"strings"
	"testing"
)

func TestLoadRequiresCoreCredentials(t *testing.T) {
	t.Setenv("AI_API_KEY", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("KAPSO_API_KEY", "")
	t.Setenv("KAPSO_PHONE_NUMBER_ID", "")
	t.Setenv("KAPSO_WEBHOOK_SECRET", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected missing configuration error")
	}

	for _, name := range []string{"AI_API_KEY", "DATABASE_URL"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not mention %s", err, name)
		}
	}
}

func TestLoadAppliesDeepSeekDefaultsAndNormalizesURLs(t *testing.T) {
	t.Setenv("AI_API_KEY", "deepseek-key")
	t.Setenv("AI_BASE_URL", "https://example.ai/v1/")
	t.Setenv("AI_MODEL", "")
	t.Setenv("DATABASE_URL", "postgres://db/chamie")
	t.Setenv("HISTORY_LIMIT", "")
	t.Setenv("KAPSO_API_KEY", "kapso-key")
	t.Setenv("KAPSO_PHONE_NUMBER_ID", "phone-id")
	t.Setenv("KAPSO_WEBHOOK_SECRET", "webhook-secret")
	t.Setenv("KAPSO_BASE_URL", "https://example.kapso/v24.0/")
	t.Setenv("HTTP_PORT", "9000")
	t.Setenv("SYSTEM_PROMPT_PATH", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.AIBaseURL != "https://example.ai/v1" {
		t.Errorf("AIBaseURL = %q", cfg.AIBaseURL)
	}
	if cfg.AIModel != "deepseek-chat" {
		t.Errorf("AIModel = %q", cfg.AIModel)
	}
	if cfg.HistoryLimit != 30 {
		t.Errorf("HistoryLimit = %d", cfg.HistoryLimit)
	}
	if cfg.KapsoBaseURL != "https://example.kapso/v24.0" {
		t.Errorf("KapsoBaseURL = %q", cfg.KapsoBaseURL)
	}
	if cfg.HTTPAddr != ":9000" {
		t.Errorf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.SystemPromptPath != "./prompts/system.md" {
		t.Errorf("SystemPromptPath = %q", cfg.SystemPromptPath)
	}
}

func TestLoadSucceedsWithoutKapso(t *testing.T) {
	t.Setenv("AI_API_KEY", "deepseek-key")
	t.Setenv("DATABASE_URL", "postgres://db/chamie")
	t.Setenv("KAPSO_API_KEY", "")
	t.Setenv("KAPSO_PHONE_NUMBER_ID", "")
	t.Setenv("KAPSO_WEBHOOK_SECRET", "")
	t.Setenv("HISTORY_LIMIT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.KapsoAPIKey != "" || cfg.KapsoPhoneNumberID != "" || cfg.KapsoWebhookSecret != "" {
		t.Errorf("Kapso fields should be empty, got %#v", cfg)
	}
}

func TestLoadRejectsInvalidHistoryLimit(t *testing.T) {
	t.Setenv("AI_API_KEY", "deepseek-key")
	t.Setenv("DATABASE_URL", "postgres://db/chamie")
	t.Setenv("KAPSO_API_KEY", "kapso-key")
	t.Setenv("KAPSO_PHONE_NUMBER_ID", "phone-id")
	t.Setenv("HISTORY_LIMIT", "0")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "HISTORY_LIMIT") {
		t.Fatalf("Load() error = %v, want HISTORY_LIMIT validation", err)
	}
}
