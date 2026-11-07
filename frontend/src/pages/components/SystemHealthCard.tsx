import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Skeleton,
  StatusDot,
  type StatusState,
} from '@/components/ui'
import { useSystemStatus } from '@/lib/queries'
import type { MarketState } from '@/types/api'

const MARKET_STATE: Record<MarketState, StatusState> = {
  OPEN: 'online',
  CLOSED: 'offline',
  ROLLOVER: 'pending',
  UNKNOWN: 'idle',
}

function Row({ state, label, value }: { state: StatusState; label: string; value: string }) {
  return (
    <li className="flex items-center gap-3 py-2.5 text-[13px]">
      <StatusDot state={state} label={`${label}: ${value}`} />
      <span>{label}</span>
      <span className="ml-auto text-right text-muted-foreground">{value}</span>
    </li>
  )
}

/** SystemHealthCard merangkum kesiapan pasar, engine, dan koneksi broker. */
export function SystemHealthCard() {
  const { data, isLoading } = useSystemStatus()

  if (isLoading) return <Skeleton className="h-44 w-full" />
  if (!data) return null

  const brokerState: StatusState = data.brokers.any_error
    ? 'pending'
    : data.brokers.connected > 0
      ? 'online'
      : 'idle'

  return (
    <Card>
      <CardHeader>
        <CardTitle>Kesiapan sistem</CardTitle>
        <CardDescription>Kondisi pasar, engine, dan sesi broker saat ini.</CardDescription>
      </CardHeader>
      <CardContent>
        <ul className="divide-y divide-border">
          <Row state={MARKET_STATE[data.market.state]} label="Pasar" value={data.market.label} />

          {data.engines.map((engine) => (
            <Row
              key={engine.strategy}
              state={engine.connected ? 'online' : 'offline'}
              label={`Engine ${engine.strategy}`}
              value={engine.connected ? engine.engine_version || 'aktif' : 'tidak terhubung'}
            />
          ))}

          {data.engines.length === 0 && (
            <Row state="idle" label="Engine" value="belum dikonfigurasi" />
          )}

          <Row
            state={brokerState}
            label="Akun broker"
            value={`${data.brokers.connected} dari ${data.brokers.total} tersambung`}
          />
        </ul>
      </CardContent>
    </Card>
  )
}
