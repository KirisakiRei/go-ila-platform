import { useCallback, useEffect, useRef, useState } from 'react'

/**
 * useIdleTimer menandai saat pengguna berhenti berinteraksi.
 *
 * Yang dihitung hanya interaksi nyata (klik, ketik, gulir, pindah tab).
 * Permintaan data berkala TIDAK dihitung: tanpa pemisahan ini, dashboard yang
 * memuat ulang data setiap 15 detik akan selalu terlihat aktif dan peringatan
 * tidak pernah muncul (plans/23 §4.3).
 *
 * Penjagaan sebenarnya tetap di server; timer ini hanya memberi kesempatan
 * pengguna menyelamatkan pekerjaannya.
 */

type Options = {
  /** Menit tanpa aktivitas sebelum peringatan muncul. */
  warnAfterMinutes: number
  /** Detik hitung mundur pada dialog peringatan. */
  countdownSeconds: number
  /** Matikan peringatan; batas diam di server tidak ikut berubah. */
  enabled: boolean
  onTimeout: () => void
}

type IdleState = {
  warningVisible: boolean
  remainingSeconds: number
  stay: () => void
  leaveNow: () => void
}

const ACTIVITY_EVENTS = ['mousedown', 'keydown', 'touchstart', 'scroll', 'visibilitychange'] as const

export function useIdleTimer({
  warnAfterMinutes,
  countdownSeconds,
  enabled,
  onTimeout,
}: Options): IdleState {
  const [warningVisible, setWarningVisible] = useState(false)
  const [remainingSeconds, setRemainingSeconds] = useState(countdownSeconds)

  const idleTimer = useRef<number | null>(null)
  const countdownTimer = useRef<number | null>(null)
  const channel = useRef<BroadcastChannel | null>(null)

  const clearTimers = useCallback(() => {
    if (idleTimer.current !== null) window.clearTimeout(idleTimer.current)
    if (countdownTimer.current !== null) window.clearInterval(countdownTimer.current)
    idleTimer.current = null
    countdownTimer.current = null
  }, [])

  const startCountdown = useCallback(() => {
    setWarningVisible(true)
    setRemainingSeconds(countdownSeconds)

    countdownTimer.current = window.setInterval(() => {
      setRemainingSeconds((prev) => {
        if (prev <= 1) {
          window.clearInterval(countdownTimer.current!)
          countdownTimer.current = null
          onTimeout()
          return 0
        }
        return prev - 1
      })
    }, 1000)
  }, [countdownSeconds, onTimeout])

  /**
   * armIdle hanya mengatur timer; tidak menyentuh state sehingga aman dipanggil
   * dari effect. Perubahan state ditangani terpisah di resetIdle.
   */
  const armIdle = useCallback(() => {
    clearTimers()
    if (!enabled) return

    idleTimer.current = window.setTimeout(startCountdown, warnAfterMinutes * 60_000)
  }, [clearTimers, enabled, startCountdown, warnAfterMinutes])

  const resetIdle = useCallback(() => {
    setWarningVisible(false)
    setRemainingSeconds(countdownSeconds)
    armIdle()
  }, [armIdle, countdownSeconds])

  // Satu tab menjadi pemimpin timer; tab lain mendengarkan agar tab lama
  // tidak keluar saat tab baru sedang dipakai (plans/23 §4.4).
  useEffect(() => {
    if (!enabled) {
      clearTimers()
      return
    }

    // Timer dipasang tanpa mengubah state, jadi aman dari render pertama.
    armIdle()

    const onActivity = () => {
      resetIdle()
      channel.current?.postMessage({ type: 'activity' })
    }

    for (const event of ACTIVITY_EVENTS) {
      window.addEventListener(event, onActivity, { passive: true })
    }

    if (typeof BroadcastChannel !== 'undefined') {
      channel.current = new BroadcastChannel('ila-idle')
      channel.current.onmessage = (message) => {
        if (message.data?.type === 'activity') resetIdle()
      }
    }

    return () => {
      for (const event of ACTIVITY_EVENTS) {
        window.removeEventListener(event, onActivity)
      }
      channel.current?.close()
      channel.current = null
      clearTimers()
    }
  }, [armIdle, clearTimers, enabled, resetIdle])

  const stay = useCallback(() => {
    resetIdle()
    channel.current?.postMessage({ type: 'activity' })
  }, [resetIdle])

  const leaveNow = useCallback(() => {
    clearTimers()
    onTimeout()
  }, [clearTimers, onTimeout])

  return { warningVisible, remainingSeconds, stay, leaveNow }
}
