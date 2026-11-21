import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { motion, useReducedMotion } from 'framer-motion'
import { ArrowRight, BellRing, TrendingDown, TrendingUp } from 'lucide-react'
import { Button, Card, CardContent, EmptyState, PageHeader, Skeleton } from '@/components/ui'
import { usePositions, useSignals, useSystemStatus } from '@/lib/queries'
import { formatDateTime, formatMoney, formatR } from '@/lib/format'
import { cn } from '@/lib/utils'
import { useAuthContext } from '@/features/auth/hooks/use-auth'
import { PageHeaderActions } from './components/PageHeaderActions'
import { OnboardingCard, OnboardingModal } from '@/features/onboarding'
import { AccountStatusCard } from './components/AccountStatusCard'
import { DashboardGreetingBar } from './components/DashboardGreetingBar'
import { DashboardRadarHero } from './components/DashboardRadarHero'
import { DashboardQuickActions } from './components/DashboardQuickActions'
import { KondisiHariIniRingkas } from './components/KondisiHariIni'
import { DashboardOnboardingWidget } from './components/DashboardOnboardingWidget'
import { MobileMoreDrawer } from '@/components/layout/MobileMoreDrawer'
import { POSISI_LIMIT_DASHBOARD } from '@/features/signals/lib/kondisi-pasar'
import { useInviteOnly } from '@/lib/use-invite-only'
import {
  DashboardEngineStatusPills,
  DashboardKpiStrip,
} from '@/features/dashboard/components/DashboardKpiStrip'
import { PerformanceCurveCard } from '@/features/dashboard/components/PerformanceCurveCard'
import { RadarPriorityCard } from '@/features/dashboard/components/RadarPriorityCard'
import { BrokerStatusCard } from '@/features/dashboard/components/BrokerStatusCard'
import {
  EASE_OUT,
  findActionableSignals,
  namaTimeCycle,
} from '@/features/dashboard/lib/dashboard-metrics'
import type { PositionStatus } from '@/types/api'

export { POSISI_LIMIT_DASHBOARD }

function statusBadge(status: PositionStatus): { label: string; className: string } {
  switch (status) {
    case 'OPEN':
      return {
        label: 'OPEN',
        className: 'bg-sky-500/12 text-sky-700 dark:text-sky-400',
      }
    case 'CLOSED_TP':
      return {
        label: 'TP',
        className: 'bg-emerald-500/12 text-emerald-700 dark:text-emerald-400',
      }
    case 'CLOSED_SL':
      return {
        label: 'SL',
        className: 'bg-rose-500/12 text-rose-700 dark:text-rose-400',
      }
    case 'CLOSED_MANUAL':
      return {
        label: 'MANUAL',
        className: 'bg-secondary text-foreground',
      }
    default:
      return {
        label: 'UNTRACKED',
        className: 'bg-secondary text-muted-foreground',
      }
  }
}

function ActionableSignalBanner() {
  const reduceMotion = useReducedMotion()
  const { data: signals } = useSignals(50)
  const actionable = useMemo(() => findActionableSignals(signals), [signals])
  const top = actionable[0]

  if (!top) return null

  const cycle = namaTimeCycle(top.strategy_version, top.htf, top.ltf)
  const isSell = top.direction === 'SELL'

  return (
    <motion.div
      initial={reduceMotion ? { opacity: 0 } : { opacity: 0, scale: 0.98 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ duration: 0.22, ease: EASE_OUT }}
    >
      <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-amber-500/40 bg-amber-500/[0.08] px-4 py-3">
        <div className="flex min-w-0 items-center gap-3">
          <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-amber-500/20 text-amber-700 dark:text-amber-400">
            <BellRing className="h-4 w-4" />
          </span>
          <div className="min-w-0">
            <p className="text-xs font-bold text-foreground">
              Setup Siap Entry: {top.symbol} {top.direction} ·{' '}
              <span className="font-mono">
                {top.strategy_version === 'ila-2.0'
                  ? `${cycle} (${isSell ? '-CISD' : '+CISD'})`
                  : `ILA 3.0 (${top.htf} → ${top.ltf})`}
              </span>
            </p>
            <p className="truncate text-[11.5px] text-muted-foreground">
              {actionable.length > 1
                ? `${actionable.length} sinyal siap entry aktif sekarang. Buka halaman Signals untuk detail & konfirmasi.`
                : top.summary || 'Konfirmasi LTF CISD terpenuhi dalam Time Cycle aktif.'}
            </p>
          </div>
        </div>
        <Button asChild size="sm" className="h-8 shrink-0 gap-1.5 text-xs active:scale-[0.97]">
          <Link to={`/sinyal?pair=${encodeURIComponent(top.symbol)}`}>
            Buka Sinyal
            <ArrowRight className="h-3.5 w-3.5" />
          </Link>
        </Button>
      </div>
    </motion.div>
  )
}

function RecentPositionsCard() {
  const { data, isLoading } = usePositions(POSISI_LIMIT_DASHBOARD)
  const { data: status } = useSystemStatus()

  const recent = (data ?? []).slice(0, 6)

  return (
    <Card className="flex h-full flex-col border-border/80 bg-card/95">
      <CardContent className="flex flex-1 flex-col gap-3.5 p-5">
        <div className="flex items-center justify-between gap-2">
          <div>
            <h2 className="text-sm font-bold tracking-tight text-foreground">Posisi Terkini</h2>
            <p className="text-[11.5px] text-muted-foreground">
              {status
                ? `Waktu server ${formatDateTime(status.server_time)}`
                : 'Memuat waktu server…'}
            </p>
          </div>
          <Link
            to="/posisi"
            className="inline-flex items-center gap-1 text-xs font-semibold text-foreground transition-colors hover:text-muted-foreground"
          >
            Semua Posisi
            <ArrowRight className="h-3.5 w-3.5" />
          </Link>
        </div>

        {isLoading ? (
          <Skeleton className="h-44 w-full rounded-lg" />
        ) : recent.length === 0 ? (
          <div className="flex flex-1 items-center justify-center">
            <EmptyState
              title="Belum ada posisi trading"
              description="Posisi akan muncul otomatis setelah sinyal dieksekusi pada akun broker Anda."
              action={
                <Button asChild variant="outline" size="sm" className="active:scale-[0.97]">
                  <Link to="/sinyal">Buka Signals</Link>
                </Button>
              }
            />
          </div>
        ) : (
          <ul className="divide-y divide-border/70 rounded-xl border border-border/80">
            {recent.map((position) => {
              const badge = statusBadge(position.status)
              const isWin =
                position.pnl_known && (position.profit_loss > 0 || position.outcome_r > 0)
              const isLoss =
                position.pnl_known && (position.profit_loss < 0 || position.outcome_r < 0)
              const pnlTone = isWin
                ? 'text-emerald-600 dark:text-emerald-400'
                : isLoss
                  ? 'text-rose-600 dark:text-rose-400'
                  : 'text-foreground'

              return (
                <li key={position.id}>
                  <Link
                    to="/posisi"
                    className="flex flex-wrap items-center gap-x-3 gap-y-1.5 px-3.5 py-2.5 text-[12.5px] transition-colors duration-150 hover:bg-secondary/35"
                  >
                    <div className="flex items-center gap-2">
                      <span className="font-semibold text-foreground">{position.symbol}</span>
                      <span
                        className={cn(
                          'inline-flex items-center gap-0.5 rounded px-1.5 py-0.5 font-mono text-[10px] font-bold',
                          position.direction === 'BUY'
                            ? 'bg-emerald-500/12 text-emerald-700 dark:text-emerald-400'
                            : 'bg-rose-500/12 text-rose-700 dark:text-rose-400',
                        )}
                      >
                        {position.direction === 'BUY' ? (
                          <TrendingUp className="h-2.5 w-2.5" />
                        ) : (
                          <TrendingDown className="h-2.5 w-2.5" />
                        )}
                        {position.direction}
                      </span>
                      <span className="rounded bg-secondary px-1.5 py-0.5 font-mono text-[10px] font-medium text-muted-foreground">
                        {position.strategy_version === 'ila-2.0' ? 'ILA 2.0' : 'ILA 3.0'}
                      </span>
                      <span
                        className={cn(
                          'rounded px-1.5 py-0.5 font-mono text-[10px] font-semibold',
                          badge.className,
                        )}
                      >
                        {badge.label}
                      </span>
                      {position.is_dry_run && (
                        <span className="rounded border border-border px-1.5 py-0.5 text-[9.5px] font-semibold uppercase tracking-wide text-muted-foreground">
                          Demo
                        </span>
                      )}
                    </div>

                    <span className="ml-auto flex items-center gap-2 font-mono tabular-nums">
                      {position.pnl_known ? (
                        <>
                          <span className={cn('font-semibold', pnlTone)}>
                            {formatMoney(position.profit_loss)}
                          </span>
                          <span className={cn('text-[11.5px] font-semibold', pnlTone)}>
                            ({formatR(position.outcome_r)})
                          </span>
                        </>
                      ) : (
                        <span
                          className="text-muted-foreground"
                          title="Hasil belum dilaporkan broker"
                        >
                          —
                        </span>
                      )}
                    </span>
                  </Link>
                </li>
              )
            })}
          </ul>
        )}
      </CardContent>
    </Card>
  )
}

export default function DashboardPage() {
  const { user } = useAuthContext()
  const inviteOnly = useInviteOnly()
  const [onboardingOpen, setOnboardingOpen] = useState(false)
  const [moreDrawerOpen, setMoreDrawerOpen] = useState(false)

  return (
    <>
      {/* ============================================================ */}
      {/* 1. TAMPILAN MOBILE (< md) & TABLET PORTRAIT (md..lg)         */}
      {/* ============================================================ */}
      <div className="flex flex-col gap-4 lg:hidden">
        <DashboardGreetingBar />
        <ActionableSignalBanner />
        <div className="flex flex-col gap-4 md:grid md:grid-cols-2 md:items-start">
          <DashboardRadarHero />
          <div className="flex flex-col gap-4">
            <DashboardQuickActions
              onOpenMore={() => setMoreDrawerOpen(true)}
              onOpenOnboarding={() => setOnboardingOpen(true)}
            />
            <KondisiHariIniRingkas />
          </div>
        </div>
        <PerformanceCurveCard />
        <div className="hidden md:grid md:grid-cols-2 md:gap-4">
          <RecentPositionsCard />
          <BrokerStatusCard />
        </div>
        <DashboardOnboardingWidget onOpenOnboarding={() => setOnboardingOpen(true)} />
      </div>

      {/* ============================================================ */}
      {/* 2. TAMPILAN DESKTOP & TABLET LANDSCAPE (>= lg) — Bento       */}
      {/* ============================================================ */}
      <div className="hidden lg:flex lg:flex-col lg:gap-5">
        <PageHeader
          title="Dashboard"
          description="Ringkasan performa trading, radar Time Cycle terdekat entry, posisi terkini, dan status broker."
          actions={
            <div className="flex flex-wrap items-center gap-2.5">
              <DashboardEngineStatusPills />
              <PageHeaderActions />
            </div>
          }
        />

        <AccountStatusCard />

        <OnboardingCard />

        {/* Banner Aksi Cepat — hanya muncul bila ada sinyal READY_TO_ENTRY */}
        <ActionableSignalBanner />

        {/* Baris 1: 4 Kartu KPI */}
        <DashboardKpiStrip />

        {/* Baris 2: Kurva Performa (7) + Radar Terpanas (5) */}
        <div className="grid gap-5 lg:grid-cols-12">
          <div className="lg:col-span-7">
            <PerformanceCurveCard />
          </div>
          <div className="lg:col-span-5">
            <RadarPriorityCard />
          </div>
        </div>

        {/* Baris 3: Posisi Terkini (7) + Akun Broker Aktif (5) */}
        <div className="grid gap-5 lg:grid-cols-12">
          <div className="lg:col-span-7">
            <RecentPositionsCard />
          </div>
          <div className="lg:col-span-5">
            <BrokerStatusCard />
          </div>
        </div>
      </div>

      {/* Modal Onboarding */}
      <OnboardingModal
        forceOpen={onboardingOpen}
        onCloseForce={() => setOnboardingOpen(false)}
      />

      {/* Bottom Sheet Drawer */}
      {user && (
        <MobileMoreDrawer
          open={moreDrawerOpen}
          onClose={() => setMoreDrawerOpen(false)}
          user={user}
          inviteOnly={inviteOnly}
        />
      )}
    </>
  )
}
