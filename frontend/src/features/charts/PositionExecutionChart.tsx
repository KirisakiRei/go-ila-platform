import { useMemo, useState } from 'react'
import { Skeleton } from '@/components/ui'
import { useMarketCandles } from '@/lib/queries'
import { formatMoney, formatNumber, formatPrice, formatR } from '@/lib/format'
import { priceDigits } from '@/lib/pricePrecision'
import type { Position } from '@/types/api'
import { PriceChart, type ChartMarkerSpec, type PriceLineSpec, type SegmentSpec } from './PriceChart'
import { chartTheme } from './lib/theme'

type PositionExecutionChartProps = {
  position: Position
  /** false menunda permintaan candle saat accordion tertutup. */
  enabled?: boolean
  /** Paksa timeframe tertentu; bila kosong dipilih otomatis dari umur posisi. */
  timeframe?: string
  height?: number
}

function toUnix(value: string | null | undefined): number {
  if (!value) return 0
  const ms = new Date(value).getTime()
  return Number.isFinite(ms) ? Math.floor(ms / 1000) : 0
}

const BAR_SECONDS: Record<string, number> = {
  M15: 15 * 60,
  H1: 60 * 60,
  H4: 4 * 60 * 60,
  D1: 24 * 60 * 60,
}

const POSITION_LIMIT = 500

/**
 * pilihTimeframe memilih timeframe terkecil yang jendelanya masih memuat titik
 * entry. Tanpa ini posisi berumur lebih dari ~5 hari kehilangan penanda entry
 * karena chart hanya memuat 500 lilin terakhir.
 */
function pilihTimeframe(entryTime: number, now: number): string {
  if (entryTime <= 0) return 'M15'
  const age = Math.max(0, now - entryTime)
  for (const tf of ['M15', 'H1', 'H4', 'D1']) {
    const seconds = BAR_SECONDS[tf] ?? 0
    if (seconds > 0 && age <= POSITION_LIMIT * seconds) return tf
  }
  return 'D1'
}

/**
 * PositionExecutionChart menampilkan jejak satu posisi di atas candle.
 *
 * Semua titik diambil dari angka yang dilaporkan broker/engine
 * (open_price, stop_loss, take_profit, exit_price) — tidak ada harga yang
 * dihitung ulang di sini agar posisi visual selalu sama dengan yang dieksekusi.
 */
export function PositionExecutionChart({
  position,
  enabled = true,
  timeframe,
  height = 320,
}: PositionExecutionChartProps) {
  const theme = chartTheme()

  // Presisi mengikuti pair: EURUSD 5 desimal, XAUUSD 2, JPY 3. Tanpa ini
  // level entry/SL/TP tampil membulat dan terlihat tidak sesuai broker.
  const digits = priceDigits(position.symbol, position.open_price)

  const entryTime = toUnix(position.opened_at)
  const exitTime = toUnix(position.closed_at)
  const isClosed = position.status !== 'OPEN'
  const hasExit = isClosed && exitTime > 0 && position.exit_price > 0

  // Umur posisi menentukan timeframe: titik entry harus masuk jendela data.
  // Waktu acuan dikunci sekali agar hasilnya stabil antar render.
  const [nowSec] = useState(() => Math.floor(Date.now() / 1000))
  const chartTimeframe = useMemo(
    () => timeframe || pilihTimeframe(entryTime, nowSec),
    [timeframe, entryTime, nowSec],
  )

  const query = useMarketCandles(position.symbol, chartTimeframe, POSITION_LIMIT, enabled)

  const candles = query.data?.candles ?? []
  const windowStart = candles.length > 0 ? (candles[0]?.time ?? 0) : 0
  const entryDiLuarJendela = entryTime > 0 && windowStart > 0 && entryTime < windowStart

  const { priceLines, markers, segments } = useMemo(() => {
    const lines: PriceLineSpec[] = []
    const marks: ChartMarkerSpec[] = []
    const links: SegmentSpec[] = []

    const isBuy = position.direction === 'BUY'

    if (entryTime > 0 && position.open_price > 0) {
      marks.push({
        time: entryTime,
        position: isBuy ? 'belowBar' : 'aboveBar',
        color: isBuy ? theme.up : theme.down,
        shape: isBuy ? 'arrowUp' : 'arrowDown',
        text: `${position.direction} ${formatNumber(position.lot_size, 2)} @ ${formatPrice(position.open_price, digits)}`,
      })

      lines.push({
        price: position.open_price,
        color: theme.entry,
        title: 'Entry',
        style: 'dashed',
        width: 1,
      })
    }

    if (position.stop_loss > 0) {
      lines.push({
        price: position.stop_loss,
        color: theme.stopLoss,
        title: `SL ${formatPrice(position.stop_loss, digits)}`,
        style: 'solid',
        width: 1,
      })
    }

    if (position.take_profit > 0) {
      lines.push({
        price: position.take_profit,
        color: theme.takeProfit,
        title: `TP ${formatPrice(position.take_profit, digits)}`,
        style: 'solid',
        width: 1,
      })
    }

    if (hasExit) {
      const menang = position.profit_loss >= 0
      marks.push({
        time: exitTime,
        position: menang ? 'aboveBar' : 'belowBar',
        color: menang ? theme.profit : theme.loss,
        shape: 'circle',
        text: `${formatMoney(position.profit_loss)} (${formatR(position.outcome_r)})`,
      })

      links.push({
        fromTime: entryTime,
        fromPrice: position.open_price,
        toTime: exitTime,
        toPrice: position.exit_price,
        color: menang ? theme.profit : theme.loss,
      })
    }

    return { priceLines: lines, markers: marks, segments: links }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    position.id,
    position.direction,
    position.open_price,
    position.stop_loss,
    position.take_profit,
    position.exit_price,
    position.profit_loss,
    position.outcome_r,
    position.lot_size,
    position.status,
    entryTime,
    exitTime,
    hasExit,
    theme,
    digits,
  ])

  if (query.isLoading) {
    return <Skeleton className="w-full rounded-xl" style={{ height }} />
  }

  return (
    <PriceChart
      candles={candles}
      priceLines={priceLines}
      markers={markers}
      segments={segments}
      height={height}
      digits={digits}
      focusTime={entryTime || undefined}
      emptyMessage={
        query.isError
          ? 'Chart posisi belum bisa dimuat. Coba lagi beberapa saat.'
          : 'Candle untuk rentang posisi ini tidak tersedia.'
      }
      footer={
        <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
          <span>
            {position.symbol} · {chartTimeframe}
          </span>
          <span className="tabular">Entry {formatPrice(position.open_price, digits)}</span>
          {hasExit && (
            <span className="tabular">
              Exit {formatPrice(position.exit_price, digits)}
            </span>
          )}
          {position.pnl_known && (
            <span
              className="ml-auto tabular font-semibold"
              style={{ color: position.profit_loss >= 0 ? theme.profit : theme.loss }}
            >
              {formatMoney(position.profit_loss)} ({formatR(position.outcome_r)})
            </span>
          )}
          {entryDiLuarJendela && (
            <span className="w-full text-[10.5px] text-muted-foreground">
              Entry berada di luar jendela candle yang tersedia; penanda entry tidak digambar.
            </span>
          )}
        </div>
      }
    />
  )
}
