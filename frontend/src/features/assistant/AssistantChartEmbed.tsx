import { Suspense, lazy, useMemo, useState } from 'react'
import { BarChart2, ChevronDown, ChevronUp } from 'lucide-react'
import { Button, Skeleton } from '@/components/ui'
import type { ChartMarkerSpec, PriceLineSpec } from '@/features/charts/PriceChart'
import { useChartTheme } from '@/features/charts/lib/theme'
import type { ZoneSpec } from '@/features/charts/lib/zonePrimitive'
import { priceDigits } from '@/lib/pricePrecision'
import { useMarketCandles } from '@/lib/queries'
import type { AssistantChartSpec } from './lib/chart-spec'

// lightweight-charts (±150 kB) hanya diunduh saat ada chart yang benar-benar dibuka.
const LazyPriceChart = lazy(() =>
  import('@/features/charts/PriceChart').then((mod) => ({ default: mod.PriceChart })),
)

type AssistantChartEmbedProps = {
  spec: AssistantChartSpec
  /** Hanya pesan terbaru yang langsung memuat canvas TradingView agar memori HP tetap ringan. */
  autoLoad?: boolean
}

export function AssistantChartEmbed({ spec, autoLoad = false }: AssistantChartEmbedProps) {
  const [expanded, setExpanded] = useState(autoLoad)
  const theme = useChartTheme()

  const query = useMarketCandles(spec.symbol, spec.timeframe, 300, expanded, {
    staticSnapshot: true,
  })
  const candles = useMemo(() => query.data?.candles ?? [], [query.data])

  const digits = useMemo(() => {
    const sample = candles[0]?.close ?? spec.lines[0]?.price ?? 0
    return priceDigits(spec.symbol, sample)
  }, [candles, spec.lines, spec.symbol])

  const priceLines = useMemo<PriceLineSpec[]>(() => {
    return spec.lines.map((lv) => {
      const color =
        lv.kind === 'entry'
          ? theme.entry
          : lv.kind === 'sl'
            ? theme.stopLoss
            : lv.kind === 'tp'
              ? theme.takeProfit
              : lv.kind === 'cisd'
                ? theme.cisd
                : lv.kind === 'sweep'
                  ? theme.sweep
                  : theme.neutral
      return {
        price: lv.price,
        color,
        title: lv.label,
        style: lv.kind === 'entry' ? 'solid' : 'dashed',
        width: lv.kind === 'entry' ? 2 : 1,
        axisLabelVisible: true,
      }
    })
  }, [spec.lines, theme])

  const markers = useMemo<ChartMarkerSpec[]>(() => {
    return spec.markers.map((m) => ({
      time: m.time,
      position: m.position === 'below' ? 'belowBar' : 'aboveBar',
      color: m.position === 'below' ? theme.up : theme.sweep,
      shape: m.position === 'below' ? 'arrowUp' : 'arrowDown',
      text: m.label,
    }))
  }, [spec.markers, theme])

  const zones = useMemo<ZoneSpec[]>(() => {
    return spec.zones.map((z) => ({
      from: z.bottom,
      to: z.top,
      color: theme.fvg,
      borderColor: theme.fvgBorder,
      label: z.label,
      timeFrom: z.timeFrom,
      timeTo: z.timeTo,
    }))
  }, [spec.zones, theme])

  const outOfWindow = useMemo(() => {
    if (!spec.focusTime || candles.length === 0) return false
    const firstTime = candles[0]?.time ?? 0
    return firstTime > 0 && spec.focusTime < firstTime
  }, [spec.focusTime, candles])

  return (
    <figure className="my-2 overflow-hidden rounded-xl border border-border bg-card">
      <figcaption className="flex flex-wrap items-center justify-between gap-2 border-b border-border/80 px-3 py-2">
        <div className="flex min-w-0 items-center gap-2">
          <BarChart2 className="h-3.5 w-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
          <span className="truncate text-[12.5px] font-semibold tracking-tight text-foreground">
            {spec.title ?? `${spec.symbol} (${spec.timeframe})`}
          </span>
        </div>

        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setExpanded((v) => !v)}
          className="press h-7 gap-1 px-2 text-[11.5px]"
        >
          {expanded ? (
            <>
              <ChevronUp className="h-3.5 w-3.5" aria-hidden="true" />
              <span>Sembunyikan chart</span>
            </>
          ) : (
            <>
              <ChevronDown className="h-3.5 w-3.5" aria-hidden="true" />
              <span>Tampilkan chart</span>
            </>
          )}
        </Button>
      </figcaption>

      {expanded && (
        <div className="p-2">
          {query.isError ? (
            <p className="px-2 py-4 text-[12.5px] text-muted-foreground">
              Data candle {spec.symbol} {spec.timeframe} gagal dimuat dari server.
            </p>
          ) : (
            <Suspense fallback={<Skeleton className="h-[260px] w-full rounded-lg" />}>
              <LazyPriceChart
                candles={candles}
                timeframe={spec.timeframe}
                priceLines={priceLines}
                markers={markers}
                zones={zones}
                focusTime={spec.focusTime}
                digits={digits}
                height={260}
                loading={query.isLoading}
                seriesKey={`${spec.symbol}:${spec.timeframe}:assistant`}
                fullscreenTitle={spec.title ?? `${spec.symbol} (${spec.timeframe})`}
                emptyMessage={`Data candle ${spec.symbol} ${spec.timeframe} belum tersedia.`}
              />
            </Suspense>
          )}
          {outOfWindow && (
            <p className="mt-1.5 px-1 text-[11.5px] text-muted-foreground">
              Waktu kejadian sudah melewati jendela 300 candle terakhir pada feed saat ini.
            </p>
          )}
        </div>
      )}
    </figure>
  )
}
