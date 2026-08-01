import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Bubble, BubbleContent } from "@/components/ui/bubble"
import {
  Message,
  MessageAvatar,
  MessageContent,
  MessageHeader,
} from "@/components/ui/message"
import {
  MessageScroller,
  MessageScrollerButton,
  MessageScrollerContent,
  MessageScrollerItem,
  MessageScrollerProvider,
  MessageScrollerViewport,
} from "@/components/ui/message-scroller"
import { Marker, MarkerContent, MarkerIcon } from "@/components/ui/marker"
import { Spinner } from "@/components/ui/spinner"
import type { ChatMessage } from "@/types/chat"

function ChamieAvatar() {
  return (
    <Avatar>
      <AvatarImage src="/chamie-char.png" alt="Chamie mascot" />
      <AvatarFallback>Ch</AvatarFallback>
    </Avatar>
  )
}

function MessageRow({ message }: { message: ChatMessage }) {
  const isUser = message.role === "user"

  return (
    <Message align={isUser ? "end" : "start"}>
      {!isUser && (
        <MessageAvatar>
          <ChamieAvatar />
        </MessageAvatar>
      )}
      <MessageContent>
        {!isUser && <MessageHeader>Chamie</MessageHeader>}
        <Bubble variant={isUser ? "default" : "muted"}>
          <BubbleContent className="whitespace-pre-wrap">
            {message.content}
          </BubbleContent>
        </Bubble>
      </MessageContent>
    </Message>
  )
}

export function ChatTranscript({
  messages,
  isSending,
}: {
  messages: ChatMessage[]
  isSending: boolean
}) {
  const lastMessage = messages[messages.length - 1]
  const showThinking =
    isSending && (!lastMessage || lastMessage.role === "user")

  return (
    <MessageScrollerProvider>
      <MessageScroller className="flex-1">
        <MessageScrollerViewport>
          <MessageScrollerContent className="gap-6 p-4">
            {messages.map((message) => (
              <MessageScrollerItem
                key={message.id}
                scrollAnchor={message.role === "user"}
              >
                <MessageRow message={message} />
              </MessageScrollerItem>
            ))}
            {showThinking && (
              <MessageScrollerItem scrollAnchor={false}>
                <Marker role="status">
                  <MarkerIcon>
                    <Spinner />
                  </MarkerIcon>
                  <MarkerContent>Chamie is thinking…</MarkerContent>
                </Marker>
              </MessageScrollerItem>
            )}
          </MessageScrollerContent>
        </MessageScrollerViewport>
        <MessageScrollerButton />
      </MessageScroller>
    </MessageScrollerProvider>
  )
}
