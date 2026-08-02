import { ChatComposer } from "@/components/chat/chat-composer"
import { ChatHeader } from "@/components/chat/chat-header"
import { ChatTranscript } from "@/components/chat/chat-transcript"
import { useChat } from "@/hooks/useChat"

export function ChatPage() {
  const { messages, isLoading, isSending, send } = useChat()

  return (
    <div className="flex h-svh justify-center">
      <div className="flex h-full w-full max-w-3xl flex-col">
        <ChatHeader />
        <ChatTranscript messages={messages} isSending={isSending} />
        <ChatComposer isSending={isLoading || isSending} onSend={send} />
      </div>
    </div>
  )
}
