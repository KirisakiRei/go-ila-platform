import { useMemo } from 'react'
import { useChartTheme } from '@/features/charts/lib/theme'
import type { DiagramLevel, DiagramSpec } from './lib/diagram-spec'

type CandleDiagramProps = {
  spec: DiagramSpec
}

const SVG_W = 640
const SVG_H = 270
const PAD_LEFT = 20
const PAD_RIGHT = 132
const PAD_TOP = 30
const PAD_BOTTOM = 34

/**
 * Merender ilustrasi skematik candlestick New ILA 2.0 secara deterministik
 * dengan SVG React murni (tanpa dangerouslySetInnerHTML).
 */
export function CandleDiagram({ spec }: CandleDiagramProps) {
  const theme = useChartTheme()

  const layout = useMemo(() => {
    let minVal = Infinity
    let maxVal = -Infinity
    for (const c of spec.candles) {
      if (c.l < minVal) minVal = c.l
      if (c.h > maxVal) maxVal = c.h
    }
    for (const lv of spec.levels) {
      if (lv.price < minVal) minVal = lv.price
      if (lv.price > maxVal) maxVal = lv.price
    }
    for (const z of spec.zones) {
      if (z.bottom < minVal) minVal = z.bottom
      if (z.top > maxVal) maxVal = z.top
    }
    if (!Number.isFinite(minVal) || !Number.isFinite(maxVal)) {
      minVal = 0
      maxVal = 100
    }
    const span = Math.max(maxVal - minVal, 1e-6)
    const lo = minVal - span * 0.16
    const hi = maxVal + span * 0.16
    const plotW = SVG_W - PAD_LEFT - PAD_RIGHT
    const plotH = SVG_H - PAD_TOP - PAD_BOTTOM
    const toY = (price: number) => PAD_TOP + ((hi - price) / (hi - lo)) * plotH
    const step = plotW / spec.candles.length
    const toX = (idx: number) => PAD_LEFT + step * (idx + 0.5)
    return { plotW, plotH, toY, toX, bodyW: Math.max(10, Math.min(28, step * 0.54)) }
  }, [spec])

  const levelColor = (kind: DiagramLevel['kind']) => {
    switch (kind) {
      case 'entry':
        return theme.entry
      case 'sl':
        return theme.stopLoss
      case 'tp':
        return theme.takeProfit
      case 'cisd':
        return theme.cisd
      case 'sweep':
        return theme.sweep
      default:
        return theme.neutral
    }
  }

  // Label level di sisi kanan didorong turun bila terlalu rapat agar tidak menumpuk.
  const placedLabels = useMemo(() => {
    const items = spec.levels
      .map((lv, idx) => ({ ...lv, idx, y: layout.toY(lv.price), labelY: layout.toY(lv.price) }))
      .sort((a, b) => a.y - b.y)
    const minGap = 17
    for (let i = 1; i < items.length; i++) {
      const prev = items[i - 1]
      const cur = items[i]
      if (prev && cur && cur.labelY - prev.labelY < minGap) {
        cur.labelY = prev.labelY + minGap
      }
    }
    return items
  }, [spec.levels, layout])

  return (
    <figure className="my-2 overflow-hidden rounded-xl border border-border bg-card">
      {(spec.title || spec.subtitle) && (
        <figcaption className="flex flex-wrap items-center justify-between gap-2 border-b border-border/80 px-3 py-2">
          <span className="text-[12.5px] font-semibold tracking-tight text-foreground">
            {spec.title ?? 'Ilustrasi Pola New ILA 2.0'}
          </span>
          {spec.subtitle && (
            <span className="rounded bg-secondary px-1.5 py-0.5 text-[11px] font-medium text-muted-foreground">
              {spec.subtitle}
            </span>
          )}
        </figcaption>
      )}

      <div className="w-full" style={{ backgroundColor: theme.background }}>
        <svg
          viewBox={`0 0 ${SVG_W} ${SVG_H}`}
          role="img"
          aria-label={spec.title ?? 'Diagram candlestick New ILA 2.0'}
          className="block h-auto w-full select-none"
        >
          {[0.2, 0.5, 0.8].map((ratio) => {
            const y = PAD_TOP + layout.plotH * ratio
            return (
              <line
                key={ratio}
                x1={PAD_LEFT}
                y1={y}
                x2={SVG_W - PAD_RIGHT}
                y2={y}
                stroke={theme.grid}
                strokeWidth={1}
              />
            )
          })}

          {spec.zones.map((zone, zIdx) => {
            const yTop = layout.toY(zone.top)
            const yBot = layout.toY(zone.bottom)
            const x1 =
              zone.from_idx !== undefined
                ? layout.toX(zone.from_idx) - layout.bodyW * 0.7
                : PAD_LEFT
            const x2 =
              zone.to_idx !== undefined
                ? layout.toX(zone.to_idx) + layout.bodyW * 0.7
                : SVG_W - PAD_RIGHT
            const height = Math.max(4, yBot - yTop)
            return (
              <g key={zIdx}>
                <rect
                  x={x1}
                  y={yTop}
                  width={Math.max(12, x2 - x1)}
                  height={height}
                  rx={3}
                  fill={theme.fvg}
                  stroke={theme.fvgBorder}
                  strokeWidth={1}
                  strokeDasharray="3 3"
                />
                {zone.label && (
                  <text
                    x={x1 + 5}
                    y={yTop + Math.min(13, height / 2 + 4)}
                    fontSize={10}
                    fontWeight={600}
                    fill={theme.entry}
                  >
                    {zone.label}
                  </text>
                )}
              </g>
            )
          })}

          {placedLabels.map((lv) => {
            const color = levelColor(lv.kind)
            const lineEndX = SVG_W - PAD_RIGHT
            return (
              <g key={lv.idx}>
                <line
                  x1={PAD_LEFT}
                  y1={lv.y}
                  x2={lineEndX}
                  y2={lv.y}
                  stroke={color}
                  strokeWidth={1.25}
                  strokeDasharray={lv.kind === 'entry' ? undefined : '4 3'}
                />
                <line
                  x1={lineEndX}
                  y1={lv.y}
                  x2={lineEndX + 6}
                  y2={lv.labelY}
                  stroke={color}
                  strokeWidth={1}
                  strokeOpacity={0.6}
                />
                <text x={lineEndX + 10} y={lv.labelY + 3.5} fontSize={10.5} fontWeight={600} fill={color}>
                  {lv.label}
                </text>
              </g>
            )
          })}

          {spec.candles.map((c, idx) => {
            const cx = layout.toX(idx)
            const color = c.c >= c.o ? theme.up : theme.down
            const yHigh = layout.toY(c.h)
            const yLow = layout.toY(c.l)
            const yOpen = layout.toY(c.o)
            const yClose = layout.toY(c.c)
            const bodyTop = Math.min(yOpen, yClose)
            const topNote = c.marker ?? c.tag
            return (
              <g key={idx}>
                <line
                  x1={cx}
                  y1={yHigh}
                  x2={cx}
                  y2={yLow}
                  stroke={color}
                  strokeWidth={1.7}
                  strokeLinecap="round"
                />
                <rect
                  x={cx - layout.bodyW / 2}
                  y={bodyTop}
                  width={layout.bodyW}
                  height={Math.max(2.5, Math.abs(yClose - yOpen))}
                  rx={2}
                  fill={color}
                />
                {topNote && (
                  <text
                    x={cx}
                    y={Math.max(13, yHigh - 7)}
                    textAnchor="middle"
                    fontSize={10}
                    fontWeight={600}
                    fill={theme.text}
                  >
                    {topNote}
                  </text>
                )}
                {c.label && (
                  <text
                    x={cx}
                    y={Math.min(SVG_H - 9, yLow + 14)}
                    textAnchor="middle"
                    fontSize={10}
                    fontWeight={500}
                    fill={theme.text}
                  >
                    {c.label}
                  </text>
                )}
              </g>
            )
          })}
        </svg>
      </div>
    </figure>
  )
}
