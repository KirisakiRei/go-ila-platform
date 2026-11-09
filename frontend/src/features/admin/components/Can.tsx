import type { ReactNode } from 'react'
import { usePermissions } from '../hooks/use-permissions'

type CanProps = {
  /**
   * Satu permission yang dibutuhkan. Kosong berarti selalu tampil, supaya
   * pemakaian yang belum selesai tidak menyembunyikan isi halaman.
   */
  permission?: string
  /** Beberapa permission; cukup salah satu dimiliki. */
  anyOf?: readonly string[]
  children: ReactNode
  /** Ditampilkan saat tidak berwenang, misalnya tombol pengganti yang mati. */
  fallback?: ReactNode
}

/**
 * Can menampilkan isi hanya bila admin berwenang.
 *
 * Ini pembatas tampilan, bukan pengaman: server tetap menolak permintaan tanpa
 * permission dengan 403. Gunakan untuk menyembunyikan tombol aksi destruktif
 * agar admin tidak menekan tombol yang pasti gagal.
 */
export function Can({ permission, anyOf, children, fallback = null }: CanProps) {
  const { can, canAny } = usePermissions()

  const allowed = permission ? can(permission) : canAny(anyOf)
  return <>{allowed ? children : fallback}</>
}
