import { Link } from 'react-router-dom'
import { motion, useReducedMotion } from 'framer-motion'
import { Activity, ArrowUpRight, Gauge, Target, TrendingDown, TrendingUp } from 'lucide-react'
import { Card, CardContent, Skeleton, StatusDot } from '@/components/ui'
import { usePositions, useSystemStatus } from '@/lib/queries'
import { publicContentApi } from '@/lib/system-api'
import { useQuery } from '@tanstack/react-query'
import { formatMoney, formatR } from '@/lib/format'
import { cn } from '@/lib/utils'
import {
  POSISI_LIMIT_DASHBOARD,
  jamTutupMenyusul,
  kondisiPasar,
  tanggalWIB,
} from '@/features/signals/lib/kondisi-pasar'
import { EASE_OUT, buildEquityCurve } from '../lib/dashboard-metrics'

export function DashboardKpiStrip() {
  const reduceMotion = useReducedMotion()
  const { data: status, isLoading: loadingStatus } = useSystemStatus()
  const { data: positions, isLoading: loadingPos } = usePositions(POSISI_LIMIT_DASHBOARD)
  const { data: publik } = useQuery({
    queryKey: ['public', 'status'],
    queryFn: publicContentApi.status,
  })

  if (loadingStatus && loadingPos) {
    return (
      <div className="grid grid-cols-2 gap-2.5 sm:gap-4 xl:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-[96px] w-full rounded-xl sm:h-[116px]" />
        ))}
      </div>
    )
  }

  const daily = status?.daily
  const pasar = kondisiPasar(status?.market, publik?.market)
  const curve30d = buildEquityCurve(positions, '30d')

  const closedHariIni = daily
    ? (positions ?? []).filter((p) => p.closed_at && tanggalWIB(p.closed_at) === daily.day)
    : []
  const plHariIni = closedHariIni.reduce((sum, p) => sum + (p.profit_loss ?? 0), 0)
  const rHariIni = closedHariIni.reduce((sum, p) => sum + (p.outcome_r ?? 0), 0)
  const adaClosedHariIni = closedHariIni.length > 0

  const openPositions = (positions ?? []).filter((p) => p.status === 'OPEN')
  const openCount = daily?.open_count ?? openPositions.length
  const totalTrades = daily?.total_trades ?? 0
  const maxTrades = Math.max(1, daily?.max_trades ?? 3)
  const kuotaRatio = Math.min(1, totalTrades / maxTrades)
  const sisaKuota = Math.max(0, maxTrades - totalTrades)

  const jadwalBerikut = publik?.market?.next_change_pair?.wib
    ? `${jamTutupMenyusul(publik.market) ? 'Tutup' : publik.market.next_label} ${publik.market.next_change_pair.wib} WIB`
    : null

  const cardMotion = (index: number) =>
    reduceMotion
      ? { initial: { opacity: 0 }, animate: { opacity: 1 }, transition: { duration: 0.16 } }
      : {
          initial: { opacity: 0, y: 8 },
          animate: { opacity: 1, y: 0 },
          transition: { duration: 0.24, delay: index * 0.045, ease: EASE_OUT },
        }

  const primaryR = adaClosedHariIni ? rHariIni : curve30d.totalR
  const primaryPnL = adaClosedHariIni ? plHariIni : curve30d.totalPnL
  const hasPrimaryValue = adaClosedHariIni || curve30d.count > 0
  const primaryTone = !hasPrimaryValue
    ? 'text-muted-foreground'
    : primaryR > 0 || primaryPnL > 0
      ? 'text-emerald-600 dark:text-emerald-400'
      : primaryR < 0 || primaryPnL < 0
        ? 'text-rose-600 dark:text-rose-400'
        : 'text-foreground'

  return (
    <div className="grid grid-cols-2 gap-2.5 sm:gap-4 xl:grid-cols-4">
      {/* 1. P/L & R */}
      <motion.div {...cardMotion(0)}>
        <Card className="h-full border-border/80 bg-card/95 transition-colors duration-150 hover:border-foreground/25">
          <CardContent className="flex h-full min-w-0 flex-col justify-between p-3 sm:p-4">
            <div className="flex items-center justify-between gap-1.5">
              <span className="truncate text-[10.5px] font-semibold uppercase tracking-wider text-muted-foreground sm:text-[11px]">
                {adaClosedHariIni ? 'P/L Hari Ini' : 'Performa 30H'}
              </span>
              {primaryR < 0 ? (
                <TrendingDown className="h-3.5 w-3.5 shrink-0 text-rose-500 sm:h-4 sm:w-4" />
              ) : (
                <TrendingUp className="h-3.5 w-3.5 shrink-0 text-emerald-500 sm:h-4 sm:w-4" />
              )}
            </div>
            <div className="mt-1.5 sm:mt-2">
              <p className={cn('truncate font-mono text-lg font-bold tabular-nums tracking-tight sm:text-xl', primaryTone)}>
                {hasPrimaryValue ? formatR(primaryR) : '0,00R'}
              </p>
              <p className="mt-0.5 flex items-center gap-1 truncate text-[11px] text-muted-foreground sm:text-xs">
                <span className={cn('truncate font-mono font-semibold tabular-nums', primaryTone)}>
                  {hasPrimaryValue ? formatMoney(primaryPnL) : 'Belum ada tutup'}
                </span>
                {adaClosedHariIni && curve30d.count > 0 && (
                  <span className="hidden sm:inline">· 30H {formatR(curve30d.totalR)}</span>
                )}
              </p>
            </div>
          </CardContent>
        </Card>
      </motion.div>

      {/* 2. Win Rate 30H */}
      <motion.div {...cardMotion(1)}>
        <Card className="h-full border-border/80 bg-card/95 transition-colors duration-150 hover:border-foreground/25">
          <CardContent className="flex h-full min-w-0 flex-col justify-between p-3 sm:p-4">
            <div className="flex items-center justify-between gap-1.5">
              <span className="truncate text-[10.5px] font-semibold uppercase tracking-wider text-muted-foreground sm:text-[11px]">
                Win Rate (30H)
              </span>
              <Target className="h-3.5 w-3.5 shrink-0 text-muted-foreground sm:h-4 sm:w-4" />
            </div>
            <div className="mt-1.5 sm:mt-2">
              <div className="flex items-baseline gap-1.5 sm:gap-2">
                <p className="font-mono text-lg font-bold tabular-nums tracking-tight text-foreground sm:text-xl">
                  {curve30d.winRate !== null ? `${curve30d.winRate}%` : '—'}
                </p>
                <span className="truncate font-mono text-[11px] text-muted-foreground tabular-nums sm:text-xs">
                  {curve30d.wins}W/{curve30d.losses}L
                </span>
              </div>
              <div className="mt-1.5 h-1.5 w-full overflow-hidden rounded-full bg-secondary sm:mt-2">
                <motion.div
                  className="h-full origin-left rounded-full bg-emerald-500"
                  initial={reduceMotion ? false : { scaleX: 0 }}
                  animate={{ scaleX: (curve30d.winRate ?? 0) / 100 }}
                  transition={{ duration: 0.45, ease: EASE_OUT }}
                />
              </div>
            </div>
          </CardContent>
        </Card>
      </motion.div>

      {/* 3. Posisi Aktif */}
      <motion.div {...cardMotion(2)}>
        <Link to="/posisi" className="block h-full transition-transform duration-150 ease-out active:scale-[0.98]">
          <Card className="h-full border-border/80 bg-card/95 transition-colors duration-150 hover:border-foreground/30">
            <CardContent className="flex h-full min-w-0 flex-col justify-between p-3 sm:p-4">
              <div className="flex items-center justify-between gap-1.5">
                <span className="truncate text-[10.5px] font-semibold uppercase tracking-wider text-muted-foreground sm:text-[11px]">
                  Posisi Aktif
                </span>
                <ArrowUpRight className="h-3.5 w-3.5 shrink-0 text-muted-foreground sm:h-4 sm:w-4" />
              </div>
              <div className="mt-1.5 sm:mt-2">
                <div className="flex items-baseline gap-1.5 sm:gap-2">
                  <p className="font-mono text-lg font-bold tabular-nums tracking-tight text-foreground sm:text-xl">
                    {openCount}
                  </p>
                  <span className="truncate text-[11px] text-muted-foreground sm:text-xs">
                    {openCount > 0 ? 'berjalan' : 'kosong'}
                  </span>
                </div>
                <p className="mt-0.5 truncate text-[11px] text-muted-foreground sm:text-xs">
                  {openPositions.length > 0
                    ? openPositions
                        .slice(0, 2)
                        .map((p) => `${p.symbol} ${p.direction}`)
                        .join(' · ')
                    : 'Pantau LTF CISD'}
                </p>
              </div>
            </CardContent>
          </Card>
        </Link>
      </motion.div>

      {/* 4. Kuota & Kondisi Pasar */}
      <motion.div {...cardMotion(3)}>
        <Card className="h-full border-border/80 bg-card/95 transition-colors duration-150 hover:border-foreground/25">
          <CardContent className="flex h-full min-w-0 flex-col justify-between p-3 sm:p-4">
            <div className="flex items-center justify-between gap-1.5">
              <div className="flex min-w-0 items-center gap-1.5">
                <StatusDot
                  state={
                    pasar.state === 'OPEN'
                      ? 'online'
                      : pasar.state === 'ROLLOVER'
                        ? 'pending'
                        : 'offline'
                  }
                  label={pasar.label}
                />
                <span className="truncate text-[10.5px] font-semibold uppercase tracking-wider text-foreground sm:text-[11px]">
                  {pasar.label}
                </span>
              </div>
              <Gauge className="h-3.5 w-3.5 shrink-0 text-muted-foreground sm:h-4 sm:w-4" />
            </div>

            <div className="mt-1.5 sm:mt-2">
              <div className="flex items-baseline justify-between gap-1.5">
                <span className="text-[11px] text-muted-foreground sm:text-xs">Kuota</span>
                <span className="font-mono text-xs font-bold tabular-nums text-foreground sm:text-sm">
                  {totalTrades}/{maxTrades}
                </span>
              </div>
              <div className="mt-1.5 h-1.5 w-full overflow-hidden rounded-full bg-secondary">
                <motion.div
                  className={cn(
                    'h-full origin-left rounded-full',
                    daily?.locked
                      ? 'bg-rose-500'
                      : sisaKuota > 0
                        ? 'bg-foreground'
                        : 'bg-amber-500',
                  )}
                  initial={reduceMotion ? false : { scaleX: 0 }}
                  animate={{ scaleX: kuotaRatio }}
                  transition={{ duration: 0.45, ease: EASE_OUT }}
                />
              </div>
              <p className="mt-1 flex items-center justify-between gap-1 truncate text-[10.5px] text-muted-foreground sm:mt-1.5 sm:text-[11px]">
                <span className="truncate">
                  {daily?.locked
                    ? daily.lock_reason || 'Entry ditahan'
                    : jadwalBerikut ?? `Sisa ${sisaKuota} slot`}
                </span>
                {publik?.time_pair?.ny && (
                  <span className="hidden shrink-0 font-mono text-[10.5px] sm:inline">
                    NY {publik.time_pair.ny}
                  </span>
                )}
              </p>
            </div>
          </CardContent>
        </Card>
      </motion.div>
    </div>
  )
}

/** Sub-strip ringkas di atas untuk koneksi Engine ILA 2.0 & 3.0. */
export function DashboardEngineStatusPills() {
  const { data: status } = useSystemStatus()
  if (!status || status.engines.length === 0) return null

  return (
    <div className="flex flex-wrap items-center gap-2">
      {status.engines.map((engine) => (
        <span
          key={engine.strategy}
          className="inline-flex items-center gap-1.5 rounded-full border border-border bg-card px-2.5 py-1 text-[11px] font-medium text-muted-foreground"
        >
          <StatusDot
            state={engine.connected ? 'online' : 'offline'}
            label={`${engine.strategy} ${engine.connected ? 'aktif' : 'nonaktif'}`}
          />
          <Activity className="h-3 w-3 opacity-60" />
          <span className="font-mono text-foreground">
            {engine.strategy === 'ila-2.0' ? 'ILA 2.0' : 'ILA 3.0'}
          </span>
          {engine.engine_version && (
            <span className="font-mono text-[10px] opacity-70">v{engine.engine_version}</span>
          )}
        </span>
      ))}
    </div>
  )
}
