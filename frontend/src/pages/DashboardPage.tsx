import { useSystemStatus, usePositions } from '@/lib/queries'
import { formatDateTime, formatMoney, formatR } from '@/lib/format'
import {
  Alert,
  Badge,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Skeleton,
  StatusDot,
  type StatusState,
} from '@/components/ui'
import type { MarketState } from '@/types/api'

const MARKET_TONE: Record<MarketState, StatusState> = {
  OPEN: 'online',
  CLOSED: 'offline',
  ROLLOVER: 'pending',
  UNKNOWN: 'idle',
}

function DailyCard() {
  const { data, isLoading } = useSystemStatus()
  if (isLoading) return <Skeleton className="h-40 w-full" />
  if (!data) return null

  const { daily } = data

  return (
    <Card>
      <CardHeader>
        <CardTitle>Batasan Harian</CardTitle>
        <CardDescription>
          {daily.day} (WIB) · maksimal {daily.max_trades} trade per hari
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <div className="flex items-center justify-between text-[13px]">
          <span className="text-muted-foreground">Trade hari ini</span>
          <span className="font-semibold">
            {daily.total_trades} / {daily.max_trades}
          </span>
        </div>
        <div className="flex gap-4 text-[13px]">
          <span className="text-success">Menang: {daily.win_count}</span>
          <span className="text-destructive">Kalah: {daily.loss_count}</span>
          <span className="text-muted-foreground">Terbuka: {daily.open_count}</span>
        </div>
        {daily.locked ? (
          <Alert variant="warning">{daily.lock_reason || 'Entry baru ditahan sampai besok.'}</Alert>
        ) : (
          <p className="text-[13px] text-success">Siap menerima setup berikutnya.</p>
        )}
      </CardContent>
    </Card>
  )
}

function SystemCard() {
  const { data, isLoading, isError, error } = useSystemStatus()

  if (isLoading) return <Skeleton className="h-48 w-full" />
  if (isError) return <Alert>{(error as Error).message}</Alert>
  if (!data) return null

  return (
    <Card>
      <CardHeader>
        <CardTitle>Status Sistem</CardTitle>
        <CardDescription>
          Waktu server {formatDateTime(data.server_time)} · hari dihitung {data.daily_timezone}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <ul className="flex flex-col divide-y divide-border">
          <li className="flex items-center gap-3 py-2.5">
            <StatusDot state={MARKET_TONE[data.market.state]} label={data.market.label} />
            <span className="text-[13px]">Pasar</span>
            <span className="ml-auto text-[13px] text-muted-foreground">{data.market.label}</span>
          </li>

          {data.engines.map((engine) => (
            <li key={engine.strategy} className="flex items-center gap-3 py-2.5">
              <StatusDot
                state={engine.connected ? 'online' : 'idle'}
                label={engine.connected ? 'engine aktif' : 'engine tidak terjangkau'}
              />
              <span className="text-[13px]">Engine {engine.strategy}</span>
              <span className="ml-auto text-[13px] text-muted-foreground">
                {engine.connected ? engine.engine_version || 'aktif' : 'tidak terhubung'}
              </span>
            </li>
          ))}

          <li className="flex items-center gap-3 py-2.5">
            <StatusDot
              state={
                data.brokers.any_error ? 'offline' : data.brokers.connected > 0 ? 'online' : 'idle'
              }
              label="akun broker"
            />
            <span className="text-[13px]">Akun broker</span>
            <span className="ml-auto text-[13px] text-muted-foreground">
              {data.brokers.connected} / {data.brokers.total} tersambung
            </span>
          </li>
        </ul>
      </CardContent>
    </Card>
  )
}

function RecentPositions() {
  const { data, isLoading } = usePositions(5)

  if (isLoading) return <Skeleton className="h-32 w-full" />
  if (!data || data.length === 0) {
    return (
      <p className="rounded-lg border border-dashed border-border p-8 text-center text-[13px] text-muted-foreground">
        Belum ada posisi. Notifikasi akan muncul saat engine menemukan setup.
      </p>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Posisi terbaru</CardTitle>
      </CardHeader>
      <CardContent>
        <ul className="flex flex-col divide-y divide-border">
          {data.map((position) => (
            <li key={position.id} className="flex items-center gap-2 py-2.5 text-[13px]">
              <span className="font-medium">{position.symbol}</span>
              <Badge variant={position.direction === 'BUY' ? 'success' : 'destructive'}>
                {position.direction}
              </Badge>
              {position.is_dry_run && <Badge variant="default">Simulasi</Badge>}
              <span
                className={`ml-auto font-medium ${
                  position.profit_loss >= 0 ? 'text-success' : 'text-destructive'
                }`}
              >
                {formatMoney(position.profit_loss)} ({formatR(position.outcome_r)})
              </span>
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  )
}

export default function DashboardPage() {
  return (
    <div className="flex flex-col gap-4">
      <header className="flex flex-col gap-1">
        <h1 className="text-xl font-semibold tracking-tight">Ringkasan</h1>
        <p className="text-[13px] text-muted-foreground">
          Pantau kesiapan sistem, batas harian, dan posisi terakhir.
        </p>
      </header>

      <SystemCard />
      <DailyCard />
      <RecentPositions />
    </div>
  )
}
