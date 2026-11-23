import { clampText, isFiniteNum, parseUnixSec } from './parse-utils'

export type ChartLineInput = {
  price: number
  label: string
  kind: 'entry' | 'sl' | 'tp' | 'cisd' | 'sweep' | 'level'
}

export type ChartMarkerInput = {
  time: number
  label: string
  position: 'above' | 'below'
}

export type ChartZoneInput = {
  top: number
  bottom: number
  label?: string
  timeFrom?: number
  timeTo?: number
}

export type AssistantChartSpec = {
  title?: string
  symbol: string
  timeframe: string
  focusTime?: number
  lines: ChartLineInput[]
  markers: ChartMarkerInput[]
  zones: ChartZoneInput[]
}

const TIMEFRAME_ALIASES: Record<string, string> = {
  M1: 'M1',
  '1M': 'M1',
  M5: 'M5',
  '5M': 'M5',
  M15: 'M15',
  '15M': 'M15',
  H1: 'H1',
  '1H': 'H1',
  H4: 'H4',
  '4H': 'H4',
  D1: 'D1',
  D: 'D1',
  '1D': 'D1',
  W: 'W',
  W1: 'W',
  '1W': 'W',
}

/**
 * Memvalidasi blok `ila-chart` dari model. Simbol dan timeframe harus cocok
 * dengan rute market yang sudah ada; sisanya dibuang diam-diam agar input
 * aneh tidak pernah mencapai chart.
 */
export function parseChartSpec(rawJSON: string): AssistantChartSpec | null {
  try {
    const parsed = JSON.parse(rawJSON) as Record<string, unknown>
    if (!parsed || typeof parsed !== 'object') return null

    const rawSym = typeof parsed.symbol === 'string' ? parsed.symbol.trim().toUpperCase() : ''
    if (!/^[A-Z0-9._-]{2,20}$/.test(rawSym)) return null

    const rawTf = typeof parsed.timeframe === 'string' ? parsed.timeframe.trim().toUpperCase() : ''
    const timeframe = TIMEFRAME_ALIASES[rawTf]
    if (!timeframe) return null

    const lines: ChartLineInput[] = []
    if (Array.isArray(parsed.lines)) {
      for (const item of parsed.lines.slice(0, 8)) {
        if (!item || typeof item !== 'object') continue
        const row = item as Record<string, unknown>
        const label = clampText(row.label, 28)
        if (!isFiniteNum(row.price) || !label) continue
        const kind =
          row.kind === 'entry' ||
          row.kind === 'sl' ||
          row.kind === 'tp' ||
          row.kind === 'cisd' ||
          row.kind === 'sweep'
            ? row.kind
            : 'level'
        lines.push({ price: row.price, label, kind })
      }
    }

    const markers: ChartMarkerInput[] = []
    if (Array.isArray(parsed.markers)) {
      for (const item of parsed.markers.slice(0, 12)) {
        if (!item || typeof item !== 'object') continue
        const row = item as Record<string, unknown>
        const time = parseUnixSec(row.time)
        const label = clampText(row.label, 24)
        if (!time || !label) continue
        markers.push({ time, label, position: row.position === 'below' ? 'below' : 'above' })
      }
    }

    const zones: ChartZoneInput[] = []
    if (Array.isArray(parsed.zones)) {
      for (const item of parsed.zones.slice(0, 6)) {
        if (!item || typeof item !== 'object') continue
        const row = item as Record<string, unknown>
        if (!isFiniteNum(row.top) || !isFiniteNum(row.bottom)) continue
        zones.push({
          top: Math.max(row.top, row.bottom),
          bottom: Math.min(row.top, row.bottom),
          label: clampText(row.label, 24),
          timeFrom: parseUnixSec(row.time_from),
          timeTo: parseUnixSec(row.time_to),
        })
      }
    }

    return {
      title: clampText(parsed.title, 72),
      symbol: rawSym,
      timeframe,
      focusTime: parseUnixSec(parsed.focus_time),
      lines,
      markers,
      zones,
    }
  } catch {
    return null
  }
}
