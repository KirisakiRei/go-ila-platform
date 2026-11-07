import { usePositions } from '@/lib/queries'
import { formatDateTime, formatMoney, formatNumber, formatR } from '@/lib/format'
import {
  Alert,
  Badge,
  Skeleton,
  StatusPill,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui'
import type { PositionStatus } from '@/types/api'

const STATUS_LABEL: Record<PositionStatus, string> = {
  OPEN: 'Terbuka',
  CLOSED_TP: 'Take Profit',
  CLOSED_SL: 'Stop Loss',
  CLOSED_MANUAL: 'Ditutup manual',
  UNTRACKED: 'Tanpa catatan',
}

const STATUS_TONE: Record<PositionStatus, 'success' | 'danger' | 'muted' | 'info'> = {
  OPEN: 'info',
  CLOSED_TP: 'success',
  CLOSED_SL: 'danger',
  CLOSED_MANUAL: 'muted',
  UNTRACKED: 'muted',
}

export default function PositionsPage() {
  const { data, isLoading, isError, error } = usePositions()

  return (
    <div className="flex flex-col gap-4">
      <header className="flex flex-col gap-1">
        <h1 className="text-xl font-semibold tracking-tight">Posisi</h1>
        <p className="text-[13px] text-muted-foreground">
          Posisi dikelola broker lewat SL/TP. Penutupan dilakukan di aplikasi broker Anda.
        </p>
      </header>

      {isLoading && <Skeleton className="h-48 w-full" />}
      {isError && <Alert>{(error as Error).message}</Alert>}

      {data?.length === 0 && (
        <p className="rounded-lg border border-dashed border-border p-8 text-center text-[13px] text-muted-foreground">
          Belum ada posisi yang tercatat.
        </p>
      )}

      {data && data.length > 0 && (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Simbol</TableHead>
              <TableHead>Arah</TableHead>
              <TableHead>Lot</TableHead>
              <TableHead>Buka</TableHead>
              <TableHead>SL / TP</TableHead>
              <TableHead>Hasil</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Waktu</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.map((position) => (
              <TableRow key={position.id}>
                <TableCell className="font-medium">
                  {position.symbol}
                  {position.is_dry_run && (
                    <Badge variant="default" className="ml-2">
                      Simulasi
                    </Badge>
                  )}
                </TableCell>
                <TableCell>
                  <Badge variant={position.direction === 'BUY' ? 'success' : 'destructive'}>
                    {position.direction}
                  </Badge>
                </TableCell>
                <TableCell>{formatNumber(position.lot_size, 2)}</TableCell>
                <TableCell>{formatNumber(position.open_price, 5)}</TableCell>
                <TableCell className="text-muted-foreground">
                  {formatNumber(position.stop_loss, 5)} / {formatNumber(position.take_profit, 5)}
                </TableCell>
                <TableCell>
                  <span className={position.profit_loss >= 0 ? 'text-success' : 'text-destructive'}>
                    {formatMoney(position.profit_loss)}
                  </span>
                  <span className="ml-1 text-muted-foreground">
                    ({formatR(position.outcome_r)})
                  </span>
                </TableCell>
                <TableCell>
                  <StatusPill tone={STATUS_TONE[position.status]}>
                    {STATUS_LABEL[position.status]}
                  </StatusPill>
                </TableCell>
                <TableCell className="text-muted-foreground">
                  {formatDateTime(position.opened_at)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  )
}
