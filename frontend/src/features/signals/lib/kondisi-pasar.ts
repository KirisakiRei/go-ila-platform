import type { MarketState } from '@/types/api'
import type { MarketWindow } from '@/types/system'

/**
 * Kondisi pasar yang ditampilkan dashboard.
 *
 * `disimpulkan` dicatat supaya pengembang bisa membedakan "engine benar-benar
 * melaporkan" dari "disimpulkan dari jadwal". Pengguna tidak perlu melihatnya.
 */
export type KondisiPasar = {
  state: MarketState
  label: string
  disimpulkan: boolean
}

const LABEL: Record<MarketState, string> = {
  OPEN: 'Pasar Buka',
  CLOSED: 'Pasar Tutup',
  ROLLOVER: 'Rollover',
  UNKNOWN: 'Belum diketahui',
}

/**
 * Jumlah posisi yang diambil dashboard.
 *
 * Dipakai bersama oleh daftar posisi terkini dan ringkasan P/L hari ini supaya
 * keduanya membaca cache yang sama. Kunci cache menyertakan limit, jadi angka
 * yang berbeda berarti dua permintaan terpisah ke /positions setiap penyegaran.
 * Batas kecil seperti 5 juga tidak cukup untuk menghitung P/L satu hari penuh.
 */
export const POSISI_LIMIT_DASHBOARD = 100

/**
 * kondisiPasar menyatukan dua sumber status pasar yang berbeda.
 *
 * `/api/v1/system/status` mengirim `market.state` yang bisa `UNKNOWN` karena
 * engine tidak selalu menyegarkannya. `/api/v1/public/status` mengirim
 * `market.open` yang dihitung dari jadwal sesi nyata. Dulu dashboard hanya
 * membaca sumber pertama, sehingga satu layar bisa menampilkan "Tidak diketahui"
 * dan "Pasar buka" sekaligus — bertentangan dan membingungkan.
 *
 * Urutannya: laporan engine lebih dulu (paling spesifik, termasuk ROLLOVER),
 * lalu jadwal publik, lalu `UNKNOWN` sebagai jalan terakhir.
 */
export function kondisiPasar(
  engine: { state: MarketState; label: string } | null | undefined,
  jadwal: MarketWindow | null | undefined,
): KondisiPasar {
  const state = engine?.state

  // Rollover dan OPEN/CLOSED dari engine lebih dipercaya: engine tahu keadaan
  // sesi saat itu, sedangkan jadwal hanya tahu jam buka-tutup.
  if (state === 'ROLLOVER') {
    return { state, label: LABEL.ROLLOVER, disimpulkan: false }
  }
  if (state === 'OPEN' || state === 'CLOSED') {
    return { state, label: engine?.label || LABEL[state], disimpulkan: false }
  }

  // Engine belum tahu. Jadwal publik tetap bisa menjawab.
  if (jadwal) {
    const dariJadwal: MarketState = jadwal.open ? 'OPEN' : 'CLOSED'
    return { state: dariJadwal, label: LABEL[dariJadwal], disimpulkan: true }
  }

  return { state: 'UNKNOWN', label: LABEL.UNKNOWN, disimpulkan: false }
}

/** jamTutupMenyusul menandai apakah label jadwal berikutnya berarti "tutup". */
export function jamTutupMenyusul(jadwal: MarketWindow | null | undefined): boolean {
  if (!jadwal?.next_label) return false
  return /close|tutup/i.test(jadwal.next_label)
}

/**
 * tanggalWIB mengubah waktu ISO menjadi kunci tanggal `YYYY-MM-DD` dalam WIB.
 *
 * Server menghitung hari akuntansi (`daily.day`) dengan batas tengah malam WIB,
 * sementara timestamp posisi dikirim dalam UTC. Mengambil sepuluh karakter
 * pertama dari string UTC akan menggeser hari untuk transaksi yang terjadi
 * antara 00:00–07:00 WIB (17:00–24:00 UTC hari sebelumnya).
 */
export function tanggalWIB(iso: string | null | undefined): string {
  if (!iso) return ''
  const waktu = new Date(iso)
  if (Number.isNaN(waktu.getTime())) return ''

  // en-CA menghasilkan format YYYY-MM-DD, sama dengan bentuk `daily.day`.
  return new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Jakarta',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(waktu)
}
