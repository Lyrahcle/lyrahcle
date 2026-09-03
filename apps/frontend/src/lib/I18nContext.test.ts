import { describe, expect, it } from 'vitest'
import { englishTitleCase, formatTitleByLanguage } from './I18nContext'

describe('englishTitleCase', () => {
  it('capitalizes significant title words and preserves acronyms', () => {
    expect(englishTitleCase('register satellite')).toBe('Register Satellite')
    expect(englishTitleCase('pass prediction & orbital events')).toBe('Pass Prediction & Orbital Events')
    expect(englishTitleCase('SDR agent and API security')).toBe('SDR Agent and API Security')
  })
})


describe('formatTitleByLanguage', () => {
  it('uses English Title Case only for English', () => {
    expect(formatTitleByLanguage('en', 'remote config signing keys')).toBe('Remote Config Signing Keys')
  })

  it('uses sentence-style capitalization for Spanish and French titles', () => {
    expect(formatTitleByLanguage('es', 'Claves de Firma de Configuración Remota')).toBe('Claves de firma de configuración remota')
    expect(formatTitleByLanguage('fr', 'Plan de Contrôle')).toBe('Plan de contrôle')
  })

  it('does not rewrite German title capitalization', () => {
    expect(formatTitleByLanguage('de', 'Remote-Konfigurationsschlüssel')).toBe('Remote-Konfigurationsschlüssel')
  })

  it('preserves acronyms in Romance-language titles', () => {
    expect(formatTitleByLanguage('es', 'Monitor RF de API')).toBe('Monitor RF de API')
  })
})
