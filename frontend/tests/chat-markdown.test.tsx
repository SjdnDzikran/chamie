import { expect, test } from "bun:test"
import { renderToStaticMarkup } from "react-dom/server"

import { ChatMarkdown } from "../src/components/chat/chat-markdown"

test("renders markdown with styled elements", () => {
  const html = renderToStaticMarkup(
    <ChatMarkdown>
      {"**hello** and `inline`\n\n- one\n- two\n\n```js\nconst x = 1\n```"}
    </ChatMarkdown>
  )

  expect(html).toContain("<strong")
  expect(html).toContain("hello")
  expect(html).toContain("<code")
  expect(html).toContain("inline")
  expect(html).toContain("<ul")
  expect(html).toContain("<li")
  expect(html).toContain("one")
  expect(html).toContain("<pre")
  expect(html).toContain("const x = 1")
})

test("escapes raw HTML instead of rendering it", () => {
  const html = renderToStaticMarkup(
    <ChatMarkdown>{'<script>alert("x")</script>'}</ChatMarkdown>
  )

  expect(html).not.toContain("<script>")
  expect(html).toContain("alert")
})

test("opens links in a new tab", () => {
  const html = renderToStaticMarkup(
    <ChatMarkdown>{"[Chamie](https://example.com)"}</ChatMarkdown>
  )

  expect(html).toContain('href="https://example.com"')
  expect(html).toContain('target="_blank"')
  expect(html).toContain('rel="noreferrer"')
})
