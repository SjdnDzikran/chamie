import { ChatComposer } from "@/components/chat/chat-composer"
import { ChatHeader } from "@/components/chat/chat-header"
import { ChatTranscript } from "@/components/chat/chat-transcript"
import { useChat } from "@/hooks/useChat"

export function ChatPage() {
  const { messages, isSending, send } = useChat()

  return (
    <div className="flex h-svh flex-col">
      <ChatHeader />
      <ChatTranscript messages={messages} isSending={isSending} />
      <ChatComposer isSending={isSending} onSend={send} />
    </div>
  )
}
