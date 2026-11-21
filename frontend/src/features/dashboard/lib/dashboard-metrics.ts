import { ILA2_SLOTS, ILA3_SLOTS } from '@/features/signals/lib/signal-slots'
import { stageOf } from '@/features/signals/lib/slot-stage'
import type { Position, Signal, StrategyVersion } from '@/types/api'

export type CurvePeriod = '7d' | '30d' | 'all'
export type CurveMetric = 'r' | 'pnl'

export type EquityPoint = {
  index: number
  positionId: string
  symbol: string
  direction: Position['direction']
  strategy: StrategyVersion
  closedAt: string
  deltaR: number
  deltaPnL: number
  cumR: number
  cumPnL: number
  status: Position['status']
}

export type EquityCurveSummary = {
  points: EquityPoint[]
  totalR: number
  totalPnL: number
  wins: number
  losses: number
  winRate: number | null
  count: number
  maxDrawdownR: number
}

export type RadarHighlight = {
  symbol: string
  strategy: StrategyVersion
  cycleName: string
  tfPair: string
  stageLabel: string
  stepCount: number
  totalSteps: number
  rank: number
  ready: boolean
  direction?: 'BUY' | 'SELL'
  signal?: Signal
}

export const EASE_OUT = [0.23, 1, 0.32, 1] as const

/**
 * buildSmoothCurvePath mengubah urutan titik (x terurut naik) menjadi path SVG
 * kurva halus menggunakan monotone cubic Hermite spline (Fritsch-Carlson),
 * sehingga melengkung lembut tanpa overshoot melewati puncak/lembah data nyata.
 */
export function buildSmoothCurvePath(points: Array<{ x: number; y: number }>): string {
  const n = points.length
  if (n === 0) return ''
  const p0 = points[0]!
  if (n === 1) return `M ${p0.x.toFixed(1)} ${p0.y.toFixed(1)}`
  if (n === 2) {
    const p1 = points[1]!
    return `M ${p0.x.toFixed(1)} ${p0.y.toFixed(1)} L ${p1.x.toFixed(1)} ${p1.y.toFixed(1)}`
  }

  const dx: number[] = []
  const slopes: number[] = []
  for (let i = 0; i < n - 1; i++) {
    const stepX = points[i + 1]!.x - points[i]!.x || 1
    dx.push(stepX)
    slopes.push((points[i + 1]!.y - points[i]!.y) / stepX)
  }

  const tangents = new Array<number>(n).fill(0)
  tangents[0] = slopes[0]!
  tangents[n - 1] = slopes[n - 2]!
  for (let i = 1; i < n - 1; i++) {
    const prev = slopes[i - 1]!
    const next = slopes[i]!
    tangents[i] = prev * next <= 0 ? 0 : (prev + next) / 2
  }

  for (let i = 0; i < n - 1; i++) {
    const d = slopes[i]!
    if (Math.abs(d) < 1e-9) {
      tangents[i] = 0
      tangents[i + 1] = 0
      continue
    }
    const a = tangents[i]! / d
    const b = tangents[i + 1]! / d
    const h = Math.hypot(a, b)
    if (h > 3) {
      const scale = 3 / h
      tangents[i] = scale * a * d
      tangents[i + 1] = scale * b * d
    }
  }

  let path = `M ${p0.x.toFixed(1)} ${p0.y.toFixed(1)}`
  for (let i = 0; i < n - 1; i++) {
    const curr = points[i]!
    const next = points[i + 1]!
    const seg = dx[i]! / 3
    const cp1x = curr.x + seg
    const cp1y = curr.y + tangents[i]! * seg
    const cp2x = next.x - seg
    const cp2y = next.y - tangents[i + 1]! * seg
    path += ` C ${cp1x.toFixed(1)} ${cp1y.toFixed(1)}, ${cp2x.toFixed(1)} ${cp2y.toFixed(1)}, ${next.x.toFixed(1)} ${next.y.toFixed(1)}`
  }

  return path
}

function normTF(s: string): string {
  const n = (s || '').toUpperCase().replace(/[\s\-_]+/g, '')
  switch (n) {
    case 'D':
    case 'D1':
    case '1D':
      return 'D1'
    case 'W':
    case 'W1':
    case '1W':
      return 'W'
    case '4H':
    case 'H4':
      return 'H4'
    case '1H':
    case 'H1':
      return 'H1'
    case '15M':
    case 'M15':
      return 'M15'
    case '5M':
    case 'M5':
      return 'M5'
    case '1M':
    case 'M1':
      return 'M1'
    default:
      return n
  }
}

/** Label tahap ringkas selaras istilah New ILA 2.0 & ILA 3.0. */
export function labelTahapDashboard(
  strategy: StrategyVersion,
  status: string,
): { label: string; step: number; total: number; rank: number; ready: boolean } {
  const s = (status || '').toUpperCase().trim()

  if (strategy === 'ila-2.0') {
    switch (s) {
      case 'READY_TO_ENTRY':
      case 'OPPORTUNITY':
        return { label: 'Siap Entry', step: 3, total: 3, rank: 0, ready: true }
      case 'WAIT_LTF_CISD':
        return { label: 'LTF CISD (C3)', step: 2, total: 3, rank: 20, ready: false }
      case 'WAIT_HTF_CANDLE_CLOSE':
        return { label: 'HTF C2 Closure', step: 1, total: 3, rank: 30, ready: false }
      case 'WAIT_HTF_SWEEP':
        return { label: 'HTF Key Level', step: 1, total: 3, rank: 40, ready: false }
      case 'INVALIDATED':
      case 'CANCELLED':
        return { label: 'Tidak Berlaku', step: 0, total: 3, rank: 90, ready: false }
      default:
        return { label: 'Scan Cycle', step: 0, total: 3, rank: 99, ready: false }
    }
  }

  const st = stageOf(strategy, s)
  const step = st.ready ? 3 : st.rank <= 45 ? 2 : st.rank <= 70 ? 1 : 0
  return { label: st.label, step, total: 3, rank: st.rank, ready: st.ready }
}

/** Cari nama Time Cycle baku berdasarkan pasangan HTF/LTF sinyal. */
export function namaTimeCycle(strategy: StrategyVersion, htf?: string, ltf?: string): string {
  const h = normTF(htf ?? '')
  const l = normTF(ltf ?? '')
  if (strategy === 'ila-3.0') {
    const found = ILA3_SLOTS.find((x) => normTF(x.htf) === h && normTF(x.ltf) === l)
    return found?.cycle ?? 'Quarterly Cycle'
  }
  const found = ILA2_SLOTS.find((x) => normTF(x.htf) === h && normTF(x.ltf) === l)
  return found?.cycle ?? 'Time Cycle'
}

/**
 * buildEquityCurve menyusun kurva pertumbuhan R & P/L kumulatif dari posisi tertutup
 * sesuai periode pilihan (7H / 30H / Semua) tanpa perlu endpoint backend baru.
 */
export function buildEquityCurve(
  positions: Position[] | undefined,
  period: CurvePeriod,
  now = new Date(),
): EquityCurveSummary {
  const cutoffMs =
    period === '7d'
      ? now.getTime() - 7 * 24 * 60 * 60 * 1000
      : period === '30d'
        ? now.getTime() - 30 * 24 * 60 * 60 * 1000
        : Number.NEGATIVE_INFINITY

  const closed = (positions ?? [])
    .filter((p) => {
      if (p.status === 'OPEN' || !p.pnl_known || !p.closed_at) return false
      const t = new Date(p.closed_at).getTime()
      return Number.isFinite(t) && t >= cutoffMs
    })
    .sort((a, b) => new Date(a.closed_at!).getTime() - new Date(b.closed_at!).getTime())

  let cumR = 0
  let cumPnL = 0
  let peakR = 0
  let maxDrawdownR = 0
  let wins = 0
  let losses = 0

  const points: EquityPoint[] = closed.map((p, idx) => {
    const r = Number.isFinite(p.outcome_r) ? p.outcome_r : 0
    const pnl = Number.isFinite(p.profit_loss) ? p.profit_loss : 0
    cumR += r
    cumPnL += pnl
    if (cumR > peakR) peakR = cumR
    const dd = peakR - cumR
    if (dd > maxDrawdownR) maxDrawdownR = dd

    if (p.status === 'CLOSED_TP' || pnl > 0) wins += 1
    else if (p.status === 'CLOSED_SL' || pnl < 0) losses += 1

    return {
      index: idx + 1,
      positionId: p.id,
      symbol: p.symbol,
      direction: p.direction,
      strategy: p.strategy_version,
      closedAt: p.closed_at!,
      deltaR: r,
      deltaPnL: pnl,
      cumR: Number(cumR.toFixed(2)),
      cumPnL: Number(cumPnL.toFixed(2)),
      status: p.status,
    }
  })

  const totalDecided = wins + losses
  const winRate = totalDecided > 0 ? Math.round((wins / totalDecided) * 100) : null

  return {
    points,
    totalR: Number(cumR.toFixed(2)),
    totalPnL: Number(cumPnL.toFixed(2)),
    wins,
    losses,
    winRate,
    count: points.length,
    maxDrawdownR: Number(maxDrawdownR.toFixed(2)),
  }
}

function signalUpdatedMs(s: Signal): number {
  return new Date(s.updated_at || s.published_at).getTime()
}

/**
 * buildRadarHighlights meranking pair watchlist berdasarkan setup terdekat ke entry
 * (Siap Entry -> LTF CISD -> HTF C2 Closure -> HTF Key Level -> Scan Cycle).
 */
export function buildRadarHighlights(
  signals: Signal[] | undefined,
  activeSymbols: string[],
  limit = 4,
): RadarHighlight[] {
  const symbols = activeSymbols.length > 0
    ? activeSymbols
    : Array.from(new Set((signals ?? []).map((s) => s.symbol.toUpperCase()))).sort()

  const list: RadarHighlight[] = symbols.map((sym) => {
    const candidates = (signals ?? []).filter((s) => s.symbol.toUpperCase() === sym)
    if (candidates.length === 0) {
      return {
        symbol: sym,
        strategy: 'ila-2.0',
        cycleName: 'Time Cycle',
        tfPair: 'H4 → M15',
        stageLabel: 'Scan Cycle',
        stepCount: 0,
        totalSteps: 3,
        rank: 99,
        ready: false,
      }
    }

    // Ambil kandidat paling dekat entry; tie-break berdasarkan update terbaru.
    const best = candidates.reduce((a, b) => {
      const sa = labelTahapDashboard(a.strategy_version, a.status)
      const sb = labelTahapDashboard(b.strategy_version, b.status)
      if (sb.rank !== sa.rank) return sb.rank < sa.rank ? b : a
      return signalUpdatedMs(b) > signalUpdatedMs(a) ? b : a
    })

    const stage = labelTahapDashboard(best.strategy_version, best.status)
    const dir =
      best.direction === 'BUY' || best.direction === 'SELL' ? best.direction : undefined

    return {
      symbol: sym,
      strategy: best.strategy_version,
      cycleName: namaTimeCycle(best.strategy_version, best.htf, best.ltf),
      tfPair: best.htf && best.ltf ? `${best.htf} → ${best.ltf}` : 'Multi-TF',
      stageLabel: stage.label,
      stepCount: stage.step,
      totalSteps: stage.total,
      rank: stage.rank,
      ready: stage.ready,
      direction: dir,
      signal: best,
    }
  })

  return list
    .sort((a, b) => {
      if (a.rank !== b.rank) return a.rank - b.rank
      const ta = a.signal ? signalUpdatedMs(a.signal) : 0
      const tb = b.signal ? signalUpdatedMs(b.signal) : 0
      if (ta !== tb) return tb - ta
      return a.symbol.localeCompare(b.symbol)
    })
    .slice(0, limit)
}

/** Sinyal aktif yang sudah siap entry / butuh tindakan pengguna. */
export function findActionableSignals(signals: Signal[] | undefined): Signal[] {
  return (signals ?? [])
    .filter((s) => {
      const st = (s.status || '').toUpperCase()
      return st === 'READY_TO_ENTRY' || st === 'OPPORTUNITY'
    })
    .sort((a, b) => signalUpdatedMs(b) - signalUpdatedMs(a))
}
