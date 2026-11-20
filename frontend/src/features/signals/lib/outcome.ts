import { formatMoney, formatR } from '@/lib/format'
import type { Signal } from '@/types/api'
import { SIGNAL_LABEL } from './status'

export type OutcomeTone = 'tp' | 'sl' | 'active' | 'ready' | 'neutral' | 'muted'

export type SignalOutcomeDescriptor = {
  label: string
  tone: OutcomeTone
  executedBadge?: boolean
  subtext?: string
  resolvedAt?: string | null
}

export const OUTCOME_BADGE_CLASS: Record<OutcomeTone, string> = {
  tp: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400',
  sl: 'border-rose-500/30 bg-rose-500/10 text-rose-700 dark:text-rose-400',
  active: 'border-amber-500/35 bg-amber-500/10 text-amber-700 dark:text-amber-400',
  ready: 'border-foreground bg-foreground text-background',
  neutral: 'border-border bg-secondary/70 text-foreground',
  muted: 'border-border/80 bg-secondary/40 text-muted-foreground',
}

export function isRadarProgressSignal(signal: Signal): boolean {
  if (signal.signal_id?.startsWith('status-')) return true
  const st = (signal.status || '').toUpperCase()
  if (st.startsWith('WAIT_') || st.startsWith('ILA3_')) {
    return !(signal.entry_price > 0)
  }
  return false
}

function isActionableLive(signal: Signal): boolean {
  if (signal.status !== 'READY_TO_ENTRY' && signal.status !== 'OPPORTUNITY') {
    return false
  }
  if (signal.can_confirm) return true
  const expMs = new Date(signal.expires_at).getTime()
  if (!Number.isFinite(expMs)) return true
  return expMs > Date.now()
}

/**
 * describeSignalOutcome menerjemahkan status sinyal + posisi nyata + forward-eval
 * menjadi satu badge hasil akhir yang jelas bagi trader di Histori Setup Sinyal.
 */
export function describeSignalOutcome(signal: Signal): SignalOutcomeDescriptor {
  if (isRadarProgressSignal(signal)) {
    return {
      label: SIGNAL_LABEL[signal.status] ?? signal.status,
      tone: 'neutral',
    }
  }

  if (isActionableLive(signal) && !signal.is_executed) {
    return {
      label: SIGNAL_LABEL[signal.status] ?? 'Siap entry',
      tone: 'ready',
    }
  }

  if (signal.is_executed) {
    const posStatus = signal.position_status
    const hasR = signal.pnl_known && typeof signal.outcome_r === 'number' && Number.isFinite(signal.outcome_r)
    const pnlText =
      signal.pnl_known && typeof signal.profit_loss === 'number' && Number.isFinite(signal.profit_loss)
        ? formatMoney(signal.profit_loss)
        : undefined

    if (posStatus === 'OPEN') {
      return {
        label: 'Sedang Berjalan',
        tone: 'active',
        executedBadge: true,
      }
    }
    if (posStatus === 'CLOSED_TP') {
      return {
        label: hasR ? `Hit TP (${formatR(signal.outcome_r ?? 0)})` : 'Hit TP',
        tone: 'tp',
        executedBadge: true,
        subtext: pnlText,
        resolvedAt: signal.closed_at,
      }
    }
    if (posStatus === 'CLOSED_SL') {
      return {
        label: hasR ? `Hit SL (${formatR(signal.outcome_r ?? 0)})` : 'Hit SL',
        tone: 'sl',
        executedBadge: true,
        subtext: pnlText,
        resolvedAt: signal.closed_at,
      }
    }
    if (posStatus === 'CLOSED_MANUAL') {
      return {
        label: hasR ? `Tutup Manual (${formatR(signal.outcome_r ?? 0)})` : 'Tutup Manual',
        tone: 'neutral',
        executedBadge: true,
        subtext: pnlText,
        resolvedAt: signal.closed_at,
      }
    }
    return {
      label: 'Dieksekusi',
      tone: 'neutral',
      executedBadge: true,
    }
  }

  switch (signal.eval_status) {
    case 'TP': {
      const rVal =
        typeof signal.eval_r_lower === 'number' && Number.isFinite(signal.eval_r_lower) && signal.eval_r_lower > 0
          ? signal.eval_r_lower
          : signal.risk_reward
      return {
        label: rVal > 0 ? `Hit TP (${formatR(rVal)})` : 'Hit TP',
        tone: 'tp',
        resolvedAt: signal.eval_at,
      }
    }
    case 'SL': {
      const rVal =
        typeof signal.eval_r_lower === 'number' && Number.isFinite(signal.eval_r_lower) && signal.eval_r_lower < 0
          ? signal.eval_r_lower
          : -1
      return {
        label: `Hit SL (${formatR(rVal)})`,
        tone: 'sl',
        resolvedAt: signal.eval_at,
      }
    }
    case 'RUNNING':
      return {
        label: 'Sedang Berjalan',
        tone: 'active',
      }
    case 'NO_ENTRY':
      return {
        label: 'Tidak Tersentuh',
        tone: 'muted',
        resolvedAt: signal.eval_at,
      }
    case 'UNKNOWN':
      return {
        label: 'Data Terbatas',
        tone: 'muted',
        subtext: signal.eval_note || undefined,
      }
    case 'PENDING':
      return {
        label: 'Menunggu Evaluasi',
        tone: 'neutral',
      }
    default:
      break
  }

  if (signal.status === 'CANCELLED' || signal.status === 'INVALIDATED') {
    return {
      label: SIGNAL_LABEL[signal.status] ?? 'Struktur batal',
      tone: 'muted',
    }
  }
  if (signal.status === 'ILA3_TARGET_REACHED') {
    return {
      label: 'Hit TP',
      tone: 'tp',
    }
  }

  return {
    label: 'Menunggu Evaluasi',
    tone: 'neutral',
  }
}

function canonClientTF(tf: string): string {
  const n = (tf || '').toUpperCase().replace(/[\s\-_]+/g, '')
  switch (n) {
    case 'D':
    case 'D1':
    case '1D':
      return 'D'
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
    case '30M':
    case 'M30':
      return 'M30'
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

/**
 * dedupeSetupSignals menyaring pengulangan setup identik di sisi klien sebagai
 * pengaman berlapis di atas deduplikasi kohort backend.
 */
export function dedupeSetupSignals(signals: Signal[]): Signal[] {
  const seenIds = new Set<string>()
  const uniqueById: Signal[] = []
  for (const s of signals) {
    if (s.signal_id) {
      if (seenIds.has(s.signal_id)) continue
      seenIds.add(s.signal_id)
    }
    uniqueById.push(s)
  }

  const radar: Signal[] = []
  const setups: Signal[] = []
  for (const s of uniqueById) {
    if (isRadarProgressSignal(s)) {
      radar.push(s)
    } else {
      setups.push(s)
    }
  }

  const chronological = [...setups].sort(
    (a, b) => new Date(a.published_at).getTime() - new Date(b.published_at).getTime(),
  )

  const WINDOW_MS = 12 * 60 * 60 * 1000
  const deduped: Signal[] = []
  const rootsByKey = new Map<string, number[]>()

  for (const item of chronological) {
    if (!(item.entry_price > 0 && item.stop_loss > 0 && item.take_profit > 0)) {
      deduped.push(item)
      continue
    }
    const key = [
      item.symbol.toUpperCase(),
      item.strategy_version,
      canonClientTF(item.htf),
      canonClientTF(item.ltf),
      item.direction.toUpperCase(),
      Math.round(item.entry_price * 1e6),
      Math.round(item.stop_loss * 1e6),
      Math.round(item.take_profit * 1e6),
    ].join('|')

    const pubMs = new Date(item.published_at).getTime()
    const bucket = rootsByKey.get(key) ?? []
    let matchedIdx = -1

    for (let i = bucket.length - 1; i >= 0; i--) {
      const idx = bucket[i]
      if (idx === undefined) continue
      const cand = deduped[idx]
      if (!cand) continue
      const candPub = new Date(cand.published_at).getTime()
      if (pubMs - candPub > WINDOW_MS) continue
      const doneAt = cand.closed_at || cand.eval_at
      const isFinished =
        cand.eval_status === 'TP' ||
        cand.eval_status === 'SL' ||
        (cand.is_executed && cand.position_status && cand.position_status !== 'OPEN')
      if (isFinished && doneAt && new Date(doneAt).getTime() <= pubMs) {
        continue
      }
      matchedIdx = idx
      break
    }

    if (matchedIdx < 0) {
      const nextIdx = deduped.length
      deduped.push({ ...item })
      bucket.push(nextIdx)
      rootsByKey.set(key, bucket)
    } else {
      const root = deduped[matchedIdx]
      if (root) {
        if (item.is_executed && !root.is_executed) {
          deduped[matchedIdx] = {
            ...item,
            published_at: root.published_at,
            eval_status: item.eval_status ?? root.eval_status,
            eval_r_lower: item.eval_r_lower || root.eval_r_lower,
            eval_at: item.eval_at ?? root.eval_at,
          }
        } else if (!root.eval_status && item.eval_status) {
          root.eval_status = item.eval_status
          root.eval_r_lower = item.eval_r_lower
          root.eval_at = item.eval_at
          root.eval_price = item.eval_price
          root.eval_note = item.eval_note
        }
      }
    }
  }

  deduped.sort((a, b) => new Date(b.published_at).getTime() - new Date(a.published_at).getTime())
  return [...radar, ...deduped]
}
