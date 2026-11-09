import { Link } from 'react-router-dom'
import { Button, PageHeader, Section, Skeleton, Stat, StatGrid } from '@/components/ui'
import { EmptyState } from '@/components/ui'
import { usePositions, useSystemStatus } from '@/lib/queries'
import { formatDateTime, formatMoney, formatR } from '@/lib/format'
import type { MarketState } from '@/types/api'
import { PageHeaderActions } from './components/PageHeaderActions'
import { DailyLimitCard } from './components/DailyLimitCard'
import { SystemHealthCard } from './components/SystemHealthCard'
import { MarketSessionCard } from '@/features/trading/components/MarketSessionCard'
import { OnboardingCard } from '@/features/onboarding'
import { AccountStatusCard } from './components/AccountStatusCard'
import { RadarPantauanCard } from './components/RadarPantauanCard'

function SummaryStats() {
  const { data, isLoading } = useSystemStatus()

  if (isLoading) {
    return (
      <StatGrid>
        {[0, 1, 2, 3].map((i) => (
          <Skeleton key={i} className="h-24 w-full" />
        ))}
      </StatGrid>
    )
  }
  if (!data) return null

  const market: Record<MarketState, string> = {
    OPEN: 'Open',
    CLOSED: 'Closed',
    ROLLOVER: 'Rollover',
    UNKNOWN: 'Standby',
  }

  return (
    <StatGrid>
      <Stat label="Status Pasar" value={market[data.market.state]} hint={data.market.label} />
      <Stat
        label="Trades Hari Ini"
        value={`${data.daily.total_trades} / ${data.daily.max_trades}`}
        hint={data.daily.day ? `Sesi ${data.daily.day} WIB` : undefined}
      />
      <Stat
        label="Win / Loss"
        value={`${data.daily.win_count} / ${data.daily.loss_count}`}
        hint={`${data.daily.open_count} posisi aktif`}
      />
      <Stat
        label="Broker Accounts"
        value={`${data.brokers.connected} / ${data.brokers.total}`}
        hint={data.brokers.any_error ? 'Ada akun perlu sinkronisasi' : 'Semua tersambung'}
      />
    </StatGrid>
  )
}

function RecentPositions() {
  const { data, isLoading } = usePositions(5)

  if (isLoading) return <Skeleton className="h-32 w-full" />

  if (!data || data.length === 0) {
    return (
      <EmptyState
        title="Belum ada posisi trading"
        description="Posisi akan muncul otomatis setelah sinyal dieksekusi pada akun broker Anda."
        action={
          <Button asChild variant="outline" size="sm">
            <Link to="/sinyal">Buka Signals</Link>
          </Button>
        }
      />
    )
  }

  return (
    <ul className="divide-y divide-border border border-border">
      {data.map((position) => (
        <li
          key={position.id}
          className="flex flex-wrap items-center gap-x-3 gap-y-1 px-4 py-3 text-[13px]"
        >
          <span className="font-semibold">{position.symbol}</span>
          <span className="text-muted-foreground">{position.direction}</span>
          {position.is_dry_run && (
            <span className="text-[11px] uppercase tracking-wide text-muted-foreground">
              Demo
            </span>
          )}
          <span className="ml-auto tabular font-medium">
            {position.pnl_known ? (
              <>
                {formatMoney(position.profit_loss)}{' '}
                <span className="text-muted-foreground font-mono">({formatR(position.outcome_r)})</span>
              </>
            ) : (
              // Hasil belum dilaporkan broker: 0 akan terbaca sebagai "impas".
              <span className="text-muted-foreground" title="Hasil belum dilaporkan broker">
                —
              </span>
            )}
          </span>
        </li>
      ))}
    </ul>
  )
}

export default function DashboardPage() {
  const { data: status } = useSystemStatus()

  return (
    <>
      <PageHeader
        title="Dashboard"
        description="Status operasional pasar, ringkasan performa trading harian, dan posisi aktif."
        actions={<PageHeaderActions />}
      />

      <AccountStatusCard />

      <OnboardingCard />

      <SummaryStats />

      {/* Radar Pair Pantauan Real-Time */}
      <RadarPantauanCard />

      <div className="grid gap-4 lg:grid-cols-2">
        <DailyLimitCard />
        <SystemHealthCard />
      </div>

      <MarketSessionCard />

      <Section
        title="Posisi Terkini"
        description={
          status ? `Waktu server ${formatDateTime(status.server_time)}` : 'Memuat waktu server…'
        }
        actions={
          <Button asChild variant="ghost" size="sm">
            <Link to="/posisi">Semua Posisi</Link>
          </Button>
        }
      >
        <RecentPositions />
      </Section>
    </>
  )
}
