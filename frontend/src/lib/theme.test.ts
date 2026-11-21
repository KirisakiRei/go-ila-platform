import { afterEach, describe, expect, it } from 'vitest'
import {
  THEME_STORAGE_KEY,
  readThemePreference,
  resolveTheme,
  setThemePreference,
} from './theme'

describe('user dashboard theme store', () => {
  afterEach(() => {
    window.localStorage.removeItem(THEME_STORAGE_KEY)
  })

  it('defaults to light when storage is empty or invalid', () => {
    expect(readThemePreference()).toBe('light')

    window.localStorage.setItem(THEME_STORAGE_KEY, 'neon-purple')
    expect(readThemePreference()).toBe('light')
  })

  it('persists light, dark, and system preferences', () => {
    setThemePreference('dark')
    expect(readThemePreference()).toBe('dark')
    expect(resolveTheme('dark')).toBe('dark')

    setThemePreference('system')
    expect(readThemePreference()).toBe('system')
    expect(['light', 'dark']).toContain(resolveTheme('system'))

    setThemePreference('light')
    expect(readThemePreference()).toBe('light')
    expect(resolveTheme('light')).toBe('light')
  })
})
