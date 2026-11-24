import { useId, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { AnimatePresence, motion, useReducedMotion } from 'framer-motion'
import { ArrowRight, LineChart, Maximize2 } from 'lucide-react'
import { Button, Card, CardContent, Skeleton } from '@/components/ui'
import { ChartFullscreenModal } from '@/features/charts/ChartFullscreenModal'
import { usePositions } from '@/lib/queries'
import { formatDateTime, formatMoney, formatR } from '@/lib/format'
import { cn } from '@/lib/utils'
import { POSISI_LIMIT_DASHBOARD } from '@/features/signals/lib/kondisi-pasar'
import {
  type CurveMetric,
  type CurvePeriod,
  EASE_OUT,
  buildEquityCurve,
  buildSmoothCurvePath,
} from '../lib/dashboard-metrics'

const PERIODS: Array<{ id: CurvePeriod; label: string }> = [
  { id: '7d', label: '7H' },
  { id: '30d', label: '30H' },
  { id: 'all', label: 'Semua' },
]

const METRICS: Array<{ id: CurveMetric; label: string }> = [
  { id: 'r', label: 'R' },
  { id: 'pnl', label: 'P/L' },
]

const SVG_W = 560
const SVG_H = 190
const PAD = { top: 18, right: 16, bottom: 24, left: 46 }

export function PerformanceCurveCard() {
  const areaGradId = useId()
  const strokeGradId = useId()
  const reduceMotion = useReducedMotion()
  const [period, setPeriod] = useState<CurvePeriod>('30d')
  const [metric, setMetric] = useState<CurveMetric>('r')
  const [hoverIdx, setHoverIdx] = useState<number | null>(null)
  const [fullscreen, setFullscreen] = useState(false)

  const { data: positions, isLoading } = usePositions(POSISI_LIMIT_DASHBOARD)
  const summary = useMemo(() => buildEquityCurve(positions, period), [positions, period])

  const series = useMemo(() => {
    const plotW = SVG_W - PAD.left - PAD.right
    const plotH = SVG_H - PAD.top - PAD.bottom

    // Mulai dari titik baseline 0 agar kurva 1 trade tetap punya lintasan bermakna.
    const rawValues = [0, ...summary.points.map((pt) => (metric === 'r' ? pt.cumR : pt.cumPnL))]
    const minVal = Math.min(...rawValues, 0)
    const maxVal = Math.max(...rawValues, 0)
    const span = maxVal - minVal || 1
    const padSpan = span * 0.14
    const lo = minVal - padSpan
    const hi = maxVal + padSpan
    const range = hi - lo || 1

    const toX = (i: number) =>
      rawValues.length <= 1 ? PAD.left + plotW : PAD.left + (i / (rawValues.length - 1)) * plotW
    const toY = (v: number) => PAD.top + ((hi - v) / range) * plotH

    const coords = rawValues.map((v, i) => ({ x: toX(i), y: toY(v), value: v }))
    const linePath = buildSmoothCurvePath(coords)

    const zeroY = toY(0)
    const zeroPct = Math.max(0, Math.min(100, ((zeroY - PAD.top) / plotH) * 100))
    const first = coords[0]!
    const last = coords[coords.length - 1]!
    const areaPath = `${linePath} L ${last.x.toFixed(1)} ${zeroY.toFixed(1)} L ${first.x.toFixed(1)} ${zeroY.toFixed(1)} Z`

    return { coords, linePath, areaPath, zeroY, zeroPct, hi, lo, minVal, maxVal }
  }, [summary.points, metric])

  const headlineVal = metric === 'r' ? summary.totalR : summary.totalPnL
  const isPositive = headlineVal > 0
  const isNegative = headlineVal < 0

  const pointColor = (v: number) => (v > 0 ? '#10b981' : v < 0 ? '#f43f5e' : '#71717a')
  const zeroPct = series.zeroPct
  const strokeSoftUp =
    series.minVal >= 0 ? zeroPct : Math.max(0, zeroPct - 3.5)
  const strokeSoftDown =
    series.maxVal <= 0 ? zeroPct : Math.min(100, zeroPct + 3.5)
  const areaSoftUp = Math.max(0, zeroPct - 2)
  const areaSoftDown = Math.min(100, zeroPct + 2)
  const activePoint =
    hoverIdx !== null && hoverIdx > 0 ? summary.points[hoverIdx - 1] ?? null : null
  const activeCoord = hoverIdx !== null ? series.coords[hoverIdx] ?? null : null

  const controls = (
    <div className="flex items-center gap-1.5">
      {/* Metric Toggle */}
      <div className="inline-flex rounded-lg border border-border bg-secondary/40 p-0.5">
        {METRICS.map((m) => (
          <button
            key={m.id}
            type="button"
            onClick={() => {
              setMetric(m.id)
              setHoverIdx(null)
            }}
            className={cn(
              'press rounded-md px-2 py-1 text-[11px] font-semibold',
              metric === m.id
                ? 'bg-card text-foreground shadow-xs'
                : 'text-muted-foreground hover:text-foreground',
            )}
          >
            {m.label}
          </button>
        ))}
      </div>

      {/* Period Toggle */}
      <div className="inline-flex rounded-lg border border-border bg-secondary/40 p-0.5">
        {PERIODS.map((p) => (
          <button
            key={p.id}
            type="button"
            onClick={() => {
              setPeriod(p.id)
              setHoverIdx(null)
            }}
            className={cn(
              'press rounded-md px-2.5 py-1 text-[11px] font-semibold',
              period === p.id
                ? 'bg-foreground text-background shadow-xs'
                : 'text-muted-foreground hover:text-foreground',
            )}
          >
            {p.label}
          </button>
        ))}
      </div>
    </div>
  )

  const renderCurve = (isExpanded: boolean) => {
    const strokeId = isExpanded ? `${strokeGradId}-fs` : strokeGradId
    const areaId = isExpanded ? `${areaGradId}-fs` : areaGradId
    return (
      <div className={cn('relative flex flex-1 flex-col', isExpanded && 'min-h-0 justify-between p-4 sm:p-6')}>
        <svg
          viewBox={`0 0 ${SVG_W} ${SVG_H}`}
          className={cn(
            'w-full select-none overflow-visible',
            isExpanded ? 'min-h-0 flex-1' : 'h-auto max-h-[210px]',
          )}
          preserveAspectRatio="xMidYMid meet"
          role="img"
          aria-label={`Kurva performa kumulatif ${summary.count} trade`}
          onPointerLeave={() => setHoverIdx(null)}
        >
          <defs>
            {/* Gradasi stroke vertikal: hijau di atas 0, transisi lembut di garis 0, merah di bawah 0 */}
            <linearGradient
              id={strokeId}
              gradientUnits="userSpaceOnUse"
              x1="0"
              y1={PAD.top}
              x2="0"
              y2={SVG_H - PAD.bottom}
            >
              <stop offset="0%" stopColor="#10b981" />
              <stop offset={`${strokeSoftUp.toFixed(2)}%`} stopColor="#10b981" />
              <stop offset={`${strokeSoftDown.toFixed(2)}%`} stopColor="#f43f5e" />
              <stop offset="100%" stopColor="#f43f5e" />
            </linearGradient>

            {/* Gradasi area vertikal: memudar ke bening tepat di garis 0 agar split warna seamless */}
            <linearGradient
              id={areaId}
              gradientUnits="userSpaceOnUse"
              x1="0"
              y1={PAD.top}
              x2="0"
              y2={SVG_H - PAD.bottom}
            >
              <stop offset="0%" stopColor="#10b981" stopOpacity="0.26" />
              <stop offset={`${areaSoftUp.toFixed(2)}%`} stopColor="#10b981" stopOpacity="0.05" />
              <stop offset={`${zeroPct.toFixed(2)}%`} stopColor="#10b981" stopOpacity="0.01" />
              <stop offset={`${zeroPct.toFixed(2)}%`} stopColor="#f43f5e" stopOpacity="0.01" />
              <stop offset={`${areaSoftDown.toFixed(2)}%`} stopColor="#f43f5e" stopOpacity="0.05" />
              <stop offset="100%" stopColor="#f43f5e" stopOpacity="0.26" />
            </linearGradient>
          </defs>

          {/* Garis baseline 0 */}
          <line
            x1={PAD.left}
            y1={series.zeroY}
            x2={SVG_W - PAD.right}
            y2={series.zeroY}
            className="stroke-border"
            strokeDasharray="4 4"
            strokeWidth="1"
          />
          <text
            x={PAD.left - 8}
            y={series.zeroY + 3}
            textAnchor="end"
            className="fill-muted-foreground font-mono text-[10px]"
          >
            0
          </text>
          <text
            x={PAD.left - 8}
            y={PAD.top + 6}
            textAnchor="end"
            className="fill-muted-foreground font-mono text-[10px]"
          >
            {metric === 'r' ? `${series.hi.toFixed(1)}R` : Math.round(series.hi)}
          </text>
          <text
            x={PAD.left - 8}
            y={SVG_H - PAD.bottom}
            textAnchor="end"
            className="fill-muted-foreground font-mono text-[10px]"
          >
            {metric === 'r' ? `${series.lo.toFixed(1)}R` : Math.round(series.lo)}
          </text>

          {/* Area fill */}
          <motion.path
            key={`area-${isExpanded ? 'fs' : 'in'}-${period}-${metric}-${summary.count}`}
            d={series.areaPath}
            fill={`url(#${areaId})`}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ duration: reduceMotion ? 0.12 : 0.35, ease: EASE_OUT }}
          />

          {/* Garis kurva utama beranimasi */}
          <motion.path
            key={`line-${isExpanded ? 'fs' : 'in'}-${period}-${metric}-${summary.count}`}
            d={series.linePath}
            fill="none"
            stroke={`url(#${strokeId})`}
            strokeWidth="2.4"
            strokeLinecap="round"
            strokeLinejoin="round"
            initial={reduceMotion ? { opacity: 0 } : { pathLength: 0, opacity: 0.4 }}
            animate={{ pathLength: 1, opacity: 1 }}
            transition={{
              duration: reduceMotion ? 0.15 : 0.55,
              ease: EASE_OUT,
            }}
          />

          {/* Crosshair vertikal saat hover */}
          {activeCoord && (
            <line
              x1={activeCoord.x}
              y1={PAD.top}
              x2={activeCoord.x}
              y2={SVG_H - PAD.bottom}
              className="stroke-foreground/30"
              strokeDasharray="3 3"
              strokeWidth="1"
            />
          )}

          {/* Titik-titik trade interaktif */}
          {series.coords.slice(1).map((c, idx) => {
            const coordIndex = idx + 1
            const isLast = coordIndex === series.coords.length - 1
            const isHovered = hoverIdx === coordIndex
            const dotFill = pointColor(c.value)
            return (
              <g key={coordIndex}>
                <circle
                  cx={c.x}
                  cy={c.y}
                  r={isHovered ? 5 : isLast ? 3.8 : 2.5}
                  fill={dotFill}
                  className="stroke-background"
                  strokeWidth={isHovered || isLast ? 2 : 1}
                />
                {/* Hit target lebar biar mudah di-hover maupun di-tap di mobile/tablet */}
                <circle
                  cx={c.x}
                  cy={c.y}
                  r={14}
                  fill="transparent"
                  className="cursor-pointer"
                  onPointerEnter={() => setHoverIdx(coordIndex)}
                  onClick={() =>
                    setHoverIdx((prev) => (prev === coordIndex ? null : coordIndex))
                  }
                />
              </g>
            )
          })}
        </svg>

        {/* Detail Hover Bar (tanpa layout shift) */}
        <div className="mt-1 flex min-h-[24px] shrink-0 items-center justify-between border-t border-border/60 pt-2 text-[11.5px]">
          <AnimatePresence mode="wait">
            {activePoint ? (
              <motion.div
                key={activePoint.positionId}
                initial={{ opacity: 0, y: 2 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0 }}
                transition={{ duration: 0.14, ease: EASE_OUT }}
                className="flex w-full flex-wrap items-center justify-between gap-2"
              >
                <span className="font-medium text-foreground">
                  #{activePoint.index} ·{' '}
                  <span className="font-semibold">{activePoint.symbol}</span>{' '}
                  <span className="text-muted-foreground">{activePoint.direction}</span> ·{' '}
                  <span className="font-mono text-[11px] uppercase text-muted-foreground">
                    {activePoint.strategy === 'ila-2.0' ? 'ILA 2.0' : 'ILA 3.0'}
                  </span>
                </span>
                <span className="flex items-center gap-2 font-mono tabular-nums">
                  <span
                    className={cn(
                      'font-semibold',
                      activePoint.deltaR > 0 || activePoint.deltaPnL > 0
                        ? 'text-emerald-600 dark:text-emerald-400'
                        : activePoint.deltaR < 0 || activePoint.deltaPnL < 0
                          ? 'text-rose-600 dark:text-rose-400'
                          : 'text-foreground',
                    )}
                  >
                    {formatR(activePoint.deltaR)} ({formatMoney(activePoint.deltaPnL)})
                  </span>
                  <span className="text-[11px] text-muted-foreground">
                    {formatDateTime(activePoint.closedAt)}
                  </span>
                </span>
              </motion.div>
            ) : (
              <motion.div
                key="idle"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                transition={{ duration: 0.14 }}
                className="flex w-full items-center justify-between text-muted-foreground"
              >
                <span className="truncate">Pilih titik kurva untuk detail trade</span>
                <span className="font-mono text-[11px]">
                  {summary.wins}W · {summary.losses}L
                </span>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </div>
    )
  }

  return (
    <>
    <Card className="flex h-full flex-col border-border/80 bg-card/95">
      <CardContent className="flex flex-1 flex-col gap-3 p-3.5 sm:gap-4 sm:p-5">
        {/* Header & Controls */}
        <div className="flex flex-wrap items-start justify-between gap-2.5 sm:gap-3">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <h2 className="text-sm font-bold tracking-tight text-foreground">Kurva Performa</h2>
              <span className="rounded-md bg-secondary px-1.5 py-0.5 font-mono text-[10.5px] font-semibold text-muted-foreground">
                {summary.count} trade
              </span>
            </div>
            <div className="mt-1 flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
              <span
                className={cn(
                  'font-mono text-xl font-bold tabular-nums tracking-tight sm:text-2xl',
                  summary.count === 0
                    ? 'text-muted-foreground'
                    : isPositive
                      ? 'text-emerald-600 dark:text-emerald-400'
                      : isNegative
                        ? 'text-rose-600 dark:text-rose-400'
                        : 'text-foreground',
                )}
              >
                {summary.count === 0
                  ? metric === 'r'
                    ? '0,00R'
                    : formatMoney(0)
                  : metric === 'r'
                    ? formatR(summary.totalR)
                    : formatMoney(summary.totalPnL)}
              </span>
              {summary.count > 0 && (
                <span className="font-mono text-xs text-muted-foreground tabular-nums">
                  {metric === 'r' ? formatMoney(summary.totalPnL) : formatR(summary.totalR)}
                  {summary.maxDrawdownR > 0 ? ` · Max DD -${summary.maxDrawdownR.toFixed(2)}R` : ''}
                </span>
              )}
            </div>
          </div>

          <div className="flex items-center gap-1.5">
            {controls}
            {summary.count > 0 && (
              <button
                type="button"
                onClick={() => setFullscreen(true)}
                aria-label="Perbesar kurva performa ke layar penuh"
                title="Layar penuh"
                className="press inline-flex h-7 w-7 items-center justify-center rounded-lg border border-border bg-secondary/40 text-muted-foreground hover:border-foreground/30 hover:text-foreground"
              >
                <Maximize2 className="h-3.5 w-3.5" aria-hidden="true" />
              </button>
            )}
          </div>
        </div>

        {/* Chart Area */}
        {isLoading ? (
          <Skeleton className="h-[190px] w-full rounded-lg" />
        ) : summary.count === 0 ? (
          <div className="flex flex-1 flex-col items-center justify-center rounded-xl border border-dashed border-border bg-secondary/20 px-4 py-8 text-center">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-secondary text-muted-foreground">
              <LineChart className="h-5 w-5" />
            </div>
            <p className="mt-3 text-sm font-semibold text-foreground">
              Belum ada posisi tertutup pada periode ini
            </p>
            <p className="mt-1 max-w-sm text-xs leading-relaxed text-muted-foreground">
              Kurva pertumbuhan R &amp; P/L akan tergambar otomatis setelah eksekusi Time Cycle
              selesai di broker.
            </p>
            <div className="mt-4 flex items-center gap-2">
              <Button asChild size="sm" variant="outline" className="h-8 text-xs active:scale-[0.97]">
                <Link to="/sinyal">
                  Pantau Radar
                  <ArrowRight className="ml-1.5 h-3.5 w-3.5" />
                </Link>
              </Button>
              <Button asChild size="sm" variant="ghost" className="h-8 text-xs active:scale-[0.97]">
                <Link to="/backtest">Uji Backtest</Link>
              </Button>
            </div>
          </div>
        ) : (
          renderCurve(false)
        )}
      </CardContent>
    </Card>

    <ChartFullscreenModal
      open={fullscreen}
      onClose={() => setFullscreen(false)}
      ariaLabel="Tampilan layar penuh kurva performa"
      title={
        <span className="inline-flex flex-wrap items-center gap-2">
          <span>Kurva Performa</span>
          <span className="rounded-md bg-secondary px-1.5 py-0.5 font-mono text-[10.5px] font-semibold text-muted-foreground">
            {summary.count} trade
          </span>
          <span
            className={cn(
              'font-mono text-xs font-bold tabular-nums sm:text-sm',
              isPositive
                ? 'text-emerald-600 dark:text-emerald-400'
                : isNegative
                  ? 'text-rose-600 dark:text-rose-400'
                  : 'text-foreground',
            )}
          >
            {metric === 'r' ? formatR(summary.totalR) : formatMoney(summary.totalPnL)}
          </span>
        </span>
      }
      toolbar={controls}
    >
      {renderCurve(true)}
    </ChartFullscreenModal>
    </>
  )
}
