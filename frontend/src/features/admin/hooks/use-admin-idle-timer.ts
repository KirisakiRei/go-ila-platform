import { useCallback, useEffect, useRef, useState } from 'react'
import { adminAuthApi } from '@/lib/admin-api'
import { ApiError } from '@/lib/api-error'

/** Batas diam panel admin sebelum peringatan muncul (plans/23 §2). */
const WARN_AFTER_MS = 10 * 60_000
/** Jendela konfirmasi sebelum sesi ditutup (plans/23 §4.2). */
const COUNTDOWN_SECONDS = 60

/**
 * Jarak minimum antar heartbeat.
 *
 * Sesi admin berlaku dua jam, jadi memperpanjang cookie tiap klik tidak
 * diperlukan. Sebelumnya setiap pointerdown, keydown, wheel, dan touchstart
 * mengirim heartbeat; beberapa detik menggulir daftar sudah cukup menghabiskan
 * kuota pembatas laju yang dipakai bersama endpoint data (plans/21 D4).
 */
const HEARTBEAT_MIN_INTERVAL_MS = 4 * 60_000

/**
 * Masa tunggu setelah server menolak dengan 429.
 *
 * Menahan pengiriman lebih lama daripada jendela pembatas laju membuat data
 * admin tetap bisa diambil setelah kuota pulih.
 */
const RATE_LIMIT_BACKOFF_MS = 60_000

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
 * Dua hal dipisahkan dengan sengaja:
 *
 *  1. Reset idle — murni lokal, dijalankan pada setiap interaksi.
 *  2. Heartbeat — permintaan jaringan, dibatasi intervalnya.
 *
 * Tanpa pemisahan ini aktivitas pengguna membanjiri API dan request data ikut
 * ditolak bersama heartbeat. Hanya interaksi nyata yang dihitung; permintaan
 * data berkala tidak dianggap aktivitas.
 */
export function useAdminIdleTimer({ enabled, onTimeout }: Options): AdminIdleState {
  const [warningVisible, setWarningVisible] = useState(false)
  const [remainingSeconds, setRemainingSeconds] = useState(COUNTDOWN_SECONDS)

  const idleTimer = useRef<number | null>(null)
  const countdownTimer = useRef<number | null>(null)
  const firedRef = useRef(false)

  // Callback disimpan di ref agar perubahan identitas fungsi dari pemanggil
  // tidak memasang ulang listener dan tidak memicu heartbeat tambahan.
  const onTimeoutRef = useRef(onTimeout)
  useEffect(() => {
    onTimeoutRef.current = onTimeout
  }, [onTimeout])

  const lastPingAt = useRef(0)
  const pingInFlight = useRef(false)
  const pingBlockedUntil = useRef(0)

  const clearTimers = useCallback(() => {
    if (idleTimer.current !== null) window.clearTimeout(idleTimer.current)
    if (countdownTimer.current !== null) window.clearInterval(countdownTimer.current)
    idleTimer.current = null
    countdownTimer.current = null
  }, [])

  /**
   * ping memperpanjang cookie sesi, dibatasi tiga cara: satu permintaan
   * berjalan pada satu waktu, jarak minimum antar permintaan, dan jeda setelah
   * server menolak karena kuota.
   */
  const ping = useCallback(() => {
    if (typeof document !== 'undefined' && document.visibilityState === 'hidden') return

    const now = Date.now()
    if (pingInFlight.current) return
    if (now < pingBlockedUntil.current) return
    if (now - lastPingAt.current < HEARTBEAT_MIN_INTERVAL_MS) return

    lastPingAt.current = now
    pingInFlight.current = true
    void adminAuthApi
      .heartbeat()
      .catch((error: unknown) => {
        // Kuota pembatas laju habis: berhenti sejenak supaya endpoint data
        // tidak ikut tertahan. Kegagalan lain diabaikan, sama seperti
        // sebelumnya, karena sesi tetap diperiksa server pada request berikut.
        if (error instanceof ApiError && error.status === 429) {
          pingBlockedUntil.current = Date.now() + RATE_LIMIT_BACKOFF_MS
        }
      })
      .finally(() => {
        pingInFlight.current = false
      })
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
            setWarningVisible(false)
            onTimeoutRef.current()
          }
          return 0
        }
        return value - 1
      })
    }, 1000)
  }, [])

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
    // ping sudah membatasi dirinya sendiri, jadi aman dipanggil pada setiap
    // interaksi tanpa mengirim permintaan beruntun.
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

  const leaveNow = useCallback(() => {
    clearTimers()
    firedRef.current = true
    setWarningVisible(false)
    onTimeoutRef.current()
  }, [clearTimers])

  return {
    warningVisible,
    remainingSeconds,
    stay: resetIdle,
    leaveNow,
  }
}
