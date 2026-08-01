import { HugeiconsIcon } from "@hugeicons/react"
import { Moon01Icon, Sun01Icon } from "@hugeicons/core-free-icons"

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { useTheme } from "@/components/theme-provider"

export function ChatHeader() {
  const { theme, setTheme } = useTheme()

  return (
    <header className="flex h-16 shrink-0 items-center gap-3 border-b px-4">
      <Avatar size="lg">
        <AvatarImage src="/chamie-char.png" alt="Chamie mascot" />
        <AvatarFallback>Ch</AvatarFallback>
      </Avatar>
      <div className="flex min-w-0 flex-1 flex-col">
        <span className="truncate font-medium">Chamie</span>
        <span className="text-xs text-muted-foreground">English Tutor</span>
      </div>
      <Button
        variant="outline"
        size="icon"
        aria-label="Toggle dark mode"
        onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
      >
        <HugeiconsIcon
          icon={theme === "dark" ? Sun01Icon : Moon01Icon}
          strokeWidth={2}
        />
      </Button>
    </header>
  )
}
