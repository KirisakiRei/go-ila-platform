import { useCallback, useEffect, useRef, useState } from 'react'
import { adminAuthApi } from '@/lib/admin-api'

/** Batas diam panel admin sebelum peringatan muncul (plans/23 §2). */
const WARN_AFTER_MS = 10 * 60_000
/** Jendela konfirmasi sebelum sesi ditutup (plans/23 §4.2). */
const COUNTDOWN_SECONDS = 60

type Options = {
  enabled: boolean
  onTimeout: () => void
}

export type AdminIdleState = {
  warningVisible: boolean
  remainingSeconds: number
  stay: () => void
  leaveNow: () => void
}

/**
 * useAdminIdleTimer mengawasi diam pengguna di panel admin.
 *
 * Hanya interaksi nyata yang dihitung; permintaan data berkala diabaikan agar
 * dashboard yang memuat ulang tiap 15 detik tidak dianggap sebagai aktivitas.
 * Selama pengguna aktif, cookie sesi diperpanjang lewat heartbeat.
 */
export function useAdminIdleTimer({ enabled, onTimeout }: Options): AdminIdleState {
  const [warningVisible, setWarningVisible] = useState(false)
  const [remainingSeconds, setRemainingSeconds] = useState(COUNTDOWN_SECONDS)

  const idleTimer = useRef<number | null>(null)
  const countdownTimer = useRef<number | null>(null)
  const firedRef = useRef(false)

  const clearTimers = useCallback(() => {
    if (idleTimer.current !== null) window.clearTimeout(idleTimer.current)
    if (countdownTimer.current !== null) window.clearInterval(countdownTimer.current)
    idleTimer.current = null
    countdownTimer.current = null
  }, [])

  const ping = useCallback(() => {
    void adminAuthApi.heartbeat().catch(() => undefined)
  }, [])

  const startCountdown = useCallback(() => {
    setWarningVisible(true)
    setRemainingSeconds(COUNTDOWN_SECONDS)

    countdownTimer.current = window.setInterval(() => {
      setRemainingSeconds((value) => {
        if (value <= 1) {
          if (countdownTimer.current !== null) window.clearInterval(countdownTimer.current)
          countdownTimer.current = null
          if (!firedRef.current) {
            firedRef.current = true
            onTimeout()
          }
          return 0
        }
        return value - 1
      })
    }, 1000)
  }, [onTimeout])

  const armIdle = useCallback(() => {
    clearTimers()
    if (!enabled) return
    idleTimer.current = window.setTimeout(startCountdown, WARN_AFTER_MS)
  }, [clearTimers, enabled, startCountdown])

  const resetIdle = useCallback(() => {
    firedRef.current = false
    setWarningVisible(false)
    setRemainingSeconds(COUNTDOWN_SECONDS)
    armIdle()
    ping()
  }, [armIdle, ping])

  useEffect(() => {
    if (!enabled) {
      clearTimers()
      return
    }

    firedRef.current = false
    armIdle()
    ping()

		const events = ['pointerdown', 'keydown', 'wheel', 'touchstart'] as const
    const onActivity = () => resetIdle()

    events.forEach((name) => window.addEventListener(name, onActivity, { passive: true }))
    return () => {
      events.forEach((name) => window.removeEventListener(name, onActivity))
      clearTimers()
    }
  }, [enabled, armIdle, resetIdle, clearTimers, ping])

  return {
    warningVisible,
    remainingSeconds,
    stay: resetIdle,
    leaveNow: () => {
      clearTimers()
      firedRef.current = true
      onTimeout()
    },
  }
}
