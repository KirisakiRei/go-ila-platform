/**
 * Bantuan tanggal untuk backtest.
 *
 * Semua waktu dari server berupa Unix detik UTC, sedangkan `input[type=date]`
 * bekerja dengan teks "YYYY-MM-DD" di zona waktu peramban. Dua fungsi di sini
 * menjadi satu-satunya tempat penerjemahan itu, supaya tidak ada bagian lain
 * yang menebak sendiri dan meleset sehari karena perbedaan zona.
 */

const BULAN_SINGKAT = [
  'Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun',
  'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des',
]

/** formatTanggalInput mengubah Date menjadi teks untuk input tanggal. */
export function formatTanggalInput(tanggal: Date): string {
  const tahun = tanggal.getFullYear()
  const bulan = String(tanggal.getMonth() + 1).padStart(2, '0')
  const hari = String(tanggal.getDate()).padStart(2, '0')
  return `${tahun}-${bulan}-${hari}`
}

/**
 * dariTanggalInput mengubah teks input tanggal menjadi Unix detik.
 *
 * Awal hari dipilih secara sengaja: rentang backtest bersifat inklusif, dan
 * memakai tengah hari akan memotong candle pagi di hari pertama.
 */
export function dariTanggalInput(teks: string): number | null {
  if (!teks) return null
  const bagian = teks.split('-').map(Number)
  const [tahun, bulan, hari] = bagian
  if (bagian.length !== 3 || tahun === undefined || bulan === undefined || hari === undefined) {
    return null
  }
  if ([tahun, bulan, hari].some((angka) => Number.isNaN(angka))) return null
  const tanggal = new Date(tahun, bulan - 1, hari, 0, 0, 0, 0)
  return Math.floor(tanggal.getTime() / 1000)
}

/** formatTanggalPanjang menulis tanggal seperti "18 Okt 2026". */
export function formatTanggalPanjang(unixDetik: number): string {
  if (!unixDetik) return '—'
  const tanggal = new Date(unixDetik * 1000)
  return `${tanggal.getDate()} ${BULAN_SINGKAT[tanggal.getMonth()]} ${tanggal.getFullYear()}`
}

/** formatWaktu menulis tanggal dan jam untuk kolom tabel. */
export function formatWaktu(unixDetik: number): string {
  if (!unixDetik) return '—'
  const tanggal = new Date(unixDetik * 1000)
  const jam = String(tanggal.getHours()).padStart(2, '0')
  const menit = String(tanggal.getMinutes()).padStart(2, '0')
  return `${tanggal.getDate()} ${BULAN_SINGKAT[tanggal.getMonth()]} ${jam}:${menit}`
}

/** formatDurasi menulis lama posisi terbuka dalam jam dan hari. */
export function formatDurasi(detik: number): string {
  if (detik <= 0) return '—'
  if (detik < 3600) return `${Math.round(detik / 60)} menit`
  if (detik < 86_400) return `${Math.round(detik / 3600)} jam`
  return `${Math.round(detik / 86_400)} hari`
}

/** hitungBulan memberi gambaran kasar panjang data, bukan angka resmi. */
export function hitungBulan(mulaiUnix: number, selesaiUnix: number): number {
  if (!mulaiUnix || !selesaiUnix) return 0
  const hari = (selesaiUnix - mulaiUnix) / 86_400
  return Math.max(1, Math.round(hari / 30))
}
