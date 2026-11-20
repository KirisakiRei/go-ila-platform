import { useMemo, useState } from 'react'
import { History, X } from 'lucide-react'
import { Badge, Button, Skeleton } from '@/components/ui'
import { useMarketCandles } from '@/lib/queries'
import { useLiveCandleStream } from '@/lib/use-live-candle-stream'
import { formatDateTime, formatPrice } from '@/lib/format'
import { priceDigits } from '@/lib/pricePrecision'
import { cn } from '@/lib/utils'
import type { Signal } from '@/types/api'
import { SIGNAL_LABEL, getIla3Mtf, isFinal } from '@/features/signals/lib/status'
import { mapSignalsToSlots } from '@/features/signals/lib/signal-slots'
import { OUTCOME_BADGE_CLASS, describeSignalOutcome } from '@/features/signals/lib/outcome'
import { PriceChart, type ChartMarkerSpec, type PriceLineSpec } from './PriceChart'
import { arahBearish } from './lib/direction'
import { chartTheme } from './lib/theme'
import type { ZoneSpec } from './lib/zonePrimitive'

type StrategyFilter = 'ila-2.0' | 'ila-3.0'

type TradingRadarChartProps = {
  symbol: string | null
  signals: Signal[]
  strategy: StrategyFilter
  /** Kunci slot pasangan timeframe yang dipilih pada TimeframeRadarMatrix. */
  selectedSlotKey?: string | null
  /** Sinyal histori yang dipilih untuk ditampilkan sebagai potongan statis. */
  snapshotSignal?: Signal | null
  onClearSnapshot?: () => void
  className?: string
  /** false menunda permintaan candle & stream (mis. chart mobile yang terlipat). */
  enabled?: boolean
}

type Tab = {
  timeframe: string
  label: string
}

const SUPPORTED_TIMEFRAMES = new Set([
  '1M', 'M1', '3M', 'M3', '5M', 'M5', '15M', 'M15', '30M', 'M30',
  '1H', 'H1', '2H', 'H2', '4H', 'H4', '1D', 'D1', 'D', '1W', 'W1', 'W',
])

const BAR_SECONDS: Record<string, number> = {
  '1M': 60,
  M1: 60,
  '3M': 180,
  M3: 180,
  '5M': 300,
  M5: 300,
  '15M': 900,
  M15: 900,
  '30M': 1800,
  M30: 1800,
  '1H': 3600,
  H1: 3600,
  '4H': 14400,
  H4: 14400,
  D: 86400,
  D1: 86400,
  '1D': 86400,
  W: 604800,
  W1: 604800,
  '1W': 604800,
}

function timeframeValid(timeframe: string): boolean {
  return SUPPORTED_TIMEFRAMES.has((timeframe || '').toUpperCase().trim())
}

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

function toUnixSec(iso: string | null | undefined): number {
  if (!iso) return 0
  const ms = new Date(iso).getTime()
  return Number.isFinite(ms) ? Math.floor(ms / 1000) : 0
}

function isLiveOverlayRelevant(sig: Signal | undefined | null): sig is Signal {
  if (!sig) return false
  if (!isFinal(sig.status)) return true
  if (sig.eval_status === 'RUNNING') return true
  if (sig.is_executed && sig.position_status === 'OPEN') return true
  return false
}

export function TradingRadarChart({
  symbol,
  signals,
  strategy,
  selectedSlotKey,
  snapshotSignal = null,
  onClearSnapshot,
  className,
  enabled = true,
}: TradingRadarChartProps) {
  const isSnapshotMode = Boolean(snapshotSignal)

  const strategySignals = useMemo(
    () => signals.filter((s) => s.strategy_version === strategy),
    [signals, strategy],
  )

  const slots = useMemo(
    () => mapSignalsToSlots(strategySignals, strategy),
    [strategySignals, strategy],
  )

  const activeSlot = useMemo(() => {
    if (selectedSlotKey) {
      const matched = slots.find((s) => s.key === selectedSlotKey)
      if (matched) return matched
    }
    return slots.find((s) => s.status === 'READY') ?? slots[0] ?? null
  }, [slots, selectedSlotKey])

  // Dalam mode live, overlay hanya digambar jika slot tersebut punya setup yang
  // sedang berjalan / dipantau agar chart scanning tidak dikotori garis basi.
  const signal = useMemo(() => {
    if (snapshotSignal) return snapshotSignal
    const slotSig = activeSlot?.latestSignal ?? null
    return isLiveOverlayRelevant(slotSig) ? slotSig : null
  }, [snapshotSignal, activeSlot])

  const tabs = useMemo<Tab[]>(() => {
    if (snapshotSignal) {
      const htf = snapshotSignal.htf?.toUpperCase() || ''
      const ltf = snapshotSignal.ltf?.toUpperCase() || ''
      if (snapshotSignal.strategy_version === 'ila-3.0') {
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
    }

    if (!activeSlot) return []
    const htf = activeSlot.htf.toUpperCase()
    const ltf = activeSlot.ltf.toUpperCase()

    if (strategy === 'ila-3.0') {
      const mtf = (activeSlot.mtf || getIla3Mtf(htf, ltf)).toUpperCase()
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
  }, [snapshotSignal, activeSlot, strategy])

  const [nowSec] = useState(() => Math.floor(Date.now() / 1000))
  const modeContextKey = snapshotSignal?.signal_id ? `snap:${snapshotSignal.signal_id}` : 'live'
  const [tabState, setTabState] = useState<{ contextKey: string; index: number }>({
    contextKey: modeContextKey,
    index: 0,
  })

  // Dalam mode potongan histori, bawaan langsung membuka tab Trigger (LTF)
  // tanpa useEffect berantai; klik manual pada tab tetap dihormati.
  const rawIndex =
    tabState.contextKey === modeContextKey
      ? tabState.index
      : snapshotSignal && tabs.length > 0
        ? tabs.length - 1
        : 0
  const tabIndex = Math.min(rawIndex, Math.max(tabs.length - 1, 0))
  const activeTab = tabs[tabIndex]

  const snapshotFocusTime = useMemo(() => {
    if (!snapshotSignal) return undefined
    const pub = toUnixSec(snapshotSignal.published_at)
    return pub > 0 ? pub : undefined
  }, [snapshotSignal])

  const candleLimit = useMemo(() => {
    if (!snapshotSignal || !activeTab) return 200
    const barSec = BAR_SECONDS[activeTab.timeframe.toUpperCase()] ?? 900
    const pubSec = toUnixSec(snapshotSignal.published_at)
    if (pubSec <= 0) return 300
    const barsFromNow = Math.ceil(Math.max(0, nowSec - pubSec) / barSec) + 80
    return Math.min(500, Math.max(250, barsFromNow))
  }, [snapshotSignal, activeTab, nowSec])

  const chartEnabled = enabled && Boolean(symbol) && tabs.length > 0

  const { isLive } = useLiveCandleStream({
    symbol,
    timeframe: activeTab?.timeframe,
    limit: candleLimit,
    enabled: chartEnabled && !isSnapshotMode,
  })

  const query = useMarketCandles(
    symbol,
    activeTab?.timeframe ?? 'H4',
    candleLimit,
    chartEnabled,
    {
      staticSnapshot: isSnapshotMode,
      liveStreaming: isLive,
    },
  )

  const theme = chartTheme()
  const candles = query.data?.candles ?? []
  const feed = query.data?.feed ?? ''

  const digits = priceDigits(symbol, candles[0]?.close ?? signal?.entry_price)

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

        if (sweep.stop_loss > 0) {
          lines.push({
            price: sweep.stop_loss,
            color: theme.stopLoss,
            title: `SL Sweep ${formatPrice(sweep.stop_loss, digits)}`,
            style: 'dotted',
            width: 1,
          })
        }
        if (sweep.target_dol > 0) {
          lines.push({
            price: sweep.target_dol,
            color: theme.takeProfit,
            title: `Target DOL ${formatPrice(sweep.target_dol, digits)}`,
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
          label: `${sweep.fvg.type || 'FVG'} ${formatPrice(sweep.fvg.bottom, digits)} - ${formatPrice(sweep.fvg.top, digits)}`,
          timeFrom: start,
        })
      }
    }

    // --- Timeframe trigger: CISD, entry, SL, TP, dan penanda histori --------
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
          title: `Entry ${formatPrice(signal.entry_price, digits)}`,
          style: 'solid',
          width: 2,
        })
      }
      if (signal.stop_loss > 0) {
        lines.push({
          price: signal.stop_loss,
          color: theme.stopLoss,
          title: `SL ${formatPrice(signal.stop_loss, digits)}`,
          style: 'solid',
          width: 1,
        })
      }
      if (signal.take_profit > 0) {
        lines.push({
          price: signal.take_profit,
          color: theme.takeProfit,
          title: `TP ${formatPrice(signal.take_profit, digits)}`,
          style: 'solid',
          width: 1,
        })
      }

      if (isSnapshotMode) {
        const isBuy = signal.direction === 'BUY'
        const pubTime = toUnixSec(signal.published_at)
        if (pubTime > 0) {
          marks.push({
            time: pubTime,
            position: isBuy ? 'belowBar' : 'aboveBar',
            color: isBuy ? theme.up : theme.down,
            shape: isBuy ? 'arrowUp' : 'arrowDown',
            text: `Setup ${signal.direction}`,
          })
        }

        const resolvedIso = signal.closed_at || signal.eval_at
        const resolvedSec = toUnixSec(resolvedIso)
        const hitTP =
          signal.position_status === 'CLOSED_TP' || (!signal.is_executed && signal.eval_status === 'TP')
        const hitSL =
          signal.position_status === 'CLOSED_SL' || (!signal.is_executed && signal.eval_status === 'SL')

        if (resolvedSec > 0 && (hitTP || hitSL)) {
          marks.push({
            time: resolvedSec,
            position: hitTP ? (isBuy ? 'aboveBar' : 'belowBar') : isBuy ? 'belowBar' : 'aboveBar',
            color: hitTP ? theme.takeProfit : theme.stopLoss,
            shape: 'circle',
            text: hitTP ? 'TP' : 'SL',
          })
        }
      }
    }

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
  }, [signal, activeTab, tabIndex, tabs.length, theme, digits, isSnapshotMode])

  if (!symbol) {
    return (
      <div className={cn('rounded-xl border border-border bg-card p-6 text-center text-xs text-muted-foreground', className)}>
        Pilih pair terlebih dahulu untuk melihat chart.
      </div>
    )
  }

  const snapshotOutcome = snapshotSignal ? describeSignalOutcome(snapshotSignal) : null

  return (
    <div className={cn('space-y-3', className)}>
      {snapshotSignal && snapshotOutcome && (
        <div className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-border bg-secondary/50 px-3 py-2 text-xs">
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <span className="inline-flex items-center gap-1 font-semibold text-foreground">
              <History className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
              <span>
                Potongan Histori · {snapshotSignal.direction} {snapshotSignal.htf}/{snapshotSignal.ltf}
              </span>
            </span>
            <span
              className={cn(
                'inline-flex items-center rounded-md border px-2 py-0.5 text-[10.5px] font-semibold',
                OUTCOME_BADGE_CLASS[snapshotOutcome.tone],
              )}
            >
              {snapshotOutcome.label}
            </span>
            <span className="text-[11px] text-muted-foreground">
              {formatDateTime(snapshotSignal.published_at)}
            </span>
          </div>

          {onClearSnapshot && (
            <Button
              size="sm"
              variant="outline"
              onClick={onClearSnapshot}
              className="h-7 gap-1 px-2.5 text-[11px]"
            >
              <X className="h-3 w-3" />
              Kembali ke Live Chart
            </Button>
          )}
        </div>
      )}

      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-1.5">
          {tabs.length === 0 ? (
            <span className="text-xs text-muted-foreground">Belum ada timeframe untuk digambar.</span>
          ) : (
            tabs.map((tab, index) => (
              <button
                key={`${tab.timeframe}-${index}`}
                type="button"
                onClick={() => setTabState({ contextKey: modeContextKey, index })}
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

        <div className="flex flex-wrap items-center gap-1.5">
          {!isSnapshotMode && isLive && (
            <span
              className="inline-flex items-center gap-1 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider text-emerald-700 dark:text-emerald-400"
              title="Aliran harga candle berjalan secara langsung"
            >
              <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
              Live
            </span>
          )}
          {isSnapshotMode && (
            <Badge variant="outline" className="text-[10px]">
              Statis
            </Badge>
          )}
          {feed && (
            <Badge variant="outline" className="text-[10px] uppercase tracking-wide">
              {feed}
            </Badge>
          )}
          {signal?.feed_source && feed && signal.feed_source !== feed && (
            <Badge variant="muted" className="text-[10px]" title="Setup dihitung sebelum feed pair diubah">
              Setup: {signal.feed_source}
            </Badge>
          )}
          {!isSnapshotMode && signal && (
            <Badge variant="muted" className="text-[10px]">
              {SIGNAL_LABEL[signal.status] ?? signal.status}
            </Badge>
          )}
          {!isSnapshotMode && !signal && activeSlot && (
            <Badge variant="muted" className="text-[10px]">
              Scanning · {activeSlot.label}
            </Badge>
          )}
        </div>
      </div>

      {query.isLoading && candles.length === 0 ? (
        <Skeleton className="h-[340px] w-full rounded-xl" />
      ) : (
        <PriceChart
          seriesKey={`${symbol}:${activeTab?.timeframe ?? ''}:${feed}:${snapshotSignal?.signal_id ?? 'live'}`}
          candles={candles}
          priceLines={priceLines}
          markers={markers}
          zones={zones}
          focusTime={snapshotFocusTime}
          height={340}
          digits={digits}
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
              <LegendItem color={theme.takeProfit} label="TP" />
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

function sweepLevel(sweep: NonNullable<Signal['htf_sweep']>): number {
  const bearish = arahBearish(sweep.direction)
  const swept = sweep.swept_candle

  const level = bearish ? (swept?.high ?? 0) : (swept?.low ?? 0)
  return Number.isFinite(level) && level > 0 ? level : 0
}

function waktuSweepDetik(sweep: NonNullable<Signal['htf_sweep']>): number {
  const kandidat = [sweep.sweep_time, sweep.sweep_candle?.timestamp]
  for (const nilai of kandidat) {
    if (!nilai) continue
    const ms = new Date(nilai).getTime()
    if (Number.isFinite(ms)) return Math.floor(ms / 1000)
  }
  return 0
}
