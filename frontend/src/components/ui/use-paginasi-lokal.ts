import { useMemo, useState } from 'react'

/** PILIHAN_UKURAN adalah jumlah baris per halaman yang lazim dipakai. */
export const PILIHAN_UKURAN = [10, 20, 50, 100] as const

/** UKURAN_BAWAAN adalah jumlah baris per halaman bila pengguna belum memilih. */
export const UKURAN_BAWAAN = 20

export type HasilPaginasi<T> = {
  /** Potongan data yang tampil pada halaman sekarang. */
  baris: T[]
  /** Halaman yang sedang tampil, dimulai dari 1. */
  page: number
  setPage: (page: number) => void
  totalPages: number
  total: number
  pageSize: number
  setPageSize: (size: number) => void
}

/**
 * usePaginasiLokal memecah daftar yang sudah dimuat seluruhnya menjadi halaman.
 *
 * Dipakai halaman yang endpoint-nya mengembalikan seluruh baris (mis. invoice
 * pengguna, kupon admin). Server-side pagination tetap lebih baik untuk data
 * besar; hook ini menyatukan perilakunya agar semua tabel punya kontrol yang
 * sama tanpa setiap halaman menulis ulang logika yang mudah salah.
 *
 * Nomor halaman disimpan apa adanya, lalu dijepit saat dipakai. Cara ini
 * menggantikan effect yang mengoreksi halaman: effect seperti itu memicu
 * render berantai, dan tetap menyisakan satu frame dengan halaman kosong.
 */
export function usePaginasiLokal<T>(data: T[] | undefined, ukuranAwal = UKURAN_BAWAAN): HasilPaginasi<T> {
  const [pageDiminta, setPageDiminta] = useState(1)
  const [pageSize, setPageSizeState] = useState(ukuranAwal)

  const total = data?.length ?? 0
  const totalPages = pageSize > 0 ? Math.ceil(total / pageSize) : 0

  // Data bisa menyusut (filter, hapus baris) sementara nomor halaman tetap.
  // Dijepit ke halaman terakhir yang benar-benar ada, minimal halaman 1.
  const page = totalPages === 0 ? 1 : Math.min(Math.max(1, pageDiminta), totalPages)

  const baris = useMemo(() => {
    if (!data || data.length === 0) return []
    const mulai = (page - 1) * pageSize
    // slice aman walau `mulai` melewati panjang data: hasilnya array kosong,
    // bukan galat, sehingga render tidak pernah gagal.
    return data.slice(mulai, mulai + pageSize)
  }, [data, page, pageSize])

  const setPage = (berikut: number) => {
    const batas = Math.max(1, totalPages)
    setPageDiminta(Math.min(Math.max(1, berikut), batas))
  }

  const setPageSize = (ukuran: number) => {
    setPageSizeState(ukuran)
    // Pembagian halaman berubah, jadi halaman 4 dari ukuran lama belum tentu
    // ada. Kembali ke halaman pertama.
    setPageDiminta(1)
  }

  return { baris, page, setPage, totalPages, total, pageSize, setPageSize }
}
