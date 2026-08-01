import { useCallback, useRef, useState } from "react"
import { toast } from "sonner"

import { getOrCreateSessionID } from "@/lib/session"
import { streamMessage } from "@/services/chat"
import type { ChatMessage } from "@/types/chat"

export function useChat() {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [isSending, setIsSending] = useState(false)
  const nextID = useRef(1)

  const append = useCallback(
    (message: Omit<ChatMessage, "id" | "createdAt">) => {
      setMessages((current) => [
        ...current,
        { ...message, id: nextID.current++, createdAt: new Date() },
      ])
    },
    []
  )

  const updateAssistant = useCallback(
    (id: number, updater: (current: string) => string) => {
      setMessages((current) =>
        current.map((message) =>
          message.id === id
            ? { ...message, content: updater(message.content) }
            : message
        )
      )
    },
    []
  )

  const send = useCallback(
    async (content: string) => {
      append({ role: "user", content })
      setIsSending(true)

      let assistantID: number | null = null
      try {
        await streamMessage(getOrCreateSessionID(), content, (delta) => {
          if (assistantID === null) {
            assistantID = nextID.current++
            setMessages((current) => [
              ...current,
              {
                id: assistantID as number,
                role: "assistant",
                content: "",
                createdAt: new Date(),
              },
            ])
          }
          updateAssistant(assistantID as number, (previous) => previous + delta)
        })
      } catch {
        toast.error("Failed to get a reply. Please try again.")
      } finally {
        setIsSending(false)
      }
    },
    [append, updateAssistant]
  )

  return { messages, isSending, send }
}
