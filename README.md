# Chamie

Chamie is a small WhatsApp AI tutor for Indonesian school students. Students can ask English questions, get help understanding homework, or practice through free-form conversation.

This repository contains only the backend. WhatsApp is the user interface.

## Architecture

```text
WhatsApp
  -> Kapso webhook
  -> Chamie Go service
  -> PostgreSQL recent conversation history
  -> DeepSeek OpenAI-compatible chat completion
  -> Kapso WhatsApp reply
```

Kapso is used only as the WhatsApp transport. Chamie owns its conversation data in PostgreSQL and calls DeepSeek directly. There are no agent tools, workflow engine, escalation flow, Slack integration, or frontend dashboard.

The MVP should run as a single application instance. Per-student message ordering is enforced in process; add a distributed event-claim mechanism before horizontally scaling the service.

## Stack

- Go 1.26
- Standard `net/http`
- PostgreSQL through `pgx/v5`
- DeepSeek through its OpenAI-compatible API
- Kapso WhatsApp API and webhooks

## Setup

1. Create a PostgreSQL database.
2. Copy the example environment file:

```bash
cp .env.example .env
```

3. Fill in `AI_API_KEY`, `DATABASE_URL`, `KAPSO_API_KEY`, `KAPSO_PHONE_NUMBER_ID`, and `KAPSO_WEBHOOK_SECRET`.
4. Replace `prompts/system.md` with the reviewed Chamie prompt before production use.
5. Run the service:

```bash
go run ./cmd/chamie
```

The service applies its database migration automatically at startup.

## Environment

| Variable | Required | Default | Purpose |
|---|---:|---|---|
| `AI_API_KEY` | Yes | - | DeepSeek API key |
| `AI_BASE_URL` | No | `https://api.deepseek.com` | OpenAI-compatible API base URL |
| `AI_MODEL` | No | `deepseek-chat` | Chat completion model |
| `DATABASE_URL` | Yes | - | PostgreSQL connection URL |
| `HISTORY_LIMIT` | No | `30` | Recent messages sent to the model |
| `KAPSO_API_KEY` | Yes | - | Kapso project API key |
| `KAPSO_PHONE_NUMBER_ID` | Yes | - | WhatsApp phone number ID |
| `KAPSO_WEBHOOK_SECRET` | Yes | - | HMAC webhook signature secret |
| `KAPSO_BASE_URL` | No | `https://api.kapso.ai/meta/whatsapp/v24.0` | Kapso WhatsApp API base URL |
| `HTTP_PORT` | No | `8080` | HTTP listen port |
| `SYSTEM_PROMPT_PATH` | No | `./prompts/system.md` | External tutor prompt file |

## Kapso Webhook

Configure Kapso to send message events to:

```text
https://your-domain.example/webhooks/whatsapp
```

The endpoint verifies Kapso's HMAC-SHA256 signature, accepts inbound text messages, and uses `X-Idempotency-Key` to avoid processing completed deliveries twice. Other message types and outbound message events are ignored. Processing completes before the webhook is acknowledged so Kapso can retry transient failures. `GET /health` provides a basic process health check.

## Development

```bash
go test ./...
go build ./cmd/chamie
```

## Data

Conversation messages are stored in `conversation_messages`, while `webhook_events` tracks completed Kapso deliveries. The model receives only the latest `HISTORY_LIMIT` messages, in chronological order. There are no student profiles or progress-tracking records in the MVP.

## License

[MIT](LICENSE)
