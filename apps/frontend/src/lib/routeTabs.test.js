import { describe, expect, it, afterEach } from 'vitest'
import { DEFAULT_TAB, getTabFromLocation, setTabInLocation, subscribeTabChanges } from './routeTabs.js'

afterEach(() => {
  window.location.hash = ''
})

describe('routeTabs', () => {
  it('falls back to the default tab when the hash is empty', () => {
    window.location.hash = ''
    expect(getTabFromLocation()).toBe(DEFAULT_TAB)
  })

  it('falls back to the default tab for an unknown hash value', () => {
    window.location.hash = '#/not-a-real-tab'
    expect(getTabFromLocation()).toBe(DEFAULT_TAB)
  })

  it('reads a known tab id from the hash', () => {
    window.location.hash = '#/stations'
    expect(getTabFromLocation()).toBe('stations')
  })

  it('writes a valid tab id to the hash', () => {
    setTabInLocation('incidents')
    expect(window.location.hash).toBe('#/incidents')
  })

  it('ignores unknown tab ids when writing', () => {
    window.location.hash = '#/orbital'
    setTabInLocation('not-a-real-tab')
    expect(window.location.hash).toBe('#/orbital')
  })

  it('notifies subscribers on hash change', () => {
    window.location.hash = '#/orbital'
    let seen = null
    const unsubscribe = subscribeTabChanges((tab) => {
      seen = tab
    })
    window.location.hash = '#/settings'
    window.dispatchEvent(new HashChangeEvent('hashchange'))
    expect(seen).toBe('settings')
    unsubscribe()
  })
})
