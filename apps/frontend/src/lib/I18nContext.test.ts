import { describe, expect, it } from 'vitest'
import { englishTitleCase } from './I18nContext'

describe('englishTitleCase', () => {
  it('capitalizes significant title words and preserves acronyms', () => {
    expect(englishTitleCase('register satellite')).toBe('Register Satellite')
    expect(englishTitleCase('pass prediction & orbital events')).toBe('Pass Prediction & Orbital Events')
    expect(englishTitleCase('SDR agent and API security')).toBe('SDR Agent and API Security')
  })
})
