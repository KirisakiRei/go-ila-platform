import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useIdleTimer } from '@/features/auth/hooks/use-idle-timer'

type Options = Parameters<typeof useIdleTimer>[0]

function renderIdle(overrides: Partial<Options> = {}) {
  const onTimeout = vi.fn()
  const hook = renderHook((props: Options) => useIdleTimer(props), {
    initialProps: {
      warnAfterMinutes: 25,
      countdownSeconds: 60,
      enabled: true,
      onTimeout,
      ...overrides,
    },
  })
  return { hook, onTimeout }
}

describe('useIdleTimer', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('menampilkan peringatan setelah batas diam', () => {
    const { hook } = renderIdle()

    expect(hook.result.current.warningVisible).toBe(false)

    act(() => {
      vi.advanceTimersByTime(25 * 60_000)
    })

    expect(hook.result.current.warningVisible).toBe(true)
  })

  it('mengeluarkan pengguna setelah hitung mundur habis', () => {
    const { onTimeout } = renderIdle()

    act(() => {
      vi.advanceTimersByTime(25 * 60_000)
    })
    expect(onTimeout).not.toHaveBeenCalled()

    act(() => {
      vi.advanceTimersByTime(60_000)
    })
    expect(onTimeout).toHaveBeenCalledTimes(1)
  })

  it('aktivitas menghapus peringatan dan memulai ulang timer', () => {
    const { hook, onTimeout } = renderIdle()

    act(() => {
      vi.advanceTimersByTime(25 * 60_000)
    })
    expect(hook.result.current.warningVisible).toBe(true)

    act(() => {
      hook.result.current.stay()
    })
    expect(hook.result.current.warningVisible).toBe(false)

    // Durasi penuh belum lewat, jadi belum boleh keluar.
    act(() => {
      vi.advanceTimersByTime(20 * 60_000)
    })
    expect(onTimeout).not.toHaveBeenCalled()
  })

  it('timer tidak berjalan saat dimatikan pengguna', () => {
    const { hook, onTimeout } = renderIdle({ enabled: false })

    act(() => {
      vi.advanceTimersByTime(60 * 60_000)
    })

    expect(hook.result.current.warningVisible).toBe(false)
    expect(onTimeout).not.toHaveBeenCalled()
  })

  it('tombol keluar langsung mengakhiri sesi', () => {
    const { hook, onTimeout } = renderIdle()

    act(() => {
      vi.advanceTimersByTime(25 * 60_000)
    })
    act(() => {
      hook.result.current.leaveNow()
    })

    expect(onTimeout).toHaveBeenCalledTimes(1)
  })
})
