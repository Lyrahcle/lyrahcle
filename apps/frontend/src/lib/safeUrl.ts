export function safeOtpauthUrl(value: unknown): string | null {
  if (typeof value !== 'string' || value.length === 0) return null
  try {
    const url = new URL(value)
    return url.protocol === 'otpauth:' ? value : null
  } catch {
    return null
  }
}
