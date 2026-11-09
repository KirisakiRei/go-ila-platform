import { Link, useParams } from 'react-router-dom'
import type { ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ArrowLeft } from 'lucide-react'
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
        title="Perjalanan sinyal"
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

          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Ringkasan sinyal</CardTitle>
              <CardDescription className="text-xs">
                {data.signal.summary || 'Tanpa ringkasan dari engine.'}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <dl className="grid gap-2 text-[12px] sm:grid-cols-2 lg:grid-cols-4">
                <Field label="Arah" value={data.signal.direction || '—'} />
                <Field label="Entry" value={formatNumber(data.signal.entry_price)} />
                <Field label="Stop loss" value={formatNumber(data.signal.stop_loss)} />
                <Field label="Take profit" value={formatNumber(data.signal.take_profit)} />
                <Field label="Risk/reward" value={`1 : ${formatNumber(data.signal.risk_reward)}`} />
                <Field label="Strategi" value={data.signal.strategy_version || '—'} />
                <Field label="Diterbitkan" value={data.signal.published_at?.wib ?? '—'} />
                <Field label="Kedaluwarsa" value={data.signal.expires_at?.wib ?? '—'} />
              </dl>
              {data.signal.cancel_reason && (
                <p className="mt-3 text-[12px] text-muted-foreground">
                  Alasan pembatalan: {data.signal.cancel_reason}
                </p>
              )}
            </CardContent>
          </Card>

          <TraceTable
            title="Penerima"
            description="Nasib sinyal pada setiap pelanggan."
            empty="Belum ada pelanggan yang menerima sinyal ini."
            head={['Pelanggan', 'Akun', 'Status', 'Alasan / posisi', 'Waktu']}
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
