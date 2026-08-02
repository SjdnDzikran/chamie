import type { ComponentProps } from "react"
import Markdown from "react-markdown"
import remarkGfm from "remark-gfm"

import { cn } from "@/lib/utils"

const components = {
  p: ({ className, ...props }: ComponentProps<"p">) => (
    <p className={cn("my-2 first:mt-0 last:mb-0", className)} {...props} />
  ),
  strong: ({ className, ...props }: ComponentProps<"strong">) => (
    <strong className={cn("font-semibold", className)} {...props} />
  ),
  em: ({ className, ...props }: ComponentProps<"em">) => (
    <em className={cn("italic", className)} {...props} />
  ),
  h1: ({ className, ...props }: ComponentProps<"h1">) => (
    <h1
      className={cn(
        "my-2 text-base font-semibold first:mt-0 last:mb-0",
        className
      )}
      {...props}
    />
  ),
  h2: ({ className, ...props }: ComponentProps<"h2">) => (
    <h2
      className={cn(
        "my-2 text-base font-semibold first:mt-0 last:mb-0",
        className
      )}
      {...props}
    />
  ),
  h3: ({ className, ...props }: ComponentProps<"h3">) => (
    <h3
      className={cn(
        "my-2 text-sm font-semibold first:mt-0 last:mb-0",
        className
      )}
      {...props}
    />
  ),
  ul: ({ className, ...props }: ComponentProps<"ul">) => (
    <ul
      className={cn("my-2 list-disc pl-5 first:mt-0 last:mb-0", className)}
      {...props}
    />
  ),
  ol: ({ className, ...props }: ComponentProps<"ol">) => (
    <ol
      className={cn("my-2 list-decimal pl-5 first:mt-0 last:mb-0", className)}
      {...props}
    />
  ),
  li: ({ className, ...props }: ComponentProps<"li">) => (
    <li className={cn("my-1", className)} {...props} />
  ),
  blockquote: ({ className, ...props }: ComponentProps<"blockquote">) => (
    <blockquote
      className={cn(
        "my-2 border-l-2 border-border pl-3 text-muted-foreground first:mt-0 last:mb-0",
        className
      )}
      {...props}
    />
  ),
  code: ({ className, ...props }: ComponentProps<"code">) => {
    const isBlock = className?.includes("language-")
    return (
      <code
        className={
          isBlock
            ? cn("font-mono", className)
            : cn(
                "rounded-md bg-primary/10 px-1 py-0.5 font-mono text-[0.9em]",
                className
              )
        }
        {...props}
      />
    )
  },
  pre: ({ className, ...props }: ComponentProps<"pre">) => (
    <pre
      className={cn(
        "my-2 overflow-x-auto rounded-xl bg-muted p-3 font-mono text-sm first:mt-0 last:mb-0",
        className
      )}
      {...props}
    />
  ),
  a: ({ className, href, children, ...props }: ComponentProps<"a">) => (
    <a
      className={cn(
        "font-medium text-primary underline underline-offset-2",
        className
      )}
      href={href}
      target="_blank"
      rel="noreferrer"
      {...props}
    >
      {children}
    </a>
  ),
  hr: ({ className, ...props }: ComponentProps<"hr">) => (
    <hr className={cn("my-3 border-border", className)} {...props} />
  ),
}

export function ChatMarkdown({ children }: { children: string }) {
  return (
    <div className="wrap-break-word">
      <Markdown remarkPlugins={[remarkGfm]} components={components}>
        {children}
      </Markdown>
    </div>
  )
}
