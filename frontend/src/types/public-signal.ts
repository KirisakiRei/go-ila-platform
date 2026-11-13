/**
 * Tampilan ringkas sinyal untuk halaman publik.
 *
 * Bentuk ini SENGAJA terpisah dari tipe sinyal milik pelanggan. Yang tampil di
 * sini hanya arah dan waktu — tanpa harga masuk, batas kerugian, atau target.
 * Memisahkan tipenya membuat field harga tidak bisa ikut bocor hanya karena
 * suatu saat tipe pelanggan ditambah.
 */

/** Kategori instrumen, dipakai untuk label di kartu. */
export type PublicSignalCategory = 'forex' | 'indices' | 'commodity' | 'crypto'

/** Keadaan pasar saat ini, diambil dari status layanan publik. */
export type PublicMarketState = {
  open: boolean
  /** Label siap tampil, mis. "Pasar buka". */
  label: string
}

export type PublicSignalSnapshot = {
  /** BULLISH atau BEARISH. */
  direction: string
  /** Rentang waktu sumber sinyal, mis. H4 dan M15. Kosong berarti belum diketahui. */
  htf: string
  ltf: string
  /**
   * ACTIVE berarti ada sinyal siap pakai, WATCHING berarti radar masih menunggu
   * konfirmasi, EXPIRED berarti arahnya sudah lewat.
   */
  status: string
  /** Waktu sinyal pertama terbit. Untuk baris radar nilainya beku sejak awal. */
  published_at: string
  /** Waktu status terakhir diperbarui di server; dipakai untuk menampilkan kesegaran data. */
  updated_at?: string
}

export type PublicSignalPreviewItem = {
  symbol: string
  display_name: string
  category: PublicSignalCategory
  has_signal: boolean
  signal?: PublicSignalSnapshot
}

export type PublicSignalPreview = {
  /** Waktu status ini disusun server. */
  generated_at: string
  market: PublicMarketState
  items: PublicSignalPreviewItem[]
}
