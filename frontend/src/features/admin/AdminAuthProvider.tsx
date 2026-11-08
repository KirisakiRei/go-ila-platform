import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { adminAuthApi } from '@/lib/admin-api'
import { AdminAuthContext, type AdminAuthContextValue, type AdminUser } from './admin-auth-context'

type Status = 'restoring' | 'authenticated' | 'anonymous'

/**
 * AdminAuthProvider memulihkan sesi admin dari cookie HttpOnly milik pintu
 * admin, bukan dari sesi pelanggan.
 *
 * Pemisahan ini disengaja: admin dan pelanggan memakai cookie berbeda di
 * server, sehingga masuk sebagai pelanggan tidak otomatis membuka panel admin
 * dan keluar dari panel admin tidak mengeluarkan sesi pelanggan.
 */
export function AdminAuthProvider({ children }: { children: ReactNode }) {
  const [admin, setAdmin] = useState<AdminUser | null>(null)
  const [status, setStatus] = useState<Status>('restoring')
  const queryClient = useQueryClient()

  useEffect(() => {
    let active = true

    adminAuthApi
      .me()
      .then((profile) => {
        if (!active) return
        setAdmin({
          id: profile.user.id,
          name: profile.user.name,
          email: profile.user.email,
          role: profile.role.code || profile.user.role,
        })
        setStatus('authenticated')
      })
      .catch(() => {
        // 401 berarti belum masuk, dan kesalahan lain tetap berarti panel tidak
        // boleh terbuka. Keduanya berakhir sebagai "belum masuk" agar panel
        // tidak menggantung di layar memuat.
        if (!active) return
        setStatus('anonymous')
      })

    return () => {
      active = false
    }
  }, [])

  const signIn = useCallback(
    (next: AdminUser) => {
      setAdmin(next)
      setStatus('authenticated')
      // Data panel sebelumnya milik sesi lain; jangan pernah dipakai ulang.
      queryClient.clear()
    },
    [queryClient],
  )

  const signOut = useCallback(() => {
    void adminAuthApi.logout().catch(() => undefined)
    setAdmin(null)
    setStatus('anonymous')
    queryClient.clear()
  }, [queryClient])

  const value = useMemo<AdminAuthContextValue>(
    () => ({
      admin,
      isLoading: status === 'restoring',
      signIn,
      signOut,
    }),
    [admin, status, signIn, signOut],
  )

  return <AdminAuthContext.Provider value={value}>{children}</AdminAuthContext.Provider>
}
