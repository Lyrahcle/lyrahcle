import '@testing-library/jest-dom'
import { afterEach, beforeEach, expect, vi } from 'vitest'
import { toHaveNoViolations } from 'jest-axe'

expect.extend(toHaveNoViolations)

let unexpectedConsole = []
let warnSpy
let errorSpy

beforeEach(() => {
  unexpectedConsole = []
  warnSpy = vi.spyOn(console, 'warn').mockImplementation((...args) => {
    unexpectedConsole.push({ level: 'warn', args })
  })
  errorSpy = vi.spyOn(console, 'error').mockImplementation((...args) => {
    unexpectedConsole.push({ level: 'error', args })
  })
})

afterEach(() => {
  warnSpy?.mockRestore()
  errorSpy?.mockRestore()
  if (!unexpectedConsole.length) return

  const messages = unexpectedConsole
    .map(({ level, args }) => `[console.${level}] ${args.map((value) => value instanceof Error ? value.stack || value.message : String(value)).join(' ')}`)
    .join('\n')
  throw new Error(`Unexpected console output during test:\n${messages}`)
})

if (!window.matchMedia) {
  window.matchMedia = (query) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  })
}
