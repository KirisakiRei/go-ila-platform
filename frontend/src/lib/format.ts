/** Pemformatan angka dan waktu untuk tampilan Indonesia. */

export function formatNumber(value: number, digits = 2): string {
  if (!Number.isFinite(value)) return '—'
  return new Intl.NumberFormat('id-ID', {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
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

/** Waktu lokal pengguna; nilai kosong menjadi tanda pisah. */
export function formatDateTime(value: string | null | undefined): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return new Intl.DateTimeFormat('id-ID', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)
}

export function formatDate(value: string | null | undefined): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return new Intl.DateTimeFormat('id-ID', { dateStyle: 'medium' }).format(date)
}
