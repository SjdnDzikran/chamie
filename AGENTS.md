# AGENTS.md

## Purpose

Chamie is a backend-only WhatsApp English tutor for Indonesian school students. It receives WhatsApp messages through Kapso, stores bounded conversation context in PostgreSQL, calls DeepSeek through an OpenAI-compatible API, and replies through Kapso.

## Scope

- Free-form tutoring, homework guidance, and conversation practice
- No frontend UI
- No tools, web search, escalation, Slack, workflow graph, profiles, or progress tracking
- `prompts/system.md` is externally owned content and should remain easy to replace

## Layout

- `cmd/chamie/main.go`: application wiring and HTTP server
- `internal/ai/`: OpenAI-compatible chat completion client
- `internal/chat/`: message orchestration
- `internal/config/`: environment configuration
- `internal/conversation/`: conversation store interface and implementations
- `internal/db/`: PostgreSQL connection and embedded migrations
- `internal/whatsapp/`: Kapso sender and webhook handler
- `prompts/system.md`: replaceable tutor instructions

## Commands

- `go test ./...`
- `go build ./cmd/chamie`
- `go run ./cmd/chamie`

## Rules

- Keep Kapso limited to WhatsApp transport; application state belongs in PostgreSQL.
- Keep the model integration OpenAI-compatible rather than DeepSeek-specific.
- Do not introduce an agent framework or tool loop without a concrete requirement.
- Preserve bounded context through `HISTORY_LIMIT`.
- Never commit `.env` or API credentials.
- Add tests before changing behavior.
