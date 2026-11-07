import type { Signal, SignalStatus } from '@/types/api'
import { formatDateTime, formatNumber } from '@/lib/format'
import {
  Badge,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  StatusPill,
} from '@/components/ui'

const STATUS_LABEL: Record<SignalStatus, string> = {
  READY_TO_ENTRY: 'Siap entry',
  ACKED: 'Diterima',
  REJECTED: 'Tertahan',
  EXPIRED: 'Kedaluwarsa',
  CANCELLED: 'Dibatalkan',
  SUPERSEDED: 'Digantikan',
  CONSUMED: 'Sudah dieksekusi',
  SKIPPED: 'Dilewati',
}

const STATUS_TONE: Record<SignalStatus, 'success' | 'warning' | 'danger' | 'muted' | 'info'> = {
  READY_TO_ENTRY: 'success',
  ACKED: 'info',
  REJECTED: 'warning',
  EXPIRED: 'muted',
  CANCELLED: 'muted',
  SUPERSEDED: 'muted',
  CONSUMED: 'success',
  SKIPPED: 'muted',
}

/** Satu sinyal: harga, alasan, dan status akhir yang bisa dibaca manusia. */
export function SignalCard({ signal }: { signal: Signal }) {
  return (
    <Card>
      <CardHeader className="flex-row items-start justify-between gap-3">
        <div className="flex flex-col gap-1">
          <div className="flex items-center gap-2">
            <CardTitle>{signal.symbol}</CardTitle>
            <Badge variant={signal.direction === 'BUY' ? 'success' : 'destructive'}>
              {signal.direction}
            </Badge>
            {signal.delayed && <Badge variant="default">Telat 15 menit</Badge>}
          </div>
          <CardDescription>
            {signal.strategy_version} · {formatDateTime(signal.published_at)}
          </CardDescription>
        </div>
        <StatusPill tone={STATUS_TONE[signal.status]}>{STATUS_LABEL[signal.status]}</StatusPill>
      </CardHeader>

      <CardContent className="flex flex-col gap-3">
        <dl className="grid grid-cols-2 gap-x-4 gap-y-2 text-[13px] sm:grid-cols-4">
          <div>
            <dt className="text-muted-foreground">Entry</dt>
            <dd className="font-medium">{formatNumber(signal.entry_price, 5)}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Stop Loss</dt>
            <dd className="font-medium">{formatNumber(signal.stop_loss, 5)}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Take Profit</dt>
            <dd className="font-medium">{formatNumber(signal.take_profit, 5)}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">R:R</dt>
            <dd className="font-medium">1 : {formatNumber(signal.risk_reward, 2)}</dd>
          </div>
        </dl>

        {signal.summary && <p className="text-[13px] text-muted-foreground">{signal.summary}</p>}

        {signal.reject_reason && (
          <p className="text-[13px] text-warning">Ditahan: {signal.reject_reason}</p>
        )}

        {signal.next_action && (
          <p className="text-[13px] font-medium">Langkah berikutnya: {signal.next_action}</p>
        )}

        <p className="text-[11px] text-muted-foreground">
          Kedaluwarsa {formatDateTime(signal.expires_at)}
        </p>
      </CardContent>
    </Card>
  )
}
