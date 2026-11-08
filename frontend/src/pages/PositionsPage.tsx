import { useState } from 'react'
import {
  Badge,
  Button,
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
  Card,
  Input,
  Label,
} from '@/components/ui'
import { usePositions } from '@/lib/queries'
import { downloadFile } from '@/lib/http'
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

/**
 * PositionsPage menampilkan posisi dan menyediakan unduhan riwayat.
 * Cetak biru CSV memuat waktu WIB dan NY sekaligus agar tidak ambigu saat
 * dibuka di luar Indonesia.
 */
export default function PositionsPage() {
  const { data, isLoading, isError, error } = usePositions()
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [downloading, setDownloading] = useState(false)
  const [downloadNotice, setDownloadNotice] = useState<string | null>(null)

  const download = async () => {
    setDownloading(true)
    setDownloadNotice(null)
    try {
      const name = await downloadFile('/positions/export', { from, to })
      setDownloadNotice(`Berkas ${name} disiapkan.`)
    } catch (cause) {
      setDownloadNotice(cause instanceof Error ? cause.message : 'Gagal mengunduh riwayat.')
    } finally {
      setDownloading(false)
    }
  }

  return (
    <>
      <PageHeader
        title="Posisi"
        description="Posisi dikelola broker melalui stop loss dan take profit. Penutupan dilakukan di aplikasi broker Anda."
      />

      <Card className="p-3">
        <div className="grid gap-2 sm:grid-cols-3">
          <div className="flex flex-col gap-1">
            <Label htmlFor="export-from">Dari tanggal</Label>
            <Input id="export-from" type="date" value={from} onChange={(event) => setFrom(event.target.value)} />
          </div>
          <div className="flex flex-col gap-1">
            <Label htmlFor="export-to">Sampai tanggal</Label>
            <Input id="export-to" type="date" value={to} onChange={(event) => setTo(event.target.value)} />
          </div>
          <div className="flex items-end">
            <Button variant="outline" size="sm" disabled={downloading} onClick={() => void download()}>
              {downloading ? 'Menyiapkan…' : 'Unduh riwayat (CSV)'}
            </Button>
          </div>
        </div>
        <p className="mt-2 text-[11px] text-muted-foreground">
          Tanggal mengikuti hari akuntansi WIB; di dalam berkas tersedia kolom waktu WIB dan NY.
        </p>
        {downloadNotice && <Alert className="mt-2">{downloadNotice}</Alert>}
      </Card>

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
                  {position.pnl_known ? (
                    <>
                      <span className="tabular">{formatMoney(position.profit_loss)}</span>
                      <span className="ml-1 tabular text-muted-foreground">
                        ({formatR(position.outcome_r)})
                      </span>
                    </>
                  ) : (
                    <span className="text-muted-foreground" title="Hasil belum dilaporkan broker">
                      —
                    </span>
                  )}
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
