const SESSION_KEY = "chamie.session"

export function getOrCreateSessionID(): string {
  const existing = localStorage.getItem(SESSION_KEY)
  if (existing) {
    return existing
  }
  const sessionID = crypto.randomUUID()
  localStorage.setItem(SESSION_KEY, sessionID)
  return sessionID
}
