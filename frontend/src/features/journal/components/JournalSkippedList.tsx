import { useState } from 'react'
import { ChevronDown } from 'lucide-react'
import { EmptyState } from '@/components/ui'
import { SkippedEvidenceChart } from '@/features/charts/SkippedEvidenceChart'
import { formatDateTime, formatPrice, formatR } from '@/lib/format'
import { priceDigits } from '@/lib/pricePrecision'
import { useIsMobile } from '@/lib/use-mobile-viewport'
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
function hasilHipotetis(item: JournalSkipped): {
  label: string
  tone: 'positif' | 'negatif' | 'netral'
} {
  switch (item.eval_status) {
    case 'PENDING':
      return { label: 'Menunggu penilaian', tone: 'netral' }
    case 'RUNNING':
      return { label: 'Sedang berjalan', tone: 'netral' }
    case 'TP':
      return {
        label:
          item.eval_r_lower != null && Number.isFinite(item.eval_r_lower)
            ? `TP (${formatR(item.eval_r_lower)})`
            : 'TP',
        tone: 'positif',
      }
    case 'SL':
      return { label: 'SL (-1R)', tone: 'negatif' }
    case 'NO_ENTRY':
      return { label: 'Tidak masuk', tone: 'netral' }
    case 'UNKNOWN':
      return { label: 'Tidak dapat dipastikan', tone: 'netral' }
    default:
      // Tahap radar awal belum punya level entry, jadi TP/SL memang tidak bisa
      // dinilai. Ditampilkan apa adanya agar baris tidak terlihat kosong.
      if (!(item.entry_price > 0) && item.status.startsWith('WAIT')) {
        return { label: 'Belum ada level entry', tone: 'netral' }
      }
      return { label: 'Menunggu penilaian', tone: 'netral' }
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
  if (!Number.isFinite(ms) || ms < 0) return ''
  if (ms < 60_000) return '<1 menit'
  const menit = Math.floor(ms / 60_000)
  if (menit < 60) return `${menit} menit`
  const jam = Math.floor(menit / 60)
  if (jam < 24) return `${jam} jam ${menit % 60} menit`
  return `${Math.floor(jam / 24)} hari ${jam % 24} jam`
}

function evidenceObject(value: unknown): Record<string, unknown> {
  try {
    const parsed: unknown = typeof value === 'string' ? JSON.parse(value) : value
    return parsed !== null && typeof parsed === 'object' && !Array.isArray(parsed)
      ? (parsed as Record<string, unknown>)
      : {}
  } catch {
    return {}
  }
}

function buktiSimulasi(value: string | undefined, digits: number) {
  const envelope = evidenceObject(value)
  const evaluation = envelope.evaluation == null ? envelope : evidenceObject(envelope.evaluation)
  const teoritis =
    envelope.entry_basis != null
      ? envelope.entry_basis === 'signal_level_at_publication'
      : evaluation.method === 'signal_forward_v1'
  const candle = evidenceObject(evaluation.candle)
  const validPrice = (price: unknown): price is number =>
    typeof price === 'number' && Number.isFinite(price) && price > 0
  const prices = [candle.open, candle.high, candle.low, candle.close]
  const validTime =
    typeof candle.time === 'number' &&
    Number.isFinite(candle.time) &&
    candle.time > 0 &&
    Number.isFinite(new Date(candle.time * 1000).getTime())
  const validCandle =
    validTime &&
    prices.every(validPrice) &&
    Number(candle.high) >=
      Math.max(Number(candle.open), Number(candle.close), Number(candle.low)) &&
    Number(candle.low) <= Math.min(Number(candle.open), Number(candle.close))
  return {
    teoritis,
    candle: validCandle
      ? `Candle tersimpan: ${formatDateTime(new Date(Number(candle.time) * 1000).toISOString())} · O ${formatPrice(Number(candle.open), digits)} · H ${formatPrice(Number(candle.high), digits)} · L ${formatPrice(Number(candle.low), digits)} · C ${formatPrice(Number(candle.close), digits)}`
      : null,
    incomplete: evaluation.incomplete === true,
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
  const isMobile = useIsMobile()
  // Hanya satu baris yang membuka chart, supaya membuka beberapa sekaligus
  // tidak memicu banyak permintaan candle dari feed yang sama.
  const [terbuka, setTerbuka] = useState<string | null>(null)
  if (items.length === 0) {
    return (
      <EmptyState
        title="Tidak ada peluang terlewat"
        description="Tidak ada sinyal tidak tereksekusi yang ditemukan untuk filter ini."
      />
    )
  }

  return (
    <section aria-label="Simulasi sinyal tidak tereksekusi" className="space-y-3">
      <p className="text-xs text-muted-foreground">
        Simulasi hasil sinyal yang tidak menjadi transaksi akun. Bukan PnL akun.
      </p>
      <ul className="divide-y divide-border rounded-xl border border-border bg-card">
        {items.map((item) => {
          const kunci = `${item.signal_id}-${item.occurred_at}`
          const pasangan = item.htf && item.ltf ? `${item.htf} → ${item.ltf}` : ''
          const hasil = hasilHipotetis(item)
          const alasan = alasanTerbaca(item)
          const digits = priceDigits(item.symbol, item.entry_price)
          const selesai = item.eval_status === 'TP' || item.eval_status === 'SL'
          const hargaExit = item.eval_exit_price ?? item.eval_price
          const bukti = buktiSimulasi(item.eval_evidence, digits)
          const teoritis = bukti.teoritis
          const hargaEntry = teoritis ? item.entry_price : item.eval_entry_price
          // Durasi mengikuti data evaluasi terakhir, bukan menganggap feed masih berjalan.
          const akhirDurasi = selesai
            ? item.eval_at
            : item.eval_status === 'RUNNING'
              ? item.eval_checked_at
              : undefined
          const durasi =
            item.eval_entry_at && akhirDurasi ? durasiRingkas(item.eval_entry_at, akhirDurasi) : ''

          return (
            <li key={kunci} className="flex min-w-0 flex-col gap-2 break-words p-3 sm:p-4">
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
                    <span className="text-[11px] text-muted-foreground">
                      {item.strategy_version}
                    </span>
                  )}
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  {
                    <span
                      className={`rounded border px-1.5 py-0.5 text-[10.5px] font-medium ${TONE_CLASS[hasil.tone]}`}
                    >
                      Simulasi · {hasil.label}
                    </span>
                  }
                </div>
              </div>

              {item.entry_price > 0 && (
                <p className="tabular text-[11px] text-muted-foreground">
                  Rencana entry {formatPrice(item.entry_price, digits)} · SL{' '}
                  {formatPrice(item.stop_loss, digits)} · TP {formatPrice(item.take_profit, digits)}
                  {item.risk_reward > 0 ? ` · RR ${formatR(item.risk_reward)}` : ''}
                </p>
              )}

              <p className="text-xs text-muted-foreground">
                Akun: <span>{labelStatus(item.status)}</span>
                {alasan && (
                  <>
                    {' '}
                    · <span>{alasan}</span>
                  </>
                )}
              </p>
              {(!isMobile || kunci === terbuka) && (
                <>
                  {item.eval_note && (
                    <p className="text-xs text-muted-foreground">Catatan simulasi: {item.eval_note}</p>
                  )}
                  <div className="grid gap-1 text-xs text-muted-foreground sm:grid-cols-2">
                    <p className="tabular">Waktu sinyal: {formatDateTime(item.occurred_at)}</p>
                    <p className="tabular">
                      {teoritis ? 'Entry simulasi (teoritis)' : 'Entry simulasi'}:{' '}
                      {formatDateTime(item.eval_entry_at)}
                      {hargaEntry != null && hargaEntry > 0
                        ? ` · Harga ${formatPrice(hargaEntry, digits)}`
                        : ''}
                    </p>
                    {teoritis && (
                      <p>
                        Entry pada level sinyal saat publikasi, bukan fill broker atau waktu retest.
                      </p>
                    )}
                    <p className="tabular">
                      Exit simulasi: {selesai ? formatDateTime(item.eval_at) : '—'}
                      {selesai && hargaExit != null && hargaExit > 0
                        ? ` · Harga ${formatPrice(hargaExit, digits)}`
                        : ''}
                    </p>
                    {item.eval_checked_at && (
                      <p className="tabular">
                        Terakhir dinilai: {formatDateTime(item.eval_checked_at)}
                      </p>
                    )}
                    {item.eval_status === 'RUNNING' && (
                      <p>Durasi berjalan: {durasi || '—'} · hingga penilaian terakhir, belum keluar</p>
                    )}
                    {selesai && durasi && <p>Durasi simulasi: {durasi}</p>}
                  </div>
                </>
              )}
              {isMobile && !(item.entry_price > 0) && (
                <p className="tabular text-[11px] text-muted-foreground">
                  Waktu sinyal: {formatDateTime(item.occurred_at)}
                </p>
              )}
              {item.entry_price > 0 && (
                <>
                  <button
                    type="button"
                    onClick={() => setTerbuka(kunci === terbuka ? null : kunci)}
                    aria-expanded={terbuka === kunci}
                    aria-controls={`bukti-${kunci}`}
                    className="inline-flex w-fit items-center gap-1 rounded-md border border-border px-2 py-1 text-[10.5px] font-semibold text-foreground transition-colors duration-150 hover:bg-secondary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring motion-safe:active:scale-[0.98] motion-reduce:transition-none"
                  >
                    <ChevronDown
                      aria-hidden="true"
                      className={`h-3 w-3 motion-reduce:transition-none transition-transform duration-150 ${terbuka === kunci ? 'rotate-180' : ''}`}
                    />
                    {terbuka === kunci ? 'Tutup bukti harga' : 'Lihat bukti harga'}
                  </button>
                  {terbuka === kunci && (
                    <div id={`bukti-${kunci}`} className="min-w-0">
                      <div className="mb-2 space-y-1 text-xs text-muted-foreground">
                        <p className="tabular">
                          {bukti.candle ?? 'Bukti candle tersimpan tidak tersedia.'}
                        </p>
                        {bukti.incomplete && (
                          <p>Bukti tidak lengkap; urutan harga tidak dapat dipastikan.</p>
                        )}
                      </div>
                      <SkippedEvidenceChart item={item} enabled height={280} />
                    </div>
                  )}
                </>
              )}
            </li>
          )
        })}
      </ul>
    </section>
  )
}
