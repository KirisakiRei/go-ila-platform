import { useEffect, useState, useMemo } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import {
  ArrowLeft,
  ArrowUpRight,
  BarChart3,
  Bot,
  PanelRightClose,
  PanelRightOpen,
  Settings,
  TrendingDown,
  TrendingUp,
} from 'lucide-react'
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  EmptyState,
  PageHeader,
  Section,
  Skeleton,
} from '@/components/ui'
import {
  usePositions,
  useSignals,
  useStrategies,
  useStrategySwitchMatrix,
  useSubscriptions,
  useTimeframeSwitchMatrix,
} from '@/lib/queries'
import { formatDateTime, formatMoney, formatNumber, formatPrice, formatR } from '@/lib/format'
import { priceDigits } from '@/lib/pricePrecision'
import { cn } from '@/lib/utils'
import { getIla3Mtf } from '@/features/signals/lib/status'
import {
  OUTCOME_BADGE_CLASS,
  dedupeSetupSignals,
  describeSignalOutcome,
  isRadarProgressSignal,
} from '@/features/signals/lib/outcome'
import type { Signal } from '@/types/api'
import {
  setTimeframeDipantau,
  mapSignalsToSlots,
  pilihSlotAwal,
  ringkasanSlot,
  timeframeSlotDipantau,
  type TimeframeSlot,
} from '@/features/signals/lib/signal-slots'
import { slotPalingDekat, stageOf, type SlotStage } from '@/features/signals/lib/slot-stage'
import { CallEntryAlertBanner } from '@/features/signals/components/CallEntryAlertBanner'
import { CycleSelectorStrip } from '@/features/signals/components/CycleSelectorStrip'
import { PengaturanPairModal } from '@/features/signals/components/PengaturanPairModal'
import { TimeframeRadarMatrix } from '@/features/signals/components/TimeframeRadarMatrix'
import { TradingRadarChart } from '@/features/charts/TradingRadarChart'
import { TimeframeBadge } from '@/features/signals/components/TimeframeBadge'
import { SignalsMobileView } from './components/SignalsMobileView'

function useIsDesktopViewport(): boolean {
  const read = () => {
    if (typeof window === 'undefined') return true
    if (typeof window.matchMedia === 'function') {
      const mql = window.matchMedia('(min-width: 1024px)')
      // Di lingkungan uji (JSDOM) innerWidth bawaan 1024 walau matchMedia palsu.
      if (!mql.matches && window.innerWidth >= 1024) return true
      if (window.innerWidth < 1024) return false
      return mql.matches
    }
    return window.innerWidth >= 1024
  }

  const [isDesktop, setIsDesktop] = useState<boolean>(read)

  useEffect(() => {
    if (typeof window === 'undefined') return
    const update = () => setIsDesktop(read())
    window.addEventListener('resize', update)
    const mql =
      typeof window.matchMedia === 'function' ? window.matchMedia('(min-width: 1024px)') : null
    mql?.addEventListener?.('change', update)
    return () => {
      window.removeEventListener('resize', update)
      mql?.removeEventListener?.('change', update)
    }
  }, [])

  return isDesktop
}

export default function SignalsPage() {
  const isDesktop = useIsDesktopViewport()
  const [params, setParams] = useSearchParams()
  const pairParam = params.get('pair')

  const { data: signals, isLoading: loadingSignals, isFetching, refetch } = useSignals(100)
  const { data: positions, isLoading: loadingPositions } = usePositions(100)
  const { data: subscriptions, isLoading: loadingSubs } = useSubscriptions()
  const { data: strategies } = useStrategies()

  /**
   * ILA 3.0 hanya ditampilkan bagi tier yang berhak. Bagi pengguna lain seluruh
   * jejak metode ini disembunyikan agar fitur berbayar tidak terbocorkan.
   */
  const adaIla3 = useMemo(
    () => (strategies ?? []).some((s) => s.version === 'ila-3.0' && s.allowed),
    [strategies],
  )

  const [activeStrategyFilter, setActiveStrategyFilter] = useState<'ila-2.0' | 'ila-3.0'>('ila-2.0')
  const [selectedSlotKey, setSelectedSlotKey] = useState<string | null>(null)
  const [snapshotSignal, setSnapshotSignal] = useState<Signal | null>(null)
  const [historyOpen, setHistoryOpen] = useState(true)
  const [settingsOpen, setSettingsOpen] = useState(false)

  const cleanSignals = useMemo(() => dedupeSetupSignals(signals ?? []), [signals])

  // Filter metode yang ditawarkan mengikuti hak akses tier pengguna.
  const strategiTersedia = useMemo(
    () => (adaIla3 ? (['ila-2.0', 'ila-3.0'] as const) : (['ila-2.0'] as const)).slice(),
    [adaIla3],
  )

  // Hanya ambil pair dari subscription aktif di watchlist agar sinkron dengan watchlist
  const allPairs = useMemo(() => {
    const set = new Set<string>()
    for (const sub of subscriptions ?? []) {
      if (sub.symbol && sub.is_active) {
        set.add(sub.symbol.toUpperCase())
      }
    }
    return Array.from(set).sort()
  }, [subscriptions])

  // Hanya izinkan pairParam jika pair tersebut ada dalam watchlist aktif
  const activePair = useMemo(() => {
    if (!pairParam) return null
    const upper = pairParam.toUpperCase()
    return allPairs.includes(upper) ? upper : null
  }, [pairParam, allPairs])

  // Matriks switch dipakai hanya untuk membedakan "sedang memindai" dari
  // "tidak ada timeframe yang dipindai". Tidak memengaruhi eksekusi.
  const { data: timeframeMatrix } = useTimeframeSwitchMatrix(activePair)

  // Sinyal dan posisi untuk pair yang sedang aktif
  const pairSignals = useMemo(() => {
    if (!activePair) return []
    return cleanSignals.filter((s) => s.symbol.toUpperCase() === activePair)
  }, [cleanSignals, activePair])

  const pairPositions = useMemo(() => {
    if (!activePair) return []
    return (positions ?? []).filter((p) => p.symbol.toUpperCase() === activePair)
  }, [positions, activePair])

  // Filter berdasarkan metode (ILA 2.0 / ILA 3.0). Tidak ada mode campuran
  // supaya chart dan matriks timeframe tidak pernah tercampur antar metode.
  const filteredSignals = useMemo(() => {
    return pairSignals.filter((s) => s.strategy_version === activeStrategyFilter)
  }, [pairSignals, activeStrategyFilter])

  const timeframeAktif = useMemo(
    () => setTimeframeDipantau(timeframeMatrix, activeStrategyFilter),
    [timeframeMatrix, activeStrategyFilter],
  )

  const slotsMetode = useMemo(
    () => mapSignalsToSlots(filteredSignals, activeStrategyFilter),
    [filteredSignals, activeStrategyFilter],
  )

  const effectiveSlotKey = useMemo(() => {
    if (selectedSlotKey && slotsMetode.some((s) => s.key === selectedSlotKey)) {
      return selectedSlotKey
    }
    return pilihSlotAwal(slotsMetode, timeframeAktif)
  }, [selectedSlotKey, slotsMetode, timeframeAktif])

  // Daftar riwayat hanya berisi setup sungguhan yang memiliki harga entry,
  // bukan baris pantauan radar (`status-*` / `WAIT_*` berharga 0).
  const signalsRiwayat = useMemo(() => {
    return filteredSignals
      .filter((s) => !isRadarProgressSignal(s))
      .sort((a, b) => new Date(b.published_at).getTime() - new Date(a.published_at).getTime())
  }, [filteredSignals])

  const filteredPositions = useMemo(() => {
    return pairPositions.filter((p) => p.strategy_version === activeStrategyFilter)
  }, [pairPositions, activeStrategyFilter])

  // Pagination untuk riwayat setup sinyal agar tidak melar panjang ke bawah
  const [historyPage, setHistoryPage] = useState(1)
  // 4 per halaman agar chart di atasnya tetap lega dibaca.
  const historyPerPage = 4
  const totalHistoryPages = Math.ceil(signalsRiwayat.length / historyPerPage) || 1
  const paginatedSignals = useMemo(() => {
    const start = (historyPage - 1) * historyPerPage
    return signalsRiwayat.slice(start, start + historyPerPage)
  }, [signalsRiwayat, historyPage])

  const handleSelectPair = (symbol: string, strategy?: 'ila-2.0' | 'ila-3.0') => {
    if (strategy) setActiveStrategyFilter(strategy)
    setSelectedSlotKey(null)
    setSnapshotSignal(null)
    setHistoryPage(1)
    setParams({ pair: symbol })
  }

  const handleBackToPairs = () => {
    setSelectedSlotKey(null)
    setSnapshotSignal(null)
    params.delete('pair')
    setParams(params)
  }

  const isLoading = loadingSignals || loadingSubs

  // Putaran pada tombol "Muat Ulang" hanya mewakili permintaan yang diminta
  // pengguna. Penyegaran berkala berjalan di latar belakang, dan menampilkan
  // spinner tiap 15 detik membuat halaman tampak terus dimuat ulang.
  const [memuatManual, setMemuatManual] = useState(false)
  const muatUlang = async () => {
    setMemuatManual(true)
    try {
      await refetch()
    } finally {
      setMemuatManual(false)
    }
  }

  // Buka modal pengaturan saat tombol "Aktifkan pemantauan" di radar diklik.
  useEffect(() => {
    if (!activePair) return
    const onOpenSettings = () => setSettingsOpen(true)
    window.addEventListener('ila:open-timeframe-settings', onOpenSettings)
    return () => window.removeEventListener('ila:open-timeframe-settings', onOpenSettings)
  }, [activePair])

  if (!isDesktop) {
    return (
      <>
        <SignalsMobileView
          signals={cleanSignals}
          allPairs={allPairs}
          activePair={activePair}
          onSelectPair={handleSelectPair}
          onBackToPairs={handleBackToPairs}
          isLoading={isLoading}
          isFetching={isFetching}
          refetch={refetch}
          adaIla3={adaIla3}
          activeStrategyFilter={activeStrategyFilter}
          onSelectStrategy={(strat) => {
            setActiveStrategyFilter(strat)
            setSelectedSlotKey(null)
            setSnapshotSignal(null)
            setHistoryPage(1)
          }}
          filteredSignals={filteredSignals}
          filteredPositions={filteredPositions}
          timeframeAktif={timeframeAktif}
          selectedSlotKey={effectiveSlotKey}
          onSelectSlot={(key) => {
            setSelectedSlotKey(key)
            setSnapshotSignal(null)
          }}
          snapshotSignal={snapshotSignal}
          onSelectSnapshot={setSnapshotSignal}
          onOpenSettings={() => setSettingsOpen(true)}
        />
        {activePair && (
          <PengaturanPairModal
            open={settingsOpen}
            onClose={() => setSettingsOpen(false)}
            symbol={activePair}
            initialStrategy={activeStrategyFilter}
            adaIla3={adaIla3}
          />
        )}
      </>
    )
  }

  return (
    <div className="space-y-6">
        <PageHeader
          title={activePair ? `Signals — ${activePair}` : 'Signals & Radar'}
          description={
            activePair
              ? `Ringkasan sinyal aktif${adaIla3 ? ', analisis per metode (ILA 2.0 & ILA 3.0),' : ''} serta riwayat eksekusi order pada pair ${activePair}.`
              : 'Pusat sinyal dan radar pergerakan pasar untuk seluruh watchlist pair Anda.'
          }
          actions={
            <div className="flex items-center gap-2">
              {activePair && (
                <Button variant="outline" size="sm" onClick={handleBackToPairs} className="gap-1.5 text-xs">
                  <ArrowLeft className="h-3.5 w-3.5" />
                  Semua Pair
                </Button>
              )}
              <Button
                variant="outline"
                size="sm"
                onClick={() => void muatUlang()}
                isLoading={memuatManual}
                className="text-xs"
              >
                Muat Ulang
              </Button>
              {activePair && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setSettingsOpen(true)}
                  className="gap-1.5 text-xs"
                >
                  <Settings className="h-3.5 w-3.5" />
                  Pengaturan
                </Button>
              )}
            </div>
          }
        />

      {/* Skeleton hanya untuk pemuatan pertama. Saat data sudah pernah ada,
          perpindahan pair atau penyegaran berkala tidak mengosongkan layar. */}
      {isLoading && !signals ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-56 w-full rounded-xl" />
          ))}
        </div>
      ) : activePair ? (
        /* DETAIL VIEW PER PAIR */
        <div className="space-y-6">
          {/* Tab Utama Pemisah Metode: ILA 2.0 vs ILA 3.0 */}
          <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border">
            <div
              role="tablist"
              aria-label={`Metode analisis untuk ${activePair}`}
              className="flex items-center gap-1"
            >
              {strategiTersedia.map((strat) => {
                const isSelected = activeStrategyFilter === strat
                return (
                  <button
                    key={strat}
                    type="button"
                    role="tab"
                    aria-selected={isSelected}
                    onClick={() => {
                      setActiveStrategyFilter(strat)
                      setSelectedSlotKey(null)
                      setSnapshotSignal(null)
                      setHistoryPage(1)
                    }}
                    className={cn(
                      'relative -mb-px border-b-2 px-4 py-2.5 text-sm font-semibold transition-colors',
                      isSelected
                        ? 'border-foreground text-foreground'
                        : 'border-transparent text-muted-foreground hover:text-foreground',
                    )}
                  >
                    {strat === 'ila-2.0' ? 'ILA 2.0' : 'ILA 3.0'}
                  </button>
                )
              })}
            </div>
            <div className="pb-2 text-xs text-muted-foreground">
              {activeStrategyFilter === 'ila-2.0' ? 'New ILA 2.0 · Time Cycle' : 'ILA 3.0'} ·{' '}
              {signalsRiwayat.length} setup · {filteredPositions.length} posisi
            </div>
          </div>

          {/* Call Entry Alert Banner (Pinned di Paling Atas saat Ada Sinyal Siap Eksekusi) */}
          <CallEntryAlertBanner signals={filteredSignals} />

          {/* Kartu Pemilih Time Cycle dengan Status Overview */}
          <CycleSelectorStrip
            signals={filteredSignals}
            strategy={activeStrategyFilter}
            timeframeAktif={timeframeAktif}
            activeSlotKey={effectiveSlotKey}
            onSelectSlot={(key) => {
              setSelectedSlotKey(key)
              setSnapshotSignal(null)
            }}
          />

          {/* Sinyal Aktif / Terakhir & Histori Setup (Histori bisa collapse/expand) */}
          <div className="grid gap-6 lg:grid-cols-12">
            <div className={historyOpen ? 'lg:col-span-8' : 'lg:col-span-12'}>
              <Section
                title={
                  activeStrategyFilter === 'ila-2.0'
                    ? 'Radar Time Cycle'
                    : 'Radar Analisis Timeframe'
                }
                description={
                  activeStrategyFilter === 'ila-2.0'
                    ? `Pantauan progres time cycle New ILA 2.0 untuk ${activePair} (HTF C2 → LTF CISD → Entry).`
                    : `Pemantauan matriks timeframe multi-tier untuk ${activePair}.`
                }
                actions={
                  !historyOpen ? (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setHistoryOpen(true)}
                      className="h-7 gap-1.5 px-2.5 text-xs"
                    >
                      <PanelRightOpen className="h-3.5 w-3.5" />
                      Tampilkan Histori ({signalsRiwayat.length})
                    </Button>
                  ) : undefined
                }
              >
                <TimeframeRadarMatrix
                  signals={filteredSignals}
                  strategy={activeStrategyFilter}
                  symbol={activePair}
                  timeframeAktif={timeframeAktif}
                  activeSlotKey={effectiveSlotKey}
                  hideSlotTabs
                  onSelectSlot={(key) => {
                    setSelectedSlotKey(key)
                    setSnapshotSignal(null)
                  }}
                />
              </Section>
            </div>

            {/* Riwayat Setup Sinyal Pair Ini dengan Pagination (Collapsible) */}
            {historyOpen && (
              <div className="lg:col-span-4">
                <Section
                  title="Histori Setup Sinyal"
                  description={`${signalsRiwayat.length} setup unik pada ${activePair} · klik untuk pratinjau chart`}
                  actions={
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setHistoryOpen(false)}
                      className="h-7 gap-1 px-2 text-xs text-muted-foreground hover:text-foreground"
                      title="Sembunyikan Histori Setup Sinyal"
                    >
                      <PanelRightClose className="h-3.5 w-3.5" />
                      Sembunyikan
                    </Button>
                  }
                >
                {signalsRiwayat.length === 0 ? (
                  <Card>
                    <CardContent className="py-6 text-center text-xs text-muted-foreground">
                      Belum ada riwayat setup pada pair {activePair} untuk metode{' '}
                      {activeStrategyFilter}.
                      <span className="mt-1 block text-[11px] text-muted-foreground/80">
                        Sinyal hanya dikirim untuk pasangan timeframe yang dipantau.
                        Nyalakan minimal satu switch metode {activeStrategyFilter} pada
                        pair ini di pengaturan switch timeframe.
                      </span>
                    </CardContent>
                  </Card>
                ) : (
                  <div className="space-y-2.5">
                    {paginatedSignals.map((signal) => {
                      const outcome = describeSignalOutcome(signal)
                      const decimals = priceDigits(signal.symbol, signal.entry_price)
                      const isSelected = snapshotSignal?.signal_id === signal.signal_id

                      return (
                        <button
                          key={signal.signal_id}
                          type="button"
                          onClick={() =>
                            setSnapshotSignal((prev) =>
                              prev?.signal_id === signal.signal_id ? null : signal,
                            )
                          }
                          aria-pressed={isSelected}
                          className={cn(
                            'w-full rounded-xl border bg-card p-3.5 text-left transition-all',
                            isSelected
                              ? 'border-foreground ring-1 ring-foreground/20'
                              : 'border-border hover:border-zinc-400',
                          )}
                        >
                          <div className="flex flex-wrap items-center justify-between gap-2">
                            <div className="flex flex-wrap items-center gap-1.5">
                              <span
                                className={cn(
                                  'inline-flex items-center gap-1 text-xs font-bold',
                                  signal.direction === 'BUY'
                                    ? 'text-emerald-600 dark:text-emerald-400'
                                    : signal.direction === 'SELL'
                                      ? 'text-rose-600 dark:text-rose-400'
                                      : 'text-muted-foreground',
                                )}
                              >
                                {signal.direction === 'BUY' ? (
                                  <TrendingUp className="h-3.5 w-3.5" />
                                ) : signal.direction === 'SELL' ? (
                                  <TrendingDown className="h-3.5 w-3.5" />
                                ) : null}
                                {signal.direction}
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
                              {outcome.executedBadge && (
                                <Badge variant="outline" className="px-1.5 py-0 text-[9.5px]">
                                  Dieksekusi
                                </Badge>
                              )}
                            </div>
                            <span
                              className={cn(
                                'inline-flex items-center rounded-md border px-2 py-0.5 text-[10.5px] font-semibold',
                                OUTCOME_BADGE_CLASS[outcome.tone],
                              )}
                            >
                              {outcome.label}
                            </span>
                          </div>

                          {signal.entry_price > 0 && (
                            <div className="mt-2.5 grid grid-cols-3 gap-2 border-t border-border pt-2 text-[11.5px]">
                              <div className="min-w-0">
                                <p className="text-[10px] uppercase text-muted-foreground">Entry</p>
                                <p className="truncate font-mono font-semibold tabular-nums">
                                  {formatNumber(signal.entry_price, decimals)}
                                </p>
                              </div>
                              <div className="min-w-0">
                                <p className="text-[10px] uppercase text-muted-foreground">SL / TP</p>
                                <p className="truncate font-mono text-[10.5px] tabular-nums text-muted-foreground">
                                  {formatNumber(signal.stop_loss, decimals)} /{' '}
                                  {formatNumber(signal.take_profit, decimals)}
                                </p>
                              </div>
                              <div className="min-w-0 text-right">
                                <p className="text-[10px] uppercase text-muted-foreground">R:R</p>
                                <p className="truncate font-mono font-semibold tabular-nums text-foreground">
                                  {signal.risk_reward > 0
                                    ? `1:${formatNumber(signal.risk_reward, 2)}`
                                    : '—'}
                                </p>
                              </div>
                            </div>
                          )}

                          {outcome.executedBadge && outcome.subtext && (
                            <div className="mt-2 flex items-center justify-between rounded-lg bg-secondary/50 px-2.5 py-1 text-[11px]">
                              <span className="text-muted-foreground">PnL Eksekusi</span>
                              <span
                                className={cn(
                                  'font-mono font-semibold tabular-nums',
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

                          <div className="mt-2 flex flex-wrap items-center justify-between gap-1 text-[10.5px] text-muted-foreground">
                            <span>Terbit: {formatDateTime(signal.published_at)}</span>
                            {outcome.resolvedAt ? (
                              <span>Selesai: {formatDateTime(outcome.resolvedAt)}</span>
                            ) : signal.delayed ? (
                              <span className="font-semibold uppercase text-amber-600">Delayed 15m</span>
                            ) : (
                              <span className="text-[10px] font-medium text-foreground/70">
                                {isSelected ? 'Sedang tampil di chart' : 'Lihat chart'}
                              </span>
                            )}
                          </div>
                        </button>
                      )
                    })}

                    {/* Pagination Controls */}
                    {totalHistoryPages > 1 && (
                      <div className="mt-3 flex items-center justify-between border-t border-border pt-2.5 text-xs">
                        <span className="text-[11px] text-muted-foreground">
                          Halaman {historyPage} dari {totalHistoryPages}
                        </span>
                        <div className="flex items-center gap-1.5">
                          <Button
                            size="sm"
                            variant="outline"
                            className="h-7 px-2.5 text-xs"
                            disabled={historyPage <= 1}
                            onClick={() => setHistoryPage((p) => Math.max(1, p - 1))}
                          >
                            Sebelumnya
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            className="h-7 px-2.5 text-xs"
                            disabled={historyPage >= totalHistoryPages}
                            onClick={() => setHistoryPage((p) => Math.min(totalHistoryPages, p + 1))}
                          >
                            Selanjutnya
                          </Button>
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </Section>
              </div>
            )}
          </div>

          {/* Chart Setup Full-Width (sejajar penuh seperti tabel Riwayat Eksekusi Order) */}
          {isDesktop && (
            <Section
              title="Chart Setup"
              description={`Lilin ${activePair} dari feed yang sama dengan evaluasi engine.`}
            >
              <TradingRadarChart
                symbol={activePair}
                signals={filteredSignals}
                strategy={activeStrategyFilter}
                selectedSlotKey={effectiveSlotKey}
                snapshotSignal={snapshotSignal}
                onClearSnapshot={() => setSnapshotSignal(null)}
              />
            </Section>
          )}

          {/* Riwayat Posisi / Eksekusi Order */}
          <Section
            title="Riwayat Eksekusi Order"
            description={`Catatan entry posisi riil broker yang dieksekusi dari sinyal ${activePair}.`}
          >
            {loadingPositions ? (
              <Skeleton className="h-32 w-full" />
            ) : filteredPositions.length === 0 ? (
              <EmptyState
                title="Belum ada riwayat posisi"
                description={`Belum ada eksekusi transaksi yang tercatat pada instrumen ${activePair}.`}
              />
            ) : (
              <div className="overflow-x-auto rounded-xl border border-border bg-card">
                <table className="w-full text-left text-xs">
                  <thead className="border-b border-border bg-secondary/50 text-muted-foreground">
                    <tr>
                      <th className="px-3.5 py-2.5 font-medium">Order / Waktu</th>
                      <th className="px-3.5 py-2.5 font-medium">Arah</th>
                      <th className="px-3.5 py-2.5 font-medium">Metode</th>
                      <th className="px-3.5 py-2.5 font-medium">Lot</th>
                      <th className="px-3.5 py-2.5 font-medium">Open Price</th>
                      <th className="px-3.5 py-2.5 font-medium">Status / PnL</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border">
                    {filteredPositions.map((pos) => (
                      <tr key={pos.id} className="hover:bg-secondary/30 transition-colors">
                        <td className="px-3.5 py-2.5 font-mono">
                          <p className="font-semibold text-foreground">
                            #{pos.ticket_number || pos.id.slice(0, 8)}
                          </p>
                          <p className="text-[10.5px] text-muted-foreground">
                            {formatDateTime(pos.opened_at)}
                          </p>
                        </td>
                        <td className="px-3.5 py-2.5">
                          <span
                            className={cn(
                              'inline-flex items-center gap-1 font-semibold',
                              pos.direction === 'BUY'
                                ? 'text-emerald-600 dark:text-emerald-400'
                                : 'text-rose-600 dark:text-rose-400',
                            )}
                          >
                            {pos.direction === 'BUY' ? (
                              <TrendingUp className="h-3 w-3" />
                            ) : (
                              <TrendingDown className="h-3 w-3" />
                            )}
                            {pos.direction}
                          </span>
                        </td>
                        <td className="px-3.5 py-2.5">
                          <Badge variant="outline" className="text-[10px]">
                            {pos.strategy_version}
                          </Badge>
                        </td>
                        <td className="px-3.5 py-2.5 tabular-nums font-medium">{pos.lot_size}</td>
                        <td className="px-3.5 py-2.5 tabular-nums">
                          {formatNumber(pos.open_price, 5)}
                        </td>
                        <td className="px-3.5 py-2.5">
                          <div className="space-y-0.5">
                            <Badge
                              variant={pos.status.startsWith('CLOSED_TP') ? 'solid' : 'outline'}
                              className="text-[10px]"
                            >
                              {pos.status}
                            </Badge>
                            {pos.pnl_known && (
                              <p
                                className={cn(
                                  'text-[11px] font-semibold tabular-nums',
                                  pos.profit_loss >= 0 ? 'text-emerald-600' : 'text-rose-600',
                                )}
                              >
                                {formatMoney(pos.profit_loss)} ({formatR(pos.outcome_r)})
                              </p>
                            )}
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </Section>

          {/* Modal Pengaturan Metode & Time Cycle per Akun */}
          <PengaturanPairModal
            open={settingsOpen}
            onClose={() => setSettingsOpen(false)}
            symbol={activePair}
            initialStrategy={activeStrategyFilter}
            adaIla3={adaIla3}
          />
        </div>
      ) : (
        /* OVERVIEW GRID VIEW PER PAIR */
        <div className="space-y-6">
          {allPairs.length === 0 ? (
            <Card>
              <CardContent className="py-12 text-center">
                <Bot className="mx-auto h-10 w-10 text-muted-foreground/60" />
                <h3 className="mt-3 text-base font-semibold">Watchlist Pair Kosong</h3>
                <p className="mx-auto mt-1 max-w-sm text-xs text-muted-foreground">
                  Tambahkan simbol instrumen (misal XAUUSD, EURUSD) pada halaman Watchlist Pair agar sinyal trading
                  otomatis muncul di sini.
                </p>
                <div className="mt-5">
                  <Button size="sm" asChild>
                    <Link to="/strategi">Kelola Watchlist Pair</Link>
                  </Button>
                </div>
              </CardContent>
            </Card>
          ) : (
            <div className="grid items-start gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {allPairs.map((symbol) => {
                const pairSigs = cleanSignals.filter((s) => s.symbol.toUpperCase() === symbol)
                const pairSetupCount = pairSigs.filter((s) => !isRadarProgressSignal(s)).length
                const pairPos = (positions ?? []).filter((p) => p.symbol.toUpperCase() === symbol)
                const hasActiveSub = (subscriptions ?? []).some(
                  (sub) => sub.symbol.toUpperCase() === symbol && sub.is_active,
                )

                return (
                  <PairOverviewCard
                    key={symbol}
                    symbol={symbol}
                    pairSignals={pairSigs}
                    setupCount={pairSetupCount}
                    orderCount={pairPos.length}
                    hasActiveSub={hasActiveSub}
                    adaIla3={adaIla3}
                    onOpenDetail={(strat) => handleSelectPair(symbol, strat)}
                  />
                )
              })}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

type OverviewMethodTab = {
  strategy: 'ila-2.0' | 'ila-3.0'
  slots: TimeframeSlot[]
  ringkas: ReturnType<typeof ringkasanSlot>
}

function PairOverviewCard({
  symbol,
  pairSignals,
  setupCount,
  orderCount,
  hasActiveSub,
  adaIla3,
  onOpenDetail,
}: {
  symbol: string
  pairSignals: Signal[]
  setupCount: number
  orderCount: number
  hasActiveSub: boolean
  adaIla3: boolean
  onOpenDetail: (strategy?: 'ila-2.0' | 'ila-3.0') => void
}) {
  const { data: stratMatrix, isLoading: loadingStrat } = useStrategySwitchMatrix(symbol)
  const { data: tfMatrix, isLoading: loadingTf } = useTimeframeSwitchMatrix(symbol)

  const metodeTampil = useMemo<OverviewMethodTab[]>(() => {
    const kandidat: Array<'ila-2.0' | 'ila-3.0'> = adaIla3 ? ['ila-2.0', 'ila-3.0'] : ['ila-2.0']
    const hasAccountConfig = Boolean(stratMatrix && stratMatrix.accounts.length > 0)
    const hasil: OverviewMethodTab[] = []

    for (const strategy of kandidat) {
      // Bila akun sudah terkonfigurasi, metode hanya tampil jika aktif di minimal 1 akun.
      if (hasAccountConfig) {
        const nyala = stratMatrix!.accounts.some((a) => Boolean(a.strategies[strategy]))
        if (!nyala) continue
      }

      const semuaSlot = mapSignalsToSlots(
        pairSignals.filter((s) => s.strategy_version === strategy),
        strategy,
      )
      const aktifSet = setTimeframeDipantau(tfMatrix, strategy)
      const slotDipantau =
        aktifSet !== null
          ? semuaSlot.filter((slot) => timeframeSlotDipantau(aktifSet, slot))
          : semuaSlot

      // Jika semua cycle metode ini dimatikan, sembunyikan metode dari kartu.
      if (slotDipantau.length === 0) continue

      hasil.push({
        strategy,
        slots: slotDipantau,
        ringkas: ringkasanSlot(slotDipantau),
      })
    }

    return hasil
  }, [adaIla3, pairSignals, stratMatrix, tfMatrix])

  const [tabPilihan, setTabPilihan] = useState<'ila-2.0' | 'ila-3.0'>('ila-2.0')
  const tabAktif =
    metodeTampil.find((m) => m.strategy === tabPilihan) ?? metodeTampil[0] ?? null
  const terdekat = tabAktif ? slotPalingDekat(tabAktif.slots, tabAktif.strategy) : null

  return (
    <Card className="flex flex-col overflow-hidden border-border bg-card transition-all hover:border-zinc-400 hover:shadow-sm">
      <CardHeader className="border-b border-border/80 pb-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-foreground text-background font-bold text-xs">
              {symbol.slice(0, 3)}
            </span>
            <div>
              <CardTitle className="text-base font-bold tracking-tight">{symbol}</CardTitle>
              <p className="text-[11px] text-muted-foreground">
                {hasActiveSub ? 'Dipantau aktif' : 'Riwayat arsip'}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-1.5">
            {hasActiveSub && (
              <span className="flex h-2 w-2 rounded-full bg-emerald-500" title="Aktif dipantau" />
            )}
            <Badge variant="outline" className="text-[10px]">
              {setupCount} Setup
            </Badge>
          </div>
        </div>
      </CardHeader>

      <CardContent className="space-y-3 pt-3.5">
        {loadingStrat || loadingTf ? (
          <Skeleton className="h-24 w-full rounded-lg" />
        ) : !tabAktif ? (
          <div className="rounded-lg border border-dashed border-border/80 bg-secondary/20 px-3 py-5 text-center text-xs text-muted-foreground">
            Belum ada metode atau time cycle yang dipantau.
          </div>
        ) : (
          <div className="space-y-2.5">
            {/* Mini-tab di dalam kartu (hanya muncul bila kedua metode aktif & punya cycle dipantau) */}
            {metodeTampil.length > 1 ? (
              <div
                role="tablist"
                aria-label={`Metode aktif ${symbol}`}
                className="grid grid-cols-2 gap-1 rounded-lg bg-secondary/60 p-1"
              >
                {metodeTampil.map((m) => {
                  const selected = m.strategy === tabAktif.strategy
                  return (
                    <button
                      key={m.strategy}
                      type="button"
                      role="tab"
                      aria-selected={selected}
                      onClick={() => setTabPilihan(m.strategy)}
                      className={cn(
                        'flex items-center justify-center gap-1.5 rounded-md px-2 py-1 text-[11px] font-semibold transition-all',
                        selected
                          ? 'bg-card text-foreground shadow-xs'
                          : 'text-muted-foreground hover:text-foreground',
                      )}
                    >
                      <span>{m.strategy === 'ila-2.0' ? 'ILA 2.0' : 'ILA 3.0'}</span>
                      {m.ringkas.siap > 0 ? (
                        <span className="rounded bg-emerald-600 px-1 py-0.5 font-mono text-[9px] font-bold leading-none text-white">
                          {m.ringkas.siap}
                        </span>
                      ) : (
                        <span className="font-mono text-[10px] text-muted-foreground">
                          {m.slots.length}
                        </span>
                      )}
                    </button>
                  )
                })}
              </div>
            ) : (
              <div className="flex items-center justify-between text-xs">
                <span className="font-semibold text-foreground">
                  {tabAktif.strategy === 'ila-2.0' ? 'ILA 2.0' : 'ILA 3.0'}
                </span>
                <span className="text-[10.5px] tabular-nums text-muted-foreground">
                  {tabAktif.ringkas.siap > 0
                    ? `${tabAktif.ringkas.siap} ready`
                    : `${tabAktif.slots.length} cycle dipantau`}
                </span>
              </div>
            )}

            {/* Daftar baris khusus cycle yang diaktifkan pantauannya */}
            <div className="space-y-1.5">
              {tabAktif.slots.map((slot) => (
                <CycleOverviewRow
                  key={slot.key}
                  slot={slot}
                  stage={stageOf(tabAktif.strategy, slot.latestSignal?.status ?? '')}
                  tersorot={slot.key === terdekat}
                />
              ))}
            </div>
          </div>
        )}

        <div className="flex items-center justify-between border-t border-border/50 pt-2.5 text-[11px] text-muted-foreground">
          <span className="flex items-center gap-1">
            <BarChart3 className="h-3.5 w-3.5" />
            {orderCount} Eksekusi Order
          </span>
          <Button
            variant="ghost"
            size="sm"
            className="h-7 gap-1 px-2 text-xs font-semibold hover:text-foreground"
            onClick={() => onOpenDetail(tabAktif?.strategy)}
          >
            Rincian
            <ArrowUpRight className="h-3 w-3" />
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

function CycleOverviewRow({
  slot,
  stage,
  tersorot,
}: {
  slot: TimeframeSlot
  stage: SlotStage
  tersorot: boolean
}) {
  const signal = slot.latestSignal
  const arah = signal?.direction
  const punyaArah = arah === 'BUY' || arah === 'SELL'

  return (
    <div
      className={cn(
        'flex items-center justify-between gap-2 rounded-lg border px-2.5 py-2 text-xs',
        stage.ready
          ? 'border-emerald-500/60 bg-emerald-500/[0.04]'
          : tersorot
            ? 'border-foreground/40 bg-secondary/60'
            : 'border-border/60 bg-secondary/25',
      )}
    >
      <div className="min-w-0">
        <p className="truncate text-[11.5px] font-semibold text-foreground leading-tight">
          {slot.cycle ?? slot.label}
        </p>
        <p className="font-mono text-[10px] text-muted-foreground leading-tight mt-0.5">
          {slot.label}
        </p>
      </div>

      <div className="shrink-0 text-right">
        <div className="flex items-center justify-end gap-1.5">
          {punyaArah && (
            <span
              className={cn(
                'inline-flex items-center gap-0.5 font-mono text-[10.5px] font-bold',
                arah === 'BUY'
                  ? 'text-emerald-600 dark:text-emerald-400'
                  : 'text-rose-600 dark:text-rose-400',
              )}
            >
              {arah === 'BUY' ? (
                <TrendingUp className="h-3 w-3" />
              ) : (
                <TrendingDown className="h-3 w-3" />
              )}
              {arah}
            </span>
          )}
          <span
            className={cn(
              'text-[10.5px]',
              stage.ready ? 'font-bold text-foreground' : 'text-muted-foreground',
            )}
          >
            {stage.label}
          </span>
        </div>
        {stage.ready && signal && signal.entry_price > 0 && (
          <p className="font-mono text-[10px] tabular-nums text-muted-foreground mt-0.5">
            @ {formatPrice(signal.entry_price, priceDigits(signal.symbol, signal.entry_price))}
          </p>
        )}
      </div>
    </div>
  )
}
