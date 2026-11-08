import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { authApi } from '@/lib/auth-api'
import { onUnauthorized, readCsrfToken, selamaPemulihan } from '@/lib/http'
import type { User } from '@/types/api'
import { AuthContext, type AuthContextValue } from './auth-context'
import { IdleWarningDialog } from './components/IdleWarningDialog'
import { useIdleTimer } from './hooks/use-idle-timer'

type SessionStatus = 'restoring' | 'authenticated' | 'anonymous'

/** Batas diam sebelum peringatan dan sebelum keluar (plans/23 §2). */
const IDLE_WARN_MINUTES = 25
const IDLE_LOGOUT_MINUTES = 30
const IDLE_COUNTDOWN_SECONDS = 60

/** Alasan keluar disimpan agar halaman masuk bisa menjelaskannya. */
export const LOGOUT_REASON_KEY = 'ila_logout_reason'

/**
 * AuthProvider memulihkan sesi dari cookie HttpOnly, menyediakan aksi
 * masuk/keluar, dan memutus tampilan seketika saat backend menolak sesi.
 * Tidak ada token yang disimpan di sisi klien.
 */
export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [status, setStatus] = useState<SessionStatus>('restoring')
  const queryClient = useQueryClient()

  // Mencegah peringatan berulang saat keluar dipicu dua kali.
  const signingOut = useRef(false)

  const finishSignOut = useCallback(
    (reason?: 'idle' | 'expired') => {
      if (signingOut.current) return
      signingOut.current = true

      if (reason) {
        // Disimpan di sessionStorage, bukan localStorage: cukup untuk satu tab
        // dan hilang saat browser ditutup.
        sessionStorage.setItem(LOGOUT_REASON_KEY, reason)
      }

      setUser(null)
      setStatus('anonymous')
      queryClient.clear()

      // Izinkan masuk lagi tanpa memuat ulang halaman.
      window.setTimeout(() => {
        signingOut.current = false
      }, 0)
    },
    [queryClient],
  )

  const signOut = useCallback(
    (reason?: 'idle') => {
      void authApi.logout().catch(() => undefined)
      finishSignOut(reason)
    },
    [finishSignOut],
  )

  const signIn = useCallback((nextUser: User) => {
    sessionStorage.removeItem(LOGOUT_REASON_KEY)
    setUser(nextUser)
    setStatus('authenticated')
  }, [])

  // Pemulihan sesi hanya sekali saat aplikasi dimuat.
  useEffect(() => {
    let cancelled = false

    async function restore() {
      try {
        // 401 di sini berarti belum masuk, bukan sesi berakhir. Tanpa penanda
        // ini pengunjung baru melihat "Sesi Anda berakhir" saat pertama membuka
        // halaman masuk.
        const currentUser = await selamaPemulihan(() => authApi.me())
        if (cancelled) return

        // Cookie CSRF bisa hilang (sesi browser dibersihkan sebagian),
        // sedangkan permintaan tulis pertama akan ditolak tanpa itu.
        if (!readCsrfToken()) {
          await authApi.csrf().catch(() => undefined)
        }

        setUser(currentUser)
        setStatus('authenticated')
      } catch {
        if (!cancelled) setStatus('anonymous')
      }
    }

    void restore()
    return () => {
      cancelled = true
    }
  }, [])

  // Sesi yang ditolak backend langsung dibuang agar UI tidak bertahan
  // pada keadaan yang sudah tidak sah. Handler dilepas saat komponen bongkar
  // supaya tidak tertinggal menunjuk state yang sudah usang.
  useEffect(() => {
    onUnauthorized(() => finishSignOut('expired'))
    return () => onUnauthorized(null)
  }, [finishSignOut])

  const auth = useIdleTimer({
    // Peringatan muncul lebih dulu; batas keluar ditegakkan server.
    warnAfterMinutes: IDLE_WARN_MINUTES,
    countdownSeconds: IDLE_COUNTDOWN_SECONDS,
    enabled: status === 'authenticated' && (user?.idle_warning ?? true),
    onTimeout: () => signOut('idle'),
  })

  // Cadangan bila tab lain sudah lebih dulu keluar.
  useEffect(() => {
    if (status !== 'authenticated' || !user?.idle_warning) return

    const timer = window.setTimeout(
      () => {
        finishSignOut('idle')
      },
      IDLE_LOGOUT_MINUTES * 60_000 + 10_000,
    )
    return () => window.clearTimeout(timer)
  }, [status, finishSignOut, user?.idle_warning])

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      isAuthenticated: status === 'authenticated',
      isLoading: status === 'restoring',
      signIn,
      signOut: () => signOut(),
    }),
    [user, status, signIn, signOut],
  )

  return (
    <AuthContext.Provider value={value}>
      {children}
      {auth.warningVisible && (
        <IdleWarningDialog
          remainingSeconds={auth.remainingSeconds}
          onStay={auth.stay}
          onLeaveNow={auth.leaveNow}
        />
      )}
    </AuthContext.Provider>
  )
}
