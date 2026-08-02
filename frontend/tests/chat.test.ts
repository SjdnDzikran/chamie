import { afterEach, expect, test } from "bun:test"

import { loadHistory } from "../src/services/chat"

const originalFetch = globalThis.fetch

afterEach(() => {
  globalThis.fetch = originalFetch
})

test("loadHistory requests the session and converts persisted timestamps", async () => {
  globalThis.fetch = (async (input) => {
    expect(input).toBe("/api/chat/history?session_id=session-1")
    return new Response(
      JSON.stringify({
        messages: [
          {
            id: "message-1",
            role: "user",
            content: "hello",
            created_at: "2026-08-02T10:00:00Z",
          },
        ],
      }),
      { status: 200, headers: { "Content-Type": "application/json" } }
    )
  }) as typeof fetch

  const messages = await loadHistory("session-1")

  expect(messages).toEqual([
    {
      id: "message-1",
      role: "user",
      content: "hello",
      createdAt: new Date("2026-08-02T10:00:00Z"),
    },
  ])
})
