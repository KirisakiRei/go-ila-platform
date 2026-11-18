import { useEffect, useState, useMemo } from 'react'
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
import {
  Badge,
  Button,
  EmptyState,
  MobileSegmentedTabs,
  Pagination,
  Skeleton,
  usePaginasiLokal,
} from '@/components/ui'
import { formatDateTime, formatMoney, formatNumber } from '@/lib/format'
import { priceDigits } from '@/lib/pricePrecision'
import { cn } from '@/lib/utils'
import { getIla3Mtf, isFinal } from '@/features/signals/lib/status'
import {
  OUTCOME_BADGE_CLASS,
  dedupeSetupSignals,
  describeSignalOutcome,
  isRadarProgressSignal,
} from '@/features/signals/lib/outcome'
import { CallEntryAlertBanner } from '@/features/signals/components/CallEntryAlertBanner'
import { TimeframeRadarMatrix } from '@/features/signals/components/TimeframeRadarMatrix'
import { SaklarMetodeAkun } from '@/features/signals/components/SaklarMetodeAkun'
import { SaklarTimeframeAkun } from '@/features/signals/components/SaklarTimeframeAkun'
import { TimeframeBadge } from '@/features/signals/components/TimeframeBadge'
import { TradingRadarChart } from '@/features/charts/TradingRadarChart'
import type { Position, Signal } from '@/types/api'

type TabFilter = 'all' | 'ready' | 'monitoring' | 'history'
type PairSection = 'radar' | 'sinyal' | 'setelan'

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
  selectedSlotKey?: string | null
  onSelectSlot?: (key: string) => void
  snapshotSignal?: Signal | null
  onSelectSnapshot?: (signal: Signal | null) => void
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
  selectedSlotKey = null,
  onSelectSlot,
  snapshotSignal = null,
  onSelectSnapshot,
}: SignalsMobileViewProps) {
  const [activeTab, setActiveTab] = useState<TabFilter>('all')
  const [pairSection, setPairSection] = useState<PairSection>('radar')
  // Chart dilipat secara bawaan agar daftar sinyal tetap jadi fokus utama di
  // layar kecil; trader membukanya saat butuh melihat struktur.
  const [chartOpen, setChartOpen] = useState(false)
  const [posisiHalaman, setPosisiHalaman] = useState(1)
  const POSISI_PER_HALAMAN = 4
  const totalHalamanPosisi = Math.ceil(filteredPositions.length / POSISI_PER_HALAMAN) || 1
  const posisiTampil = filteredPositions.slice(
    (posisiHalaman - 1) * POSISI_PER_HALAMAN,
    posisiHalaman * POSISI_PER_HALAMAN,
  )

  // Saat tombol "Aktifkan pemantauan" di TimeframeRadarMatrix ditekan,
  // pindah otomatis ke segmen Setelan agar elemen #pengaturan-timeframe tampil.
  useEffect(() => {
    const onOpenSettings = () => setPairSection('setelan')
    window.addEventListener('ila:open-timeframe-settings', onOpenSettings)
    return () => window.removeEventListener('ila:open-timeframe-settings', onOpenSettings)
  }, [])

  const baseSignals = useMemo(() => {
    const source = activePair
      ? filteredSignals
      : signals.filter((s) => {
          const activePairSet = new Set(allPairs.map((p) => p.toUpperCase()))
          const matchPair = activePairSet.has(s.symbol.toUpperCase())
          const matchStrategy = s.strategy_version === activeStrategyFilter
          return matchPair && matchStrategy
        })
    return dedupeSetupSignals(source)
  }, [activePair, filteredSignals, signals, allPairs, activeStrategyFilter])

  const tabFilteredSignals = useMemo(() => {
    const filtered = baseSignals.filter((s) => {
      const isRadarRow = isRadarProgressSignal(s)
      const isReady =
        !isRadarRow &&
        (s.status === 'READY_TO_ENTRY' ||
          s.status === 'OPPORTUNITY' ||
          s.status.toUpperCase().includes('READY')) &&
        !isFinal(s.status)

      if (activeTab === 'ready') return isReady
      if (activeTab === 'monitoring') return isRadarRow || s.eval_status === 'RUNNING'
      if (activeTab === 'history') return !isRadarRow && !isReady
      // Tab "Semua" fokus pada setup nyata (memiliki harga Entry/SL/TP) agar
      // tidak tercampur baris radar mentah berharga 0.
      return !isRadarRow
    })

    return [...filtered].sort(
      (a, b) => new Date(b.published_at).getTime() - new Date(a.published_at).getTime(),
    )
  }, [baseSignals, activeTab])

  const signalPaginasi = usePaginasiLokal(tabFilteredSignals, 8)

  const tabCounts = useMemo(() => {
    let allSetups = 0
    let readyCount = 0
    let monitoringCount = 0
    let historyCount = 0

    for (const s of baseSignals) {
      const isRadarRow = isRadarProgressSignal(s)
      const isReady =
        !isRadarRow &&
        (s.status === 'READY_TO_ENTRY' ||
          s.status === 'OPPORTUNITY' ||
          s.status.toUpperCase().includes('READY')) &&
        !isFinal(s.status)

      if (!isRadarRow) {
        allSetups++
      }
      if (isReady) {
        readyCount++
      }
      if (isRadarRow || s.eval_status === 'RUNNING') {
        monitoringCount++
      }
      if (!isRadarRow && !isReady) {
        historyCount++
      }
    }

    return {
      all: allSetups,
      ready: readyCount,
      monitoring: monitoringCount,
      history: historyCount,
    }
  }, [baseSignals])

  const signalListBlock = (
    <>
      {/* 4-Tab Filter: Semua | Siap Entry | Pantau HTF | Riwayat */}
      <div className="grid grid-cols-4 gap-1 rounded-xl border border-border bg-secondary/50 p-1">
        <button
          type="button"
          onClick={() => {
            setActiveTab('all')
            signalPaginasi.setPage(1)
          }}
          className={cn(
            'flex min-w-0 flex-col items-center justify-center rounded-lg px-1 py-1.5 text-[10.5px] font-semibold transition-all',
            activeTab === 'all'
              ? 'bg-card text-foreground shadow-xs'
              : 'text-muted-foreground hover:text-foreground',
          )}
        >
          <span className="truncate">Semua</span>
          <span className="text-[9.5px] font-normal opacity-70">({tabCounts.all})</span>
        </button>

        <button
          type="button"
          onClick={() => {
            setActiveTab('ready')
            signalPaginasi.setPage(1)
          }}
          className={cn(
            'flex min-w-0 flex-col items-center justify-center rounded-lg px-1 py-1.5 text-[10.5px] font-semibold transition-all',
            activeTab === 'ready'
              ? 'bg-card text-emerald-700 shadow-xs dark:text-emerald-400'
              : 'text-muted-foreground hover:text-foreground',
          )}
        >
          <span className="inline-flex max-w-full items-center gap-0.5">
            <Zap className="h-3 w-3 shrink-0" />
            <span className="truncate">Siap</span>
          </span>
          <span className="text-[9.5px] font-normal opacity-70">({tabCounts.ready})</span>
        </button>

        <button
          type="button"
          onClick={() => {
            setActiveTab('monitoring')
            signalPaginasi.setPage(1)
          }}
          className={cn(
            'flex min-w-0 flex-col items-center justify-center rounded-lg px-1 py-1.5 text-[10.5px] font-semibold transition-all',
            activeTab === 'monitoring'
              ? 'bg-card text-foreground shadow-xs'
              : 'text-muted-foreground hover:text-foreground',
          )}
        >
          <span className="truncate">Pantau</span>
          <span className="text-[9.5px] font-normal opacity-70">({tabCounts.monitoring})</span>
        </button>

        <button
          type="button"
          onClick={() => {
            setActiveTab('history')
            signalPaginasi.setPage(1)
          }}
          className={cn(
            'flex min-w-0 flex-col items-center justify-center rounded-lg px-1 py-1.5 text-[10.5px] font-semibold transition-all',
            activeTab === 'history'
              ? 'bg-card text-foreground shadow-xs'
              : 'text-muted-foreground hover:text-foreground',
          )}
        >
          <span className="truncate">Riwayat</span>
          <span className="text-[9.5px] font-normal opacity-70">({tabCounts.history})</span>
        </button>
      </div>

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
          {signalPaginasi.baris.map((signal) => {
            const isBuy = signal.direction === 'BUY'
            const isSell = signal.direction === 'SELL'
            const isReady =
              !isFinal(signal.status) &&
              (signal.status === 'READY_TO_ENTRY' || signal.status === 'OPPORTUNITY')
            const decimals = priceDigits(signal.symbol, signal.entry_price)
            const outcome = describeSignalOutcome(signal)
            const isRadarRow = isRadarProgressSignal(signal)

            return (
              <div
                key={signal.signal_id}
                className="flex flex-col gap-3 rounded-2xl border border-border bg-card p-3.5 shadow-xs transition-colors hover:border-zinc-400"
              >
                {/* Header Kartu */}
                <div className="flex items-start justify-between gap-2 border-b border-border/70 pb-3">
                  <div className="flex min-w-0 flex-1 items-center gap-2">
                    <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-foreground text-xs font-bold text-background">
                      {signal.symbol.slice(0, 3)}
                    </div>

                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-1.5">
                        <span className="text-sm font-bold tracking-tight text-foreground">
                          {signal.symbol}
                        </span>
                        <TimeframeBadge
                          htf={signal.htf}
                          mtf={
                            signal.strategy_version === 'ila-3.0'
                              ? getIla3Mtf(signal.htf, signal.ltf)
                              : undefined
                          }
                          ltf={signal.ltf}
                        />
                      </div>
                      <div className="mt-0.5 flex flex-wrap items-center gap-1.5 text-[11px] text-muted-foreground">
                        <Badge variant="outline" className="px-1.5 py-0 text-[9.5px]">
                          {signal.strategy_version}
                        </Badge>
                        {outcome.executedBadge && (
                          <Badge variant="outline" className="px-1.5 py-0 text-[9.5px]">
                            Dieksekusi
                          </Badge>
                        )}
                        {isReady && (
                          <span className="inline-flex items-center gap-0.5 text-[10px] font-bold uppercase tracking-wide text-emerald-700 dark:text-emerald-400">
                            <Zap className="h-3 w-3" />
                            Prioritas
                          </span>
                        )}
                      </div>
                    </div>
                  </div>

                  <div className="flex shrink-0 flex-col items-end gap-1">
                    <div
                      className={cn(
                        'inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-bold',
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

                    <span
                      className={cn(
                        'inline-flex items-center rounded-md border px-2 py-0.5 text-[10px] font-semibold',
                        OUTCOME_BADGE_CLASS[outcome.tone],
                      )}
                    >
                      {outcome.label}
                    </span>
                  </div>
                </div>

                {/* 2x2 Grid Metric di layar sempit (4 kolom di >= sm) bebas tabrakan */}
                {!isRadarRow && (
                  <div className="grid grid-cols-2 gap-2 rounded-xl border border-border/80 bg-secondary/40 p-2.5 sm:grid-cols-4">
                    <div className="min-w-0">
                      <span className="block text-[9.5px] font-bold uppercase tracking-wider text-muted-foreground">
                        Entry
                      </span>
                      <span className="block truncate font-mono text-xs font-bold tabular-nums text-foreground">
                        {signal.entry_price > 0
                          ? formatNumber(signal.entry_price, decimals)
                          : '—'}
                      </span>
                    </div>

                    <div className="min-w-0 text-right sm:text-left">
                      <span className="block text-[9.5px] font-bold uppercase tracking-wider text-muted-foreground">
                        R:R
                      </span>
                      <span className="block truncate font-mono text-xs font-bold tabular-nums text-foreground">
                        {signal.risk_reward > 0
                          ? `1:${formatNumber(signal.risk_reward, 2)}`
                          : '—'}
                      </span>
                    </div>

                    <div className="min-w-0">
                      <span className="block text-[9.5px] font-bold uppercase tracking-wider text-muted-foreground">
                        Stop Loss
                      </span>
                      <span className="block truncate font-mono text-xs font-semibold tabular-nums text-rose-600 dark:text-rose-400">
                        {signal.stop_loss > 0 ? formatNumber(signal.stop_loss, decimals) : '—'}
                      </span>
                    </div>

                    <div className="min-w-0 text-right">
                      <span className="block text-[9.5px] font-bold uppercase tracking-wider text-muted-foreground">
                        Take Profit
                      </span>
                      <span className="block truncate font-mono text-xs font-semibold tabular-nums text-emerald-600 dark:text-emerald-400">
                        {signal.take_profit > 0
                          ? formatNumber(signal.take_profit, decimals)
                          : '—'}
                      </span>
                    </div>
                  </div>
                )}

                {outcome.executedBadge && outcome.subtext && (
                  <div className="flex items-center justify-between rounded-xl bg-secondary/50 px-3 py-1.5 text-xs">
                    <span className="text-[11px] text-muted-foreground">PnL Eksekusi</span>
                    <span
                      className={cn(
                        'font-mono text-xs font-bold tabular-nums',
                        outcome.tone === 'tp'
                          ? 'text-emerald-600 dark:text-emerald-400'
                          : outcome.tone === 'sl'
                            ? 'text-rose-600 dark:text-rose-400'
                            : 'text-foreground',
                      )}
                    >
                      {outcome.subtext}
                    </span>
                  </div>
                )}

                {signal.summary && (
                  <p className="line-clamp-2 text-[11.5px] leading-relaxed text-muted-foreground">
                    {signal.summary}
                  </p>
                )}

                <div className="flex flex-wrap items-center justify-between gap-2 pt-0.5 text-[11px] text-muted-foreground">
                  <div className="flex min-w-0 flex-wrap items-center gap-1.5">
                    <Clock className="h-3 w-3 shrink-0" />
                    <span>{formatDateTime(signal.published_at)}</span>
                    {signal.delayed && (
                      <span className="rounded bg-amber-500/10 px-1.5 py-[1px] text-[9.5px] font-bold text-amber-700 dark:text-amber-400">
                        Delayed
                      </span>
                    )}
                  </div>

                  {!activePair ? (
                    <button
                      type="button"
                      onClick={() => {
                        setPairSection('radar')
                        onSelectPair(signal.symbol)
                      }}
                      className="inline-flex items-center gap-1 font-semibold text-foreground hover:underline"
                    >
                      <span>Buka Pair</span>
                      <ChevronRight className="h-3.5 w-3.5" />
                    </button>
                  ) : !isRadarRow && onSelectSnapshot ? (
                    <button
                      type="button"
                      onClick={() => {
                        onSelectSnapshot(signal)
                        setPairSection('radar')
                        setChartOpen(true)
                      }}
                      className="inline-flex items-center gap-1 font-semibold text-foreground hover:underline"
                    >
                      <span>Lihat Chart</span>
                      <ChevronRight className="h-3.5 w-3.5" />
                    </button>
                  ) : null}
                </div>
              </div>
            )
          })}

          {signalPaginasi.totalPages > 1 && (
            <Pagination
              page={signalPaginasi.page}
              totalPages={signalPaginasi.totalPages}
              shown={signalPaginasi.baris.length}
              total={signalPaginasi.total}
              unit="sinyal"
              onPageChange={signalPaginasi.setPage}
            />
          )}
        </div>
      )}
    </>
  )

  return (
    <div className="flex flex-col gap-3.5 lg:hidden">
      {/* 1. Header: Live Radar Status + Refetch Button */}
      <div className="flex items-center justify-between gap-2 rounded-2xl border border-border bg-card p-3.5 shadow-xs">
        <div className="flex min-w-0 flex-1 items-center gap-2.5">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-foreground text-background">
            <Radio className="h-[1.125rem] w-[1.125rem] animate-pulse" />
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-1.5">
              <h2 className="truncate text-sm font-bold tracking-tight text-foreground">
                {activePair ? `Radar · ${activePair}` : 'Signals & Radar'}
              </h2>
              <span className="flex h-2 w-2 shrink-0 rounded-full bg-emerald-500 ring-2 ring-emerald-500/20" />
            </div>
            <p className="truncate text-[11px] text-muted-foreground">
              {activePair ? 'Radar timeframe, sinyal, & setelan akun' : 'Pemantauan multi-tier live'}
            </p>
          </div>
        </div>

        <div className="flex shrink-0 items-center gap-1.5">
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

      {/* 2. Chip Filter Pair (Wrap Rapi — Bebas Horizontal Scroll) */}
      {allPairs.length > 0 && (
        <div className="flex flex-wrap items-center gap-1.5">
          <button
            type="button"
            onClick={onBackToPairs}
            className={cn(
              'flex items-center gap-1 rounded-xl px-2.5 py-1.5 text-xs font-semibold transition-all active:scale-95',
              !activePair
                ? 'bg-foreground text-background shadow-xs'
                : 'border border-border bg-card text-muted-foreground hover:text-foreground',
            )}
          >
            <Layers className="h-3.5 w-3.5" />
            <span>Semua</span>
            <span
              className={cn(
                'ml-0.5 rounded-full px-1.5 py-[1px] text-[9px]',
                !activePair
                  ? 'bg-background/20 text-background'
                  : 'bg-secondary text-muted-foreground',
              )}
            >
              {allPairs.length}
            </span>
          </button>

          {allPairs.map((pair) => {
            const isPairActive = activePair === pair
            const pairSignalCount = signals.filter(
              (s) => s.symbol.toUpperCase() === pair && !isFinal(s.status),
            ).length

            return (
              <button
                key={pair}
                type="button"
                onClick={() => {
                  onSelectPair(pair)
                }}
                className={cn(
                  'flex items-center gap-1 rounded-xl px-2.5 py-1.5 text-xs font-semibold transition-all active:scale-95',
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
                      'ml-0.5 rounded-full px-1.5 py-[1px] text-[9px]',
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

      {/* 3. Strategy Filter (jika user memiliki akses ILA 3.0) */}
      {adaIla3 && (
        <div className="flex items-center justify-between gap-2 rounded-xl border border-border bg-card px-3 py-2 text-xs">
          <span className="text-[11px] font-semibold text-muted-foreground">Metode Analisis:</span>
          <div className="flex items-center gap-1">
            {(['ila-2.0', 'ila-3.0'] as const).map((strat) => (
              <button
                key={strat}
                type="button"
                onClick={() => {
                  onSelectStrategy(strat)
                  signalPaginasi.setPage(1)
                }}
                className={cn(
                  'rounded-lg px-2.5 py-1 text-[11px] font-semibold transition-all',
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

      {/* 4. Call Entry Alert Banner (Pinned jika ada sinyal siap eksekusi) */}
      <CallEntryAlertBanner signals={baseSignals} />

      {/* 5. Jika Pair Terpilih: Segmented Control 3 Bagian (Radar | Sinyal & Order | Setelan) */}
      {activePair ? (
        <div className="space-y-3.5">
          <MobileSegmentedTabs<PairSection>
            ariaLabel={`Bagian detail pair ${activePair}`}
            value={pairSection}
            onChange={setPairSection}
            options={[
              { id: 'radar', label: 'Radar' },
              { id: 'sinyal', label: 'Sinyal & Order', count: tabCounts.all },
              { id: 'setelan', label: 'Setelan Pair' },
            ]}
          />

          {pairSection === 'radar' && (
            <div className="space-y-3.5">
              {/* Collapsible Chart Setup */}
              <div className="overflow-hidden rounded-2xl border border-border bg-card shadow-xs">
                <button
                  type="button"
                  onClick={() => setChartOpen((open) => !open)}
                  aria-expanded={chartOpen}
                  className="flex w-full items-center justify-between gap-2 px-3.5 py-3 text-left"
                >
                  <span className="flex min-w-0 items-center gap-2.5">
                    <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-xl bg-secondary">
                      <BarChart3 className="h-4 w-4 text-foreground" />
                    </span>
                    <span className="min-w-0">
                      <span className="block truncate text-xs font-bold text-foreground">
                        Chart Setup ({activePair})
                      </span>
                      <span className="block truncate text-[10.5px] text-muted-foreground">
                        Lilin, struktur, dan level order live
                      </span>
                    </span>
                  </span>
                  <ChevronRight
                    className={cn(
                      'h-4 w-4 shrink-0 text-muted-foreground transition-transform',
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
                      selectedSlotKey={selectedSlotKey}
                      snapshotSignal={snapshotSignal}
                      onClearSnapshot={onSelectSnapshot ? () => onSelectSnapshot(null) : undefined}
                      enabled={chartOpen}
                    />
                  </div>
                )}
              </div>

              <div className="rounded-2xl border border-border bg-card p-3.5 shadow-xs">
                <TimeframeRadarMatrix
                  signals={filteredSignals}
                  strategy={activeStrategyFilter}
                  symbol={activePair}
                  timeframeAktif={timeframeAktif}
                  activeSlotKey={selectedSlotKey}
                  onSelectSlot={(key) => {
                    onSelectSlot?.(key)
                    onSelectSnapshot?.(null)
                  }}
                />
              </div>
            </div>
          )}

          {pairSection === 'sinyal' && (
            <div className="space-y-4">
              {signalListBlock}

              <div className="space-y-2.5 rounded-2xl border border-border bg-card p-3.5 shadow-xs">
                <div className="flex items-center justify-between gap-2">
                  <h4 className="text-xs font-bold uppercase tracking-wider text-muted-foreground">
                    Eksekusi Order Terakhir
                  </h4>
                  <Badge variant="outline" className="text-[10px]">
                    {filteredPositions.length} Order
                  </Badge>
                </div>

                {filteredPositions.length === 0 ? (
                  <p className="py-2 text-xs text-muted-foreground">
                    Belum ada eksekusi order pada pair {activePair}.
                  </p>
                ) : (
                  <div className="space-y-2">
                    {posisiTampil.map((pos) => (
                      <div
                        key={pos.id}
                        className="flex min-w-0 items-center justify-between gap-2 rounded-xl border border-border/80 bg-secondary/30 p-2.5 text-xs"
                      >
                        <div className="min-w-0 flex-1">
                          <div className="flex flex-wrap items-center gap-1.5">
                            <span className="truncate font-semibold text-foreground">
                              #{pos.ticket_number || pos.id.slice(0, 8)}
                            </span>
                            <span
                              className={cn(
                                'text-[10px] font-bold',
                                pos.direction === 'BUY' ? 'text-emerald-600' : 'text-rose-600',
                              )}
                            >
                              {pos.direction}
                            </span>
                            <Badge variant="outline" className="px-1 text-[9px]">
                              {pos.lot_size} Lot
                            </Badge>
                          </div>
                          <p className="text-[10.5px] text-muted-foreground">
                            {formatDateTime(pos.opened_at)}
                          </p>
                        </div>

                        <div className="shrink-0 text-right">
                          <Badge
                            variant={pos.status.startsWith('CLOSED_TP') ? 'solid' : 'outline'}
                            className="text-[9.5px]"
                          >
                            {pos.status}
                          </Badge>
                          {pos.pnl_known && (
                            <p
                              className={cn(
                                'font-mono text-[11px] font-bold tabular-nums',
                                pos.profit_loss >= 0 ? 'text-emerald-600' : 'text-rose-600',
                              )}
                            >
                              {formatMoney(pos.profit_loss)}
                            </p>
                          )}
                        </div>
                      </div>
                    ))}
                    {totalHalamanPosisi > 1 && (
                      <div className="flex items-center justify-between gap-2 pt-1">
                        <button
                          type="button"
                          onClick={() => setPosisiHalaman((p) => Math.max(1, p - 1))}
                          disabled={posisiHalaman <= 1}
                          className="rounded-lg border border-border px-2.5 py-1 text-[11px] font-semibold text-foreground transition-colors duration-150 disabled:opacity-40"
                        >
                          Sebelumnya
                        </button>
                        <span className="text-[11px] text-muted-foreground">
                          {posisiHalaman} / {totalHalamanPosisi}
                        </span>
                        <button
                          type="button"
                          onClick={() =>
                            setPosisiHalaman((p) => Math.min(totalHalamanPosisi, p + 1))
                          }
                          disabled={posisiHalaman >= totalHalamanPosisi}
                          className="rounded-lg border border-border px-2.5 py-1 text-[11px] font-semibold text-foreground transition-colors duration-150 disabled:opacity-40"
                        >
                          Berikutnya
                        </button>
                      </div>
                    )}
                  </div>
                )}
              </div>
            </div>
          )}

          {pairSection === 'setelan' && (
            <div
              id="pengaturan-timeframe"
              className="scroll-mt-20 space-y-3.5 rounded-2xl border border-border bg-card p-3.5 shadow-xs"
            >
              <div>
                <h4 className="text-xs font-bold text-foreground">
                  Pengaturan Akun ({activePair})
                </h4>
                <p className="text-[11px] text-muted-foreground">
                  Aktifkan metode analisis dan pasangan timeframe yang dipantau serta diizinkan eksekusi otomatis.
                </p>
              </div>
              <SaklarMetodeAkun symbol={activePair} />
              <SaklarTimeframeAkun symbol={activePair} />
            </div>
          )}
        </div>
      ) : (
        signalListBlock
      )}
    </div>
  )
}
