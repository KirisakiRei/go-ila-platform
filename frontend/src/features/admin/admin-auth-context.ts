import { createContext, useContext } from 'react'

/**
 * AdminUser adalah identitas admin yang sedang masuk.
 *
 * Sengaja terpisah dari tipe `User` milik pelanggan: sesi admin memakai cookie
 * sendiri di server, dan mencampur keduanya membuat admin bisa tampil sebagai
 * pelanggan biasa atau sebaliknya.
 */
export type AdminUser = {
  id: string
  name: string
  email: string
  role: string
}

export type AdminAuthContextValue = {
  admin: AdminUser | null
  isLoading: boolean
  signIn: (admin: AdminUser) => void
  signOut: () => void
}

/**
 * AdminAuthContext dipisah dari provider agar hot-reload tidak memutus
 * referensi konteks.
 */
export const AdminAuthContext = createContext<AdminAuthContextValue | null>(null)

export function useAdminAuth(): AdminAuthContextValue {
  const context = useContext(AdminAuthContext)
  if (!context) {
    throw new Error('useAdminAuth harus dipakai di dalam AdminAuthProvider')
  }
  return context
}
