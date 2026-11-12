/** Pemformatan angka dan waktu untuk tampilan Indonesia. */

export function formatNumber(value: number, digits = 2): string {
  if (!Number.isFinite(value)) return '—'
  return new Intl.NumberFormat('id-ID', {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  }).format(value)
}

/**
 * formatPrice menampilkan harga pair dengan presisi penuh.
 *
 * Berbeda dari formatNumber, pemisah ribuan tidak dipakai: harga chart dibaca
 * dan disalin sebagai level (4051.31), dan titik ribuan akan menyulitkan
 * pembacaan sekaligus membuatnya tidak cocok saat ditempel ke terminal broker.
 * Pemisah desimal mengikuti locale aplikasi (koma).
 */
export function formatPrice(value: number, digits: number): string {
  if (!Number.isFinite(value)) return '—'
  return new Intl.NumberFormat('id-ID', {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
    useGrouping: false,
  }).format(value)
}

export function formatMoney(value: number): string {
  if (!Number.isFinite(value)) return '—'
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 2,
  }).format(value)
}

export function formatR(value: number): string {
  if (!Number.isFinite(value)) return '—'
  return `${value > 0 ? '+' : ''}${value.toFixed(2)}R`
}

/** Waktu standar WIB; nilai kosong menjadi tanda pisah. */
export function formatDateTime(value: string | null | undefined): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return `${new Intl.DateTimeFormat('id-ID', {
    dateStyle: 'medium',
    timeStyle: 'short',
    timeZone: 'Asia/Jakarta',
  }).format(date)} WIB`
}

/** Menghasilkan format jam New York (NY), misal "12:51 NY" */
export function formatNYTime(value: string | null | undefined): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return `${new Intl.DateTimeFormat('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone: 'America/New_York',
  }).format(date)} NY`
}

/** Menghasilkan waktu ganda: "10 Nov 2026, 00.51 WIB · 12:51 NY" */
export function formatDualDateTime(value: string | null | undefined): string {
  if (!value) return '—'
  const wib = formatDateTime(value)
  const ny = formatNYTime(value)
  if (wib === '—') return '—'
  return `${wib} · ${ny}`
}

export function formatDate(value: string | null | undefined): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return new Intl.DateTimeFormat('id-ID', { dateStyle: 'medium' }).format(date)
}

/**
 * Jarak waktu lampau dalam bahasa sehari-hari, mis. "12 menit lalu".
 *
 * Dipakai halaman publik untuk menuliskan kapan sesuatu terakhir terjadi tanpa
 * menampilkan tanggal penuh. Hasilnya selalu satu satuan, karena dua satuan
 * ("1 hari 3 jam lalu") membuat kartu tidak rata dan sulit dipindai.
 *
 * Waktu yang belum lewat (selisih negatif, mis. jam server sedikit di depan)
 * diperlakukan sebagai "baru saja", bukan tanggal masa depan.
 */
export function formatRelativeTime(value: string | null | undefined, now?: Date): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'

  const acuan = now ?? new Date()
  const detik = Math.floor((acuan.getTime() - date.getTime()) / 1000)
  if (detik < 60) return 'baru saja'

  const menit = Math.floor(detik / 60)
  if (menit < 60) return `${menit} menit lalu`

  const jam = Math.floor(menit / 60)
  if (jam < 24) return `${jam} jam lalu`

  const hari = Math.floor(jam / 24)
  if (hari < 30) return `${hari} hari lalu`

  const bulan = Math.floor(hari / 30)
  if (bulan < 12) return `${bulan} bulan lalu`

  return `${Math.floor(bulan / 12)} tahun lalu`
}
