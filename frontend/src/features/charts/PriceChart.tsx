import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import {
  CandlestickSeries,
  createChart,
  createSeriesMarkers,
  LineSeries,
  LineStyle,
  type CandlestickData,
  type IChartApi,
  type ISeriesApi,
  type IPriceLine,
  type ISeriesMarkersPluginApi,
  type LineData,
  type SeriesMarker,
  type Time,
  type UTCTimestamp,
} from 'lightweight-charts'
import { Skeleton } from '@/components/ui'
import type { MarketCandle } from '@/types/api'
import { cn } from '@/lib/utils'
import { chartTheme, isDarkMode } from './lib/theme'
import { ZonePrimitive, type ZoneSpec } from './lib/zonePrimitive'

export type PriceLineSpec = {
  price: number
  color: string
  title: string
  /** solid | dashed | dotted */
  style?: 'solid' | 'dashed' | 'dotted'
  width?: 1 | 2 | 3 | 4
  axisLabelVisible?: boolean
}

export type ChartMarkerSpec = {
  time: number
  position: 'aboveBar' | 'belowBar'
  color: string
  shape: 'arrowUp' | 'arrowDown' | 'circle' | 'square'
  text?: string
}

/**
 * SegmentSpec adalah garis penghubung dua titik, mis. entry menuju exit.
 * Dipakai pelacak eksekusi agar jejak trade terlihat seperti di cTrader.
 */
export type SegmentSpec = {
  fromTime: number
  fromPrice: number
  toTime: number
  toPrice: number
  color: string
}

type PriceChartProps = {
  candles: MarketCandle[]
  priceLines?: PriceLineSpec[]
  markers?: ChartMarkerSpec[]
  zones?: ZoneSpec[]
  segments?: SegmentSpec[]
  height?: number
  loading?: boolean
  emptyMessage?: string
  className?: string
  /**
   * Unix detik. Bila diisi, chart memusatkan tampilan di sekitar waktu ini
   * agar titik entry tidak hilang di luar layar pada trade lama.
   */
  focusTime?: number
  /** Legend ringkas di bawah chart. */
  footer?: ReactNode
}

const STYLE_MAP = {
  solid: LineStyle.Solid,
  dashed: LineStyle.Dashed,
  dotted: LineStyle.Dotted,
} as const

/**
 * sortedBarTimes mengumpulkan waktu lilin unik yang benar-benar dikirim ke
 * chart. Dipakai untuk menyelaraskan marker dan garis penghubung.
 */
function sortedBarTimes(candles: MarketCandle[]): number[] {
  const seen = new Set<number>()
  const times: number[] = []
  for (const candle of candles) {
    if (!Number.isFinite(candle.time) || seen.has(candle.time)) continue
    seen.add(candle.time)
    times.push(candle.time)
  }
  times.sort((a, b) => a - b)
  return times
}

/**
 * snapKeBar menggeser sebuah waktu ke lilin yang menaunginya.
 *
 * Lightweight-charts hanya menggambar marker pada waktu yang ada di data,
 * sedangkan waktu eksekusi broker tidak selalu jatuh tepat di grid timeframe
 * (mis. entry 10:07 pada chart M15). Tanpa penyelarasan ini marker dan garis
 * entry→exit hilang tanpa pesan. Waktu di luar jendela data mengembalikan null
 * supaya titik tidak digambar di lilin yang salah.
 */
function snapKeBar(time: number, barTimes: number[]): number | null {
  if (barTimes.length === 0 || !Number.isFinite(time)) return null

  const first = barTimes[0]
  const last = barTimes[barTimes.length - 1]
  if (first === undefined || last === undefined) return null
  if (time < first || time > last) return null

  let lo = 0
  let hi = barTimes.length - 1
  while (lo <= hi) {
    const mid = (lo + hi) >> 1
    const current = barTimes[mid]
    if (current === undefined) break
    if (current === time) return current
    if (current < time) lo = mid + 1
    else hi = mid - 1
  }
  // hi kini menunjuk lilin terakhir yang waktunya <= time.
  const snapped = hi >= 0 ? barTimes[hi] : undefined
  return snapped ?? null
}

/**
 * PriceChart adalah pembungkus tipis lightweight-charts.
 *
 * Data candle wajib terurut menaik dengan waktu unik; pengurutan dilakukan di
 * sini sebagai jaring pengaman terakhir agar chart tidak pernah mogok karena
 * payload dari jaringan.
 */
export function PriceChart({
  candles,
  priceLines = [],
  markers = [],
  zones = [],
  segments = [],
  height = 380,
  loading = false,
  emptyMessage = 'Belum ada data candle untuk ditampilkan.',
  className,
  focusTime,
  footer,
}: PriceChartProps) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const chartRef = useRef<IChartApi | null>(null)
  const seriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null)
  const priceLinesRef = useRef<IPriceLine[]>([])
  const markerApiRef = useRef<ISeriesMarkersPluginApi<Time> | null>(null)
  const zonePrimitiveRef = useRef<ZonePrimitive | null>(null)
  const segmentSeriesRef = useRef<ISeriesApi<'Line'>[]>([])
  const prevDataRef = useRef<CandlestickData<UTCTimestamp>[]>([])
  /** focusTime yang sudah diterapkan; mencegah view ter-reset setiap polling. */
  const focusAppliedRef = useRef<number | null>(null)
  const [dark, setDark] = useState(isDarkMode)

  const theme = useMemo(() => chartTheme(dark), [dark])
  const barTimes = useMemo(() => sortedBarTimes(candles), [candles])

  // Marker dan garis penghubung diselaraskan ke lilin nyata sebelum digambar.
  const snappedMarkers = useMemo(() => {
    const out: (ChartMarkerSpec & { time: number })[] = []
    for (const marker of markers) {
      const time = snapKeBar(marker.time, barTimes)
      if (time === null) continue
      out.push({ ...marker, time })
    }
    return out
  }, [markers, barTimes])

  const snappedSegments = useMemo(() => {
    const out: SegmentSpec[] = []
    for (const segment of segments) {
      const fromTime = snapKeBar(segment.fromTime, barTimes)
      const toTime = snapKeBar(segment.toTime, barTimes)
      if (fromTime === null || toTime === null || fromTime === toTime) continue
      out.push({ ...segment, fromTime, toTime })
    }
    return out
  }, [segments, barTimes])

  // Ikuti perubahan tema aplikasi tanpa memuat ulang halaman.
  useEffect(() => {
    const observer = new MutationObserver(() => setDark(isDarkMode()))
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
    return () => observer.disconnect()
  }, [])

  // Chart dibuat sekali; data dan overlay diperbarui lewat effect terpisah.
  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    const chart = createChart(container, {
      autoSize: true,
      layout: {
        background: { color: theme.background },
        textColor: theme.text,
        fontSize: 11,
      },
      grid: {
        vertLines: { color: theme.grid },
        horzLines: { color: theme.grid },
      },
      rightPriceScale: { borderColor: theme.border },
      timeScale: { borderColor: theme.border, timeVisible: true, secondsVisible: false },
      crosshair: {
        vertLine: { color: theme.crosshair, width: 1, style: LineStyle.Dashed },
        horzLine: { color: theme.crosshair, width: 1, style: LineStyle.Dashed },
      },
      handleScale: { axisPressedMouseMove: true },
    })

    const series = chart.addSeries(CandlestickSeries, {
      upColor: theme.up,
      downColor: theme.down,
      borderUpColor: theme.up,
      borderDownColor: theme.down,
      wickUpColor: theme.upWick,
      wickDownColor: theme.downWick,
      priceLineVisible: false,
      lastValueVisible: true,
    })

    chartRef.current = chart
    seriesRef.current = series
    markerApiRef.current = createSeriesMarkers(series, [])

    const zonePrimitive = new ZonePrimitive([])
    series.attachPrimitive(zonePrimitive)
    zonePrimitiveRef.current = zonePrimitive

    return () => {
      zonePrimitiveRef.current = null
      priceLinesRef.current = []
      segmentSeriesRef.current = []
      prevDataRef.current = []
      focusAppliedRef.current = null
      markerApiRef.current = null
      seriesRef.current = null
      chartRef.current = null
      chart.remove()
    }
    // Tema hanya dipakai untuk nilai awal; perubahan tema ditangani effect lain.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Perubahan tema diterapkan tanpa membuat ulang chart.
  useEffect(() => {
    const chart = chartRef.current
    const series = seriesRef.current
    if (!chart || !series) return

    chart.applyOptions({
      layout: { background: { color: theme.background }, textColor: theme.text },
      grid: { vertLines: { color: theme.grid }, horzLines: { color: theme.grid } },
      rightPriceScale: { borderColor: theme.border },
      timeScale: { borderColor: theme.border },
      crosshair: {
        vertLine: { color: theme.crosshair },
        horzLine: { color: theme.crosshair },
      },
    })
    series.applyOptions({
      upColor: theme.up,
      downColor: theme.down,
      borderUpColor: theme.up,
      borderDownColor: theme.down,
      wickUpColor: theme.upWick,
      wickDownColor: theme.downWick,
    })
  }, [theme])

  // Data candle: bersihkan duplikat dan urutkan menaik.
  useEffect(() => {
    const series = seriesRef.current
    const chart = chartRef.current
    if (!series || !chart) return

    const seen = new Set<number>()
    const data: CandlestickData<UTCTimestamp>[] = []
    for (const candle of [...candles].sort((a, b) => a.time - b.time)) {
      if (!Number.isFinite(candle.time) || seen.has(candle.time)) continue
      seen.add(candle.time)
      data.push({
        time: candle.time as UTCTimestamp,
        open: candle.open,
        high: candle.high,
        low: candle.low,
        close: candle.close,
      })
    }

    const prev = prevDataRef.current
    prevDataRef.current = data

    if (data.length === 0) {
      series.setData(data)
      return
    }

    const prevLast = prev.length > 0 ? (prev[prev.length - 1] as CandlestickData<UTCTimestamp>).time : null
    const focusPending = focusTime != null && focusAppliedRef.current !== focusTime

    // Jendela data sama (hanya lilin berjalan yang berubah): cukup perbarui
    // ekornya. Ini mencegah zoom dan posisi gulir pengguna tereset setiap kali
    // data disegarkan.
    if (prevLast !== null && data[0]?.time === prev[0]?.time && !focusPending) {
      const startIndex = data.findIndex((bar) => (bar.time as number) >= (prevLast as number))
      if (startIndex > 0) {
        for (let i = startIndex; i < data.length; i += 1) {
          series.update(data[i] as CandlestickData<UTCTimestamp>)
        }
        return
      }
    }

    // Jendela bergeser: muat ulang, lalu kembalikan tampilan yang sedang dilihat
    // pengguna alih-alih memaksa fit ke seluruh data.
    const range = chart.timeScale().getVisibleLogicalRange()
    series.setData(data)

    if (focusPending) {
      // Jendela 60 lilin sebelum dan 40 sesudah titik fokus: cukup untuk
      // melihat konteks struktur sebelum entry. Hanya dipakai sekali agar
      // gulir/zoom pengguna tidak dipaksa kembali pada penyegaran berikutnya.
      let index = data.findIndex((bar) => (bar.time as number) >= (focusTime as number))
      if (index < 0) index = data.length - 1
      chart.timeScale().setVisibleLogicalRange({
        from: Math.max(0, index - 60),
        to: Math.min(data.length - 1, index + 40),
      })
      focusAppliedRef.current = focusTime ?? null
      return
    }

    if (range) {
      chart.timeScale().setVisibleLogicalRange(range)
      return
    }

    chart.timeScale().fitContent()
  }, [candles, focusTime])

  // Garis harga: Entry, SL, TP, Sweep, CISD.
  useEffect(() => {
    const series = seriesRef.current
    if (!series) return

    for (const line of priceLinesRef.current) {
      series.removePriceLine(line)
    }
    priceLinesRef.current = priceLines
      .filter((line) => Number.isFinite(line.price) && line.price > 0)
      .map((line) =>
        series.createPriceLine({
          price: line.price,
          color: line.color,
          lineWidth: line.width ?? 2,
          lineStyle: STYLE_MAP[line.style ?? 'solid'],
          axisLabelVisible: line.axisLabelVisible ?? true,
          title: line.title,
        }),
      )
  }, [priceLines])

  // Marker entry/exit.
  useEffect(() => {
    const api = markerApiRef.current
    if (!api) return

    const list: SeriesMarker<Time>[] = snappedMarkers
      .filter((marker) => Number.isFinite(marker.time))
      .map((marker) => ({
        time: marker.time as UTCTimestamp,
        position: marker.position,
        color: marker.color,
        shape: marker.shape,
        text: marker.text,
      }))
      .sort((a, b) => (a.time as number) - (b.time as number))

    api.setMarkers(list)
  }, [snappedMarkers])

  // Zona harga (FVG).
  useEffect(() => {
    zonePrimitiveRef.current?.updateZones(zones)
  }, [zones])

  // Garis penghubung entry → exit.
  useEffect(() => {
    const chart = chartRef.current
    if (!chart) return

    for (const series of segmentSeriesRef.current) {
      chart.removeSeries(series)
    }
    segmentSeriesRef.current = []

    for (const segment of snappedSegments) {
      const series = chart.addSeries(LineSeries, {
        color: segment.color,
        lineWidth: 2,
        lineStyle: LineStyle.Dashed,
        priceLineVisible: false,
        lastValueVisible: false,
        crosshairMarkerVisible: false,
      })

      const data: LineData<UTCTimestamp>[] = [
        { time: segment.fromTime as UTCTimestamp, value: segment.fromPrice },
        { time: segment.toTime as UTCTimestamp, value: segment.toPrice },
      ]
      series.setData(data)
      segmentSeriesRef.current.push(series)
    }
  }, [snappedSegments])

  const hasData = candles.length > 0

  return (
    <div className={cn('relative w-full overflow-hidden rounded-xl border border-border bg-card', className)}>
      {loading && (
        <div className="absolute inset-0 z-10 flex items-center justify-center bg-card/80">
          <Skeleton className="h-full w-full" />
        </div>
      )}
      <div ref={containerRef} style={{ height }} className="w-full" />
      {!hasData && !loading && (
        <div className="absolute inset-0 flex items-center justify-center px-6 text-center text-xs text-muted-foreground">
          {emptyMessage}
        </div>
      )}
      {footer && <div className="border-t border-border px-3 py-2 text-[11px] text-muted-foreground">{footer}</div>}
    </div>
  )
}
