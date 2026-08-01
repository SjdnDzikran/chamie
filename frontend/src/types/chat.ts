export type ChatMessage = {
  id: number
  role: "user" | "assistant"
  content: string
  createdAt: Date
}

export type ChatReply = {
  reply: string
}
