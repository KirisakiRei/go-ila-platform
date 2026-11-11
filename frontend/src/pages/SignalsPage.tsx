import { useState, useMemo } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import {
  ArrowLeft,
  ArrowUpRight,
  BarChart3,
  Bot,
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
  StatusPill,
} from '@/components/ui'
import { usePositions, useSignals, useStrategies, useSubscriptions } from '@/lib/queries'
import { formatDateTime, formatMoney, formatNumber, formatR } from '@/lib/format'
import { cn } from '@/lib/utils'
import { SIGNAL_LABEL, SIGNAL_TONE } from '@/features/signals/lib/status'
import { CallEntryAlertBanner } from '@/features/signals/components/CallEntryAlertBanner'
import { TimeframeRadarMatrix } from '@/features/signals/components/TimeframeRadarMatrix'
import { TradingRadarChart } from '@/features/charts/TradingRadarChart'
import { SaklarMetodeAkun } from '@/features/signals/components/SaklarMetodeAkun'
import { SaklarTimeframeAkun } from '@/features/signals/components/SaklarTimeframeAkun'
import { TimeframeBadge } from '@/features/signals/components/TimeframeBadge'
import { SignalsMobileView } from './components/SignalsMobileView'

export default function SignalsPage() {
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

  const [activeStrategyFilter, setActiveStrategyFilter] = useState<'all' | 'ila-2.0' | 'ila-3.0'>('all')

  // Filter metode yang ditawarkan mengikuti hak akses tier pengguna.
  const strategiTersedia = useMemo(
    () =>
      (adaIla3
        ? (['all', 'ila-2.0', 'ila-3.0'] as const)
        : (['all', 'ila-2.0'] as const)
      ).slice(),
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

  // Sinyal dan posisi untuk pair yang sedang aktif
  const pairSignals = useMemo(() => {
    if (!activePair) return []
    return (signals ?? []).filter((s) => s.symbol.toUpperCase() === activePair)
  }, [signals, activePair])

  const pairPositions = useMemo(() => {
    if (!activePair) return []
    return (positions ?? []).filter((p) => p.symbol.toUpperCase() === activePair)
  }, [positions, activePair])

  // Filter berdasarkan metode (ILA 2.0 / ILA 3.0)
  const filteredSignals = useMemo(() => {
    if (activeStrategyFilter === 'all') return pairSignals
    return pairSignals.filter((s) => s.strategy_version === activeStrategyFilter)
  }, [pairSignals, activeStrategyFilter])

  const filteredPositions = useMemo(() => {
    if (activeStrategyFilter === 'all') return pairPositions
    return pairPositions.filter((p) => p.strategy_version === activeStrategyFilter)
  }, [pairPositions, activeStrategyFilter])

  // Pagination untuk riwayat setup sinyal agar tidak melar panjang ke bawah
  const [historyPage, setHistoryPage] = useState(1)
  const historyPerPage = 5
  const totalHistoryPages = Math.ceil(filteredSignals.length / historyPerPage) || 1
  const paginatedSignals = useMemo(() => {
    const start = (historyPage - 1) * historyPerPage
    return filteredSignals.slice(start, start + historyPerPage)
  }, [filteredSignals, historyPage])

  const handleSelectPair = (symbol: string) => {
    setParams({ pair: symbol })
  }

  const handleBackToPairs = () => {
    params.delete('pair')
    setParams(params)
  }

  const isLoading = loadingSignals || loadingSubs

  return (
    <>
      {/* ============================================================ */}
      {/* 1. TAMPILAN KHUSUS MOBILE (< lg) — Persis signals_radar_mobile */}
      {/* ============================================================ */}
      <SignalsMobileView
        signals={signals}
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
          setHistoryPage(1)
        }}
        filteredSignals={filteredSignals}
        filteredPositions={filteredPositions}
      />

      {/* ============================================================ */}
      {/* 2. TAMPILAN DESKTOP (>= lg) — Struktur lama, aman 100%       */}
      {/* ============================================================ */}
      <div className="hidden lg:block space-y-6">
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
                onClick={() => void refetch()}
                isLoading={isFetching}
                className="text-xs"
              >
                Muat Ulang
              </Button>
            </div>
          }
        />

      {isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-56 w-full rounded-xl" />
          ))}
        </div>
      ) : activePair ? (
        /* DETAIL VIEW PER PAIR */
        <div className="space-y-6">
          {/* Filter Bar Metode */}
          <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border bg-card p-3">
            <div className="flex items-center gap-2">
              <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                Metode Analisis:
              </span>
              <div className="flex items-center gap-1.5">
                {strategiTersedia.map((strat) => (
                  <Button
                    key={strat}
                    size="sm"
                    variant={activeStrategyFilter === strat ? 'default' : 'outline'}
                    className="h-8 text-xs font-medium"
                    onClick={() => {
                      setActiveStrategyFilter(strat)
                      setHistoryPage(1)
                    }}
                  >
                    {strat === 'all' ? 'Semua Metode' : strat === 'ila-2.0' ? 'ILA 2.0' : 'ILA 3.0'}
                  </Button>
                ))}
              </div>
            </div>

            <div className="text-xs text-muted-foreground">
              Menampilkan {filteredSignals.length} sinyal · {filteredPositions.length} posisi
            </div>
          </div>

          {/* Call Entry Alert Banner (Pinned di Paling Atas saat Ada Sinyal Siap Eksekusi) */}
          <CallEntryAlertBanner signals={filteredSignals} />

          {/* Sinyal Aktif / Terakhir & Histori Setup Langsung di Atas */}
          <div className="grid gap-6 lg:grid-cols-12">
            <div className="lg:col-span-8 space-y-6">
              <Section
                title="Radar Analisis Timeframe"
                description={`Pemantauan matriks timeframe multi-tier untuk ${activePair}.`}
              >
                <TimeframeRadarMatrix
                  signals={filteredSignals}
                  strategy={activeStrategyFilter}
                  symbol={activePair}
                />
              </Section>

              {/* Chart setup: lilin, sweep, FVG, CISD, dan level order. */}
              <Section
                title="Chart Setup"
                description={`Lilin ${activePair} dari feed yang sama dengan evaluasi engine.`}
              >
                <TradingRadarChart
                  symbol={activePair}
                  signals={filteredSignals}
                  strategy={activeStrategyFilter}
                />
              </Section>
            </div>

            {/* Riwayat Setup Sinyal Pair Ini dengan Pagination */}
            <div className="lg:col-span-4">
              <Section
                title="Histori Setup Sinyal"
                description={`${filteredSignals.length} setup terakhir pada ${activePair}`}
              >
                {filteredSignals.length === 0 ? (
                  <Card>
                    <CardContent className="py-6 text-center text-xs text-muted-foreground">
                      Belum ada riwayat setup pada pair {activePair}.
                    </CardContent>
                  </Card>
                ) : (
                  <div className="space-y-2.5">
                    {paginatedSignals.map((signal) => (
                      <div
                        key={signal.signal_id}
                        className="rounded-xl border border-border bg-card p-3.5 transition-colors hover:border-zinc-300"
                      >
                        <div className="flex items-center justify-between gap-2">
                          <div className="flex items-center gap-2">
                            <span
                              className={cn(
                                'inline-flex items-center gap-1 font-bold text-xs',
                                signal.direction === 'BUY'
                                  ? 'text-emerald-600'
                                  : signal.direction === 'SELL'
                                    ? 'text-rose-600'
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
                            <Badge variant="outline" className="text-[10.5px]">
                              {signal.strategy_version}
                            </Badge>
                            <TimeframeBadge htf={signal.htf} ltf={signal.ltf} />
                          </div>
                          <StatusPill tone={SIGNAL_TONE[signal.status]} className="text-[11px]">
                            {SIGNAL_LABEL[signal.status]}
                          </StatusPill>
                        </div>

                        {signal.entry_price > 0 && (
                          <div className="mt-2.5 grid grid-cols-3 gap-2 border-t border-border pt-2 text-[11.5px]">
                            <div>
                              <p className="text-[10px] uppercase text-muted-foreground">Entry</p>
                              <p className="font-semibold tabular-nums font-mono">
                                {formatNumber(signal.entry_price, 5)}
                              </p>
                            </div>
                            <div>
                              <p className="text-[10px] uppercase text-muted-foreground">SL / TP</p>
                              <p className="font-mono text-[10.5px] tabular-nums text-muted-foreground">
                                {formatNumber(signal.stop_loss, 4)} /{' '}
                                {formatNumber(signal.take_profit, 4)}
                              </p>
                            </div>
                            <div className="text-right">
                              <p className="text-[10px] uppercase text-muted-foreground">R:R</p>
                              <p className="font-semibold tabular-nums font-mono">
                                1:{formatNumber(signal.risk_reward, 2)}
                              </p>
                            </div>
                          </div>
                        )}

                        <div className="mt-2 flex items-center justify-between text-[10.5px] text-muted-foreground">
                          <span>Terbit: {formatDateTime(signal.published_at)}</span>
                          {signal.delayed && (
                            <span className="font-semibold uppercase text-amber-600">Delayed 15m</span>
                          )}
                        </div>
                      </div>
                    ))}

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
          </div>

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

          {/* Konfigurasi Akun untuk Pair Ini */}
          <div className="space-y-6 border-t border-border/80 pt-6">
            <div className="flex flex-col gap-1">
              <h3 className="text-base font-semibold tracking-tight text-foreground">
                Pengaturan Akun ({activePair})
              </h3>
              <p className="text-xs text-muted-foreground">
                Konfigurasi metode dan pasangan timeframe yang diizinkan untuk akun trading Anda.
              </p>
            </div>

            <Section
              title="Metode per Akun"
              description={`Tentukan metode analisis yang dijalankan tiap akun pada ${activePair}. Watchlist tetap memantau seluruh metode.`}
            >
              <SaklarMetodeAkun symbol={activePair} />
            </Section>

            <Section
              title="Timeframe per Akun"
              description={`Pilih pasangan timeframe yang dipantau dan diizinkan eksekusi otomatis untuk ${activePair}.`}
            >
              <SaklarTimeframeAkun symbol={activePair} />
            </Section>
          </div>
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
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {allPairs.map((symbol) => {
                const pairSigs = (signals ?? []).filter((s) => s.symbol.toUpperCase() === symbol)
                const pairPos = (positions ?? []).filter((p) => p.symbol.toUpperCase() === symbol)

                const latestIla2 = pairSigs.find((s) => s.strategy_version === 'ila-2.0')
                const latestIla3 = pairSigs.find((s) => s.strategy_version === 'ila-3.0')

                const hasActiveSub = (subscriptions ?? []).some(
                  (sub) => sub.symbol.toUpperCase() === symbol && sub.is_active,
                )

                return (
                  <Card
                    key={symbol}
                    className="flex flex-col justify-between overflow-hidden border-border bg-card transition-all hover:border-zinc-400 hover:shadow-sm"
                  >
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
                            {pairSigs.length} Setup
                          </Badge>
                        </div>
                      </div>
                    </CardHeader>

                    <CardContent className="space-y-3 pt-3.5">
                      {/* Sub-card ILA 2.0 */}
                      <div className="rounded-lg border border-border/70 bg-secondary/40 p-2.5">
                        <div className="flex items-center justify-between text-xs">
                          <span className="font-semibold text-foreground">ILA 2.0</span>
                          {latestIla2 ? (
                            <StatusPill tone={SIGNAL_TONE[latestIla2.status]} className="text-[10px]">
                              {SIGNAL_LABEL[latestIla2.status]}
                            </StatusPill>
                          ) : (
                            <span className="text-[11px] text-muted-foreground">Belum ada sinyal</span>
                          )}
                        </div>

                        {latestIla2 && (
                          <div className="mt-2 flex items-center justify-between text-xs">
                            <span
                              className={cn(
                                'inline-flex items-center gap-1 font-bold',
                                latestIla2.direction === 'BUY' ? 'text-emerald-600' : 'text-rose-600',
                              )}
                            >
                              {latestIla2.direction === 'BUY' ? <TrendingUp className="h-3 w-3" /> : <TrendingDown className="h-3 w-3" />}
                              {latestIla2.direction} @ {formatNumber(latestIla2.entry_price, 4)}
                            </span>
                            <span className="text-[10.5px] tabular-nums text-muted-foreground">
                              R:R 1:{formatNumber(latestIla2.risk_reward, 1)}
                            </span>
                          </div>
                        )}
                      </div>

                      {/* Sub-card ILA 3.0 — hanya untuk tier yang berhak. */}
                      {adaIla3 && (
                        <div className="rounded-lg border border-border/70 bg-secondary/40 p-2.5">
                          <div className="flex items-center justify-between text-xs">
                            <div className="flex items-center gap-1.5">
                              <span className="font-semibold text-foreground">ILA 3.0</span>
                              <Badge variant="muted" className="text-[9px]">Plus</Badge>
                            </div>
                            {latestIla3 ? (
                              <StatusPill tone={SIGNAL_TONE[latestIla3.status]} className="text-[10px]">
                                {SIGNAL_LABEL[latestIla3.status]}
                              </StatusPill>
                            ) : (
                              <span className="text-[11px] text-muted-foreground">Menunggu setup</span>
                            )}
                          </div>

                          {latestIla3 && (
                            <div className="mt-2 flex items-center justify-between text-xs">
                              <span
                                className={cn(
                                  'inline-flex items-center gap-1 font-bold',
                                  latestIla3.direction === 'BUY' ? 'text-emerald-600' : 'text-rose-600',
                                )}
                              >
                                {latestIla3.direction === 'BUY' ? <TrendingUp className="h-3 w-3" /> : <TrendingDown className="h-3 w-3" />}
                                {latestIla3.direction} @ {formatNumber(latestIla3.entry_price, 4)}
                              </span>
                              <span className="text-[10.5px] tabular-nums text-muted-foreground">
                                R:R 1:{formatNumber(latestIla3.risk_reward, 1)}
                              </span>
                            </div>
                          )}
                        </div>
                      )}

                      <div className="flex items-center justify-between pt-1 text-[11px] text-muted-foreground">
                        <span className="flex items-center gap-1">
                          <BarChart3 className="h-3.5 w-3.5" />
                          {pairPos.length} Eksekusi Order
                        </span>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 gap-1 px-2 text-xs font-semibold hover:text-foreground"
                          onClick={() => handleSelectPair(symbol)}
                        >
                          Rincian
                          <ArrowUpRight className="h-3 w-3" />
                        </Button>
                      </div>
                    </CardContent>
                  </Card>
                )
              })}
            </div>
          )}
        </div>
      )}
      </div>
    </>
  )
}
