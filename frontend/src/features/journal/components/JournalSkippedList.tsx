import { useState } from 'react'
import { ChevronDown } from 'lucide-react'
import { EmptyState } from '@/components/ui'
import { SkippedEvidenceChart } from '@/features/charts/SkippedEvidenceChart'
import { formatDateTime, formatNumber, formatR } from '@/lib/format'
import type { JournalSkipped } from '@/types/api'

/**
 * labelStatus menerjemahkan status user_signal menjadi bahasa trader.
 *
 * Status mentah seperti REJECTED tidak menjelaskan apa pun kepada pengguna.
 * Yang ia butuhkan: apakah peluangnya lewat karena harga sudah bergerak, karena
 * masa berlaku habis, atau karena ia sendiri melewatinya.
 */
function labelStatus(status: string): string {
  switch (status) {
    case 'REJECTED':
      return 'Tidak tereksekusi'
    case 'SKIPPED':
      return 'Dilewati'
    case 'EXPIRED':
      return 'Kedaluwarsa'
    case 'OPPORTUNITY':
      return 'Menunggu keputusan'
    case 'READY_TO_ENTRY':
      return 'Siap entry'
    case 'WAIT_HTF_SWEEP':
      return 'Tahap sweep HTF'
    case 'WAIT_HTF_CANDLE_CLOSE':
      return 'Tahap tutup candle HTF'
    case 'WAIT_LTF_CISD':
      return 'Tahap CISD LTF'
    default:
      // Status radar ILA 3.0 memakai prefiks yang panjang dan teknis; cukup
      // ditampilkan sebagai tahap berjalan agar tidak membingungkan.
      if (status.startsWith('WAIT_ILA3_')) return 'Tahap radar ILA 3.0'
      return status
  }
}

/**
 * alasanTerbaca mengubah alasan teknis menjadi kalimat yang bisa ditindaklanjuti.
 *
 * Pesan slippage dari Runner menyebut harga sudah bergerak jauh dari level
 * entry. Tanpa penjelasan tambahan, pengguna menyimpulkan sistem rusak padahal
 * setup-nya memang sudah tidak layak dimasuki.
 */
function alasanTerbaca(item: JournalSkipped): string {
  // Alasan penolakan adalah kondisi saat sinyal tiba, bukan kesimpulan akhir.
  // Begitu hasil evaluasi harga ada, kondisinya sudah lampau dan bisa
  // bertentangan dengan hasilnya (mis. "Pasar sedang tutup" pada sinyal yang
  // ternyata dinilai memakai candle sesi buka). Hasil evaluasi yang berlaku.
  if (item.eval_status && item.eval_status !== 'PENDING' && item.eval_note) {
    return item.eval_note
  }
  if (item.reason) return item.reason
  switch (item.status) {
    case 'OPPORTUNITY':
      return 'Eksekusi otomatis tidak aktif untuk pasangan timeframe ini. Buka sinyal untuk memutuskan sendiri.'
    case 'READY_TO_ENTRY':
      return 'Menunggu keputusan Anda sebelum masa berlakunya habis.'
    case 'EXPIRED':
      return 'Masa berlaku sinyal habis sebelum dieksekusi.'
    default:
      return ''
  }
}

/**
 * hasilHipotetis menerjemahkan hasil evaluasi harga menjadi label singkat.
 *
 * Setup yang tidak dieksekusi tetap punya nasib: jika harganya benar-benar
 * mencapai TP, peluangnya nyata; jika hanya menyentuh SL, melewatinya adalah
 * keputusan yang tepat. Tanpa label ini pengguna tidak bisa menilai itu.
 */
function hasilHipotetis(item: JournalSkipped): { label: string; tone: 'positif' | 'negatif' | 'netral' } | null {
  switch (item.eval_status) {
    case 'RUNNING':
      return { label: 'Sedang berjalan', tone: 'netral' }
    case 'TP':
      return {
        label: item.eval_r_lower ? `Tembus TP (+${formatR(item.eval_r_lower)})` : 'Tembus TP',
        tone: 'positif',
      }
    case 'SL':
      return { label: 'Tembus SL (-1R)', tone: 'negatif' }
    case 'NO_ENTRY':
      return { label: 'Harga tak sampai entry', tone: 'netral' }
    case 'UNKNOWN':
      return { label: 'Hasil ambigu', tone: 'netral' }
    default:
      // Tahap radar awal belum punya level entry, jadi TP/SL memang tidak bisa
      // dinilai. Ditampilkan apa adanya agar baris tidak terlihat kosong.
      if (!(item.entry_price > 0) && item.status.startsWith('WAIT')) {
        return { label: 'Belum ada level entry', tone: 'netral' }
      }
      return null
  }
}

/**
 * durasiRingkas menghitung selisih dua waktu menjadi teks pendek.
 *
 * Durasi sengaja dihitung saat ditampilkan, bukan disimpan: menyimpannya akan
 * membuat nilainya basi untuk setup yang masih berjalan.
 */
function durasiRingkas(mulai: string, akhir: string): string {
  const ms = new Date(akhir).getTime() - new Date(mulai).getTime()
  if (!Number.isFinite(ms) || ms <= 0) return ''
  const menit = Math.floor(ms / 60_000)
  if (menit < 60) return `${menit} menit`
  const jam = Math.floor(menit / 60)
  if (jam < 24) return `${jam} jam ${menit % 60} menit`
  return `${Math.floor(jam / 24)} hari ${jam % 24} jam`
}

const TONE_CLASS: Record<'positif' | 'negatif' | 'netral', string> = {
  positif: 'border-foreground bg-foreground text-background',
  negatif: 'border-foreground text-foreground',
  netral: 'border-border text-muted-foreground',
}

/**
 * JournalSkippedList menampilkan peluang yang tidak berakhir sebagai posisi.
 *
 * Bagian ini penting untuk kejujuran jurnal: tanpa daftar ini, pengguna hanya
 * melihat trade yang jadi dan tidak tahu berapa banyak setup yang terlewat.
 */
export function JournalSkippedList({ items }: { items: JournalSkipped[] }) {
  // Hanya satu baris yang membuka chart, supaya membuka beberapa sekaligus
  // tidak memicu banyak permintaan candle dari feed yang sama.
  const [terbuka, setTerbuka] = useState<string | null>(null)
  if (items.length === 0) {
    return (
      <EmptyState
        title="Tidak ada peluang terlewat"
        description="Semua sinyal pada periode ini berakhir sebagai posisi atau masih menunggu."
      />
    )
  }

  return (
    <ul className="divide-y divide-border rounded-xl border border-border bg-card">
      {items.map((item) => {
        const kunci = `${item.signal_id}-${item.occurred_at}`
        const pasangan = item.htf && item.ltf ? `${item.htf} → ${item.ltf}` : ''
        const hasil = hasilHipotetis(item)

        return (
          <li key={kunci} className="flex flex-col gap-1.5 p-3">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div className="flex flex-wrap items-center gap-2">
                <span className="font-semibold">{item.symbol || '—'}</span>
                {item.direction && (
                  <span className="text-[11px] font-medium uppercase text-muted-foreground">
                    {item.direction}
                  </span>
                )}
                {pasangan && (
                  <span className="text-[11px] text-muted-foreground">{pasangan}</span>
                )}
                {item.strategy_version && (
                  <span className="text-[11px] text-muted-foreground">{item.strategy_version}</span>
                )}
              </div>
              <div className="flex items-center gap-2">
                {hasil && (
                  <span
                    className={`rounded border px-1.5 py-0.5 text-[10.5px] font-medium ${TONE_CLASS[hasil.tone]}`}
                  >
                    {hasil.label}
                  </span>
                )}
                <span className="text-[11px] font-medium">{labelStatus(item.status)}</span>
              </div>
            </div>

            {item.entry_price > 0 && (
              <p className="tabular text-[11px] text-muted-foreground">
                Entry {formatNumber(item.entry_price, 5)} · SL {formatNumber(item.stop_loss, 5)} · TP{' '}
                {formatNumber(item.take_profit, 5)}
                {item.risk_reward > 0 ? ` · RR ${formatR(item.risk_reward)}` : ''}
              </p>
            )}

            {alasanTerbaca(item) && (
              <p className="text-[11px] text-muted-foreground">{alasanTerbaca(item)}</p>
            )}

            {hasil && item.eval_entry_at && (
              <p className="tabular text-[10.5px] text-muted-foreground">
                Entry {formatDateTime(item.eval_entry_at)}
                {item.eval_at && (
                  <>
                    {' · Exit '}
                    {formatDateTime(item.eval_at)}
                    {durasiRingkas(item.eval_entry_at, item.eval_at)
                      ? ` · ${durasiRingkas(item.eval_entry_at, item.eval_at)}`
                      : ''}
                  </>
                )}
                {!item.eval_at && item.eval_status === 'RUNNING' ? ' · belum keluar' : ''}
              </p>
            )}


            <p className="text-[10.5px] text-muted-foreground">{formatDateTime(item.occurred_at)}</p>
            {item.entry_price > 0 && (
              <>
                <button
                  type="button"
                  onClick={() => setTerbuka(kunci === terbuka ? null : kunci)}
                  aria-expanded={terbuka === kunci}
                  className="inline-flex w-fit items-center gap-1 rounded-md border border-border px-2 py-1 text-[10.5px] font-semibold text-foreground transition-colors duration-150 hover:bg-secondary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.98]"
                >
                  <ChevronDown
                    className={`h-3 w-3 transition-transform duration-150 ${terbuka === kunci ? 'rotate-180' : ''}`}
                  />
                  {terbuka === kunci ? 'Tutup bukti harga' : 'Lihat bukti harga'}
                </button>
                {terbuka === kunci && (
                  <SkippedEvidenceChart item={item} enabled height={280} />
                )}
              </>
            )}
          </li>
        )
      })}
    </ul>
  )
}
