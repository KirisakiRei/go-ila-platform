import { Link, useParams } from 'react-router-dom'
import type { ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  ArrowLeft,
  Compass,
  Crosshair,
  Layers,
  Target,
  Zap,
} from 'lucide-react'
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  EmptyState,
  PageHeader,
  Skeleton,
  Stat,
  StatGrid,
} from '@/components/ui'
import { adminSignalApi } from '@/lib/admin-api'
import { formatNumber } from '@/lib/format'

type AnalysisPoint = {
  id: string
  title: string
  detail: string
  iconType: 'sweep' | 'trigger' | 'target' | 'context'
}

function parseAnalysisSummary(rawSummary: string, strategyVersion?: string): AnalysisPoint[] {
  if (!rawSummary || !rawSummary.trim()) return []

  const cleaned = rawSummary.replace(/^💎\s*SETUP[^\n:]+:\s*/gi, '').trim()
  const rawParts = cleaned
    .split(/(?:•|\n)+/)
    .map((p) => p.trim())
    .filter(Boolean)

  const points: AnalysisPoint[] = []

  for (let i = 0; i < rawParts.length; i++) {
    const part = rawParts[i]
    if (!part) continue

    if (/^level:\s*entry=/i.test(part) || /^entry=/i.test(part)) {
      continue
    }

    const cleanDetail = part.replace(/^[•\s\-_]+/, '').replace(/^💎\s*/, '')
    if (!cleanDetail) continue

    if (strategyVersion === 'ila-3.0') {
      if (/htf|macro|q1/i.test(cleanDetail)) {
        points.push({
          id: `htf-${i}`,
          title: 'HTF Macro Bias & Key Level',
          detail: cleanDetail,
          iconType: 'sweep',
        })
      } else if (/mtf|reversal|retrace|internal|smt/i.test(cleanDetail)) {
        points.push({
          id: `mtf-${i}`,
          title: 'MTF Structure & Alignment',
          detail: cleanDetail,
          iconType: 'trigger',
        })
      } else if (/ltf|cisd|entry\s*1|entry\s*2|trigger|break/i.test(cleanDetail)) {
        points.push({
          id: `ltf-${i}`,
          title: 'LTF Trigger (CISD)',
          detail: cleanDetail,
          iconType: 'trigger',
        })
      } else if (/stdev|target|tp\s*1|dol|proyeksi|fibo/i.test(cleanDetail)) {
        points.push({
          id: `target-${i}`,
          title: 'Target & DOL (Draw on Liquidity)',
          detail: cleanDetail,
          iconType: 'target',
        })
      } else {
        points.push({
          id: `point-${i}`,
          title: 'Market Structure Context',
          detail: cleanDetail,
          iconType: 'context',
        })
      }
    } else {
      if (/sweep|fvg|masuk ke|menyapu|liquidity|erl|irl|htf/i.test(cleanDetail)) {
        points.push({
          id: `sweep-${i}`,
          title: 'HTF Liquidity Sweep',
          detail: cleanDetail,
          iconType: 'sweep',
        })
      } else if (/cisd|trigger|break|tembus|valid|konfirmasi|reversal|ltf/i.test(cleanDetail)) {
        points.push({
          id: `trigger-${i}`,
          title: 'LTF Trigger (CISD)',
          detail: cleanDetail,
          iconType: 'trigger',
        })
      } else if (/stdev|target|tp\s*1|dol|proyeksi|fibo/i.test(cleanDetail)) {
        points.push({
          id: `target-${i}`,
          title: 'Target & DOL (Draw on Liquidity)',
          detail: cleanDetail,
          iconType: 'target',
        })
      } else {
        points.push({
          id: `point-${i}`,
          title: 'Market Structure Context',
          detail: cleanDetail,
          iconType: 'context',
        })
      }
    }
  }

  return points
}

/**
 * AdminSignalTracePage menampilkan perjalanan satu sinyal: siapa menerima,
 * apa yang terjadi pada statusnya, akun mana yang mengeksekusi, dan posisi
 * yang lahir. Membuka halaman ini tercatat di jejak audit.
 */
export default function AdminSignalTracePage() {
  const { id = '' } = useParams()

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['admin', 'signals', 'trace', id],
    queryFn: () => adminSignalApi.trace(id),
    enabled: Boolean(id),
    // Telusur tercatat di audit, jadi jangan menarik ulang otomatis.
    staleTime: 60_000,
    refetchOnWindowFocus: false,
  })

  return (
    <>
      <PageHeader
        title="Trace Sinyal"
        description="Penerima, perubahan status, klaim eksekusi, dan posisi yang lahir dari sinyal ini."
        actions={
          <Button asChild variant="outline" size="sm">
            <Link to="/admin/sinyal">
              <ArrowLeft className="h-4 w-4" aria-hidden="true" />
              Kembali ke daftar
            </Link>
          </Button>
        }
      />

      {isLoading && (
        <div className="flex flex-col gap-2" aria-busy="true">
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className="h-20 w-full" />
          ))}
        </div>
      )}

      {isError && (
        <EmptyState
          title="Perjalanan sinyal gagal dimuat"
          description={error instanceof Error ? error.message : 'Sinyal mungkin tidak ditemukan.'}
        />
      )}

      {data && (
        <>
          <StatGrid>
            <Stat label="Simbol" value={data.signal.symbol || '—'} />
            <Stat label="Status" value={data.signal.status || '—'} />
            <Stat label="Penerima" value={String(data.signal.recipients)} />
            <Stat label="Tereksekusi" value={String(data.signal.executed)} />
          </StatGrid>

          {/* Kotak Parameter Eksekusi Utama */}
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            <Card className="p-3.5 border-l-4 border-l-primary">
              <span className="text-[10.5px] font-semibold uppercase tracking-wider text-muted-foreground">
                Entry Price
              </span>
              <p className="mt-1 text-lg font-bold tracking-tight text-foreground">
                {formatNumber(data.signal.entry_price)}
              </p>
            </Card>

            <Card className="p-3.5 border-l-4 border-l-rose-500">
              <span className="text-[10.5px] font-semibold uppercase tracking-wider text-muted-foreground">
                Stop Loss
              </span>
              <p className="mt-1 text-lg font-bold tracking-tight text-rose-600 dark:text-rose-400">
                {formatNumber(data.signal.stop_loss)}
              </p>
            </Card>

            <Card className="p-3.5 border-l-4 border-l-emerald-500">
              <span className="text-[10.5px] font-semibold uppercase tracking-wider text-muted-foreground">
                Take Profit
              </span>
              <p className="mt-1 text-lg font-bold tracking-tight text-emerald-600 dark:text-emerald-400">
                {formatNumber(data.signal.take_profit)}
              </p>
            </Card>

            <Card className="p-3.5 border-l-4 border-l-amber-500">
              <span className="text-[10.5px] font-semibold uppercase tracking-wider text-muted-foreground">
                Risk : Reward
              </span>
              <p className="mt-1 text-lg font-bold tracking-tight text-foreground">
                1 : {formatNumber(data.signal.risk_reward)}
              </p>
            </Card>
          </div>

          {/* Rincian Analisis Terstruktur */}
          <Card>
            <CardHeader className="pb-3 border-b border-border/60">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Compass className="h-4 w-4 text-primary" />
                  <CardTitle className="text-sm">Rincian Analisis Struktur Pasar</CardTitle>
                </div>
                <Badge variant="outline" className="text-[11px]">
                  {data.signal.strategy_version ? data.signal.strategy_version.toUpperCase() : 'ILA 2.0'}
                </Badge>
              </div>
              <CardDescription className="text-xs">
                Uraian liquidity sweep, konfirmasi CISD, dan target DOL dari engine analisis.
              </CardDescription>
            </CardHeader>
            <CardContent className="pt-4 space-y-4">
              {(() => {
                const points = parseAnalysisSummary(data.signal.summary || '', data.signal.strategy_version)
                if (points.length === 0) {
                  return (
                    <p className="text-[13px] leading-relaxed text-muted-foreground">
                      {data.signal.summary || 'Tanpa ringkasan analisis khusus.'}
                    </p>
                  )
                }

                return (
                  <div className="flex flex-col gap-2.5">
                    {points.map((pt) => {
                      const icon =
                        pt.iconType === 'sweep' ? (
                          <Crosshair className="h-4 w-4 shrink-0 text-amber-500 mt-0.5" />
                        ) : pt.iconType === 'trigger' ? (
                          <Zap className="h-4 w-4 shrink-0 text-emerald-500 mt-0.5" />
                        ) : pt.iconType === 'target' ? (
                          <Target className="h-4 w-4 shrink-0 text-blue-500 mt-0.5" />
                        ) : (
                          <Layers className="h-4 w-4 shrink-0 text-primary mt-0.5" />
                        )

                      return (
                        <div
                          key={pt.id}
                          className="flex items-start gap-3 rounded-lg border border-border/60 bg-muted/20 p-3"
                        >
                          {icon}
                          <div className="flex flex-col gap-0.5">
                            <span className="text-[11.5px] font-semibold text-foreground">
                              {pt.title}
                            </span>
                            <span className="text-[12.5px] leading-relaxed text-muted-foreground">
                              {pt.detail}
                            </span>
                          </div>
                        </div>
                      )
                    })}
                  </div>
                )
              })()}

              <dl className="grid gap-2 border-t border-border/60 pt-3 text-[12px] sm:grid-cols-2 lg:grid-cols-4">
                <Field label="Arah Posisi" value={data.signal.direction || '—'} />
                <Field label="Engine Version" value={data.signal.engine_version || '—'} />
                <Field label="Diterbitkan (WIB)" value={data.signal.published_at?.wib ?? '—'} />
                <Field label="Kedaluwarsa (WIB)" value={data.signal.expires_at?.wib ?? '—'} />
              </dl>

              {data.signal.cancel_reason && (
                <div className="rounded-lg border border-destructive/40 bg-destructive/10 p-2.5 text-[12px] text-destructive">
                  <strong>Alasan Pembatalan:</strong> {data.signal.cancel_reason}
                </div>
              )}
            </CardContent>
          </Card>

          <TraceTable
            title="Distribusi Pengguna"
            description="Status distribusi dan eksekusi sinyal pada setiap pengguna."
            empty="Belum ada pengguna yang menerima sinyal ini."
            head={['Pengguna', 'Akun Broker', 'Status', 'Alasan / Posisi', 'Waktu']}
            rows={data.recipients.map((row) => [
              <span key="u" className="flex flex-col">
                <span className="font-medium">{row.user_name || row.user_id}</span>
                <span className="text-[11px] text-muted-foreground">{row.user_email}</span>
              </span>,
              row.broker_account_id || '—',
              <Badge key="s" variant="muted">
                {row.status}
              </Badge>,
              row.reject_reason || row.position_id || '—',
              row.decided_at?.wib ?? row.created_at?.wib ?? '—',
            ])}
          />

          <TraceTable
            title="Perubahan status"
            description="Jejak transisi state machine per pengguna."
            empty="Belum ada transisi status tercatat."
            head={['Pengguna', 'Dari', 'Ke', 'Alasan', 'Waktu']}
            rows={data.events.map((row) => [
              row.user_id,
              row.from_status || '—',
              row.to_status,
              row.reason || '—',
              row.created_at?.wib ?? '—',
            ])}
          />

          <TraceTable
            title="Klaim eksekusi"
            description="Klaim eksekusi per akun broker. Satu sinyal bisa diklaim beberapa akun."
            empty="Belum ada klaim eksekusi."
            head={['Pengguna', 'Akun broker', 'Status', 'Waktu']}
            rows={data.executions.map((row) => [
              row.user_id,
              row.broker_account_id,
              row.status,
              row.created_at?.wib ?? '—',
            ])}
          />

          <TraceTable
            title="Posisi"
            description="Posisi yang lahir dari sinyal ini."
            empty="Belum ada posisi terbentuk."
            head={['Simbol', 'Arah', 'Status', 'Hasil', 'R', 'Dibuka']}
            rows={data.positions.map((row) => [
              row.symbol,
              row.direction,
              row.status,
              formatNumber(row.profit_loss),
              formatNumber(row.outcome_r),
              row.opened_at?.wib ?? '—',
            ])}
          />
        </>
      )}
    </>
  )
}

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col">
      <dt className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
        {label}
      </dt>
      <dd className="text-[12px]">{value}</dd>
    </div>
  )
}

function TraceTable({
  title,
  description,
  empty,
  head,
  rows,
}: {
  title: string
  description: string
  empty: string
  head: string[]
  rows: ReactNode[][]
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">{title}</CardTitle>
        <CardDescription className="text-xs">{description}</CardDescription>
      </CardHeader>
      <CardContent>
        {rows.length === 0 ? (
          <p className="text-[12px] text-muted-foreground">{empty}</p>
        ) : (
          <div className="w-full overflow-x-auto rounded-lg border border-border">
            <table className="w-full text-[13px]">
              <thead className="bg-secondary/40">
                <tr className="border-b border-border">
                  {head.map((cell) => (
                    <th
                      key={cell}
                      className="h-9 px-3 text-left text-[11px] font-semibold uppercase tracking-wider text-muted-foreground"
                    >
                      {cell}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {rows.map((row, rowIndex) => (
                  <tr key={rowIndex} className="border-b border-border last:border-0">
                    {row.map((cell, cellIndex) => (
                      <td key={cellIndex} className="px-3 py-2.5 text-[12px]">
                        {cell}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
