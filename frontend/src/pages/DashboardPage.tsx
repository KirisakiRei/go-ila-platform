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
    OPEN: 'Buka',
    CLOSED: 'Tutup',
    ROLLOVER: 'Rollover',
    UNKNOWN: 'Tidak diketahui',
  }

  return (
    <StatGrid>
      <Stat label="Pasar" value={market[data.market.state]} hint={data.market.label} />
      <Stat
        label="Trade hari ini"
        value={`${data.daily.total_trades} / ${data.daily.max_trades}`}
        hint={data.daily.day ? `Tanggal ${data.daily.day} (WIB)` : undefined}
      />
      <Stat
        label="Menang / Kalah"
        value={`${data.daily.win_count} / ${data.daily.loss_count}`}
        hint={`${data.daily.open_count} posisi terbuka`}
      />
      <Stat
        label="Akun broker"
        value={`${data.brokers.connected} / ${data.brokers.total}`}
        hint={data.brokers.any_error ? 'Ada akun bermasalah' : 'Semua tersambung'}
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
        title="Belum ada posisi"
        description="Posisi muncul setelah sinyal dieksekusi pada akun broker Anda."
        action={
          <Button asChild variant="outline" size="sm">
            <Link to="/sinyal">Lihat sinyal</Link>
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
              Simulasi
            </span>
          )}
          <span className="ml-auto tabular font-medium">
            {position.pnl_known ? (
              <>
                {formatMoney(position.profit_loss)}{' '}
                <span className="text-muted-foreground">({formatR(position.outcome_r)})</span>
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
        title="Ringkasan"
        description="Keadaan sistem, batas harian, dan posisi terakhir."
        actions={<PageHeaderActions />}
      />

      <SummaryStats />

      <div className="grid gap-4 lg:grid-cols-2">
        <DailyLimitCard />
        <SystemHealthCard />
      </div>

      <MarketSessionCard />

      <Section
        title="Posisi terbaru"
        description={
          status ? `Waktu server ${formatDateTime(status.server_time)}` : 'Memuat waktu server…'
        }
        actions={
          <Button asChild variant="ghost" size="sm">
            <Link to="/posisi">Semua posisi</Link>
          </Button>
        }
      >
        <RecentPositions />
      </Section>
    </>
  )
}
