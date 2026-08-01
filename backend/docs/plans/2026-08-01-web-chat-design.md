# Chamie Web Chat Design

**Goal:** Let users chat with Chamie through a web interface instead of WhatsApp. The web client reuses the existing tutor brain (history + DeepSeek) through a new HTTP endpoint, without Kapso or WhatsApp in the loop.

**Context:** The backend currently exposes only a Kapso webhook (`POST /webhooks/whatsapp`) and `GET /health`. Conversations are stored in PostgreSQL keyed by WhatsApp `phone_number` with `HISTORY_LIMIT` bounding the context window. The frontend is a fresh Vite + React 19 + shadcn/ui (base-rhea, Base UI, Hugeicons, Tailwind v4) starter.

**Approach summary:**
- Anonymous session: a `crypto.randomUUID()` stored in localStorage identifies a user.
- New `POST /api/chat` endpoint on the existing chamie server reuses `chat.Service`, `conversation.Store`, and `ai.Client`.
- Web conversations persist in the existing `conversation_messages` table under the key `web:<session_id>` (the `phone_number` column becomes a generic participant key).
- Frontend renders a full-height chat screen with shadcn `message` and `message-scroller` components.

---

## Backend

### `POST /api/chat`

Request:
```json
{ "session_id": "3f1c…", "message": "What is photosynthesis?" }
```

Response (200):
```json
{ "reply": "Photosynthesis is…" }
```

Errors:
- 400 for missing/empty `session_id` or `message`
- 500 for tutor failures (store, completion)

### Chat service web path

Add a method to `internal/chat/service.go` that:
1. Validates `sessionID` and `message` are non-empty.
2. Serializes per-session via the existing `phoneLocks` (keyed on `web:<sessionID>`).
3. Appends the user message (message ID `web:<sessionID>:<n>` for idempotency).
4. Loads `Recent(history)` and calls `completer.Complete` (system prompt + history).
5. Appends the assistant message.
6. Returns the reply string.

It does **not** use `BeginEvent`/`CompleteEvent` (webhook idempotency only) and does **not** call Kapso `SendText`.

### Wiring in `cmd/chamie/main.go`

Register `mux.HandleFunc("POST /api/chat", …)` with a JSON decoder/encoder and status-code mapping.

### Storage

No schema change. The existing `conversation_messages` table stores web messages under `phone_number = "web:<session_id>"`. The `HISTORY_LIMIT` still bounds the window.

### Tests

- `internal/chat/service_test.go`: table-driven tests for the web path — happy path (store + complete ordering), empty session/message errors, completer failure, empty completion.
- Follow existing test patterns (MemoryStore + fake Completer).

---

## Frontend

### Components to add (shadcn)

- `message` — chat bubble primitives: `MessageGroup`, `Message`, `MessageAvatar`, `MessageContent`, `MessageHeader`, `MessageFooter`
- `message-scroller` — auto-scrolling message list with jump-to-end button (adds `@shadcn/react`; fix the shipped `IconPlaceholder` import to a real Hugeicon)
- `input`, `textarea`, `skeleton`, `avatar`, `sonner` (error toasts), `button` (already present)

### Files

- `src/lib/session.ts` — `getOrCreateSessionID()`: reads `chamie.session` from localStorage, generates `crypto.randomUUID()` if absent.
- `src/lib/api.ts` — `sendMessage(sessionId, message): Promise<{ reply: string }>` via `POST /api/chat`.
- `src/App.tsx` — full chat screen:
  - Header: mascot avatar (`public/chamie-char.png`), "Chamie — English Tutor", dark-mode toggle
  - `MessageScroller` list: assistant left (mascot avatar), user right, timestamps in `MessageFooter`
  - Composer: `Textarea` + send `Button` (Enter sends, Shift+Enter newline), disabled + spinner while awaiting reply
  - Empty state: mascot + greeting
- `src/components/chat/` — split the screen into small components (`ChatHeader`, `ChatMessageList`, `ChatComposer`) if it grows.

### Config

- `vite.config.ts`: add `server.proxy` `/api` → `http://localhost:8080`.
- `index.html`: title `Chamie — English Tutor`, favicon → mascot.
- Copy `/mnt/c/Users/A S U S/Downloads/chamie-char.png` → `frontend/public/chamie-char.png`.

### State

Plain React state: `messages`, `isSending`, `error`. Transcript is authoritative server-side; the UI starts fresh each visit (backend already has full context via `Recent`).

---

## Verification

1. `go test ./...` (backend)
2. `go build ./cmd/chamie` (backend)
3. `bun run build` (frontend)
4. `bun run lint` (frontend)
5. Manual: `make run` in backend, `bun run dev` in frontend, send a message.
