import { useMemo } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  ArrowDownRight,
  ArrowRight,
  ArrowUpRight,
  Layers,
  Minus,
  Users,
  Zap,
} from 'lucide-react'
import {
  Button,
  Card,
  EmptyState,
  Input,
  Label,
  PageHeader,
  Select,
  Skeleton,
  StatusPill,
} from '@/components/ui'
import { adminSignalApi } from '@/lib/admin-api'
import { formatNumber } from '@/lib/format'
import { SIGNAL_LABEL, SIGNAL_TONE } from '@/features/signals/lib/status'
import type { SignalStatus } from '@/types/api'
import { Can } from '@/features/admin/components/Can'
import { cn } from '@/lib/utils'
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



function directionMeta(direction: string | undefined) {
  const dir = (direction ?? '').toUpperCase()
  if (dir === 'BULLISH' || dir === 'BUY' || dir === 'LONG') {
    return {
      label: 'Bullish',
      icon: ArrowUpRight,
      borderClass: 'border-emerald-300 dark:border-emerald-800',
      bgClass: 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-400',
      badgeClass: 'bg-emerald-600 text-white',
    }
  }
  if (dir === 'BEARISH' || dir === 'SELL' || dir === 'SHORT') {
    return {
      label: 'Bearish',
      icon: ArrowDownRight,
      borderClass: 'border-rose-300 dark:border-rose-800',
      bgClass: 'bg-rose-500/10 text-rose-700 dark:text-rose-400',
      badgeClass: 'bg-rose-600 text-white',
    }
  }
  return {
    label: dir || 'Netral',
    icon: Minus,
    borderClass: 'border-border',
    bgClass: 'bg-muted text-muted-foreground',
    badgeClass: 'bg-secondary text-secondary-foreground',
  }
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
        <div className="space-y-4">
          {/* Card Grid Daftar Pair / Sinyal */}
          <div className="grid gap-3.5 sm:grid-cols-2 lg:grid-cols-3">
            {rows.map((row) => {
              const dir = directionMeta(row.direction)
              const DirIcon = dir.icon

              return (
                <Card
                  key={row.signal_id}
                  className="flex flex-col justify-between overflow-hidden border transition-all hover:border-foreground/30 hover:shadow-sm"
                >
                  <div className="p-4">
                    {/* Header: Simbol, Arah, dan Status */}
                    <div className="flex items-start justify-between gap-2">
                      <div className="flex items-center gap-2">
                        <span className="text-[17px] font-bold tracking-tight text-foreground">
                          {row.symbol}
                        </span>
                        <span
                          className={cn(
                            'inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-[11px] font-semibold',
                            dir.bgClass,
                          )}
                        >
                          <DirIcon className="h-3.5 w-3.5" aria-hidden="true" />
                          {dir.label}
                        </span>
                      </div>

                      <span title={row.status}>
                        <StatusPill tone={SIGNAL_TONE[row.status as SignalStatus] ?? 'neutral'}>
                          {SIGNAL_LABEL[row.status as SignalStatus] ?? row.status}
                        </StatusPill>
                      </span>
                    </div>

                    {/* Subheader: Engine & Versi Strategi */}
                    <div className="mt-1 flex items-center gap-1.5 text-[11px] text-muted-foreground">
                      <Layers className="h-3 w-3 shrink-0" />
                      <span>{row.strategy_version ? row.strategy_version.toUpperCase() : 'ILA 2.0'}</span>
                      <span>·</span>
                      <span>RR {formatNumber(row.risk_reward)}</span>
                    </div>

                    {/* Parameter Harga (Entry, SL, TP) */}
                    <div className="mt-3.5 grid grid-cols-3 gap-2 rounded-lg border border-border/60 bg-muted/30 p-2.5 text-[11.5px]">
                      <div>
                        <span className="block text-[10px] uppercase tracking-wider text-muted-foreground">
                          Entry
                        </span>
                        <span className="font-semibold text-foreground">
                          {formatNumber(row.entry_price)}
                        </span>
                      </div>
                      <div>
                        <span className="block text-[10px] uppercase tracking-wider text-muted-foreground">
                          Stop Loss
                        </span>
                        <span className="font-semibold text-rose-600 dark:text-rose-400">
                          {formatNumber(row.stop_loss)}
                        </span>
                      </div>
                      <div>
                        <span className="block text-[10px] uppercase tracking-wider text-muted-foreground">
                          Take Profit
                        </span>
                        <span className="font-semibold text-emerald-600 dark:text-emerald-400">
                          {formatNumber(row.take_profit)}
                        </span>
                      </div>
                    </div>

                    {/* Ringkasan Analisis / Summary jika ada */}
                    {row.summary && (
                      <p className="mt-3 line-clamp-2 text-[12px] leading-relaxed text-muted-foreground">
                        {row.summary.replace(/^💎\s*SETUP[^\n:]+:\s*/gi, '')}
                      </p>
                    )}
                  </div>

                  {/* Kaki Card: Statistik Distribusi & Tombol Telusuri */}
                  <div className="flex items-center justify-between border-t border-border/80 bg-muted/20 px-4 py-2.5">
                    <div className="flex items-center gap-3 text-[11.5px] text-muted-foreground">
                      <span className="flex items-center gap-1" title="User penerima sinyal">
                        <Users className="h-3.5 w-3.5" />
                        <strong className="text-foreground">{row.recipients}</strong> user
                      </span>
                      <span>·</span>
                      <span className="flex items-center gap-1" title="Akun tereksekusi">
                        <Zap className="h-3.5 w-3.5 text-amber-500" />
                        <strong className="text-foreground">{row.executed}</strong> order
                      </span>
                    </div>

                    <Can permission="signal.trace">
                      <Button asChild size="sm" variant="outline" className="h-7 gap-1 px-2.5 text-[11.5px]">
                        <Link to={`/admin/sinyal/${row.signal_id}`}>
                          Detail &amp; Trace
                          <ArrowRight className="h-3 w-3" />
                        </Link>
                      </Button>
                    </Can>
                  </div>
                </Card>
              )
            })}
          </div>

          {/* Baris Pagination */}
          <Card className="flex flex-wrap items-center justify-between gap-2 p-3">
            <p className="text-[12px] text-muted-foreground">
              {meta ? `${meta.total} sinyal · halaman ${meta.page} dari ${meta.total_pages || 1}` : ''}
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
          </Card>
        </div>
      )}
    </>
  )
}
