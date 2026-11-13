import { ArrowDownRight, ArrowUpRight, Minus } from 'lucide-react'
import { cn } from '@/lib/utils'
import { formatRelativeTime } from '@/lib/format'
import type { PublicSignalPreviewItem } from '@/types/public-signal'

/**
 * Kartu ringkas pantauan pasar untuk halaman publik.
 *
 * Aturan yang dipegang:
 *  - Hanya arah dan waktu. Tidak ada harga masuk, batas kerugian, maupun target
 *    — itu bagian berlangganan, dan menampilkannya di sini sama dengan
 *    membagikan isi produk secara gratis.
 *  - Tidak ada istilah internal. Yang terbaca pengunjung adalah bahasa pasar
 *    sehari-hari, bukan nama modul, versi strategi, atau kode sinyal.
 *  - Keadaan kosong dirancang sebaik keadaan terisi. Engine bisa saja belum
 *    mengirim apa pun, dan kartu tetap harus terlihat hidup, bukan rusak.
 *  - Kedua keadaan memakai kerangka tinggi yang sama agar grid tidak melompat
 *    saat data mulai masuk.
 */

/** Label kategori dalam bahasa pembaca, bukan kode internal. */
const CATEGORY_LABEL: Record<string, string> = {
  forex: 'Forex',
  indices: 'Indeks',
  commodity: 'Logam & Komoditas',
  crypto: 'Kripto',
}

/** Nama instrumen yang dikenal pembaca, dengan simbol sebagai cadangan. */
const INSTRUMENT_NAME: Record<string, string> = {
  XAUUSD: 'Emas vs Dolar AS',
  EURUSD: 'Euro vs Dolar AS',
  NAS100: 'Indeks Teknologi AS',
  USTEC: 'Indeks Teknologi AS',
}

type Arah = 'bullish' | 'bearish' | 'unknown'

function arahDari(direction: string | undefined): Arah {
  const nilai = (direction ?? '').toUpperCase()
  if (nilai === 'BULLISH' || nilai === 'BUY' || nilai === 'LONG') return 'bullish'
  if (nilai === 'BEARISH' || nilai === 'SELL' || nilai === 'SHORT') return 'bearish'
  return 'unknown'
}

const ARAH_TAMPILAN: Record<
  Arah,
  {
    label: string
    icon: typeof ArrowUpRight
    cardBorder: string
    bgClass: string
    badgeClass: string
    textClass: string
    iconClass: string
  }
> = {
  bullish: {
    label: 'Kecenderungan naik',
    icon: ArrowUpRight,
    cardBorder: 'border-emerald-300 hover:border-emerald-400',
    bgClass: 'bg-emerald-50/40',
    badgeClass: 'border-emerald-600 bg-emerald-600 text-white',
    textClass: 'text-emerald-700',
    iconClass: 'text-emerald-600',
  },
  bearish: {
    label: 'Kecenderungan turun',
    icon: ArrowDownRight,
    cardBorder: 'border-rose-300 hover:border-rose-400',
    bgClass: 'bg-rose-50/40',
    badgeClass: 'border-rose-600 bg-rose-600 text-white',
    textClass: 'text-rose-700',
    iconClass: 'text-rose-600',
  },
  unknown: {
    label: 'Belum ada arah',
    icon: Minus,
    cardBorder: 'border-zinc-200 hover:border-zinc-300',
    bgClass: 'bg-white',
    badgeClass: 'border-zinc-300 bg-transparent text-[#3f3f46]',
    textClass: 'text-[#09090b]',
    iconClass: 'text-[#09090b]',
  },
}

/** Baris kecil berlabel di kaki kartu. */
function MetaRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-baseline justify-between gap-3">
      <span className="text-[11px] uppercase tracking-[0.08em] text-[#71717a]">{label}</span>
      <span className="text-[12px] font-medium text-[#3f3f46]">{value}</span>
    </div>
  )
}

/** Penanda keadaan di pojok kanan atas kartu. */
function StatusTag({
  hasSignal,
  marketOpen,
  badgeClass,
}: {
  hasSignal: boolean
  marketOpen: boolean
  badgeClass: string
}) {
  const gaya = !marketOpen
    ? 'border-zinc-200 bg-transparent text-[#71717a]'
    : hasSignal
      ? badgeClass
      : 'border-zinc-300 bg-transparent text-[#3f3f46]'

  const teks = !marketOpen ? 'Pasar tutup' : hasSignal ? 'Sinyal Aktif' : 'Dipantau'

  // Titik hidup hanya berdenyut saat pasar benar-benar sedang dipantau. Saat
  // pasar tutup atau sinyal sudah terkunci, titik berhenti bergerak supaya
  // gerakan tidak dipakai pada keadaan yang tidak berubah sendiri.
  const keadaanTitik = !marketOpen ? 'statis' : hasSignal ? 'penuh' : 'hidup'

  return (
    <span
      className={cn(
        'inline-flex shrink-0 items-center gap-1.5 whitespace-nowrap rounded-md border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-[0.07em]',
        gaya,
      )}
    >
      <span
        aria-hidden="true"
        className={cn(
          'status-dot',
          keadaanTitik === 'hidup' && 'status-live',
          keadaanTitik === 'statis' && 'opacity-40',
        )}
      />
      {teks}
    </span>
  )
}

export function SignalOverviewCard({
  item,
  marketOpen,
  marketLabel,
  now,
}: {
  item: PublicSignalPreviewItem
  marketOpen: boolean
  marketLabel: string
  now?: Date
}) {
  const nama = INSTRUMENT_NAME[item.symbol] ?? item.display_name ?? item.symbol
  const kategori = CATEGORY_LABEL[item.category] ?? item.display_name ?? ''

  const sinyal = item.has_signal ? item.signal : undefined
  const arah = arahDari(sinyal?.direction)
  const tampilan = ARAH_TAMPILAN[arah]
  const Icon = tampilan.icon

  // Rentang waktu hanya ditulis bila kedua sisi tersedia; setengah informasi
  // ("H4 /") lebih membingungkan daripada tidak menampilkannya sama sekali.
  const rentang = sinyal?.htf && sinyal?.ltf ? `${sinyal.htf} / ${sinyal.ltf}` : null
  const umur = sinyal ? formatRelativeTime(sinyal.published_at, now) : null

  return (
    <article
      className={cn(
        'hover-lift flex h-full flex-col rounded-xl border p-5',
        sinyal ? tampilan.cardBorder : 'border-zinc-200 hover:border-zinc-300',
        sinyal ? tampilan.bgClass : 'bg-white',
      )}
    >
      {/* Kepala: simbol, kategori, dan penanda keadaan. */}
      <header className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h3 className="truncate text-[17px] font-semibold leading-tight tracking-tight text-[#09090b]">
            {item.symbol}
          </h3>
          <p className="mt-1 truncate text-[12px] text-[#71717a]">
            {nama}
            {kategori && <span className="text-zinc-400"> · {kategori}</span>}
          </p>
        </div>
        <StatusTag
          hasSignal={Boolean(sinyal)}
          marketOpen={marketOpen}
          badgeClass={tampilan.badgeClass}
        />
      </header>

      {/* Isi: perbedaan utama antara keadaan terisi dan belum terisi. */}
      <div className="mt-5 flex min-h-[76px] flex-col justify-center border-t border-zinc-200/60 pt-4">
        {sinyal ? (
          <>
            <div className="flex items-center gap-2">
              <Icon
                className={cn('h-4 w-4 shrink-0', tampilan.iconClass)}
                aria-hidden="true"
                strokeWidth={2.5}
              />
              <p
                className={cn(
                  'text-[15px] font-semibold leading-tight tracking-tight',
                  tampilan.textClass,
                )}
              >
                {tampilan.label}
              </p>
            </div>
            <p className="mt-1.5 text-[12px] leading-relaxed text-[#52525b]">
              {rentang ? `Timeframe ${rentang}` : 'Timeframe belum tersedia'}
            </p>
          </>
        ) : (
          <>
            <p className="text-[15px] font-semibold leading-tight tracking-tight text-[#09090b]">
              Sedang dipantau
            </p>
            <p className="mt-1.5 text-[12px] leading-relaxed text-[#52525b]">
              Belum ada arah terkunci. Kartu ini berubah begitu struktur terkonfirmasi.
            </p>
          </>
        )}
      </div>

      {/* Kaki: waktu dan keadaan pasar. */}
      <footer className="mt-5 space-y-1.5 border-t border-zinc-200/60 pt-4">
        {umur ? (
          <MetaRow label="Diperbarui" value={umur} />
        ) : (
          <MetaRow label="Status" value="Menunggu konfirmasi" />
        )}
        <MetaRow label="Pasar" value={marketLabel} />
      </footer>
    </article>
  )
}

/** Bentuk kerangka saat data pertama masih dimuat. */
export function SignalOverviewCardSkeleton() {
  return (
    <div className="flex h-full flex-col rounded-xl border border-zinc-200 bg-white p-5" aria-hidden="true">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="h-4 w-20 animate-pulse rounded bg-zinc-200" />
          <div className="mt-2 h-3 w-32 animate-pulse rounded bg-zinc-100" />
        </div>
        <div className="h-4 w-16 animate-pulse rounded bg-zinc-100" />
      </div>
      <div className="mt-5 min-h-[76px] border-t border-zinc-100 pt-4">
        <div className="h-4 w-40 animate-pulse rounded bg-zinc-200" />
        <div className="mt-2 h-3 w-28 animate-pulse rounded bg-zinc-100" />
      </div>
      <div className="mt-5 space-y-2 border-t border-zinc-100 pt-4">
        <div className="h-3 w-full animate-pulse rounded bg-zinc-100" />
        <div className="h-3 w-3/4 animate-pulse rounded bg-zinc-100" />
      </div>
    </div>
  )
}
