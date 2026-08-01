# Chamie MVP Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a minimal WhatsApp English tutor backend that receives messages through Kapso, remembers a bounded conversation history in PostgreSQL, and replies through DeepSeek's OpenAI-compatible API.

**Architecture:** A long-running Go HTTP service exposes an authenticated Kapso webhook and health endpoint. The orchestrator deduplicates completed webhook events, serializes messages per phone number, stores each inbound message, loads the latest configured number of messages, requests one chat completion, and replies through Kapso. Prompt content remains in an external file so another contributor can own the tutor's personality and teaching instructions.

**Tech Stack:** Go 1.26, standard `net/http`, PostgreSQL via `pgx/v5`, DeepSeek through its OpenAI-compatible HTTP API, Kapso WhatsApp API.

---

### Task 1: Replace the JavaScript scaffold

**Files:**
- Delete: Kapso workflow scaffold files under `functions/`, `workflows/`, `scripts/`, `src/`, and `tests/`
- Create: `go.mod`
- Create: `.env.example`

1. Remove workflow-platform files that Chamie will not use.
2. Initialize a small Go module with only `pgx/v5` and `godotenv`.
3. Define required DeepSeek, PostgreSQL, Kapso, HTTP, history-limit, and prompt-path variables.

### Task 2: Configuration

**Files:**
- Test: `internal/config/config_test.go`
- Create: `internal/config/config.go`

1. Write failing tests for defaults, required values, URL normalization, and history limits.
2. Continue with config implementation after confirming the required cases.
3. Implement environment loading and validation.
4. Run the package tests and confirm they pass.

### Task 3: Bounded conversation storage

**Files:**
- Test: `internal/conversation/store_test.go`
- Create: `internal/conversation/store.go`
- Create: `internal/db/db.go`
- Create: `db/migrations/000001_conversations.up.sql`
- Create: `db/migrations/000001_conversations.down.sql`

1. Define a small store interface around appending and retrieving recent messages.
2. Write failing tests for ordering and history truncation using an in-memory implementation.
3. Implement the in-memory store for tests and local development.
4. Add a PostgreSQL implementation using `conversation_messages` plus completed webhook-event tracking.
5. Verify package tests.

### Task 4: OpenAI-compatible chat client

**Files:**
- Test: `internal/ai/client_test.go`
- Create: `internal/ai/client.go`

1. Write failing HTTP contract tests for URL, authorization, model, system prompt, history, response parsing, and API errors.
2. Implement the minimal `/chat/completions` client.
3. Verify package tests.

### Task 5: Kapso gateway

**Files:**
- Test: `internal/whatsapp/kapso_test.go`
- Create: `internal/whatsapp/kapso.go`

1. Write failing tests for outbound message payloads, inbound text parsing, idempotency keys, ignored events, and mandatory HMAC verification.
2. Implement the Kapso sender and webhook handler.
3. Verify package tests.

### Task 6: Message orchestration and server wiring

**Files:**
- Test: `internal/chat/service_test.go`
- Create: `internal/chat/service.go`
- Create: `cmd/chamie/main.go`
- Create: `prompts/system.md`

1. Write failing service tests for save-load-complete-send-save ordering, completed-event deduplication, same-phone serialization, and empty messages.
2. Implement the application service with dependency interfaces.
3. Wire PostgreSQL, DeepSeek, Kapso, webhook, health checks, and graceful shutdown.
4. Add a deliberately minimal placeholder prompt that can be replaced independently.

### Task 7: Documentation and verification

**Files:**
- Modify: `README.md`
- Modify: `AGENTS.md`

1. Document architecture, setup, environment variables, webhook URL, and local commands.
2. Run `gofmt -w .`.
3. Start the service with `make run`.
5. Inspect `git diff --check` and `git status --short`.
