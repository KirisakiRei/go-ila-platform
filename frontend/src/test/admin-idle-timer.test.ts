import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'

const heartbeat = vi.fn(() => Promise.resolve({ expires_at: '', idle_minutes: 10 }))

vi.mock('@/lib/admin-api', () => ({
  adminAuthApi: {
    heartbeat: () => heartbeat(),
  },
}))

const { useAdminIdleTimer } = await import('@/features/admin/hooks/use-admin-idle-timer')

type Options = Parameters<typeof useAdminIdleTimer>[0]

function renderIdle(overrides: Partial<Options> = {}) {
  const onTimeout = vi.fn()
  const hook = renderHook((props: Options) => useAdminIdleTimer(props), {
    initialProps: { enabled: true, onTimeout, ...overrides },
  })
  return { hook, onTimeout }
}

describe('useAdminIdleTimer', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    heartbeat.mockClear()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('menampilkan peringatan setelah 10 menit diam', () => {
    const { hook } = renderIdle()

    expect(hook.result.current.warningVisible).toBe(false)

    act(() => {
      vi.advanceTimersByTime(10 * 60_000)
    })

    expect(hook.result.current.warningVisible).toBe(true)
  })

  it('keluar otomatis setelah hitung mundur habis', () => {
    const { onTimeout } = renderIdle()

    act(() => {
      vi.advanceTimersByTime(10 * 60_000)
    })
    expect(onTimeout).not.toHaveBeenCalled()

    act(() => {
      vi.advanceTimersByTime(60_000)
    })
    expect(onTimeout).toHaveBeenCalledTimes(1)
  })

  it('tombol masih di sini mengulang timer dan memperpanjang sesi', () => {
    const { hook, onTimeout } = renderIdle()

    act(() => {
      vi.advanceTimersByTime(10 * 60_000)
    })
    heartbeat.mockClear()

    act(() => {
      hook.result.current.stay()
    })

    expect(hook.result.current.warningVisible).toBe(false)
    expect(heartbeat).toHaveBeenCalledTimes(1)

    // Hitung mundur lama tidak boleh lagi memicu keluar.
    act(() => {
      vi.advanceTimersByTime(30_000)
    })
    expect(onTimeout).not.toHaveBeenCalled()
  })

  it('tidak berjalan saat dinonaktifkan', () => {
    const { hook, onTimeout } = renderIdle({ enabled: false })

    act(() => {
      vi.advanceTimersByTime(30 * 60_000)
    })

    expect(hook.result.current.warningVisible).toBe(false)
    expect(onTimeout).not.toHaveBeenCalled()
  })
})
