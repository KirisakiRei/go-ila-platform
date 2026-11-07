import {
  Badge,
  EmptyState,
  PageHeader,
  Skeleton,
  StatusPill,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Alert,
} from '@/components/ui'
import { usePositions } from '@/lib/queries'
import { formatDateTime, formatMoney, formatNumber, formatR } from '@/lib/format'
import type { PositionStatus } from '@/types/api'

const LABEL: Record<PositionStatus, string> = {
  OPEN: 'Terbuka',
  CLOSED_TP: 'Take profit',
  CLOSED_SL: 'Stop loss',
  CLOSED_MANUAL: 'Ditutup manual',
  UNTRACKED: 'Tanpa catatan',
}

const TONE: Record<PositionStatus, 'solid' | 'outline' | 'neutral' | 'muted'> = {
  OPEN: 'solid',
  CLOSED_TP: 'outline',
  CLOSED_SL: 'outline',
  CLOSED_MANUAL: 'muted',
  UNTRACKED: 'muted',
}

export default function PositionsPage() {
  const { data, isLoading, isError, error } = usePositions()

  return (
    <>
      <PageHeader
        title="Posisi"
        description="Posisi dikelola broker melalui stop loss dan take profit. Penutupan dilakukan di aplikasi broker Anda."
      />

      {isLoading && <Skeleton className="h-56 w-full" />}
      {isError && <Alert variant="destructive">{(error as Error).message}</Alert>}

      {data && data.length === 0 && (
        <EmptyState
          title="Belum ada posisi"
          description="Posisi tercatat setelah sinyal dieksekusi pada akun broker Anda."
        />
      )}

      {data && data.length > 0 && (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Simbol</TableHead>
              <TableHead>Arah</TableHead>
              <TableHead className="text-right">Lot</TableHead>
              <TableHead className="text-right">Buka</TableHead>
              <TableHead className="text-right">SL</TableHead>
              <TableHead className="text-right">TP</TableHead>
              <TableHead className="text-right">Hasil</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="text-right">Dibuka</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.map((position) => (
              <TableRow key={position.id}>
                <TableCell className="whitespace-nowrap font-semibold">
                  <span className="flex items-center gap-1.5">
                    {position.symbol}
                    {position.is_dry_run && <Badge variant="muted">Simulasi</Badge>}
                  </span>
                </TableCell>
                <TableCell className="whitespace-nowrap text-muted-foreground">
                  {position.direction}
                </TableCell>
                <TableCell className="tabular whitespace-nowrap text-right">
                  {formatNumber(position.lot_size, 2)}
                </TableCell>
                <TableCell className="tabular whitespace-nowrap text-right">
                  {formatNumber(position.open_price, 5)}
                </TableCell>
                <TableCell className="tabular whitespace-nowrap text-right text-muted-foreground">
                  {formatNumber(position.stop_loss, 5)}
                </TableCell>
                <TableCell className="tabular whitespace-nowrap text-right text-muted-foreground">
                  {formatNumber(position.take_profit, 5)}
                </TableCell>
                <TableCell className="whitespace-nowrap text-right font-medium">
                  <span className="tabular">{formatMoney(position.profit_loss)}</span>
                  <span className="ml-1 tabular text-muted-foreground">
                    ({formatR(position.outcome_r)})
                  </span>
                </TableCell>
                <TableCell>
                  <StatusPill tone={TONE[position.status]}>{LABEL[position.status]}</StatusPill>
                </TableCell>
                <TableCell className="tabular whitespace-nowrap text-right text-muted-foreground">
                  {formatDateTime(position.opened_at)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </>
  )
}
