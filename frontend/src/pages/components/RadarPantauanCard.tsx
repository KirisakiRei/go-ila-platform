import { useMemo } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import {
  ArrowRight,
  CheckCircle2,
  Clock,
  Layers,
  TrendingDown,
  TrendingUp,
} from 'lucide-react'
import {
  Badge,
  Button,
  Card,
  EmptyState,
  Section,
  Skeleton,
  StatusPill,
} from '@/components/ui'
import { useSignals, useSubscriptions } from '@/lib/queries'
import { formatDateTime } from '@/lib/format'
import { cn } from '@/lib/utils'
import { SIGNAL_LABEL, SIGNAL_TONE } from '@/features/signals/lib/status'
import { TimeframeBadge } from '@/features/signals/components/TimeframeBadge'

export function RadarPantauanCard() {
  const navigate = useNavigate()
  const { data: subscriptions, isLoading: loadingSubs } = useSubscriptions()
  const { data: signals, isLoading: loadingSignals } = useSignals(100)

  // Ambil daftar unik simbol yang dipantau aktif oleh user
  const monitoredSymbols = useMemo(() => {
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

  // Pasangkan tiap simbol pantauan dengan sinyal terakhirnya
  const items = useMemo(() => {
    return monitoredSymbols.map((sym) => {
      const latest = (signals ?? []).find((s) => s.symbol.toUpperCase() === sym) ?? null
      return { symbol: sym, signal: latest }
    })
  }, [monitoredSymbols, signals])

  const isLoading = loadingSubs || loadingSignals

  if (isLoading) {
    return (
      <Section
        title="Radar Watchlist Pair"
        description="Pemindaian struktur pasar real-time untuk instrumen pilihan Anda."
      >
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-32 w-full rounded-xl" />
          ))}
        </div>
      </Section>
    )
  }

  if (monitoredSymbols.length === 0) {
    return (
      <Section
        title="Radar Watchlist Pair"
        description="Pemindaian struktur pasar real-time untuk instrumen pilihan Anda."
      >
        <EmptyState
          title="Watchlist masih kosong"
          description="Tambahkan instrumen ke Watchlist Pair untuk memantau sinyal dan setup pasar secara real-time."
          action={
            <Button asChild variant="outline" size="sm">
              <Link to="/strategi">Kelola Watchlist</Link>
            </Button>
          }
        />
      </Section>
    )
  }

  return (
    <Section
      title="Radar Watchlist Pair"
      description="Pemindaian struktur pasar real-time. Klik pair untuk membuka analisa lengkap."
      actions={
        <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
          <span className="relative flex h-2 w-2">
            <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75" />
            <span className="relative inline-flex h-2 w-2 rounded-full bg-emerald-500" />
          </span>
          <span className="font-mono text-[11px]">Live radar</span>
        </div>
      }
    >
      <div className="grid gap-3.5 sm:grid-cols-2 lg:grid-cols-3">
        {items.map(({ symbol, signal }) => {
          const isReady = signal?.status === 'READY_TO_ENTRY' || signal?.status === 'OPPORTUNITY'
          const isWaitLtf = signal?.status === 'WAIT_LTF_CISD'
          const isHtfDone = isWaitLtf || isReady

          return (
            <Card
              key={symbol}
              onClick={() => navigate(`/sinyal?pair=${symbol}`)}
              className={cn(
                'group relative cursor-pointer overflow-hidden border border-border bg-card p-4 transition-all duration-150 hover:-translate-y-0.5 hover:border-foreground/30 hover:shadow-md',
                isReady && 'border-emerald-500/40 bg-emerald-500/5',
                isWaitLtf && 'border-amber-500/40 bg-amber-500/5',
              )}
            >
              {/* Header Kartu: Simbol & Status */}
              <div className="flex items-center justify-between gap-2">
                <div className="flex items-center gap-2">
                  <h4 className="text-base font-bold tracking-tight text-foreground">{symbol}</h4>
                  {signal?.direction && signal.direction !== 'NEUTRAL' && (
                    <span
                      className={cn(
                        'inline-flex items-center gap-0.5 font-mono text-xs font-bold',
                        signal.direction === 'BUY'
                          ? 'text-emerald-600 dark:text-emerald-400'
                          : 'text-rose-600 dark:text-rose-400',
                      )}
                    >
                      {signal.direction === 'BUY' ? (
                        <TrendingUp className="h-3.5 w-3.5" />
                      ) : (
                        <TrendingDown className="h-3.5 w-3.5" />
                      )}
                      {signal.direction}
                    </span>
                  )}
                </div>

                {signal ? (
                  <StatusPill tone={SIGNAL_TONE[signal.status]} className="text-[10px]">
                    {SIGNAL_LABEL[signal.status]}
                  </StatusPill>
                ) : (
                  <Badge variant="outline" className="text-[10px]">
                    Memantau
                  </Badge>
                )}
              </div>

              {/* Timeframe & Strategy */}
              <div className="mt-2.5 flex items-center justify-between text-xs text-muted-foreground">
                <div className="flex items-center gap-1.5">
                  <Badge variant="outline" className="font-mono text-[9.5px]">
                    {signal?.strategy_version || 'ila-2.0'}
                  </Badge>
                  {signal && <TimeframeBadge htf={signal.htf} ltf={signal.ltf} />}
                </div>
                <ArrowRight className="h-3.5 w-3.5 text-muted-foreground transition-transform group-hover:translate-x-1" />
              </div>

              {/* Radar Step Mini Progress Bar */}
              <div className="mt-3.5 rounded-lg border border-border/60 bg-secondary/25 p-2.5 text-[11.5px]">
                <div className="flex items-center justify-between text-[10.5px] font-medium text-muted-foreground">
                  <span className="flex items-center gap-1">
                    {isHtfDone ? (
                      <CheckCircle2 className="h-3 w-3 text-emerald-500" />
                    ) : (
                      <Clock className="h-3 w-3 text-muted-foreground" />
                    )}
                    HTF: {isHtfDone ? 'Sweep Closed' : 'Memantau'}
                  </span>
                  <span>➔</span>
                  <span className="flex items-center gap-1">
                    {isReady ? (
                      <CheckCircle2 className="h-3 w-3 text-emerald-500" />
                    ) : isWaitLtf ? (
                      <Clock className="h-3 w-3 text-amber-500 animate-pulse" />
                    ) : (
                      <Layers className="h-3 w-3 text-muted-foreground" />
                    )}
                    LTF: {isReady ? 'Siap' : isWaitLtf ? 'Menunggu CISD' : 'Standby'}
                  </span>
                </div>
              </div>

              {/* Update Footer */}
              <div className="mt-3 flex items-center justify-between text-[10.5px] text-muted-foreground">
                <span>
                  {signal
                    ? `Update ${formatDateTime(signal.updated_at ?? signal.published_at)}`
                    : 'Menunggu data candle'}
                </span>
                <span className="font-medium text-foreground/80 group-hover:underline">Buka Setup</span>
              </div>
            </Card>
          )
        })}
      </div>
    </Section>
  )
}
