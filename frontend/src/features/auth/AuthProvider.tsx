import { useCallback, useMemo, useState, type ReactNode } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { authApi } from '@/lib/auth-api'
import { onUnauthorized, tokenStorage } from '@/lib/http'
import type { User } from '@/types/api'
import { AuthContext, type AuthContextValue } from './auth-context'
import { useEffect } from 'react'

type SessionStatus = 'restoring' | 'authenticated' | 'anonymous'

/**
 * AuthProvider memulihkan sesi dari token tersimpan, menyediakan aksi
 * masuk/keluar, dan memutus sesi seketika saat backend menolak token.
 */
export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [status, setStatus] = useState<SessionStatus>('restoring')
  const queryClient = useQueryClient()

  const signOut = useCallback(() => {
    tokenStorage.clear()
    setUser(null)
    setStatus('anonymous')
    queryClient.clear()
  }, [queryClient])

  const signIn = useCallback((token: string, nextUser: User) => {
    tokenStorage.set(token)
    setUser(nextUser)
    setStatus('authenticated')
  }, [])

  // Pemulihan sesi hanya sekali saat aplikasi dimuat.
  useEffect(() => {
    let cancelled = false

    async function restore() {
      if (!tokenStorage.get()) {
        if (!cancelled) setStatus('anonymous')
        return
      }

      try {
        const currentUser = await authApi.me()
        if (!cancelled) {
          setUser(currentUser)
          setStatus('authenticated')
        }
      } catch {
        if (!cancelled) {
          tokenStorage.clear()
          setStatus('anonymous')
        }
      }
    }

    void restore()
    return () => {
      cancelled = true
    }
  }, [])

  // Token yang ditolak backend langsung dibuang agar UI tidak bertahan
  // pada sesi yang sudah tidak sah.
  useEffect(() => {
    onUnauthorized(() => {
      tokenStorage.clear()
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
