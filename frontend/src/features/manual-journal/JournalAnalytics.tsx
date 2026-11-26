import { useId, useState } from 'react'
import { Card } from '@/components/ui'
import { cn } from '@/lib/utils'
import type { JournalInsight, JournalStats } from './types'
import { formatAmount, formatSignedAmount, pnlTone } from './model'
import { InfoTooltip, SegmentedControl } from './ui'

export function JournalKpiStrip({ stats }: { stats: JournalStats }) {
  const currency = stats.currency || 'USD'
  const netNum = Number(stats.net_pnl)

  const items = [
    {
      label: 'Net P&L',
      value: stats.trades ? formatSignedAmount(stats.net_pnl, currency) : formatAmount(0, currency),
      sub: `${stats.trades} trade closed`,
      tone: stats.trades ? pnlTone(netNum) : 'text-foreground',
      tip: 'Total net P&L dari trade closed sesuai filter.',
      align: 'start' as const,
    },
    {
      label: 'Win rate',
      value: stats.wins + stats.losses > 0 ? `${Number(stats.win_rate).toFixed(1)}%` : '—',
      sub: `${stats.wins}W · ${stats.losses}L · ${stats.breakeven}BE`,
      tone: 'text-foreground',
      tip: 'Win dibagi (win + loss). Breakeven tidak dihitung.',
      align: 'center' as const,
    },
    {
      label: 'Profit factor',
      value: stats.profit_factor === null ? '—' : Number(stats.profit_factor).toFixed(2),
      sub: `Avg +${formatAmount(stats.average_win, currency)} / −${formatAmount(stats.average_loss, currency)}`,
      tone: 'text-foreground',
      tip: 'Total profit dibagi total loss. Tampil — jika belum ada loss.',
      align: 'center' as const,
    },
    {
      label: 'Max drawdown',
      value: Number(stats.max_drawdown) > 0 ? `−${formatAmount(stats.max_drawdown, currency)}` : formatAmount(0, currency),
      sub: 'Dari peak equity',
      tone: Number(stats.max_drawdown) > 0 ? 'text-rose-600 dark:text-rose-400' : 'text-foreground',
      tip: 'Penurunan terbesar dari peak equity pada filter aktif.',
      align: 'end' as const,
    },
  ]

  return (
    <dl className="grid grid-cols-2 gap-3 lg:grid-cols-4">
      {items.map((item) => (
        <div
          key={item.label}
          className="rounded-xl border border-border/80 bg-card px-3.5 py-3 sm:px-4 sm:py-3.5"
        >
          <dt className="flex items-center gap-1 text-xs text-muted-foreground">
            <span>{item.label}</span>
            <InfoTooltip label={`Definisi ${item.label}`} content={item.tip} align={item.align} />
          </dt>
          <dd className={cn('mt-1.5 truncate text-lg font-semibold tabular-nums sm:text-xl', item.tone)}>
            {item.value}
          </dd>
          <p className="mt-0.5 truncate text-[11px] tabular-nums text-muted-foreground">{item.sub}</p>
        </div>
      ))}
    </dl>
  )
}

const BREAKDOWN_OPTIONS = [
  { value: 'setup', label: 'Setup' },
  { value: 'asset', label: 'Pair' },
  { value: 'session', label: 'Sesi' },
  { value: 'emotion', label: 'Emosi' },
  { value: 'mistake', label: 'Mistake' },
  { value: 'tag', label: 'Tag' },
  { value: 'side', label: 'Arah' },
] as const

export function JournalAnalytics({
  stats,
  insights,
}: {
  stats: JournalStats
  insights: JournalInsight[]
}) {
  const [breakdown, setBreakdown] = useState<(typeof BREAKDOWN_OPTIONS)[number]['value']>('setup')
  const [hoveredIdx, setHoveredIdx] = useState<number | null>(null)
  const gradId = useId()
  const currency = stats.currency || 'USD'
  const points = stats.curve ?? []

  const values = [0, ...points.map((p) => Number(p.cumulative))]
  const min = Math.min(...values)
  const max = Math.max(...values)
  const range = max - min || 1
  const padX = 20
  const plotW = 640
  const topY = 18
  const botY = 162
  const y = (n: number) => botY - ((n - min) / range) * (botY - topY)
  const x = (i: number) => padX + ((i + 1) / Math.max(points.length, 1)) * plotW

  const polylinePoints = `20,${y(0)} ${points.map((p, i) => `${x(i)},${y(Number(p.cumulative))}`).join(' ')}`
  const areaPoints = points.length
    ? `${polylinePoints} ${x(points.length - 1)},${y(0)} 20,${y(0)}`
    : ''

  const positiveCurve = Number(stats.net_pnl) >= 0
  const activePoint = hoveredIdx !== null ? points[hoveredIdx] : null

  const groups = Object.entries(stats.breakdowns?.[breakdown] ?? {}).sort(
    (a, b) => Math.abs(Number(b[1].net_pnl)) - Math.abs(Number(a[1].net_pnl)),
  )
  const scale = Math.max(...groups.map(([, g]) => Math.abs(Number(g.net_pnl))), 1)

  return (
    <div className="space-y-5">
      <div className="grid gap-4 lg:grid-cols-[1.55fr_1fr]">
        <Card className="min-w-0 p-4 sm:p-5">
          <div className="flex flex-wrap items-start justify-between gap-2">
            <div>
              <h2 className="text-sm font-semibold">Equity curve</h2>
              <p className="mt-0.5 text-xs text-muted-foreground">
                Net P&amp;L kumulatif · WIB
              </p>
            </div>
            {activePoint ? (
              <div className="text-right text-xs tabular-nums">
                <span className={cn('font-semibold', pnlTone(activePoint.cumulative))}>
                  {formatSignedAmount(activePoint.cumulative, currency)}
                </span>
                <span className="block text-[11px] text-muted-foreground">
                  {activePoint.date} WIB ({formatSignedAmount(activePoint.pnl, currency)})
                </span>
              </div>
            ) : points.length > 0 ? (
              <span className={cn('text-sm font-semibold tabular-nums', pnlTone(stats.net_pnl))}>
                {formatSignedAmount(stats.net_pnl, currency)}
              </span>
            ) : null}
          </div>

          {!points.length ? (
            <p className="py-14 text-center text-sm text-muted-foreground">
              Equity curve muncul setelah ada trade closed
            </p>
          ) : (
            <>
              <svg
                viewBox="0 0 680 184"
                role="img"
                aria-label={`Equity curve ${currency}`}
                className={cn(
                  'mt-4 w-full',
                  positiveCurve
                    ? 'text-emerald-600 dark:text-emerald-400'
                    : 'text-rose-600 dark:text-rose-400',
                )}
                onMouseLeave={() => setHoveredIdx(null)}
              >
                <defs>
                  <linearGradient id={gradId} x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stopColor="currentColor" stopOpacity="0.18" />
                    <stop offset="100%" stopColor="currentColor" stopOpacity="0" />
                  </linearGradient>
                </defs>
                <line
                  x1="20"
                  y1={y(0)}
                  x2="660"
                  y2={y(0)}
                  strokeDasharray="4 4"
                  className="stroke-border"
                />
                {areaPoints && <polygon points={areaPoints} fill={`url(#${gradId})`} />}
                <polyline
                  points={polylinePoints}
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinejoin="round"
                  strokeLinecap="round"
                />
                {points.map((p, i) => {
                  const cx = x(i)
                  const cy = y(Number(p.cumulative))
                  const isActive = hoveredIdx === i
                  return (
                    <g key={p.id}>
                      <circle
                        cx={cx}
                        cy={cy}
                        r={isActive ? 5 : 2.5}
                        fill="currentColor"
                        className="transition-[r] duration-150"
                      />
                      <circle
                        cx={cx}
                        cy={cy}
                        r="14"
                        fill="transparent"
                        className="cursor-pointer"
                        onMouseEnter={() => setHoveredIdx(i)}
                        onClick={() => setHoveredIdx(isActive ? null : i)}
                      >
                        <title>
                          {p.date} WIB: {formatSignedAmount(p.cumulative, currency)}
                        </title>
                      </circle>
                    </g>
                  )
                })}
              </svg>
              <div className="mt-1 flex justify-between gap-3 text-[11px] tabular-nums text-muted-foreground">
                <span>{points[0]?.date.slice(0, 10)}</span>
                <span>{points.length} trade</span>
                <span>{points.at(-1)?.date.slice(0, 10)}</span>
              </div>
            </>
          )}
        </Card>

        <Card className="min-w-0 p-4 sm:p-5">
          <div className="flex items-center justify-between gap-2">
            <h2 className="text-sm font-semibold">Breakdown performa</h2>
            <InfoTooltip
              label="Info breakdown performa"
              align="end"
              content="Trade dengan lebih dari satu label dihitung di tiap kategorinya."
            />
          </div>

          <div className="mt-3 overflow-x-auto pb-1 scrollbar-none">
            <SegmentedControl
              ariaLabel="Kategori breakdown"
              size="sm"
              value={breakdown}
              onChange={setBreakdown}
              options={BREAKDOWN_OPTIONS}
            />
          </div>

          {!groups.length ? (
            <p className="py-12 text-center text-sm text-muted-foreground">
              Belum ada data pada kategori ini
            </p>
          ) : (
            <ul className="mt-4 max-h-64 space-y-3 overflow-y-auto pr-1 scrollbar-thin">
              {groups.map(([label, group]) => {
                const net = Number(group.net_pnl)
                const wr = group.trades > 0 ? Math.round((group.wins / group.trades) * 100) : 0
                return (
                  <li key={label} className="space-y-1">
                    <div className="flex items-baseline justify-between gap-2 text-xs">
                      <span className="truncate font-medium text-foreground">{label}</span>
                      <span className={cn('shrink-0 font-semibold tabular-nums', pnlTone(net))}>
                        {formatSignedAmount(group.net_pnl, currency)}
                      </span>
                    </div>
                    <div className="h-1.5 overflow-hidden rounded-full bg-secondary">
                      <div
                        className={cn(
                          'h-full rounded-full',
                          net < 0
                            ? 'bg-rose-500/80 dark:bg-rose-400/80'
                            : 'bg-emerald-500/80 dark:bg-emerald-400/80',
                        )}
                        style={{
                          width: `${Math.max(4, Math.min(100, (Math.abs(net) / scale) * 100))}%`,
                        }}
                      />
                    </div>
                    <p className="text-[11px] tabular-nums text-muted-foreground">
                      {group.trades} trade · {wr}% win
                    </p>
                  </li>
                )
              })}
            </ul>
          )}
        </Card>
      </div>

      <Card className="p-4 sm:p-5">
        <div className="flex items-center justify-between gap-2">
          <h2 className="text-sm font-semibold">Catatan pola</h2>
          <span className="text-[11px] text-muted-foreground">Otomatis · berbasis aturan</span>
        </div>
        {!insights.length ? (
          <p className="mt-2 text-sm text-muted-foreground">
            Belum ada pola berisiko pada filter ini
          </p>
        ) : (
          <ul className="mt-3 space-y-2 text-sm">
            {insights.map((insight, i) => (
              <li
                key={`${insight.code}-${i}`}
                className="flex items-start gap-2.5 rounded-lg border border-border/70 bg-secondary/30 px-3 py-2 text-xs leading-relaxed text-foreground"
              >
                <span className="mt-1 h-1.5 w-1.5 shrink-0 rounded-full bg-foreground/60" aria-hidden="true" />
                <span>{insight.message}</span>
              </li>
            ))}
          </ul>
        )}
      </Card>
    </div>
  )
}
