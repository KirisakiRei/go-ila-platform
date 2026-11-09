import { useMemo } from 'react'
import { useAdminAuth } from '../admin-auth-context'
import { permissionSet } from '../lib/permissions'

export type PermissionChecker = {
  /** Seluruh permission yang dipegang admin ini. */
  permissions: string[]
  isSuperadmin: boolean
  /** can menandai apakah satu permission dimiliki. */
  can: (key?: string) => boolean
  /** canAny menandai apakah salah satu permission dimiliki. */
  canAny: (keys?: readonly string[]) => boolean
}

/**
 * usePermissions membaca permission admin yang sedang masuk.
 *
 * Server tetap penentu akhir; hook ini hanya untuk menyembunyikan menu dan
 * tombol yang pasti ditolak 403 agar panel tidak membingungkan. Daftar kosong
 * berarti tidak berwenang (deny by default).
 */
export function usePermissions(): PermissionChecker {
  const { admin } = useAdminAuth()
  const isSuperadmin = admin?.isSuperadmin ?? false

  // Daftar dari konteks dipakai langsung; memo ditambahkan agar set di bawah
  // tidak dibangun ulang pada setiap render.
  const permissions = useMemo(() => admin?.permissions ?? [], [admin])
  const set = useMemo(() => permissionSet(permissions), [permissions])

  return useMemo<PermissionChecker>(
    () => ({
      permissions,
      isSuperadmin,
      can: (key?: string) => (key ? set.has(key) : true),
      canAny: (keys?: readonly string[]) =>
        !keys || keys.length === 0 ? true : keys.some((key) => set.has(key)),
    }),
    [permissions, isSuperadmin, set],
  )
}
