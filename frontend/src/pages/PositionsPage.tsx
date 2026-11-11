import { Fragment, useState } from 'react'
import { ChevronDown, ChevronRight, LineChart } from 'lucide-react'
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
import { cn } from '@/lib/utils'
import type { Position, PositionStatus } from '@/types/api'
import { PositionExecutionChart } from '@/features/charts/PositionExecutionChart'
import { chartTheme } from '@/features/charts/lib/theme'

const LABEL: Record<PositionStatus, string> = {
  OPEN: 'Open',
  CLOSED_TP: 'Take Profit',
  CLOSED_SL: 'Stop Loss',
  CLOSED_MANUAL: 'Closed Manual',
  UNTRACKED: 'Untracked',
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
 *
 * Setiap posisi dapat dibuka sebagai accordion berisi chart eksekusi: titik
 * entry, level SL/TP broker, dan garis penghubung ke titik exit.
 */
export default function PositionsPage() {
  const { data, isLoading, isError, error } = usePositions()
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [downloading, setDownloading] = useState(false)
  const [downloadNotice, setDownloadNotice] = useState<string | null>(null)
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const theme = chartTheme()

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

  const toggle = (id: string) => setExpandedId((current) => (current === id ? null : id))

  const pnlColor = (position: Position) =>
    position.profit_loss >= 0 ? theme.profit : theme.loss

  return (
    <>
      <PageHeader
        title="Positions"
        description="Posisi aktif dan riwayat transaksi broker yang dieksekusi dari sinyal ILA."
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
              {downloading ? 'Menyiapkan…' : 'Export CSV'}
            </Button>
          </div>
        </div>
        <p className="mt-2 text-[11px] text-muted-foreground">
          Format waktu ekspor mencakup WIB dan New York (NY).
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

      {/* --- Desktop: tabel dengan accordion chart ------------------------- */}
      {data && data.length > 0 && (
        <div className="hidden lg:block">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-8" />
                <TableHead>Simbol</TableHead>
                <TableHead>Arah</TableHead>
                <TableHead className="text-right">Lot</TableHead>
                <TableHead className="text-right">Open Price</TableHead>
                <TableHead className="text-right">SL</TableHead>
                <TableHead className="text-right">TP</TableHead>
                <TableHead className="text-right">PnL</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Waktu Buka</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.map((position) => {
                const expanded = expandedId === position.id
                return (
                  <Fragment key={position.id}>
                    <TableRow
                      className={cn('cursor-pointer', expanded && 'bg-secondary/40')}
                      onClick={() => toggle(position.id)}
                    >
                      <TableCell className="text-muted-foreground">
                        <button
                          type="button"
                          aria-expanded={expanded}
                          aria-label={expanded ? 'Tutup chart posisi' : 'Buka chart posisi'}
                          className="flex h-6 w-6 items-center justify-center rounded hover:bg-secondary"
                          onClick={(event) => {
                            event.stopPropagation()
                            toggle(position.id)
                          }}
                        >
                          {expanded ? (
                            <ChevronDown className="h-4 w-4" aria-hidden="true" />
                          ) : (
                            <ChevronRight className="h-4 w-4" aria-hidden="true" />
                          )}
                        </button>
                      </TableCell>
                      <TableCell className="whitespace-nowrap font-semibold">
                        <span className="flex items-center gap-1.5">
                          {position.symbol}
                          {position.is_dry_run && <Badge variant="muted">Demo</Badge>}
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

                    {expanded && (
                      <TableRow className="bg-secondary/20 hover:bg-secondary/20">
                        <TableCell colSpan={10} className="p-3">
                          <div className="flex items-center gap-1.5 pb-2 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                            <LineChart className="h-3.5 w-3.5" aria-hidden="true" />
                            Jejak eksekusi
                          </div>
                          <PositionExecutionChart position={position} enabled={expanded} />
                        </TableCell>
                      </TableRow>
                    )}
                  </Fragment>
                )
              })}
            </TableBody>
          </Table>
        </div>
      )}

      {/* --- Mobile: kartu dengan accordion chart -------------------------- */}
      {data && data.length > 0 && (
        <div className="space-y-2 lg:hidden">
          {data.map((position) => {
            const expanded = expandedId === position.id
            return (
              <div key={position.id} className="overflow-hidden rounded-xl border border-border bg-card">
                <button
                  type="button"
                  onClick={() => toggle(position.id)}
                  aria-expanded={expanded}
                  className="flex w-full flex-col gap-2 p-3 text-left"
                >
                  <div className="flex w-full items-center justify-between gap-2">
                    <div className="flex items-center gap-2">
                      <span className="font-semibold">{position.symbol}</span>
                      <Badge variant="outline" className="text-[10px]">
                        {position.direction}
                      </Badge>
                      {position.is_dry_run && <Badge variant="muted">Demo</Badge>}
                    </div>
                    <div className="flex items-center gap-2">
                      <StatusPill tone={TONE[position.status]}>{LABEL[position.status]}</StatusPill>
                      {expanded ? (
                        <ChevronDown className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
                      ) : (
                        <ChevronRight className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
                      )}
                    </div>
                  </div>

                  <div className="grid grid-cols-3 gap-2 text-[11px]">
                    <div>
                      <p className="text-[10px] uppercase text-muted-foreground">Entry</p>
                      <p className="tabular font-semibold">{formatNumber(position.open_price, 5)}</p>
                    </div>
                    <div>
                      <p className="text-[10px] uppercase text-muted-foreground">SL / TP</p>
                      <p className="tabular text-muted-foreground">
                        {formatNumber(position.stop_loss, 2)} / {formatNumber(position.take_profit, 2)}
                      </p>
                    </div>
                    <div className="text-right">
                      <p className="text-[10px] uppercase text-muted-foreground">Hasil</p>
                      {position.pnl_known ? (
                        <p className="tabular font-semibold" style={{ color: pnlColor(position) }}>
                          {formatMoney(position.profit_loss)} ({formatR(position.outcome_r)})
                        </p>
                      ) : (
                        <p className="text-muted-foreground">—</p>
                      )}
                    </div>
                  </div>

                  <p className="text-[10.5px] text-muted-foreground">
                    Buka {formatDateTime(position.opened_at)}
                    {position.closed_at ? ` · Tutup ${formatDateTime(position.closed_at)}` : ''}
                  </p>
                </button>

                {expanded && (
                  <div className="border-t border-border p-3">
                    <PositionExecutionChart position={position} enabled={expanded} height={280} />
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}
    </>
  )
}
