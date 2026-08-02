export type ChatMessage = {
  id: string
  role: "user" | "assistant"
  content: string
  createdAt: Date
}

export type ChatReply = {
  reply: string
}

export type ChatHistory = {
  messages: Array<{
    id: string
    role: "user" | "assistant"
    content: string
    created_at: string
  }>
}
