import { useMemo, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ArrowRight, ChevronDown, TrendingDown, TrendingUp } from 'lucide-react'
import { usePairCatalog, useSignals, useSubscriptions } from '@/lib/queries'
import { formatNumber } from '@/lib/format'
import { cn } from '@/lib/utils'

export function DashboardRadarHero() {
  const navigate = useNavigate()
  const { data: subscriptions } = useSubscriptions()
  const { data: signals } = useSignals(50)
  const { data: catalog } = usePairCatalog()

  // Ambil daftar pair aktif di watchlist user
  const activePairs = useMemo(() => {
    const list: string[] = []
    const seen = new Set<string>()
    for (const sub of subscriptions ?? []) {
      if (sub.symbol && sub.is_active && !seen.has(sub.symbol.toUpperCase())) {
        seen.add(sub.symbol.toUpperCase())
        list.push(sub.symbol.toUpperCase())
      }
    }
    return list.sort()
  }, [subscriptions])

  // Peta nama instrumen ramah (misal XAUUSD -> Gold Spot)
  const namaBySymbol = useMemo(() => {
    const map = new Map<string, string>()
    for (const kategori of catalog?.categories ?? []) {
      for (const grup of kategori.groups) {
        for (const pair of grup.pairs) {
          map.set(pair.symbol, pair.name)
        }
      }
    }
    return map
  }, [catalog])

  // Default ke pair pertama atau XAUUSD jika ada
  const [selectedPair, setSelectedPair] = useState<string | null>(null)
  const currentPair =
    selectedPair && activePairs.includes(selectedPair)
      ? selectedPair
      : activePairs.includes('XAUUSD')
        ? 'XAUUSD'
        : activePairs[0] ?? null

  // Cari sinyal terbaru untuk pair yang dipilih
  const latestSignal = useMemo(() => {
    if (!currentPair) return null
    return (signals ?? []).find((s) => s.symbol.toUpperCase() === currentPair) ?? null
  }, [signals, currentPair])

  const instrumentName = currentPair
    ? namaBySymbol.get(currentPair) ?? (currentPair === 'XAUUSD' ? 'Gold Spot' : currentPair)
    : ''

  const ltfTimeframe = latestSignal?.ltf || 'M5'
  const htfTimeframe = latestSignal?.htf || 'H1'

  const decimals = currentPair?.includes('XAU') ? 2 : 4

  return (
    <section className="flex flex-col gap-2.5">
      {/* 1. Header: Sinyal Radar Pasar + Live Radar Badge + Lihat Semua */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h2 className="text-base font-bold tracking-tight text-foreground">Sinyal Radar Pasar</h2>
          <div className="flex items-center gap-1 rounded-full bg-emerald-500/10 px-2 py-0.5 text-emerald-700 dark:text-emerald-400 shrink-0">
            <span className="relative flex h-2 w-2">
              <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75" />
              <span className="relative inline-flex h-2 w-2 rounded-full bg-emerald-500" />
            </span>
            <span className="text-micro font-semibold">Live radar</span>
          </div>
        </div>

        <Link
          to="/sinyal"
          className="inline-flex items-center gap-0.5 text-xs font-semibold text-foreground hover:underline"
        >
          <span>Lihat Semua</span>
          <ArrowRight className="h-3.5 w-3.5" />
        </Link>
      </div>

      {/* 2. Horizontal Chip Scroller: Pair Filter + Timeframe Tag */}
      {activePairs.length > 0 && (
        <div className="flex items-center justify-between gap-2 overflow-x-auto pb-0.5 scrollbar-none">
          <div className="flex items-center gap-1.5 shrink-0">
            {activePairs.map((pair) => {
              const isActive = pair === currentPair
              return (
                <button
                  key={pair}
                  type="button"
                  onClick={() => setSelectedPair(pair)}
                  className={cn(
                    'flex items-center gap-1 rounded-lg px-2.5 py-1 text-xs font-medium transition-all active:scale-95 shrink-0',
                    isActive
                      ? 'border border-border bg-secondary font-semibold text-foreground shadow-xs'
                      : 'border border-transparent bg-secondary/50 text-muted-foreground hover:bg-secondary hover:text-foreground',
                  )}
                >
                  {isActive && <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />}
                  <span>{pair}</span>
                  {isActive && <ChevronDown className="h-3 w-3 text-muted-foreground" />}
                </button>
              )
            })}
          </div>

          <div className="h-4 w-px bg-border shrink-0" />

          <div className="flex items-center gap-1 shrink-0">
            <span className="rounded bg-foreground px-2 py-0.5 text-micro font-bold text-background uppercase tracking-wider">
              {ltfTimeframe}
            </span>
            <span className="rounded bg-secondary px-2 py-0.5 text-micro font-medium text-muted-foreground uppercase">
              {htfTimeframe}
            </span>
          </div>
        </div>
      )}

      {/* 3. Hero Signal Card */}
      {activePairs.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border bg-card p-6 text-center">
          <p className="text-xs font-semibold text-foreground">Watchlist Pair Kosong</p>
          <p className="mt-1 text-caption text-muted-foreground">
            Tambahkan instrumen seperti XAUUSD atau EURUSD di Watchlist Pair agar radar aktif.
          </p>
          <Link
            to="/strategi"
            className="mt-3 inline-flex items-center gap-1 rounded-lg bg-foreground px-3 py-1.5 text-xs font-medium text-background"
          >
            + Kelola Watchlist
          </Link>
        </div>
      ) : latestSignal ? (
        <div
          onClick={() => navigate(`/sinyal?pair=${currentPair}`)}
          className="flex flex-col gap-2.5 rounded-xl border border-border bg-card p-3.5 shadow-xs transition-all hover:border-zinc-400 hover:shadow-sm active:scale-[0.99] cursor-pointer"
        >
          {/* Top Row: Ticker Box + Pair + Direction + Subtitle + Status Pill */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2.5">
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-secondary font-bold text-xs text-foreground tracking-tight shrink-0">
                {currentPair?.slice(0, 3)}
              </div>
              <div className="min-w-0">
                <div className="flex items-center gap-1.5">
                  <span className="text-sm font-bold tracking-tight text-foreground">{currentPair}</span>
                  <span
                    className={cn(
                      'inline-flex items-center gap-0.5 text-xs font-bold',
                      latestSignal.direction === 'BUY'
                        ? 'text-emerald-600 dark:text-emerald-400'
                        : latestSignal.direction === 'SELL'
                          ? 'text-rose-600 dark:text-rose-400'
                          : 'text-muted-foreground',
                    )}
                  >
                    {latestSignal.direction === 'BUY' ? (
                      <TrendingUp className="h-3.5 w-3.5" />
                    ) : latestSignal.direction === 'SELL' ? (
                      <TrendingDown className="h-3.5 w-3.5" />
                    ) : null}
                    {latestSignal.direction}
                  </span>
                </div>
                <p className="truncate text-micro text-muted-foreground">
                  {instrumentName} · {latestSignal.strategy_version} · {latestSignal.htf} · {latestSignal.ltf}
                </p>
              </div>
            </div>

            <span className="rounded-md bg-foreground px-2.5 py-1 text-micro leading-none font-bold tracking-wide uppercase text-background shadow-xs shrink-0">
              {latestSignal.status === 'READY_TO_ENTRY'
                ? 'SIAP ENTRY'
                : latestSignal.status === 'WAIT_LTF_CISD'
                  ? 'WAIT CISD'
                  : latestSignal.status === 'WAIT_HTF_SWEEP'
                    ? 'PANTAU HTF'
                    : latestSignal.status.replace(/_/g, ' ')}
            </span>
          </div>

          {/* 4-Box Key Metrics Grid (Entry, SL, TP, R:R) */}
          <div className="grid grid-cols-4 divide-x divide-border/60 rounded-lg border border-border/60 bg-secondary/35 p-2 text-center">
            <div>
              <span className="block text-micro font-semibold uppercase tracking-wider text-muted-foreground">
                Entry
              </span>
              <span className="mt-0.5 block font-mono text-[12px] font-bold text-foreground">
                {latestSignal.entry_price > 0 ? formatNumber(latestSignal.entry_price, decimals) : '—'}
              </span>
            </div>
            <div>
              <span className="block text-micro font-semibold uppercase tracking-wider text-muted-foreground">
                SL
              </span>
              <span className="mt-0.5 block font-mono text-[12px] font-bold text-rose-600 dark:text-rose-400">
                {latestSignal.stop_loss > 0 ? formatNumber(latestSignal.stop_loss, decimals) : '—'}
              </span>
            </div>
            <div>
              <span className="block text-micro font-semibold uppercase tracking-wider text-muted-foreground">
                TP
              </span>
              <span className="mt-0.5 block font-mono text-[12px] font-bold text-emerald-600 dark:text-emerald-400">
                {latestSignal.take_profit > 0 ? formatNumber(latestSignal.take_profit, decimals) : '—'}
              </span>
            </div>
            <div>
              <span className="block text-micro font-semibold uppercase tracking-wider text-muted-foreground">
                R:R
              </span>
              <span className="mt-0.5 block font-mono text-[12px] font-bold text-foreground">
                {latestSignal.risk_reward > 0 ? `1:${formatNumber(latestSignal.risk_reward, 2)}` : '—'}
              </span>
            </div>
          </div>

          {/* Subline Footer */}
          <div className="flex items-center justify-between border-t border-border/60 pt-1.5 text-xs text-muted-foreground">
            <span className="truncate text-micro">
              {latestSignal.summary ||
                `Tahap 1 HTF (${latestSignal.htf}) Valid · Tahap 2 LTF (${latestSignal.ltf}) Konfirmasi Siap`}
            </span>
            <ArrowRight className="h-3.5 w-3.5 shrink-0 text-muted-foreground ml-1" />
          </div>
        </div>
      ) : (
        <div
          onClick={() => navigate(`/sinyal?pair=${currentPair}`)}
          className="flex flex-col gap-2 rounded-xl border border-border bg-card p-4 transition-colors hover:border-zinc-400 cursor-pointer"
        >
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span className="flex h-7 w-7 items-center justify-center rounded-md bg-secondary text-xs font-bold">
                {currentPair?.slice(0, 3)}
              </span>
              <div>
                <p className="text-xs font-bold text-foreground">{currentPair}</p>
                <p className="text-micro text-muted-foreground">{instrumentName}</p>
              </div>
            </div>
            <span className="rounded bg-secondary px-2 py-0.5 text-micro font-semibold text-muted-foreground">
              STANDBY
            </span>
          </div>
          <p className="text-caption text-muted-foreground">
            Belum ada setup sinyal aktif untuk {currentPair}. Radar engine terus memindai pembentukan struktur FVG & Sweep.
          </p>
          <div className="flex items-center justify-end text-xs font-medium text-foreground">
            <span>Buka Radar Detail</span>
            <ArrowRight className="h-3.5 w-3.5 ml-1" />
          </div>
        </div>
      )}
    </section>
  )
}
