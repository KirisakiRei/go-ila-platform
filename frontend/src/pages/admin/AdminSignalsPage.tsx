import { useMemo } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  Badge,
  Button,
  Card,
  EmptyState,
  Input,
  Label,
  PageHeader,
  Select,
  Skeleton,
} from '@/components/ui'
import { adminSignalApi } from '@/lib/admin-api'
import { formatNumber } from '@/lib/format'
import { Can } from '@/features/admin/components/Can'
import type { AdminSignalItem } from '@/types/admin'

const STATUS_OPTIONS: { value: string; label: string }[] = [
  { value: '', label: 'Semua status' },
  { value: 'WAIT_HTF_SWEEP', label: 'Menunggu sweep HTF' },
  { value: 'WAIT_HTF_CANDLE_CLOSE', label: 'Menunggu candle HTF' },
  { value: 'WAIT_LTF_CISD', label: 'Menunggu CISD LTF' },
  { value: 'READY_TO_ENTRY', label: 'Siap entry' },
  { value: 'INVALIDATED', label: 'Batal (invalidated)' },
  { value: 'ACKED', label: 'Dikonfirmasi user' },
  { value: 'CONSUMED', label: 'Dieksekusi' },
  { value: 'EXPIRED', label: 'Kedaluwarsa' },
  { value: 'CANCELLED', label: 'Dibatalkan' },
  { value: 'SUPERSEDED', label: 'Digantikan sinyal baru' },
  { value: 'REJECTED', label: 'Ditolak user' },
  { value: 'SKIPPED', label: 'Dilewati' },
]

/** statusLabel memakai bahasa manusia; kode asli tetap pada atribut title. */
function statusLabel(status: string): string {
  return STATUS_OPTIONS.find((item) => item.value === status)?.label ?? status
}

/** isoDate mengubah tanggal lokal menjadi YYYY-MM-DD untuk query server. */
function isoDate(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

/** defaultRange adalah tujuh hari terakhir, sesuai bawaan server. */
function defaultRange(): { from: string; to: string } {
  const to = new Date()
  const from = new Date()
  from.setDate(from.getDate() - 7)
  return { from: isoDate(from), to: isoDate(to) }
}

/**
 * AdminSignalsPage menampilkan sinyal yang diterbitkan engine.
 *
 * Rentang tanggal wajib karena tabel sinyal tumbuh terus; filter disimpan di
 * query string agar tautan bisa dibagikan dan tombol kembali bekerja.
 */
export default function AdminSignalsPage() {
  const range = useMemo(() => defaultRange(), [])
  const [params, setParams] = useSearchParams()

  const from = params.get('from') || range.from
  const to = params.get('to') || range.to
  const status = params.get('status') || ''
  const symbol = params.get('symbol') || ''
  const strategy = params.get('strategy') || ''
  const page = Math.max(1, Number(params.get('page') || '1'))

  const update = (patch: Record<string, string>, resetPage = true) => {
    const next = new URLSearchParams(params)
    for (const [key, value] of Object.entries(patch)) {
      if (value) next.set(key, value)
      else next.delete(key)
    }
    if (resetPage) next.delete('page')
    setParams(next, { replace: true })
  }

  const query = useMemo(
    () => ({ page, per_page: 25, from, to, status, symbol, strategy }),
    [page, from, to, status, symbol, strategy],
  )

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['admin', 'signals', query],
    queryFn: () => adminSignalApi.list(query),
  })

  const rows: AdminSignalItem[] = data?.data ?? []
  const meta = data?.meta

  return (
    <>
      <PageHeader
        title="Sinyal"
        description="Pemantauan sinyal yang diterbitkan analisis pasar, status distribusi ke pengguna, dan riwayat konfirmasi order."
      />

      <Card className="p-3">
        <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
          <div>
            <Label htmlFor="signal-from">Dari tanggal (WIB)</Label>
            <Input
              id="signal-from"
              type="date"
              className="mt-1"
              value={from}
              max={to}
              onChange={(event) => update({ from: event.target.value, to })}
            />
          </div>
          <div>
            <Label htmlFor="signal-to">Sampai tanggal (WIB)</Label>
            <Input
              id="signal-to"
              type="date"
              className="mt-1"
              value={to}
              min={from}
              onChange={(event) => update({ from, to: event.target.value })}
            />
          </div>
          <div>
            <Label htmlFor="signal-status">Status</Label>
            <Select
              id="signal-status"
              className="mt-1"
              value={status}
              onChange={(event) => update({ status: event.target.value })}
            >
              {STATUS_OPTIONS.map((item) => (
                <option key={item.value} value={item.value}>
                  {item.label}
                </option>
              ))}
            </Select>
          </div>
          <div>
            <Label htmlFor="signal-symbol">Simbol</Label>
            <Input
              id="signal-symbol"
              className="mt-1"
              placeholder="mis. XAUUSD"
              value={symbol}
              onChange={(event) => update({ symbol: event.target.value.toUpperCase() })}
            />
          </div>
          <div>
            <Label htmlFor="signal-strategy">Versi strategi</Label>
            <Input
              id="signal-strategy"
              className="mt-1"
              placeholder="mis. ila-2.0"
              value={strategy}
              onChange={(event) => update({ strategy: event.target.value })}
            />
          </div>
          <div className="flex items-end">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setParams(new URLSearchParams(), { replace: true })}
            >
              Atur ulang filter
            </Button>
          </div>
        </div>
      </Card>

      {isLoading && (
        <div className="flex flex-col gap-2" aria-busy="true">
          {Array.from({ length: 5 }).map((_, index) => (
            <Skeleton key={index} className="h-12 w-full" />
          ))}
        </div>
      )}

      {isError && (
        <EmptyState
          title="Daftar sinyal gagal dimuat"
          description={error instanceof Error ? error.message : 'Silakan coba lagi.'}
        />
      )}

      {!isLoading && !isError && rows.length === 0 && (
        <EmptyState
          title="Tidak ada sinyal pada rentang ini"
          description="Coba perluas rentang tanggal atau longgarkan filter."
        />
      )}

      {rows.length > 0 && (
        <Card className="overflow-x-auto">
          <table className="w-full text-[13px]">
            <thead className="bg-secondary/40">
              <tr className="border-b border-border">
                <th className="h-9 px-3 text-left text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                  Simbol
                </th>
                <th className="h-9 px-3 text-left text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                  Arah
                </th>
                <th className="h-9 px-3 text-left text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                  Status
                </th>
                <th className="h-9 px-3 text-right text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                  Penerima
                </th>
                <th className="h-9 px-3 text-right text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                  Tereksekusi
                </th>
                <th className="h-9 px-3 text-left text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                  Diterbitkan
                </th>
                <th className="h-9 px-3 text-right text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                  Aksi
                </th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.signal_id} className="border-b border-border last:border-0">
                  <td className="px-3 py-2.5">
                    <div className="flex flex-col">
                      <span className="text-[13px] font-medium">{row.symbol}</span>
                      <span className="text-[11px] text-muted-foreground">
                        {row.strategy_version || '—'} · RR {formatNumber(row.risk_reward)}
                      </span>
                    </div>
                  </td>
                  <td className="px-3 py-2.5 text-[12px]">{row.direction || '—'}</td>
                  <td className="px-3 py-2.5">
                    <Badge variant="muted" title={row.status}>
                      {statusLabel(row.status)}
                    </Badge>
                  </td>
                  <td className="px-3 py-2.5 text-right text-[12px] tabular-nums">
                    {row.recipients}
                  </td>
                  <td className="px-3 py-2.5 text-right text-[12px] tabular-nums">
                    {row.executed}
                  </td>
                  <td className="px-3 py-2.5 text-[12px] text-muted-foreground">
                    {row.published_at?.wib ?? '—'}
                  </td>
                  <td className="px-3 py-2.5 text-right">
                    <Can permission="signal.trace">
                      <Button asChild size="sm" variant="outline">
                        <Link to={`/admin/sinyal/${row.signal_id}`}>Telusuri</Link>
                      </Button>
                    </Can>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>

          <div className="flex flex-wrap items-center justify-between gap-2 border-t border-border px-3 py-2">
            <p className="text-[11px] text-muted-foreground">
              {meta ? `${meta.total} sinyal · halaman ${meta.page}/${meta.total_pages || 1}` : ''}
            </p>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => update({ page: String(page - 1) }, false)}
              >
                Sebelumnya
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={!meta || page >= (meta.total_pages || 1)}
                onClick={() => update({ page: String(page + 1) }, false)}
              >
                Berikutnya
              </Button>
            </div>
          </div>
        </Card>
      )}
    </>
  )
}
