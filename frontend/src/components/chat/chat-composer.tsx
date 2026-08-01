import { useState } from "react"
import { HugeiconsIcon } from "@hugeicons/react"
import { Airplane02Icon } from "@hugeicons/core-free-icons"

import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupTextarea,
} from "@/components/ui/input-group"
import { Spinner } from "@/components/ui/spinner"

export function ChatComposer({
  isSending,
  onSend,
}: {
  isSending: boolean
  onSend: (message: string) => void
}) {
  const [draft, setDraft] = useState("")

  const submit = () => {
    const text = draft.trim()
    if (!text || isSending) {
      return
    }
    onSend(text)
    setDraft("")
  }

  return (
    <footer className="shrink-0 p-4">
      <InputGroup>
        <InputGroupTextarea
          className="min-h-10"
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter" && !event.shiftKey) {
              event.preventDefault()
              submit()
            }
          }}
          placeholder="Ask Chamie something…"
          aria-label="Message"
          disabled={isSending}
        />
        <InputGroupAddon align="block-end" className="p-2">
          <InputGroupButton
            type="submit"
            size="icon-sm"
            onClick={submit}
            disabled={isSending || draft.trim() === ""}
            aria-label="Send message"
          >
            {isSending ? (
              <Spinner />
            ) : (
              <HugeiconsIcon icon={Airplane02Icon} strokeWidth={2} />
            )}
          </InputGroupButton>
        </InputGroupAddon>
      </InputGroup>
    </footer>
  )
}
