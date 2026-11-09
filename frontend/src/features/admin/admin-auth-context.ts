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
  /** Nama role RBAC untuk ditampilkan; kosong bila role belum ditugaskan. */
  roleName?: string
  /** Superadmin melewati sebagian pembatas tampilan (server tetap memutuskan). */
  isSuperadmin?: boolean
  /**
   * Permission yang dipegang admin ini. Daftar kosong berarti tidak berwenang;
   * lihat `features/admin/lib/permissions.ts`.
   */
  permissions: string[]
}

export type AdminAuthContextValue = {
  admin: AdminUser | null
  isLoading: boolean
  signIn: (admin: AdminUser) => void
  signOut: () => void
  /**
   * Perbarui daftar permission tanpa logout.
   *
   * Dipakai setelah role admin berubah: server sudah menerbitkan sesi baru
   * dengan permission terbaru, dan panel cukup memuat ulang profilnya agar menu
   * langsung menyesuaikan tanpa memaksa masuk ulang.
   */
  refreshPermissions: () => Promise<void>
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
