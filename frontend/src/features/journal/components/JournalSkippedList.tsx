import { EmptyState } from '@/components/ui'
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
    default:
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
    case 'TP':
      return { label: 'Akan TP', tone: 'positif' }
    case 'SL':
      return { label: 'Akan SL', tone: 'negatif' }
    case 'NO_ENTRY':
      return { label: 'Tanpa entry', tone: 'netral' }
    case 'UNKNOWN':
      return { label: 'Hasil ambigu', tone: 'netral' }
    default:
      return null
  }
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
        const pasangan = item.htf && item.ltf ? `${item.htf} → ${item.ltf}` : ''
        const hasil = hasilHipotetis(item)

        return (
          <li key={`${item.signal_id}-${item.occurred_at}`} className="flex flex-col gap-1.5 p-3">
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

            {hasil && item.eval_at && (
              <p className="text-[10.5px] text-muted-foreground">
                Dinilai {formatDateTime(item.eval_at)}
                {item.eval_note ? ` · ${item.eval_note}` : ''}
              </p>
            )}

            <p className="text-[10.5px] text-muted-foreground">{formatDateTime(item.occurred_at)}</p>
          </li>
        )
      })}
    </ul>
  )
}
