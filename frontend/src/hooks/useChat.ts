import { useCallback, useRef, useState } from "react"
import { toast } from "sonner"

import { getOrCreateSessionID } from "@/lib/session"
import { sendMessage } from "@/services/chat"
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

  const send = useCallback(
    async (content: string) => {
      append({ role: "user", content })
      setIsSending(true)

      try {
        const { reply } = await sendMessage(getOrCreateSessionID(), content)
        append({ role: "assistant", content: reply })
      } catch {
        toast.error("Failed to get a reply. Please try again.")
      } finally {
        setIsSending(false)
      }
    },
    [append]
  )

  return { messages, isSending, send }
}
