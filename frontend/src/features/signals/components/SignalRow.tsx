import { formatDateTime, formatNumber } from '@/lib/format'
import { StatusPill } from '@/components/ui'
import { cn } from '@/lib/utils'
import type { Signal } from '@/types/api'
import { SIGNAL_LABEL, SIGNAL_TONE } from '../lib/status'

/**
 * SignalRow adalah satu baris padat pada daftar sinyal.
 * Seluruh baris dapat diklik dan dapat difokus keyboard.
 */
export function SignalRow({
  signal,
  selected,
  onSelect,
}: {
  signal: Signal
  selected: boolean
  onSelect: () => void
}) {
  return (
    <li>
      <button
        type="button"
        onClick={onSelect}
        aria-pressed={selected}
        className={cn(
          'flex w-full flex-wrap items-center gap-x-3 gap-y-1 px-3 py-2.5 text-left text-[13px] transition-colors',
          selected ? 'bg-secondary' : 'hover:bg-secondary/60',
        )}
      >
        <span className="w-20 shrink-0 font-semibold">{signal.symbol}</span>
        <span className="w-12 shrink-0 text-muted-foreground">{signal.direction}</span>
        <span className="tabular w-24 shrink-0">{formatNumber(signal.entry_price, 5)}</span>
        <span className="tabular hidden w-16 shrink-0 text-muted-foreground sm:inline">
          {formatNumber(signal.risk_reward, 2)}R
        </span>
        <span className="ml-auto flex items-center gap-2">
          {signal.delayed && (
            <span className="text-micro uppercase tracking-wide text-muted-foreground">Telat</span>
          )}
          <StatusPill tone={SIGNAL_TONE[signal.status]}>{SIGNAL_LABEL[signal.status]}</StatusPill>
        </span>
        <span className="w-full text-micro text-muted-foreground sm:w-auto sm:flex-1">
          {formatDateTime(signal.published_at)}
        </span>
      </button>
    </li>
  )
}
