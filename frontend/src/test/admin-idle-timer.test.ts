import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { ApiError } from '@/lib/api-error'

const { heartbeat } = vi.hoisted(() => ({
  heartbeat: vi.fn(() => Promise.resolve({ expires_at: '', idle_minutes: 10 })),
}))

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
    // Batas interval heartbeat memakai Date.now(), jadi jam palsu harus ikut
    // maju bersama timer.
    vi.useFakeTimers({ shouldAdvanceTime: true })
    vi.setSystemTime(new Date('2026-01-01T00:00:00Z'))
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

  it('tombol masih di sini mengulang timer tanpa mengirim heartbeat beruntun', () => {
    const { hook, onTimeout } = renderIdle()

    act(() => {
      vi.advanceTimersByTime(10 * 60_000)
    })
    heartbeat.mockClear()

    // Heartbeat pertama sudah terkirim saat hook dipasang, jadi klik dalam
    // jendela interval minimum tidak boleh menambah permintaan.
    act(() => {
      hook.result.current.stay()
    })

    expect(hook.result.current.warningVisible).toBe(false)
    expect(heartbeat).not.toHaveBeenCalled()

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

/**
 * Pembatasan laju heartbeat diuji dengan jam sungguhan yang bisa dimajukan,
 * karena batas interval memakai Date.now() dan tidak bergantung pada timer.
 */
describe('useAdminIdleTimer heartbeat', () => {
  const realNow = Date.now.bind(Date)
  let offsetMs = 0

  beforeEach(() => {
    offsetMs = 0
    vi.spyOn(Date, 'now').mockImplementation(() => realNow() + offsetMs)
    heartbeat.mockClear()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('mengirim heartbeat saat interaksi pertama lalu menahan yang berikutnya', () => {
    renderIdle()

    expect(heartbeat).toHaveBeenCalledTimes(1)

    act(() => {
      window.dispatchEvent(new Event('pointerdown'))
    })

    expect(heartbeat).toHaveBeenCalledTimes(1)
  })

  it('mengirim heartbeat lagi setelah jarak minimum terlampaui', async () => {
    renderIdle()

    // Tunggu permintaan pertama selesai supaya penanda "sedang berjalan" lepas.
    await act(async () => {
      await Promise.resolve()
    })

    offsetMs += 4 * 60_000
    act(() => {
      window.dispatchEvent(new Event('pointerdown'))
    })

    expect(heartbeat).toHaveBeenCalledTimes(2)
  })

  it('menahan pengiriman selama masa jeda setelah ditolak 429', async () => {
    heartbeat.mockRejectedValueOnce(new ApiError('rate limited', 429))
    renderIdle()

    await act(async () => {
      await Promise.resolve()
      await Promise.resolve()
    })
    expect(heartbeat).toHaveBeenCalledTimes(1)

    // Jendela jeda 60 detik belum lewat, jadi interaksi tidak memicu kirim.
    offsetMs += 30_000
    act(() => {
      window.dispatchEvent(new Event('pointerdown'))
    })
    expect(heartbeat).toHaveBeenCalledTimes(1)

    // Lewati sisa masa jeda sekaligus jarak minimum heartbeat.
    offsetMs += 4 * 60_000
    act(() => {
      window.dispatchEvent(new Event('pointerdown'))
    })
    expect(heartbeat).toHaveBeenCalledTimes(2)
  })
})
