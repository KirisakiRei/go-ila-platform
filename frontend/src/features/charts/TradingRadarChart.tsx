import { useMemo, useState } from 'react'
import { Badge, Skeleton } from '@/components/ui'
import { useMarketCandles } from '@/lib/queries'
import { formatNumber } from '@/lib/format'
import { cn } from '@/lib/utils'
import type { Signal } from '@/types/api'
import { SIGNAL_LABEL, getIla3Mtf, isFinal } from '@/features/signals/lib/status'
import { PriceChart, type ChartMarkerSpec, type PriceLineSpec } from './PriceChart'
import { arahBearish } from './lib/direction'
import { chartTheme } from './lib/theme'
import type { ZoneSpec } from './lib/zonePrimitive'

type StrategyFilter = 'ila-2.0' | 'ila-3.0'

type TradingRadarChartProps = {
  symbol: string | null
  signals: Signal[]
  strategy: StrategyFilter
  className?: string
  /** false menunda permintaan candle (mis. chart mobile yang masih terlipat). */
  enabled?: boolean
}

type Tab = {
  timeframe: string
  label: string
}

/**
 * SUPPORTED_TIMEFRAMES mencerminkan MapResolution di backend market service.
 *
 * getIla3Mtf() mengembalikan "MTF" untuk kombinasi timeframe yang tidak
 * dikenal; nilai itu bukan timeframe nyata, jadi tab-nya harus dilewati agar
 * permintaan candle tidak ditolak backend dengan 400.
 */
const SUPPORTED_TIMEFRAMES = new Set([
  '1M', 'M1', '3M', 'M3', '5M', 'M5', '15M', 'M15', '30M', 'M30',
  '1H', 'H1', '2H', 'H2', '4H', 'H4', '1D', 'D1', 'D', '1W', 'W1', 'W',
])

function timeframeValid(timeframe: string): boolean {
  return SUPPORTED_TIMEFRAMES.has((timeframe || '').toUpperCase().trim())
}

/** rapikanTab membuang timeframe tak dikenal dan tab duplikat. */
function rapikanTab(tabs: Tab[]): Tab[] {
  const terpakai = new Set<string>()
  const hasil: Tab[] = []
  for (const tab of tabs) {
    const key = tab.timeframe.toUpperCase().trim()
    if (!timeframeValid(key) || terpakai.has(key)) continue
    terpakai.add(key)
    hasil.push(tab)
  }
  return hasil
}

/** sinyalAktif memilih sinyal paling relevan untuk digambar di chart. */
function sinyalAktif(signals: Signal[]): Signal | null {
  if (signals.length === 0) return null

  const berbobot = signals.filter((signal) => !isFinal(signal.status))
  const kandidat = berbobot.length > 0 ? berbobot : signals

  return (
    [...kandidat].sort((a, b) => {
      const aTime = new Date(a.updated_at || a.published_at).getTime()
      const bTime = new Date(b.updated_at || b.published_at).getTime()
      return bTime - aTime
    })[0] ?? null
  )
}

function times(signal: Signal) {
  const htf = signal.htf?.toUpperCase() || ''
  const ltf = signal.ltf?.toUpperCase() || ''
  return { htf, ltf }
}

export function TradingRadarChart({
  symbol,
  signals,
  strategy,
  className,
  enabled = true,
}: TradingRadarChartProps) {
  // Chart dikunci ke satu metode. Penyaringan ini pengaman tambahan supaya chart
  // tidak pernah memilih sinyal dari metode lain walau daftar sinyal tercampur.
  const signal = useMemo(
    () => sinyalAktif(signals.filter((s) => s.strategy_version === strategy)),
    [signals, strategy],
  )

  const tabs = useMemo<Tab[]>(() => {
    if (!signal) return []

    const { htf, ltf } = times(signal)
    if (!htf || !ltf) return []

    const ila3 = signal.strategy_version === 'ila-3.0'

    if (ila3) {
      const mtf = getIla3Mtf(htf, ltf)
      return rapikanTab([
        { timeframe: htf, label: `${htf} Macro` },
        { timeframe: mtf, label: `${mtf} Alignment` },
        { timeframe: ltf, label: `${ltf} Trigger` },
      ])
    }

    return rapikanTab([
      { timeframe: htf, label: `${htf} Setup` },
      { timeframe: ltf, label: `${ltf} Trigger` },
    ])
  }, [signal])

  const [activeIndex, setActiveIndex] = useState(0)
  const tabIndex = Math.min(activeIndex, Math.max(tabs.length - 1, 0))
  const activeTab = tabs[tabIndex]

  const query = useMarketCandles(
    symbol,
    activeTab?.timeframe ?? 'H4',
    200,
    enabled && Boolean(symbol) && tabs.length > 0,
  )

  const theme = chartTheme()
  const candles = query.data?.candles ?? []
  const feed = query.data?.feed ?? ''

  const { priceLines, markers, zones } = useMemo(() => {
    const lines: PriceLineSpec[] = []
    const marks: ChartMarkerSpec[] = []
    const areas: ZoneSpec[] = []
    if (!signal || !activeTab) return { priceLines: lines, markers: marks, zones: areas }

    const isLast = tabIndex === tabs.length - 1
    const sweep = signal.htf_sweep
    const cisd = signal.cisd

    // --- Timeframe besar: level sweep dan kotak FVG -------------------------
    if (tabIndex === 0) {
      // Sweep yang masih berjalan pun sudah punya lilin nyata dari engine, jadi
      // ikut digambar: trader perlu melihat level yang sedang diuji.
      if (sweep && (sweep.detected || sweep.sweep_in_progress)) {
        const level = sweepLevel(sweep)
        if (level > 0) {
          lines.push({
            price: level,
            color: theme.sweep,
            title: 'Sweep Likuiditas',
            style: 'dashed',
            width: 2,
          })
        }

        // Penanda diletakkan pada lilin yang menyapu (sweep candle), bukan pada
        // lilin yang tersapu: sweep terjadi ketika candle itu ditutup.
        const waktuSweep = waktuSweepDetik(sweep)
        if (waktuSweep > 0) {
          marks.push({
            time: waktuSweep,
            position: arahBearish(sweep.direction) ? 'aboveBar' : 'belowBar',
            color: theme.sweep,
            shape: 'circle',
            text: 'Sweep',
          })
        }

        // Batas risiko dan target yang dihitung engine ikut ditampilkan agar
        // angka di chart sama dengan angka yang dipakai untuk entry nanti.
        if (sweep.stop_loss > 0) {
          lines.push({
            price: sweep.stop_loss,
            color: theme.stopLoss,
            title: `SL Sweep ${formatNumber(sweep.stop_loss, 2)}`,
            style: 'dotted',
            width: 1,
          })
        }
        if (sweep.target_dol > 0) {
          lines.push({
            price: sweep.target_dol,
            color: theme.takeProfit,
            title: `Target DOL ${formatNumber(sweep.target_dol, 2)}`,
            style: 'dotted',
            width: 1,
          })
        }
      }

      if (sweep?.fvg) {
        const start = sweep.fvg.candle3_time
          ? Math.floor(new Date(sweep.fvg.candle3_time).getTime() / 1000)
          : undefined
        areas.push({
          from: sweep.fvg.top,
          to: sweep.fvg.bottom,
          color: theme.fvg,
          borderColor: theme.fvgBorder,
          label: `FVG ${formatNumber(sweep.fvg.bottom, 2)} - ${formatNumber(sweep.fvg.top, 2)}`,
          timeFrom: start,
        })
      }
    }

    // --- Timeframe trigger: CISD, entry, SL, TP -----------------------------
    if (isLast && tabs.length > 0) {
      if (cisd?.detected && cisd.cisd_level > 0) {
        lines.push({
          price: cisd.cisd_level,
          color: theme.cisd,
          title: 'CISD Trigger',
          style: 'dashed',
          width: 2,
        })

        if (cisd.break_candle) {
          const waktuBreak = Math.floor(new Date(cisd.break_candle.timestamp).getTime() / 1000)
          if (Number.isFinite(waktuBreak)) {
            marks.push({
              time: waktuBreak,
              position: arahBearish(cisd.direction) ? 'aboveBar' : 'belowBar',
              color: theme.cisd,
              shape: 'circle',
              text: 'CISD',
            })
          }
        }
      }

      if (signal.entry_price > 0) {
        lines.push({
          price: signal.entry_price,
          color: theme.entry,
          title: `Entry ${formatNumber(signal.entry_price, 2)}`,
          style: 'solid',
          width: 2,
        })
      }
      if (signal.stop_loss > 0) {
        lines.push({
          price: signal.stop_loss,
          color: theme.stopLoss,
          title: `SL ${formatNumber(signal.stop_loss, 2)}`,
          style: 'solid',
          width: 1,
        })
      }
      if (signal.take_profit > 0) {
        lines.push({
          price: signal.take_profit,
          color: theme.takeProfit,
          title: `TP ${formatNumber(signal.take_profit, 2)}`,
          style: 'solid',
          width: 1,
        })
      }
    }

    // Timeframe tengah ILA 3.0 tetap menampilkan FVG MTF bila ada.
    if (tabs.length === 3 && tabIndex === 1 && sweep?.fvg) {
      const start = sweep.fvg.candle3_time
        ? Math.floor(new Date(sweep.fvg.candle3_time).getTime() / 1000)
        : undefined
      areas.push({
        from: sweep.fvg.top,
        to: sweep.fvg.bottom,
        color: theme.fvg,
        borderColor: theme.fvgBorder,
        label: 'FVG MTF',
        timeFrom: start,
      })
    }

    return { priceLines: lines, markers: marks, zones: areas }
  }, [signal, activeTab, tabIndex, tabs.length, theme])

  if (!symbol) {
    return (
      <div className={cn('rounded-xl border border-border bg-card p-6 text-center text-xs text-muted-foreground', className)}>
        Pilih pair terlebih dahulu untuk melihat chart.
      </div>
    )
  }

  return (
    <div className={cn('space-y-3', className)}>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-1.5">
          {tabs.length === 0 ? (
            <span className="text-xs text-muted-foreground">Belum ada setup untuk digambar.</span>
          ) : (
            tabs.map((tab, index) => (
              <button
                key={`${tab.timeframe}-${index}`}
                type="button"
                onClick={() => setActiveIndex(index)}
                className={cn(
                  'rounded-lg border px-2.5 py-1 text-[11px] font-medium transition-colors',
                  index === tabIndex
                    ? 'border-foreground bg-foreground text-background'
                    : 'border-border bg-card text-muted-foreground hover:border-foreground/40',
                )}
              >
                {tab.label}
              </button>
            ))
          )}
        </div>

        <div className="flex items-center gap-1.5">
          {feed && (
            <Badge variant="outline" className="text-[10px] uppercase tracking-wide">
              {feed}
            </Badge>
          )}
          {/* Setup lama bisa dihitung pada feed berbeda bila admin baru mengganti
              feed pair; trader perlu tahu agar tidak mengira garisnya meleset. */}
          {signal?.feed_source && feed && signal.feed_source !== feed && (
            <Badge variant="muted" className="text-[10px]" title="Setup dihitung sebelum feed pair diubah">
              Setup: {signal.feed_source}
            </Badge>
          )}
          {signal && (
            <Badge variant="muted" className="text-[10px]">
              {SIGNAL_LABEL[signal.status] ?? signal.status}
            </Badge>
          )}
        </div>
      </div>

      {/* Skeleton hanya untuk pemuatan pertama. Saat data lama masih ada
          (ganti timeframe atau penyegaran berkala), chart lama dipertahankan
          agar tampilan tidak berkedip. */}
      {query.isLoading && candles.length === 0 ? (
        <Skeleton className="h-[340px] w-full rounded-xl" />
      ) : (
        <PriceChart
          candles={candles}
          priceLines={priceLines}
          markers={markers}
          zones={zones}
          height={340}
          loading={query.isFetching && candles.length === 0}
          emptyMessage={
            query.isError
              ? 'Chart belum bisa dimuat. Coba muat ulang beberapa saat lagi.'
              : 'Belum ada candle untuk timeframe ini.'
          }
          footer={
            <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
              <LegendItem color={theme.sweep} label="Sweep" />
              <LegendItem color={theme.fvgBorder ?? theme.cisd} label="FVG" />
              <LegendItem color={theme.cisd} label="CISD" />
              <LegendItem color={theme.entry} label="Entry" />
              <LegendItem color={theme.stopLoss} label="SL" />
              <LegendItem color={theme.takeProfit} label="TP / DOL" />
              <span className="ml-auto tabular">
                {symbol} · {activeTab?.timeframe ?? '-'}
              </span>
            </div>
          }
        />
      )}
    </div>
  )
}

function LegendItem({ color, label }: { color: string; label: string }) {
  return (
    <span className="inline-flex items-center gap-1.5">
      <span className="h-1.5 w-4 rounded-full" style={{ backgroundColor: color }} aria-hidden="true" />
      {label}
    </span>
  )
}

/**
 * sweepLevel menentukan harga likuiditas yang tersapu, persis seperti yang
 * dihitung engine: high lilin sebelumnya untuk struktur BEARISH, low-nya untuk
 * BULLISH.
 *
 * Semua angka diambil dari payload engine; tidak ada harga yang diturunkan
 * ulang di browser.
 */
function sweepLevel(sweep: NonNullable<Signal['htf_sweep']>): number {
  const bearish = arahBearish(sweep.direction)
  const swept = sweep.swept_candle

  const level = bearish ? (swept?.high ?? 0) : (swept?.low ?? 0)
  return Number.isFinite(level) && level > 0 ? level : 0
}

/**
 * waktuSweepDetik memilih waktu lilin sweep dari payload engine.
 *
 * Engine mengisi sweep_time sama dengan timestamp lilin penyapu; keduanya bisa
 * dipakai, sedangkan lilin tersapu waktunya lebih awal dan membuat penanda
 * tampak mundur satu candle.
 */
function waktuSweepDetik(sweep: NonNullable<Signal['htf_sweep']>): number {
  const kandidat = [sweep.sweep_time, sweep.sweep_candle?.timestamp]
  for (const nilai of kandidat) {
    if (!nilai) continue
    const ms = new Date(nilai).getTime()
    if (Number.isFinite(ms)) return Math.floor(ms / 1000)
  }
  return 0
}
