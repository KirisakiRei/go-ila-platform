/**
 * Nama halaman publik untuk keadaan kosong.
 *
 * Hanya nama yang tinggal di sini; seluruh isi halaman berasal dari server.
 */
const TITLES: Record<string, string> = {
  fitur: 'Fitur',
  harga: 'Harga',
  tentang: 'Tentang',
  bantuan: 'Pusat Bantuan',
  legal: 'Dokumen Legal',
}

/** pageTitle mengembalikan nama halaman, atau judul umum bila tidak dikenal. */
export function pageTitle(slug: string): string {
  return TITLES[slug] ?? 'Halaman'
}
