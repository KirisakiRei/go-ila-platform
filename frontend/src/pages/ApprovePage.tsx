import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Alert, Badge, Button, PageHeader, Section, Skeleton } from '@/components/ui'
import { approvalApi } from '@/lib/api'
import { queryKeys } from '@/lib/queries'
import { ApiError } from '@/lib/api-error'
import { formatDateTime, formatNumber } from '@/lib/format'

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-0.5">
      <dt className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</dt>
      <dd className="tabular text-[15px] font-semibold">{value}</dd>
    </div>
  )
}

/**
 * Halaman konfirmasi entry dari tautan bot Telegram.
 * Token sekali pakai; halaman ini tetap memerlukan sesi login.
 */
export default function ApprovePage() {
  const { token = '' } = useParams<{ token: string }>()
  const qc = useQueryClient()
  const [hasil, setHasil] = useState<string | null>(null)
  const [galat, setGalat] = useState<string | null>(null)

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['approve', token],
    queryFn: () => approvalApi.get(token),
    enabled: token.length > 0,
    retry: false,
    refetchOnWindowFocus: false,
  })

  const confirm = useMutation({
    mutationFn: () => approvalApi.confirm(token),
    onSuccess: (res) => {
      setHasil(res.message || 'Entry dikonfirmasi.')
      void qc.invalidateQueries({ queryKey: queryKeys.signals(50) })
      void qc.invalidateQueries({ queryKey: queryKeys.positions(100) })
    },
    onError: (err) => setGalat(err instanceof ApiError ? err.message : 'Konfirmasi gagal.'),
  })

  const decline = useMutation({
    mutationFn: () => approvalApi.decline(token),
    onSuccess: (res) => setHasil(res.message || 'Sinyal ditolak.'),
    onError: (err) => setGalat(err instanceof ApiError ? err.message : 'Penolakan gagal.'),
  })

  if (isLoading) return <Skeleton className="h-64 w-full" />

  if (hasil) {
    return (
      <>
        <PageHeader title="Selesai" description={hasil} />
        <Alert variant="success">{hasil}</Alert>
      </>
    )
  }

  if (isError) {
    return (
      <>
        <PageHeader title="Tautan tidak berlaku" />
        <Alert variant="destructive">
          {(error as Error).message || 'Tautan sudah kedaluwarsa atau pernah dipakai.'}
        </Alert>
      </>
    )
  }

  if (!data) return null

  return (
    <>
      <PageHeader
        title="Konfirmasi Entry"
        description="Periksa rincian di bawah sebelum meneruskan."
        actions={
          <div className="flex items-center gap-1.5">
            <Badge variant="solid">{data.direction}</Badge>
            <Badge variant="outline">{data.strategy_version}</Badge>
          </div>
        }
      />

      <Section title="Rincian posisi">
        <div className="flex flex-col gap-4 border border-border p-4">
          <dl className="grid grid-cols-2 gap-x-4 gap-y-4 sm:grid-cols-4">
            <Field label="Simbol" value={data.symbol} />
            <Field label="Harga entry" value={formatNumber(data.entry_price, 5)} />
            <Field label="Stop loss" value={formatNumber(data.stop_loss, 5)} />
            <Field label="Take profit" value={formatNumber(data.take_profit, 5)} />
          </dl>

          <div className="flex flex-wrap items-center gap-x-4 gap-y-1 border-t border-border pt-3 text-[12px] text-muted-foreground">
            <span>Rasio 1 : {formatNumber(data.risk_reward, 2)}</span>
            <span>Berlaku sampai {formatDateTime(data.expires_at)}</span>
          </div>

          {!data.can_confirm && data.blocked_reason && (
            <Alert variant="warning">{data.blocked_reason}</Alert>
          )}
          {galat && <Alert variant="destructive">{galat}</Alert>}

          {data.can_confirm && (
            <div className="flex flex-col gap-2 sm:flex-row">
              <Button
                className="flex-1"
                isLoading={confirm.isPending}
                onClick={() => {
                  setGalat(null)
                  confirm.mutate()
                }}
              >
                Konfirmasi entry
              </Button>
              <Button
                variant="outline"
                className="flex-1"
                isLoading={decline.isPending}
                onClick={() => {
                  setGalat(null)
                  decline.mutate()
                }}
              >
                Tolak
              </Button>
            </div>
          )}
        </div>
      </Section>
    </>
  )
}
