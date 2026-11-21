import { useEffect, useSyncExternalStore } from 'react'

export type ThemePreference = 'light' | 'dark' | 'system'
export type ResolvedTheme = 'light' | 'dark'

export const THEME_STORAGE_KEY = 'ila_user_theme'

const DEFAULT_META_THEME_COLOR = '#171717'
const LIGHT_META_THEME_COLOR = '#ffffff'
const DARK_META_THEME_COLOR = '#0e0e11'

const listeners = new Set<() => void>()

function emitChange() {
  for (const listener of listeners) {
    listener()
  }
}

function subscribe(listener: () => void) {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

export function isValidThemePreference(value: unknown): value is ThemePreference {
  return value === 'light' || value === 'dark' || value === 'system'
}

export function readThemePreference(): ThemePreference {
  if (typeof window === 'undefined') return 'light'
  try {
    const stored = window.localStorage.getItem(THEME_STORAGE_KEY)
    return isValidThemePreference(stored) ? stored : 'light'
  } catch {
    return 'light'
  }
}

export function readSystemTheme(): ResolvedTheme {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
    return 'light'
  }
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export function resolveTheme(preference: ThemePreference): ResolvedTheme {
  if (preference === 'system') return readSystemTheme()
  return preference
}

export function setThemePreference(preference: ThemePreference) {
  if (typeof window !== 'undefined') {
    try {
      window.localStorage.setItem(THEME_STORAGE_KEY, preference)
    } catch {
      // Abaikan jika penyimpanan lokal diblokir (mode privat / kuota penuh).
    }
  }
  emitChange()
}

function getSnapshot(): ThemePreference {
  return readThemePreference()
}

function getServerSnapshot(): ThemePreference {
  return 'light'
}

function syncDocumentTheme(resolved: ResolvedTheme) {
  if (typeof document === 'undefined') return
  const root = document.documentElement
  root.classList.toggle('dark', resolved === 'dark')

  const meta = document.querySelector('meta[name="theme-color"]')
  if (meta) {
    meta.setAttribute('content', resolved === 'dark' ? DARK_META_THEME_COLOR : LIGHT_META_THEME_COLOR)
  }
}

function clearDocumentTheme() {
  if (typeof document === 'undefined') return
  document.documentElement.classList.remove('dark')

  const meta = document.querySelector('meta[name="theme-color"]')
  if (meta) {
    meta.setAttribute('content', DEFAULT_META_THEME_COLOR)
  }
}

/**
 * Hook untuk membaca dan mengubah preferensi tema dashboard pelanggan.
 * Default untuk pengguna baru adalah 'light'.
 */
export function useUserTheme() {
  const preference = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot)
  const resolved = resolveTheme(preference)
  const isDark = resolved === 'dark'

  const toggle = () => {
    setThemePreference(isDark ? 'light' : 'dark')
  }

  return {
    preference,
    resolved,
    isDark,
    setPreference: setThemePreference,
    toggle,
  }
}

/**
 * Menerapkan kelas `.dark` pada <html> selama berada di dalam AppShell
 * (dashboard user), dan membersihkannya saat keluar agar halaman publik,
 * login, dan panel admin tidak ikut berubah.
 */
export function useApplyUserDashboardTheme() {
  const preference = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot)

  useEffect(() => {
    syncDocumentTheme(resolveTheme(preference))

    const onStorage = (event: StorageEvent) => {
      if (event.key === THEME_STORAGE_KEY) {
        emitChange()
      }
    }
    window.addEventListener('storage', onStorage)

    let mediaQuery: MediaQueryList | null = null
    const onMediaChange = () => {
      if (readThemePreference() === 'system') {
        syncDocumentTheme(readSystemTheme())
        emitChange()
      }
    }

    if (preference === 'system' && typeof window.matchMedia === 'function') {
      mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
      mediaQuery.addEventListener('change', onMediaChange)
    }

    return () => {
      window.removeEventListener('storage', onStorage)
      mediaQuery?.removeEventListener('change', onMediaChange)
    }
  }, [preference])

  useEffect(() => {
    return () => {
      clearDocumentTheme()
    }
  }, [])
}
