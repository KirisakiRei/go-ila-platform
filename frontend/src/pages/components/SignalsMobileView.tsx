import { useState, useMemo } from 'react'
import {
  ArrowLeft,
  BarChart3,
  ChevronRight,
  Clock,
  Layers,
  Radio,
  RotateCw,
  TrendingDown,
  TrendingUp,
  Zap,
} from 'lucide-react'
import { Badge, Button, EmptyState, Skeleton, StatusPill } from '@/components/ui'
import { formatDateTime, formatMoney, formatNumber } from '@/lib/format'
import { cn } from '@/lib/utils'
import { SIGNAL_LABEL, SIGNAL_TONE, getIla3Mtf, isFinal } from '@/features/signals/lib/status'
import { CallEntryAlertBanner } from '@/features/signals/components/CallEntryAlertBanner'
import { TimeframeRadarMatrix } from '@/features/signals/components/TimeframeRadarMatrix'
import { SaklarMetodeAkun } from '@/features/signals/components/SaklarMetodeAkun'
import { SaklarTimeframeAkun } from '@/features/signals/components/SaklarTimeframeAkun'
import { TimeframeBadge } from '@/features/signals/components/TimeframeBadge'
import { TradingRadarChart } from '@/features/charts/TradingRadarChart'
import type { Position, Signal } from '@/types/api'

type TabFilter = 'all' | 'ready' | 'monitoring' | 'history'

type SignalsMobileViewProps = {
  signals?: Signal[]
  allPairs: string[]
  activePair: string | null
  onSelectPair: (symbol: string) => void
  onBackToPairs: () => void
  isLoading: boolean
  isFetching: boolean
  refetch: () => Promise<unknown>
  adaIla3: boolean
  activeStrategyFilter: 'ila-2.0' | 'ila-3.0'
  onSelectStrategy: (strat: 'ila-2.0' | 'ila-3.0') => void
  filteredSignals: Signal[]
  filteredPositions: Position[]
  /**
   * Pasangan timeframe yang sedang dipantau. `null` berarti belum diketahui.
   */
  timeframeAktif?: Set<string> | null
}

export function SignalsMobileView({
  signals = [],
  allPairs,
  activePair,
  onSelectPair,
  onBackToPairs,
  isLoading,
  isFetching,
  refetch,
  adaIla3,
  activeStrategyFilter,
  onSelectStrategy,
  filteredSignals,
  filteredPositions,
  timeframeAktif,
}: SignalsMobileViewProps) {
  const [activeTab, setActiveTab] = useState<TabFilter>('all')
  // Chart dilipat secara bawaan agar daftar sinyal tetap jadi fokus utama di
  // layar kecil; trader membukanya saat butuh melihat struktur.
  const [chartOpen, setChartOpen] = useState(false)

  // Base list of signals: if activePair is chosen, filter by activePair; else use all signals matching active watchlist pairs
  const baseSignals = useMemo(() => {
    if (activePair) {
      return filteredSignals
    }
    // Only show signals for pairs in user's active watchlist
    const activePairSet = new Set(allPairs.map((p) => p.toUpperCase()))
    return signals.filter((s) => {
      const matchPair = activePairSet.has(s.symbol.toUpperCase())
      const matchStrategy = s.strategy_version === activeStrategyFilter
      return matchPair && matchStrategy
    })
  }, [activePair, filteredSignals, signals, allPairs, activeStrategyFilter])

  // Filter based on the 4 tabs: Semua, Siap Entry, Pantau HTF, Riwayat
  const tabFilteredSignals = useMemo(() => {
    return baseSignals.filter((s) => {
      const isReady =
        s.status === 'READY_TO_ENTRY' ||
        s.status === 'OPPORTUNITY' ||
        s.status.toUpperCase().includes('READY')
      // Baris radar (status `WAIT_*`) tidak pernah dihitung sebagai riwayat,
      // sekalipun namanya tidak diawali `WAIT_` persis (mis. WAIT_ILA3_LTF_CISD_1).
      // Tanpa pengecualian ini, slot yang masih memantau bisa muncul di tab
      // Riwayat dan terbaca sebagai setup yang sudah selesai.
      const isWaiting = s.signal_id?.startsWith('status-') || s.status.toUpperCase().startsWith('WAIT')
      const isFinalStatus = isFinal(s.status) && !isWaiting

      if (activeTab === 'ready') return isReady && !isFinalStatus
      if (activeTab === 'monitoring') return !isReady && !isFinalStatus
      if (activeTab === 'history') return isFinalStatus
      return true
    })
  }, [baseSignals, activeTab])

  // Tab counts
  const tabCounts = useMemo(() => {
    let readyCount = 0
    let monitoringCount = 0
    let historyCount = 0

    for (const s of baseSignals) {
      const isReady =
        s.status === 'READY_TO_ENTRY' ||
        s.status === 'OPPORTUNITY' ||
        s.status.toUpperCase().includes('READY')
      const isWaiting = s.signal_id?.startsWith('status-') || s.status.toUpperCase().startsWith('WAIT')
      const isFinalStatus = isFinal(s.status) && !isWaiting

      if (isFinalStatus) {
        historyCount++
      } else if (isReady) {
        readyCount++
      } else {
        monitoringCount++
      }
    }

    return {
      all: baseSignals.length,
      ready: readyCount,
      monitoring: monitoringCount,
      history: historyCount,
    }
  }, [baseSignals])

  return (
    <div className="flex flex-col gap-4 lg:hidden">
      {/* 1. Header: Live Radar Status + Refetch Button */}
      <div className="flex items-center justify-between rounded-2xl border border-border bg-card p-3.5 shadow-xs">
        <div className="flex items-center gap-2.5">
          <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-foreground text-background shrink-0">
            <Radio className="h-4.5 w-4.5 animate-pulse" />
          </div>
          <div>
            <div className="flex items-center gap-1.5">
              <h2 className="text-sm font-bold tracking-tight text-foreground">
                {activePair ? `Radar · ${activePair}` : 'Signals & Radar'}
              </h2>
              <span className="flex h-2 w-2 rounded-full bg-emerald-500 ring-2 ring-emerald-500/20" />
            </div>
            <p className="text-micro text-muted-foreground">
              {activePair ? 'Analisis matriks & histori order' : 'Pemantauan multi-tier live'}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-1.5">
          {activePair && (
            <Button
              variant="outline"
              size="sm"
              onClick={onBackToPairs}
              className="h-8 gap-1 px-2.5 text-xs font-medium"
            >
              <ArrowLeft className="h-3.5 w-3.5" />
              <span>Semua</span>
            </Button>
          )}
          <Button
            variant="ghost"
            size="sm"
            onClick={() => void refetch()}
            disabled={isFetching}
            className="h-8 w-8 p-0 text-muted-foreground hover:text-foreground"
            aria-label="Muat Ulang Sinyal"
          >
            <RotateCw className={cn('h-4 w-4', isFetching && 'animate-spin text-foreground')} />
          </Button>
        </div>
      </div>

      {/* 2. Horizontal Chip Scroller: Pair Filter */}
      {allPairs.length > 0 && (
        <div className="flex items-center gap-1.5 overflow-x-auto pb-1 scrollbar-none">
          {/* Chip 'Semua' */}
          <button
            type="button"
            onClick={onBackToPairs}
            className={cn(
              'flex items-center gap-1 rounded-xl px-3 py-1.5 text-xs font-semibold transition-all active:scale-95 shrink-0',
              !activePair
                ? 'bg-foreground text-background shadow-xs'
                : 'border border-border bg-card text-muted-foreground hover:text-foreground',
            )}
          >
            <Layers className="h-3.5 w-3.5" />
            <span>Semua Pair</span>
            <span
              className={cn(
                'ml-0.5 rounded-full px-1.5 py-0.2 text-micro leading-none',
                !activePair
                  ? 'bg-background/20 text-background'
                  : 'bg-secondary text-muted-foreground',
              )}
            >
              {allPairs.length}
            </span>
          </button>

          {/* List Pair Chips */}
          {allPairs.map((pair) => {
            const isPairActive = activePair === pair
            const pairSignalCount = signals.filter(
              (s) => s.symbol.toUpperCase() === pair && !isFinal(s.status),
            ).length

            return (
              <button
                key={pair}
                type="button"
                onClick={() => onSelectPair(pair)}
                className={cn(
                  'flex items-center gap-1 rounded-xl px-3 py-1.5 text-xs font-semibold transition-all active:scale-95 shrink-0',
                  isPairActive
                    ? 'bg-foreground text-background shadow-xs'
                    : 'border border-border bg-card text-muted-foreground hover:text-foreground',
                )}
              >
                {pairSignalCount > 0 && (
                  <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
                )}
                <span>{pair}</span>
                {pairSignalCount > 0 && (
                  <span
                    className={cn(
                      'ml-0.5 rounded-full px-1.5 py-0.2 text-micro leading-none',
                      isPairActive
                        ? 'bg-background/20 text-background'
                        : 'bg-secondary text-muted-foreground',
                    )}
                  >
                    {pairSignalCount}
                  </span>
                )}
              </button>
            )
          })}
        </div>
      )}

      {/* 3. 4-Tab Filter: Semua | Siap Entry | Pantau HTF | Riwayat */}
      <div className="grid grid-cols-4 gap-1 rounded-xl border border-border bg-secondary/50 p-1">
        <button
          type="button"
          onClick={() => setActiveTab('all')}
          className={cn(
            'flex flex-col items-center justify-center rounded-lg py-1.5 text-micro font-semibold transition-all',
            activeTab === 'all'
              ? 'bg-card text-foreground shadow-xs'
              : 'text-muted-foreground hover:text-foreground',
          )}
        >
          <span>Semua</span>
          <span className="text-micro font-normal opacity-70">({tabCounts.all})</span>
        </button>

        <button
          type="button"
          onClick={() => setActiveTab('ready')}
          className={cn(
            'flex flex-col items-center justify-center rounded-lg py-1.5 text-micro font-semibold transition-all',
            activeTab === 'ready'
              ? 'bg-card text-emerald-700 dark:text-emerald-400 shadow-xs'
              : 'text-muted-foreground hover:text-foreground',
          )}
        >
          <span className="inline-flex items-center gap-0.5">
            <Zap className="h-3 w-3" />
            <span>Siap Entry</span>
          </span>
          <span className="text-micro font-normal opacity-70">({tabCounts.ready})</span>
        </button>

        <button
          type="button"
          onClick={() => setActiveTab('monitoring')}
          className={cn(
            'flex flex-col items-center justify-center rounded-lg py-1.5 text-micro font-semibold transition-all',
            activeTab === 'monitoring'
              ? 'bg-card text-foreground shadow-xs'
              : 'text-muted-foreground hover:text-foreground',
          )}
        >
          <span>Pantau HTF</span>
          <span className="text-micro font-normal opacity-70">({tabCounts.monitoring})</span>
        </button>

        <button
          type="button"
          onClick={() => setActiveTab('history')}
          className={cn(
            'flex flex-col items-center justify-center rounded-lg py-1.5 text-micro font-semibold transition-all',
            activeTab === 'history'
              ? 'bg-card text-foreground shadow-xs'
              : 'text-muted-foreground hover:text-foreground',
          )}
        >
          <span>Riwayat</span>
          <span className="text-micro font-normal opacity-70">({tabCounts.history})</span>
        </button>
      </div>

      {/* 4. Strategy Filter (jika user memiliki akses ILA 3.0) */}
      {adaIla3 && (
        <div className="flex items-center justify-between gap-2 rounded-xl border border-border bg-card px-3 py-2 text-xs">
          <span className="text-micro font-semibold text-muted-foreground">Metode:</span>
          <div className="flex items-center gap-1">
            {(['ila-2.0', 'ila-3.0'] as const).map((strat) => (
              <button
                key={strat}
                type="button"
                onClick={() => onSelectStrategy(strat)}
                className={cn(
                  'rounded-lg px-2.5 py-1 text-micro leading-none font-semibold transition-all',
                  activeStrategyFilter === strat
                    ? 'bg-foreground text-background shadow-xs'
                    : 'bg-secondary text-muted-foreground hover:text-foreground',
                )}
              >
                {strat === 'ila-2.0' ? 'ILA 2.0' : 'ILA 3.0'}
              </button>
            ))}
          </div>
        </div>
      )}

      {/* 5. Call Entry Alert Banner (Pinned jika ada sinyal siap eksekusi) */}
      <CallEntryAlertBanner signals={baseSignals} />

      {/* 5b. Chart setup ringkas untuk pair yang sedang dipilih */}
      {activePair && (
        <div className="overflow-hidden rounded-2xl border border-border bg-card shadow-xs">
          <button
            type="button"
            onClick={() => setChartOpen((open) => !open)}
            aria-expanded={chartOpen}
            className="flex w-full items-center justify-between gap-2 px-3.5 py-3 text-left"
          >
            <span className="flex items-center gap-2">
              <span className="flex h-8 w-8 items-center justify-center rounded-xl bg-secondary">
                <BarChart3 className="h-4 w-4 text-foreground" />
              </span>
              <span>
                <span className="block text-xs font-bold text-foreground">Chart Setup</span>
                <span className="block text-micro text-muted-foreground">
                  Lilin, sweep, FVG, dan level order {activePair}
                </span>
              </span>
            </span>
            <ChevronRight
              className={cn(
                'h-4 w-4 text-muted-foreground transition-transform',
                chartOpen && 'rotate-90',
              )}
            />
          </button>

          {chartOpen && (
            <div className="border-t border-border p-3">
              <TradingRadarChart
                symbol={activePair}
                signals={filteredSignals}
                strategy={activeStrategyFilter}
                enabled={chartOpen}
              />
            </div>
          )}
        </div>
      )}

      {/* 6. List Sinyal (Card "PRIORITAS ILA" dkk) */}
      {isLoading ? (
        <div className="space-y-3">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-44 w-full rounded-2xl" />
          ))}
        </div>
      ) : tabFilteredSignals.length === 0 ? (
        <EmptyState
          title={
            activeTab === 'ready'
              ? 'Tidak ada sinyal siap entry'
              : activeTab === 'monitoring'
                ? 'Tidak ada setup yang dipantau'
                : 'Belum ada sinyal terdeteksi'
          }
          description={
            allPairs.length === 0
              ? 'Watchlist pair Anda masih kosong. Tambahkan instrumen di menu Watchlist agar sinyal otomatis terpantau.'
              : 'Radar sedang memindai struktur pasar. Sinyal valid akan segera muncul otomatis di sini.'
          }
        />
      ) : (
        <div className="space-y-3">
          {tabFilteredSignals.map((signal) => {
            const isBuy = signal.direction === 'BUY'
            const isSell = signal.direction === 'SELL'
            const isReady =
              signal.status === 'READY_TO_ENTRY' || signal.status === 'OPPORTUNITY'
            const decimals = signal.symbol.includes('XAU') ? 2 : 4

            return (
              <div
                key={signal.signal_id}
                className="flex flex-col gap-3 rounded-2xl border border-border bg-card p-4 shadow-xs transition-colors hover:border-zinc-400"
              >
                {/* Header Kartu: PRIORITAS ILA + Simbol + Timeframe | Status + Arah */}
                <div className="flex items-center justify-between gap-2 border-b border-border/70 pb-3">
                  <div className="flex items-center gap-2">
                    {/* Badge Prioritas atau Pair Avatar */}
                    <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-foreground text-background font-bold text-xs shrink-0">
                      {signal.symbol.slice(0, 3)}
                    </div>

                    <div>
                      <div className="flex items-center gap-1.5">
                        <span className="font-bold text-sm tracking-tight text-foreground">
                          {signal.symbol}
                        </span>
                        <TimeframeBadge
                          htf={signal.htf}
                          mtf={signal.strategy_version === 'ila-3.0' ? getIla3Mtf(signal.htf, signal.ltf) : undefined}
                          ltf={signal.ltf}
                        />
                      </div>
                      <div className="flex items-center gap-1.5 text-micro text-muted-foreground">
                        <Badge variant="outline" className="text-micro px-1.5 py-0">
                          {signal.strategy_version}
                        </Badge>
                        {isReady && (
                          <span className="inline-flex items-center gap-0.5 text-micro font-bold text-emerald-700 dark:text-emerald-400 uppercase tracking-wide">
                            <Zap className="h-3 w-3" />
                            Prioritas
                          </span>
                        )}
                      </div>
                    </div>
                  </div>

                  {/* Kanan: Direction + Status Badge */}
                  <div className="flex flex-col items-end gap-1">
                    <div
                      className={cn(
                        'inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 font-bold text-xs',
                        isBuy
                          ? 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-400'
                          : isSell
                            ? 'bg-rose-500/10 text-rose-700 dark:text-rose-400'
                            : 'bg-secondary text-muted-foreground',
                      )}
                    >
                      {isBuy ? (
                        <TrendingUp className="h-3.5 w-3.5" />
                      ) : isSell ? (
                        <TrendingDown className="h-3.5 w-3.5" />
                      ) : null}
                      <span>{signal.direction}</span>
                    </div>

                    <StatusPill tone={SIGNAL_TONE[signal.status]} className="text-micro">
                      {SIGNAL_LABEL[signal.status]}
                    </StatusPill>
                  </div>
                </div>

                {/* 4-Box Grid Metric: ENTRY | STOP LOSS | TAKE PROFIT | R:R */}
                <div className="grid grid-cols-4 gap-1.5 rounded-xl border border-border/80 bg-secondary/40 p-2">
                  <div className="flex flex-col">
                    <span className="text-micro font-bold uppercase tracking-wider text-muted-foreground">
                      Entry
                    </span>
                    <span className="font-mono text-xs font-bold tabular-nums text-foreground">
                      {signal.entry_price > 0
                        ? formatNumber(signal.entry_price, decimals)
                        : '—'}
                    </span>
                  </div>

                  <div className="flex flex-col">
                    <span className="text-micro font-bold uppercase tracking-wider text-muted-foreground">
                      Stop Loss
                    </span>
                    <span className="font-mono text-xs font-semibold tabular-nums text-rose-600 dark:text-rose-400">
                      {signal.stop_loss > 0 ? formatNumber(signal.stop_loss, decimals) : '—'}
                    </span>
                  </div>

                  <div className="flex flex-col">
                    <span className="text-micro font-bold uppercase tracking-wider text-muted-foreground">
                      Take Profit
                    </span>
                    <span className="font-mono text-xs font-semibold tabular-nums text-emerald-600 dark:text-emerald-400">
                      {signal.take_profit > 0
                        ? formatNumber(signal.take_profit, decimals)
                        : '—'}
                    </span>
                  </div>

                  <div className="flex flex-col text-right">
                    <span className="text-micro font-bold uppercase tracking-wider text-muted-foreground">
                      R:R
                    </span>
                    <span className="font-mono text-xs font-bold tabular-nums text-foreground">
                      {signal.risk_reward > 0
                        ? `1:${formatNumber(signal.risk_reward, 1)}`
                        : '—'}
                    </span>
                  </div>
                </div>

                {/* Summary / Confluence setup */}
                {signal.summary && (
                  <p className="text-caption leading-relaxed text-muted-foreground line-clamp-2">
                    {signal.summary}
                  </p>
                )}

                {/* Footer Kartu: Waktu Terbit + Tombol Aksi Detail */}
                <div className="flex items-center justify-between pt-1 text-micro text-muted-foreground">
                  <div className="flex items-center gap-1.5">
                    <Clock className="h-3 w-3" />
                    <span>{formatDateTime(signal.published_at)}</span>
                    {signal.delayed && (
                      <span className="rounded bg-amber-500/10 px-1.5 py-0.2 text-micro font-bold text-amber-700 dark:text-amber-400">
                        Delayed
                      </span>
                    )}
                  </div>

                  {!activePair && (
                    <button
                      type="button"
                      onClick={() => onSelectPair(signal.symbol)}
                      className="inline-flex items-center gap-1 font-semibold text-foreground hover:underline"
                    >
                      <span>Matriks & Order</span>
                      <ChevronRight className="h-3.5 w-3.5" />
                    </button>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* 7. Matriks & Konfigurasi saat Pair Terpilih di Mobile */}
      {activePair && (
        <div className="mt-2 space-y-4 rounded-2xl border border-border bg-card p-4 shadow-xs">
          <div className="flex items-center justify-between border-b border-border pb-3">
            <div>
              <h3 className="text-sm font-bold tracking-tight text-foreground">
                Radar Timeframe ({activePair})
              </h3>
              <p className="text-micro text-muted-foreground">
                Pemantauan konfluensi multi-tier HTF / LTF
              </p>
            </div>
            <Badge variant="outline" className="text-micro">
              {filteredPositions.length} Eksekusi
            </Badge>
          </div>

          <TimeframeRadarMatrix
            signals={filteredSignals}
            strategy={activeStrategyFilter}
            symbol={activePair}
            timeframeAktif={timeframeAktif}
          />

          {/* Riwayat Order Posisi Mobile */}
          <div className="space-y-2.5 pt-2">
            <h4 className="text-xs font-bold uppercase tracking-wider text-muted-foreground">
              Eksekusi Order Terakhir
            </h4>
            {filteredPositions.length === 0 ? (
              <p className="text-xs text-muted-foreground py-2">
                Belum ada eksekusi order pada pair {activePair}.
              </p>
            ) : (
              <div className="space-y-2">
                {filteredPositions.slice(0, 5).map((pos) => (
                  <div
                    key={pos.id}
                    className="flex items-center justify-between rounded-xl border border-border/80 bg-secondary/30 p-2.5 text-xs"
                  >
                    <div>
                      <div className="flex items-center gap-1.5">
                        <span className="font-semibold text-foreground">
                          #{pos.ticket_number || pos.id.slice(0, 8)}
                        </span>
                        <span
                          className={cn(
                            'font-bold text-micro',
                            pos.direction === 'BUY' ? 'text-emerald-600' : 'text-rose-600',
                          )}
                        >
                          {pos.direction}
                        </span>
                        <Badge variant="outline" className="text-micro px-1">
                          {pos.lot_size} Lot
                        </Badge>
                      </div>
                      <p className="text-micro text-muted-foreground">
                        {formatDateTime(pos.opened_at)}
                      </p>
                    </div>

                    <div className="text-right">
                      <Badge
                        variant={pos.status.startsWith('CLOSED_TP') ? 'solid' : 'outline'}
                        className="text-micro"
                      >
                        {pos.status}
                      </Badge>
                      {pos.pnl_known && (
                        <p
                          className={cn(
                            'font-mono text-micro font-bold tabular-nums',
                            pos.profit_loss >= 0 ? 'text-emerald-600' : 'text-rose-600',
                          )}
                        >
                          {formatMoney(pos.profit_loss)}
                        </p>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Pengaturan Akun Pair Mobile */}
          <div id="pengaturan-timeframe" className="scroll-mt-20 space-y-3 border-t border-border pt-4">
            <div>
              <h4 className="text-xs font-bold text-foreground">
                Pengaturan Akun ({activePair})
              </h4>
              <p className="text-micro text-muted-foreground">
                Aktifkan metode dan timeframe untuk eksekusi otomatis.
              </p>
            </div>
            <SaklarMetodeAkun symbol={activePair} />
            <SaklarTimeframeAkun symbol={activePair} />
          </div>
        </div>
      )}
    </div>
  )
}
