import { clampText, isFiniteNum } from './parse-utils'

export type DiagramCandle = {
  o: number
  h: number
  l: number
  c: number
  label?: string
  marker?: string
  tag?: string
}

export type DiagramLevel = {
  price: number
  label: string
  kind: 'entry' | 'sl' | 'tp' | 'cisd' | 'sweep' | 'level'
}

export type DiagramZone = {
  from_idx?: number
  to_idx?: number
  top: number
  bottom: number
  label?: string
}

export type DiagramSpec = {
  title?: string
  subtitle?: string
  candles: DiagramCandle[]
  levels: DiagramLevel[]
  zones: DiagramZone[]
}

const MAX_CANDLES = 24
const MAX_LEVELS = 8
const MAX_ZONES = 4

/**
 * Memvalidasi blok `ila-diagram` dari model menjadi data siap gambar.
 * Candle yang high/low-nya tidak membungkus open/close diperbaiki diam-diam
 * supaya batang lilin tidak pernah tergambar terbalik.
 */
export function parseDiagramSpec(rawJSON: string): DiagramSpec | null {
  try {
    const parsed = JSON.parse(rawJSON) as Record<string, unknown>
    if (!parsed || typeof parsed !== 'object' || !Array.isArray(parsed.candles)) {
      return null
    }

    const candles: DiagramCandle[] = []
    for (const item of parsed.candles.slice(0, MAX_CANDLES)) {
      if (!item || typeof item !== 'object') continue
      const row = item as Record<string, unknown>
      if (
        !isFiniteNum(row.o) ||
        !isFiniteNum(row.h) ||
        !isFiniteNum(row.l) ||
        !isFiniteNum(row.c)
      ) {
        continue
      }
      candles.push({
        o: row.o,
        h: Math.max(row.h, row.o, row.c),
        l: Math.min(row.l, row.o, row.c),
        c: row.c,
        label: clampText(row.label, 20),
        marker: clampText(row.marker, 24),
        tag: clampText(row.tag, 20),
      })
    }
    if (candles.length < 2) return null

    const levels: DiagramLevel[] = []
    if (Array.isArray(parsed.levels)) {
      for (const item of parsed.levels.slice(0, MAX_LEVELS)) {
        if (!item || typeof item !== 'object') continue
        const row = item as Record<string, unknown>
        const label = clampText(row.label, 32)
        if (!isFiniteNum(row.price) || !label) continue
        const kind =
          row.kind === 'entry' ||
          row.kind === 'sl' ||
          row.kind === 'tp' ||
          row.kind === 'cisd' ||
          row.kind === 'sweep'
            ? row.kind
            : 'level'
        levels.push({ price: row.price, label, kind })
      }
    }

    const zones: DiagramZone[] = []
    if (Array.isArray(parsed.zones)) {
      for (const item of parsed.zones.slice(0, MAX_ZONES)) {
        if (!item || typeof item !== 'object') continue
        const row = item as Record<string, unknown>
        if (!isFiniteNum(row.top) || !isFiniteNum(row.bottom)) continue
        const top = Math.max(row.top, row.bottom)
        const bottom = Math.min(row.top, row.bottom)
        if (top === bottom) continue
        const lastIdx = candles.length - 1
        zones.push({
          from_idx: isFiniteNum(row.from_idx) ? Math.max(0, Math.floor(row.from_idx)) : undefined,
          to_idx: isFiniteNum(row.to_idx) ? Math.min(lastIdx, Math.floor(row.to_idx)) : undefined,
          top,
          bottom,
          label: clampText(row.label, 24),
        })
      }
    }

    return {
      title: clampText(parsed.title, 72),
      subtitle: clampText(parsed.subtitle, 40),
      candles,
      levels,
      zones,
    }
  } catch {
    return null
  }
}
