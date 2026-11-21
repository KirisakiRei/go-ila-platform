import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import { motion, useReducedMotion } from 'framer-motion'
import { ArrowRight, Radar, TrendingDown, TrendingUp } from 'lucide-react'
import { Card, CardContent, Skeleton } from '@/components/ui'
import { useSignals, useSubscriptions } from '@/lib/queries'
import { cn } from '@/lib/utils'
import { EASE_OUT, buildRadarHighlights } from '../lib/dashboard-metrics'

export function RadarPriorityCard() {
  const reduceMotion = useReducedMotion()
  const { data: subscriptions, isLoading: loadingSub } = useSubscriptions()
  const { data: signals, isLoading: loadingSig } = useSignals(50)

  const activeSymbols = useMemo(() => {
    const seen = new Set<string>()
    for (const sub of subscriptions ?? []) {
      if (sub.symbol && sub.is_active) seen.add(sub.symbol.toUpperCase())
    }
    return Array.from(seen).sort()
  }, [subscriptions])

  const highlights = useMemo(
    () => buildRadarHighlights(signals, activeSymbols, 4),
    [signals, activeSymbols],
  )

  return (
    <Card className="flex h-full flex-col border-border/80 bg-card/95">
      <CardContent className="flex flex-1 flex-col gap-3 p-3.5 sm:gap-3.5 sm:p-5">
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <h2 className="text-sm font-bold tracking-tight text-foreground">Radar Terpanas</h2>
            <span className="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10.5px] font-semibold text-emerald-700 dark:text-emerald-400">
              <span className="relative flex h-1.5 w-1.5">
                <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75" />
                <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-emerald-500" />
              </span>
              Live
            </span>
          </div>
          <Link
            to="/sinyal"
            className="inline-flex items-center gap-1 text-xs font-semibold text-foreground transition-colors hover:text-muted-foreground"
          >
            Semua Radar
            <ArrowRight className="h-3.5 w-3.5" />
          </Link>
        </div>

        {loadingSub && loadingSig ? (
          <div className="space-y-2.5">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-[64px] w-full rounded-lg" />
            ))}
          </div>
        ) : highlights.length === 0 ? (
          <div className="flex flex-1 flex-col items-center justify-center rounded-xl border border-dashed border-border bg-secondary/20 px-4 py-8 text-center">
            <Radar className="h-6 w-6 text-muted-foreground" />
            <p className="mt-2.5 text-xs font-semibold text-foreground">
              Belum ada pair di watchlist
            </p>
            <p className="mt-1 text-[11.5px] text-muted-foreground">
              Tambahkan pair untuk mulai memantau Time Cycle &amp; LTF CISD.
            </p>
            <Link
              to="/strategi"
              className="mt-3 inline-flex items-center gap-1 text-xs font-semibold text-foreground hover:underline"
            >
              Atur Watchlist Pair
              <ArrowRight className="h-3.5 w-3.5" />
            </Link>
          </div>
        ) : (
          <div className="flex flex-1 flex-col gap-2">
            {highlights.map((item, idx) => {
              const isSell = item.direction === 'SELL'
              const isBuy = item.direction === 'BUY'
              return (
                <motion.div
                  key={`${item.symbol}-${item.strategy}-${item.tfPair}`}
                  initial={reduceMotion ? { opacity: 0 } : { opacity: 0, y: 6 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{
                    duration: reduceMotion ? 0.14 : 0.22,
                    delay: reduceMotion ? 0 : idx * 0.045,
                    ease: EASE_OUT,
                  }}
                >
                  <Link
                    to={`/sinyal?pair=${encodeURIComponent(item.symbol)}`}
                    className={cn(
                      'group flex items-center justify-between gap-2.5 rounded-xl border p-2.5 transition-colors duration-150 active:scale-[0.98] sm:gap-3 sm:p-3',
                      item.ready
                        ? 'border-emerald-500/40 bg-emerald-500/[0.06] hover:border-emerald-500/70'
                        : 'border-border/80 bg-secondary/20 hover:border-foreground/30 hover:bg-secondary/40',
                    )}
                  >
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-1.5">
                        <span className="text-xs font-bold tracking-tight text-foreground">
                          {item.symbol}
                        </span>
                        <span className="rounded bg-secondary px-1.5 py-0.5 font-mono text-[10px] font-semibold text-muted-foreground">
                          {item.strategy === 'ila-2.0' ? 'ILA 2.0' : 'ILA 3.0'}
                        </span>
                        {item.direction && (
                          <span
                            className={cn(
                              'inline-flex items-center gap-0.5 rounded px-1.5 py-0.5 font-mono text-[10px] font-bold',
                              isBuy
                                ? 'bg-emerald-500/12 text-emerald-700 dark:text-emerald-400'
                                : isSell
                                  ? 'bg-rose-500/12 text-rose-700 dark:text-rose-400'
                                  : 'bg-secondary text-muted-foreground',
                            )}
                          >
                            {isBuy ? (
                              <TrendingUp className="h-2.5 w-2.5" />
                            ) : (
                              <TrendingDown className="h-2.5 w-2.5" />
                            )}
                            {item.strategy === 'ila-2.0'
                              ? isBuy
                                ? '+CISD'
                                : '-CISD'
                              : item.direction}
                          </span>
                        )}
                      </div>
                      <p className="mt-1 truncate text-[11px] text-muted-foreground">
                        {item.cycleName} · <span className="font-mono">{item.tfPair}</span>
                      </p>
                    </div>

                    <div className="flex shrink-0 flex-col items-end gap-1.5">
                      <span
                        className={cn(
                          'rounded-md px-2 py-0.5 text-[10.5px] font-semibold',
                          item.ready
                            ? 'bg-emerald-600 text-white'
                            : item.stepCount >= 2
                              ? 'bg-amber-500/15 text-amber-700 dark:text-amber-400'
                              : 'bg-secondary text-muted-foreground',
                        )}
                      >
                        {item.stageLabel}
                      </span>

                      {/* 3-step progress mini (HTF C2 -> LTF CISD -> Entry) */}
                      <div className="flex items-center gap-1" aria-hidden="true">
                        {[1, 2, 3].map((step) => (
                          <span
                            key={step}
                            className={cn(
                              'h-1 w-4 rounded-full transition-colors',
                              step <= item.stepCount
                                ? item.ready
                                  ? 'bg-emerald-500'
                                  : 'bg-foreground'
                                : 'bg-border',
                            )}
                          />
                        ))}
                      </div>
                    </div>
                  </Link>
                </motion.div>
              )
            })}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
