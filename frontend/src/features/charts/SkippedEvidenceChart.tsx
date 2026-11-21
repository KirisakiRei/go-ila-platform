import { useMemo, useState } from 'react'
import { Skeleton } from '@/components/ui'
import { useMarketCandles } from '@/lib/queries'
import { formatNumber, formatPrice } from '@/lib/format'
import { priceDigits } from '@/lib/pricePrecision'
import type { JournalSkipped } from '@/types/api'
import { PriceChart, type ChartMarkerSpec, type PriceLineSpec } from './PriceChart'
import { useChartTheme } from './lib/theme'

type SkippedEvidenceChartProps = {
  item: JournalSkipped
  /** false menunda permintaan candle selama panel belum dibuka. */
  enabled?: boolean
  height?: number
}

const BAR_SECONDS: Record<string, number> = {
  M5: 5 * 60,
  M15: 15 * 60,
  '5M': 5 * 60,
  '15M': 15 * 60,
  H1: 60 * 60,
  H4: 4 * 60 * 60,
  D1: 24 * 60 * 60,
}

/** jumlahCandle menentukan jendela yang masih memuat titik masuk dan keluar. */
function jumlahCandle(mulai: number, akhir: number, bar: number): number {
  if (mulai <= 0 || bar <= 0) return 300
  const rentang = Math.max(0, akhir - mulai) / bar
  // Sisakan ruang kiri untuk melihat pergerakan sebelum entry, dan batasi atas
  // agar satu jendela tidak menarik ribuan candle dari feed.
  return Math.min(900, Math.max(300, Math.ceil(rentang) + 120))
}

function toUnix(value: string | null | undefined): number {
  if (!value) return 0
  const ms = new Date(value).getTime()
  return Number.isFinite(ms) ? Math.floor(ms / 1000) : 0
}

/**
 * SkippedEvidenceChart menampilkan bukti candle untuk satu setup yang tidak
 * dieksekusi: level entry, SL, TP, dan kapan harga menyentuhnya.
 *
 * Seluruh titik berasal dari angka yang tersimpan (entry_price, stop_loss,
 * take_profit, eval_entry_at, eval_at). Tidak ada harga yang dihitung ulang di
 * sini, supaya gambar dan label pada jurnal selalu menunjukkan hal yang sama.
 */
export function SkippedEvidenceChart({
  item,
  enabled = true,
  height = 320,
}: SkippedEvidenceChartProps) {
  const theme = useChartTheme()
  // Presisi mengikuti pair agar level dan label tidak tampil membulat.
  const digits = priceDigits(item.symbol, item.entry_price)
  const mulai = toUnix(item.occurred_at)
  const masuk = toUnix(item.eval_entry_at)
  const selesai = toUnix(item.eval_at)
  const akhir = Math.max(selesai, masuk, mulai)
  const [nowSec] = useState(() => Math.floor(Date.now() / 1000))

  const bar = BAR_SECONDS[item.ltf?.toUpperCase()] ?? 0
  const limit = jumlahCandle(mulai, akhir === mulai ? nowSec : akhir, bar)
  const query = useMarketCandles(item.symbol, item.ltf, limit, enabled, {
    staticSnapshot: true,
  })
  const candles = query.data?.candles ?? []

  const { priceLines, markers } = useMemo(() => {
    const lines: PriceLineSpec[] = []
    const marks: ChartMarkerSpec[] = []
    const buy = item.direction === 'BUY'

    if (item.entry_price > 0) {
      lines.push({
        price: item.entry_price,
        color: theme.entry,
        title: `Entry ${formatPrice(item.entry_price, digits)}`,
        style: 'dashed',
        width: 1,
      })
    }
    if (item.stop_loss > 0) {
      lines.push({
        price: item.stop_loss,
        color: theme.stopLoss,
        title: `SL ${formatPrice(item.stop_loss, digits)}`,
        style: 'solid',
        width: 1,
      })
    }
    if (item.take_profit > 0) {
      lines.push({
        price: item.take_profit,
        color: theme.takeProfit,
        title: `TP ${formatPrice(item.take_profit, digits)}`,
        style: 'solid',
        width: 1,
      })
    }
    if (mulai > 0) {
      marks.push({
        time: mulai,
        position: 'aboveBar',
        color: theme.neutral,
        shape: 'square',
        text: 'Sinyal diterbitkan',
      })
    }
    if (masuk > 0) {
      marks.push({
        time: masuk,
        position: buy ? 'belowBar' : 'aboveBar',
        color: buy ? theme.up : theme.down,
        shape: buy ? 'arrowUp' : 'arrowDown',
        text: `Entry simulasi ${formatPrice(item.entry_price, digits)}`,
      })
    }
    if (selesai > 0 && item.eval_status && item.eval_status !== 'NO_ENTRY') {
      const menang = item.eval_status === 'TP'
      marks.push({
        time: selesai,
        position: menang ? 'aboveBar' : 'belowBar',
        color: menang ? theme.profit : theme.loss,
        shape: 'circle',
        text:
          item.eval_status === 'TP'
            ? `TP ${formatNumber(item.eval_r_lower ?? 0, 2)}R`
            : item.eval_status === 'SL'
              ? 'SL -1R'
              : 'Tidak bisa dipastikan',
      })
    }
    return { priceLines: lines, markers: marks }
  }, [item, digits, mulai, masuk, selesai, theme])

  if (query.isLoading) return <Skeleton className="w-full rounded-xl" />
  if (candles.length === 0) {
    return (
      <p className="rounded-xl border border-border bg-card p-4 text-[11.5px] text-muted-foreground">
        Data candle untuk {item.symbol} {item.ltf} sedang tidak tersedia. Bukti harga tidak bisa
        ditampilkan.
      </p>
    )
  }

  return (
    <div className="overflow-hidden rounded-xl border border-border">
      <PriceChart
        candles={candles}
        priceLines={priceLines}
        markers={markers}
        height={height}
        digits={digits}
        focusTime={masuk || mulai}
        footer={
          <p className="text-[10.5px] leading-relaxed text-muted-foreground">
            Garis putus-putus adalah level entry, garis penuh adalah SL dan TP. Tanda panah menempel
            di candle waktu entry simulasi sesuai metode evaluasi, bukan fill broker; lingkaran
            menandai candle penentu hasil.
          </p>
        }
      />
    </div>
  )
}
