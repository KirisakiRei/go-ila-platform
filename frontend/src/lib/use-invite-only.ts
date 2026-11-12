import { useQuery } from '@tanstack/react-query'

import { publicContentApi } from '@/lib/system-api'

/**
 * useInviteOnly menandai platform sedang tertutup untuk pendaftar mandiri.
 *
 * Selama menyala, pendaftaran, uji coba, dan seluruh jalan menuju pembayaran
 * disembunyikan. Akun baru hanya lahir dari undangan admin.
 *
 * Nilainya dibaca dari `GET /api/v1/status` bersama status pemeliharaan, jadi
 * tidak ada permintaan tambahan di halaman yang sudah memakainya.
 *
 * Gagal membaca status diperlakukan sebagai TERTUTUP: itu sama dengan yang
 * dilakukan backend (`INVITE_ONLY` gagal dibaca → tertutup), sehingga UI tidak
 * pernah menawarkan pintu yang di server sudah dikunci. Status pemeliharaan di
 * halaman yang sama tetap memakai nilai bawaannya sendiri, jadi memilih aman di
 * sini tidak mengubah perilaku halaman saat jaringan bermasalah.
 */
export function useInviteOnly(): boolean {
  const { data } = useQuery({
    queryKey: ['public', 'status'],
    queryFn: publicContentApi.status,
    staleTime: 30_000,
  })

  if (!data) return true
  return data.invite_only !== false
}
