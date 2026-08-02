import type { ChatHistory, ChatMessage, ChatReply } from "@/types/chat"

export async function loadHistory(sessionID: string): Promise<ChatMessage[]> {
  const response = await fetch(
    `/api/chat/history?session_id=${encodeURIComponent(sessionID)}`
  )
  if (!response.ok) {
    throw new Error(`Chat history request failed (${response.status})`)
  }

  const history = (await response.json()) as ChatHistory
  return history.messages.map((message) => ({
    id: message.id,
    role: message.role,
    content: message.content,
    createdAt: new Date(message.created_at),
  }))
}

export async function sendMessage(
  sessionID: string,
  message: string
): Promise<ChatReply> {
  const response = await fetch("/api/chat", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ session_id: sessionID, message }),
  })

  if (!response.ok) {
    throw new Error(`Chat request failed (${response.status})`)
  }

  return (await response.json()) as ChatReply
}

export async function streamMessage(
  sessionID: string,
  message: string,
  onDelta: (delta: string) => void
): Promise<void> {
  const response = await fetch("/api/chat/stream", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ session_id: sessionID, message }),
  })

  if (!response.ok || !response.body) {
    throw new Error(`Chat stream failed (${response.status})`)
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ""

  const readEvent = async (): Promise<{
    delta?: string
    done?: boolean
    error?: string
  } | null> => {
    while (true) {
      const index = buffer.indexOf("\n\n")
      if (index !== -1) {
        const raw = buffer.slice(0, index)
        buffer = buffer.slice(index + 2)
        for (const line of raw.split("\n")) {
          if (!line.startsWith("data:")) {
            continue
          }
          const payload = line.slice(5).trim()
          if (payload === "[DONE]") {
            return { done: true }
          }
          try {
            return JSON.parse(payload)
          } catch {
            return null
          }
        }
        continue
      }

      const { done, value } = await reader.read()
      if (done) {
        return null
      }
      buffer += decoder.decode(value, { stream: true })
    }
  }

  while (true) {
    const event = await readEvent()
    if (event === null) {
      break
    }
    if (event.error) {
      throw new Error(event.error)
    }
    if (event.done) {
      return
    }
    if (event.delta) {
      onDelta(event.delta)
    }
  }
}
