import type { ChatReply } from "@/types/chat"

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
