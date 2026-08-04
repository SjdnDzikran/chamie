package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AIAPIKey           string
	AIBaseURL          string
	AIModel            string
	DatabaseURL        string
	HistoryLimit       int
	KapsoAPIKey        string
	KapsoBaseURL       string
	KapsoPhoneNumberID string
	KapsoWebhookSecret string
	HTTPAddr           string
	SystemPromptPath   string
	CORSAllowedOrigins []string
}

func Load() (*Config, error) {
	historyLimit, err := strconv.Atoi(envOr("HISTORY_LIMIT", "30"))
	if err != nil || historyLimit < 1 {
		return nil, fmt.Errorf("HISTORY_LIMIT must be a positive integer")
	}

	port := envOr("HTTP_PORT", "8080")
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	cfg := &Config{
		AIAPIKey:           strings.TrimSpace(os.Getenv("AI_API_KEY")),
		AIBaseURL:          strings.TrimRight(envOr("AI_BASE_URL", "https://api.deepseek.com"), "/"),
		AIModel:            envOr("AI_MODEL", "deepseek-chat"),
		DatabaseURL:        strings.TrimSpace(os.Getenv("DATABASE_URL")),
		HistoryLimit:       historyLimit,
		KapsoAPIKey:        strings.TrimSpace(os.Getenv("KAPSO_API_KEY")),
		KapsoBaseURL:       strings.TrimRight(envOr("KAPSO_BASE_URL", "https://api.kapso.ai/meta/whatsapp/v24.0"), "/"),
		KapsoPhoneNumberID: strings.TrimSpace(os.Getenv("KAPSO_PHONE_NUMBER_ID")),
		KapsoWebhookSecret: strings.TrimSpace(os.Getenv("KAPSO_WEBHOOK_SECRET")),
		HTTPAddr:           port,
		SystemPromptPath:   envOr("SYSTEM_PROMPT_PATH", "./prompts/system.md"),
		CORSAllowedOrigins: parseOrigins(os.Getenv("CORS_ALLOWED_ORIGINS")),
	}

	var missing []string
	for name, value := range map[string]string{
		"AI_API_KEY":   cfg.AIAPIKey,
		"DATABASE_URL": cfg.DatabaseURL,
	} {
		if value == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func parseOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			origins = append(origins, p)
		}
	}
	return origins
}
