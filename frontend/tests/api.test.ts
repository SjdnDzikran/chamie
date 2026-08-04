import { expect, test } from "bun:test"

import { apiUrl } from "../src/lib/api"

test("defaults to a relative API path", () => {
  delete process.env.VITE_API_BASE_URL
  expect(apiUrl("/api/chat/history?session_id=x")).toBe(
    "/api/chat/history?session_id=x"
  )
})

test("prefixes an absolute API base URL", () => {
  process.env.VITE_API_BASE_URL = "https://api.example.com"
  expect(apiUrl("/api/chat")).toBe("https://api.example.com/api/chat")
  delete process.env.VITE_API_BASE_URL
})

test("strips a trailing slash from the base URL", () => {
  process.env.VITE_API_BASE_URL = "https://api.example.com/"
  expect(apiUrl("/api/chat")).toBe("https://api.example.com/api/chat")
  delete process.env.VITE_API_BASE_URL
})
