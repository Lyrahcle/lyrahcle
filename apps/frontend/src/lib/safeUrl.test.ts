import { describe, expect, test } from 'vitest'
import { safeOtpauthUrl } from './safeUrl'

describe('safeOtpauthUrl', () => {
  test('accepts otpauth URLs', () => {
    const value = 'otpauth://totp/Lyrahcle:user@example.com?secret=ABC123&issuer=Lyrahcle'
    expect(safeOtpauthUrl(value)).toBe(value)
  })

  test.each([
    'javascript:alert(1)',
    'https://example.com/',
    'data:text/html,<script>alert(1)</script>',
    '//example.com/path',
    'not a url',
    '',
    null,
  ])('rejects unsafe value %j', (value) => {
    expect(safeOtpauthUrl(value)).toBeNull()
  })
})
