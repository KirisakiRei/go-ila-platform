import { TrendingDown, TrendingUp } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Signal } from '@/types/api'
import { buildIla2Timeline } from '../lib/ila2-progress'
import { mapSignalsToSlots, timeframeSlotDipantau, type TimeframeSlot } from '../lib/signal-slots'
import { SIGNAL_LABEL } from '../lib/status'

function ringkasOverview(
  slot: TimeframeSlot,
  strategy: 'ila-2.0' | 'ila-3.0',
  dipantau: boolean,
): { badge: string; badgeTone: 'ready' | 'wait' | 'off' | 'muted'; note: string } {
  if (!dipantau) {
    return { badge: 'OFF', badgeTone: 'off', note: 'Tidak dipantau' }
  }

  const sig = slot.latestSignal
  if (!sig) {
    return { badge: 'SCAN', badgeTone: 'muted', note: 'Memantau pasar...' }
  }

  if (strategy === 'ila-2.0') {
    const tl = buildIla2Timeline(sig)
    if (tl.entry.confirmed) {
      return {
        badge: 'READY',
        badgeTone: 'ready',
        note: `Entry ${sig.direction} terkonfirmasi`,
      }
    }
    if (tl.htf.state === 'invalid') {
      return { badge: 'INVALID', badgeTone: 'muted', note: 'C3 close tanpa setup' }
    }
    if (tl.ltf.state === 'active') {
      return { badge: 'WAIT CISD', badgeTone: 'wait', note: `C2 valid · ${tl.ltf.erlStatus}` }
    }
    if (sig.status === 'WAIT_HTF_CANDLE_CLOSE' || sig.htf_sweep?.sweep_in_progress) {
      return { badge: 'WAIT C2', badgeTone: 'wait', note: 'Candle C2 sedang berjalan' }
    }
    return { badge: 'WAIT C2', badgeTone: 'muted', note: 'Menunggu C2 closure' }
  }

  if (slot.status === 'READY') {
    return { badge: 'READY', badgeTone: 'ready', note: SIGNAL_LABEL[sig.status] ?? 'Siap entry' }
  }
  if (slot.status === 'WAIT_CISD') {
    return { badge: 'WAIT CISD', badgeTone: 'wait', note: SIGNAL_LABEL[sig.status] ?? 'Menunggu CISD' }
  }
  if (slot.status === 'WAIT_SWEEP') {
    return { badge: 'WAIT HTF', badgeTone: 'muted', note: SIGNAL_LABEL[sig.status] ?? 'Pantau HTF' }
  }
  if (slot.status === 'INVALIDATED') {
    return { badge: 'INVALID', badgeTone: 'muted', note: 'Struktur batal' }
  }
  return { badge: 'SCAN', badgeTone: 'muted', note: SIGNAL_LABEL[sig.status] ?? 'Scanning' }
}

/**
 * CycleSelectorStrip menampilkan kartu-kartu Time Cycle yang dapat dipilih di
 * bagian atas halaman detail pair (di bawah tab metode, di atas radar & histori).
 */
export function CycleSelectorStrip({
  signals,
  strategy,
  timeframeAktif,
  activeSlotKey,
  onSelectSlot,
}: {
  signals: Signal[]
  strategy: 'ila-2.0' | 'ila-3.0'
  timeframeAktif?: Set<string> | null
  activeSlotKey: string
  onSelectSlot: (key: string) => void
}) {
  const signalMetode = signals.filter((s) => s.strategy_version === strategy)
  const slots = mapSignalsToSlots(signalMetode, strategy)
  const aktif = timeframeAktif ?? null

  return (
    <div
      role="group"
      aria-label={strategy === 'ila-2.0' ? 'Pilih Time Cycle' : 'Pilih Siklus Timeframe'}
      className={cn(
        'grid gap-2.5',
        strategy === 'ila-2.0'
          ? 'grid-cols-2 sm:grid-cols-3 lg:grid-cols-5'
          : 'grid-cols-2 sm:grid-cols-4',
      )}
    >
      {slots.map((slot) => {
        const dipantau = timeframeSlotDipantau(aktif, slot)
        const isActive = slot.key === activeSlotKey
        const overview = ringkasOverview(slot, strategy, dipantau)
        const sig = dipantau ? slot.latestSignal : undefined
        const isBuy = sig?.direction === 'BUY'
        const isSell = sig?.direction === 'SELL'

        return (
          <button
            key={slot.key}
            type="button"
            aria-pressed={isActive}
            onClick={() => onSelectSlot(slot.key)}
            className={cn(
              'group relative flex flex-col justify-between gap-2 rounded-xl border p-3 text-left transition-all',
              isActive
                ? 'border-foreground bg-card ring-1 ring-foreground shadow-sm'
                : dipantau
                  ? overview.badgeTone === 'ready'
                    ? 'border-emerald-500/60 bg-emerald-500/[0.03] hover:border-emerald-500'
                    : 'border-border bg-card hover:border-foreground/40 hover:bg-secondary/20'
                  : 'border-dashed border-border/80 bg-secondary/15 opacity-75 hover:opacity-100 hover:border-foreground/40',
            )}
          >
            {/* Baris 1: Nama Cycle + Badge Status */}
            <div className="flex items-start justify-between gap-1.5">
              <span className="text-xs font-bold tracking-tight text-foreground leading-snug">
                {slot.cycle ?? slot.label}
              </span>
              <span
                className={cn(
                  'shrink-0 rounded px-1.5 py-0.5 font-mono text-[9.5px] font-bold leading-none',
                  overview.badgeTone === 'ready'
                    ? 'bg-emerald-600 text-white'
                    : overview.badgeTone === 'wait'
                      ? 'bg-amber-500/15 text-amber-700 dark:text-amber-400 border border-amber-500/30'
                      : overview.badgeTone === 'off'
                        ? 'bg-secondary text-muted-foreground border border-border/70'
                        : 'bg-muted text-muted-foreground',
                )}
              >
                {overview.badge}
              </span>
            </div>

            {/* Baris 2: Pasangan TF + Arah */}
            <div className="flex items-center justify-between gap-2">
              <span className="font-mono text-[11px] font-medium text-muted-foreground">
                {slot.label}
              </span>
              {(isBuy || isSell) && (
                <span
                  className={cn(
                    'inline-flex items-center gap-0.5 font-mono text-[10.5px] font-bold',
                    isBuy
                      ? 'text-emerald-600 dark:text-emerald-400'
                      : 'text-rose-600 dark:text-rose-400',
                  )}
                >
                  {isBuy ? (
                    <TrendingUp className="h-3 w-3" />
                  ) : (
                    <TrendingDown className="h-3 w-3" />
                  )}
                  {sig.direction}
                </span>
              )}
            </div>

            {/* Baris 3: Ringkasan Overview */}
            <p
              className="truncate border-t border-border/50 pt-1.5 text-[11px] text-muted-foreground"
              title={overview.note}
            >
              {overview.note}
            </p>
          </button>
        )
      })}
    </div>
  )
}
