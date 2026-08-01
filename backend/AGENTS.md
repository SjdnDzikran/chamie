# AGENTS.md

## Purpose

Chamie is a WhatsApp English tutor backend for Indonesian school students. It receives WhatsApp messages through Kapso, stores bounded conversation context in PostgreSQL, calls DeepSeek through an OpenAI-compatible API, and replies through Kapso.

## Scope

- Free-form tutoring, homework guidance, and conversation practice
- No tools, web search, escalation, Slack, workflow graph, profiles, or progress tracking
- `prompts/system.md` is externally owned content and should remain easy to replace

## Layout

- `cmd/chamie/main.go`: application wiring and HTTP server
- `internal/ai/`: OpenAI-compatible chat completion client
- `internal/chat/`: message orchestration
- `internal/config/`: environment configuration
- `internal/conversation/`: conversation store interface and implementations
- `internal/db/`: PostgreSQL connection and migration runner
- `internal/whatsapp/`: Kapso sender and webhook handler
- `db/migrations/`: versioned PostgreSQL migrations
- `cmd/migrateup/main.go`: migration up/down command
- `prompts/system.md`: replaceable tutor instructions

## Command

- `make run`

## Rules

- Keep Kapso limited to WhatsApp transport; application state belongs in PostgreSQL.
- Keep the model integration OpenAI-compatible rather than DeepSeek-specific.
- Run migrations explicitly through `make migrate-up`; do not embed schema execution in the API process.
- Do not introduce an agent framework or tool loop without a concrete requirement.
- Preserve bounded context through `HISTORY_LIMIT`.
- Never commit `.env` or API credentials.
- Add tests before changing behavior.
