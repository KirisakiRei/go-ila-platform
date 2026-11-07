/**
 * formatInvoice memformat rupiah untuk dokumen dan laporan.
 * Selalu tanpa desimal: harga paket dalam rupiah bulat.
 */
export function formatIDR(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return '—'
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(value)
}

/**
 * formatInvoice meneruskan string waktu WIB yang sudah diformat server.
 * Server yang menghitung zona, bukan browser (plans/22 §6.3).
 */
export function formatInvoice(wib: string | undefined): string {
  return wib && wib.trim() !== '' ? wib : '—'
}

/** formatPercent menampilkan potongan kupon dalam persen. */
export function formatPercent(value: number): string {
  return `${new Intl.NumberFormat('id-ID', { maximumFractionDigits: 2 }).format(value)}%`
}
