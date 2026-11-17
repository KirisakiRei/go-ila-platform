import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { useLocation } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { adminAuthApi } from '@/lib/admin-api'
import { onAdminUnauthorized, selamaPemulihan } from '@/lib/http'
import { AdminAuthContext, type AdminAuthContextValue, type AdminUser } from './admin-auth-context'
import { mapProfile } from './lib/profile'

type Status = 'idle' | 'restoring' | 'authenticated' | 'anonymous'

function isAdminRoutePath(pathname: string): boolean {
  return /(^|\/)admin(\/|$)/.test(pathname)
}

/**
 * AdminAuthProvider memulihkan sesi admin dari cookie HttpOnly milik pintu
 * admin, bukan dari sesi pelanggan.
 *
 * Pemulihan hanya dijalankan ketika pengguna benar-benar membuka rute /admin/*,
 * sehingga halaman pelanggan biasa tidak pernah menembak /admin/auth/me dan
 * balasan 401 di ranah admin tidak pernah mengganggu sesi pelanggan.
 */
export function AdminAuthProvider({ children }: { children: ReactNode }) {
  const location = useLocation()
  const onAdminRoute = isAdminRoutePath(location.pathname)
  const [admin, setAdmin] = useState<AdminUser | null>(null)
  const [status, setStatus] = useState<Status>(() => (onAdminRoute ? 'restoring' : 'idle'))
  const restoredOnce = useRef(false)
  const queryClient = useQueryClient()

  useEffect(() => {
    if (!onAdminRoute || restoredOnce.current) return
    setStatus('restoring')

    let active = true

    selamaPemulihan(() => adminAuthApi.me())
      .then((profile) => {
        if (!active) return
        restoredOnce.current = true
        setAdmin(mapProfile(profile))
        setStatus('authenticated')
      })
      .catch(() => {
        // 401 berarti belum masuk, dan kesalahan lain tetap berarti panel tidak
        // boleh terbuka. Keduanya berakhir sebagai "belum masuk" agar panel
        // tidak menggantung di layar memuat.
        if (!active) return
        restoredOnce.current = true
        setStatus('anonymous')
      })

    return () => {
      active = false
    }
  }, [onAdminRoute])

  useEffect(() => {
    onAdminUnauthorized(() => {
      setAdmin(null)
      setStatus('anonymous')
    })
    return () => onAdminUnauthorized(null)
  }, [])

  const signIn = useCallback(
    (next: AdminUser) => {
      restoredOnce.current = true
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

  const refreshPermissions = useCallback(async () => {
    const profile = await adminAuthApi.me()
    setAdmin(mapProfile(profile))
  }, [])

  const value = useMemo<AdminAuthContextValue>(
    () => ({
      admin,
      isLoading: status === 'restoring' || (onAdminRoute && status === 'idle'),
      signIn,
      signOut,
      refreshPermissions,
    }),
    [admin, onAdminRoute, status, signIn, signOut, refreshPermissions],
  )

  return <AdminAuthContext.Provider value={value}>{children}</AdminAuthContext.Provider>
}
