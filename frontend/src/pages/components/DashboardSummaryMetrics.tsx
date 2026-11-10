import { useSystemStatus } from '@/lib/queries'
import type { MarketState } from '@/types/api'

const MARKET_LABEL: Record<MarketState, string> = {
  OPEN: 'Open',
  CLOSED: 'Closed',
  ROLLOVER: 'Rollover',
  UNKNOWN: 'Standby',
}

export function DashboardSummaryMetrics() {
  const { data } = useSystemStatus()

  const marketState = data?.market?.state ? MARKET_LABEL[data.market.state] : 'Standby'
  const sessionDay = data?.daily?.day ? `Sesi ${data.daily.day}` : data?.market?.label || 'Sesi Aktif'

  const totalTrades = data?.daily?.total_trades ?? 0
  const maxTrades = data?.daily?.max_trades ?? 2

  const winCount = data?.daily?.win_count ?? 0
  const lossCount = data?.daily?.loss_count ?? 0
  const openCount = data?.daily?.open_count ?? 0

  const connectedBrokers = data?.brokers?.connected ?? 0
  const totalBrokers = data?.brokers?.total ?? 0
  const anyBrokerError = Boolean(data?.brokers?.any_error)

  return (
    <section className="flex flex-col gap-2">
      <h2 className="text-base font-bold tracking-tight text-foreground">Ringkasan Trading Hari Ini</h2>

      <div className="grid grid-cols-2 gap-2.5">
        {/* Tile 1: Status Pasar */}
        <div className="flex flex-col justify-between rounded-xl border border-border bg-card p-3 shadow-xs">
          <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
            Status Pasar
          </span>
          <div className="my-1">
            <span className="font-mono text-xl font-bold tracking-tight text-foreground sm:text-2xl">
              {marketState}
            </span>
          </div>
          <span className="truncate text-[11px] text-muted-foreground">{sessionDay}</span>
        </div>

        {/* Tile 2: Trades Hari Ini */}
        <div className="flex flex-col justify-between rounded-xl border border-border bg-card p-3 shadow-xs">
          <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
            Trades Hari Ini
          </span>
          <div className="my-1">
            <span className="font-mono text-xl font-bold tracking-tight text-foreground sm:text-2xl">
              {totalTrades} / {maxTrades}
            </span>
          </div>
          <span className="truncate text-[11px] text-muted-foreground">
            Maksimal {maxTrades} eksekusi/hari
          </span>
        </div>

        {/* Tile 3: Win / Loss */}
        <div className="flex flex-col justify-between rounded-xl border border-border bg-card p-3 shadow-xs">
          <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
            Win / Loss
          </span>
          <div className="my-1">
            <span className="font-mono text-xl font-bold tracking-tight text-foreground sm:text-2xl">
              {winCount} / {lossCount}
            </span>
          </div>
          <span className="truncate text-[11px] text-muted-foreground">{openCount} posisi aktif</span>
        </div>

        {/* Tile 4: Akun Broker */}
        <div className="flex flex-col justify-between rounded-xl border border-border bg-card p-3 shadow-xs">
          <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
            Akun Broker
          </span>
          <div className="my-1">
            <span className="font-mono text-xl font-bold tracking-tight text-foreground sm:text-2xl">
              {connectedBrokers} / {totalBrokers}
            </span>
          </div>
          <span
            className={`truncate text-[11px] font-medium ${
              anyBrokerError
                ? 'text-rose-600 dark:text-rose-400'
                : 'text-emerald-600 dark:text-emerald-400'
            }`}
          >
            {anyBrokerError ? 'Perlu sinkronisasi' : 'Semua tersambung'}
          </span>
        </div>
      </div>
    </section>
  )
}
