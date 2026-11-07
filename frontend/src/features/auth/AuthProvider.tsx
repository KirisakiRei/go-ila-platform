import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { authApi } from '@/lib/auth-api'
import { onUnauthorized, readCsrfToken } from '@/lib/http'
import type { User } from '@/types/api'
import { AuthContext, type AuthContextValue } from './auth-context'

type SessionStatus = 'restoring' | 'authenticated' | 'anonymous'

/**
 * AuthProvider memulihkan sesi dari cookie HttpOnly, menyediakan aksi
 * masuk/keluar, dan memutus tampilan seketika saat backend menolak sesi.
 * Tidak ada token yang disimpan di sisi klien.
 */
export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [status, setStatus] = useState<SessionStatus>('restoring')
  const queryClient = useQueryClient()

  const signOut = useCallback(() => {
    // Cookie dihapus backend; kegagalan jaringan tidak boleh memblokir keluar.
    void authApi.logout().catch(() => undefined)
    setUser(null)
    setStatus('anonymous')
    queryClient.clear()
  }, [queryClient])

  const signIn = useCallback((nextUser: User) => {
    setUser(nextUser)
    setStatus('authenticated')
  }, [])

  // Pemulihan sesi hanya sekali saat aplikasi dimuat.
  useEffect(() => {
    let cancelled = false

    async function restore() {
      try {
        const currentUser = await authApi.me()
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
  // pada keadaan yang sudah tidak sah.
  useEffect(() => {
    onUnauthorized(() => {
      setUser(null)
      setStatus('anonymous')
    })
  }, [])

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      isAuthenticated: status === 'authenticated',
      isLoading: status === 'restoring',
      signIn,
      signOut,
    }),
    [user, status, signIn, signOut],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
