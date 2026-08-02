import { useCallback, useEffect, useState } from "react"
import { toast } from "sonner"

import { getOrCreateSessionID } from "@/lib/session"
import { loadHistory, streamMessage } from "@/services/chat"
import type { ChatMessage } from "@/types/chat"

export function useChat() {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [isSending, setIsSending] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const [sessionID] = useState(getOrCreateSessionID)

  useEffect(() => {
    let cancelled = false
    loadHistory(sessionID)
      .then((history) => {
        if (!cancelled) {
          setMessages(history)
        }
      })
      .catch(() => {
        if (!cancelled) {
          toast.error("Failed to load chat history.")
        }
      })
      .finally(() => {
        if (!cancelled) {
          setIsLoading(false)
        }
      })
    return () => {
      cancelled = true
    }
  }, [sessionID])

  const append = useCallback(
    (message: Omit<ChatMessage, "id" | "createdAt">) => {
      setMessages((current) => [
        ...current,
        { ...message, id: crypto.randomUUID(), createdAt: new Date() },
      ])
    },
    []
  )

  const updateAssistant = useCallback(
    (id: string, updater: (current: string) => string) => {
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

      let assistantID: string | null = null
      try {
        await streamMessage(sessionID, content, (delta) => {
          if (assistantID === null) {
            assistantID = crypto.randomUUID()
            setMessages((current) => [
              ...current,
              {
                id: assistantID as string,
                role: "assistant",
                content: "",
                createdAt: new Date(),
              },
            ])
          }
          updateAssistant(assistantID as string, (previous) => previous + delta)
        })
      } catch {
        toast.error("Failed to get a reply. Please try again.")
      } finally {
        setIsSending(false)
      }
    },
    [append, sessionID, updateAssistant]
  )

  return { messages, isLoading, isSending, send }
}
