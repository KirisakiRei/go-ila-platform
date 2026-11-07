import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { approvalApi } from '@/lib/api'
import { queryKeys } from '@/lib/queries'
import { ApiError } from '@/lib/api-error'
import { formatDateTime, formatNumber } from '@/lib/format'
import {
  Alert,
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Skeleton,
} from '@/components/ui'

/**
 * Halaman konfirmasi entry dari tautan bot Telegram.
 * Token dipakai sekali; setelah konfirmasi hasil akhir ditampilkan permanen.
 */
export default function ApprovePage() {
  const { token = '' } = useParams<{ token: string }>()
  const queryClient = useQueryClient()
  const [result, setResult] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)

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
      setResult(res.message || 'Entry dikonfirmasi.')
      void queryClient.invalidateQueries({ queryKey: queryKeys.signals(50) })
      void queryClient.invalidateQueries({ queryKey: queryKeys.positions(100) })
    },
    onError: (err) => {
      setActionError(err instanceof ApiError ? err.message : 'Konfirmasi gagal.')
    },
  })

  const decline = useMutation({
    mutationFn: () => approvalApi.decline(token),
    onSuccess: (res) => setResult(res.message || 'Sinyal ditolak.'),
    onError: (err) => {
      setActionError(err instanceof ApiError ? err.message : 'Penolakan gagal.')
    },
  })

  if (isLoading) return <Skeleton className="h-64 w-full" />

  if (result) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Selesai</CardTitle>
          <CardDescription>{result}</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  if (isError) {
    return (
      <Alert>{(error as Error).message || 'Tautan tidak berlaku atau sudah kedaluwarsa.'}</Alert>
    )
  }

  if (!data) return null

  return (
    <Card className="max-w-xl">
      <CardHeader>
        <div className="flex items-center gap-2">
          <CardTitle>Konfirmasi Entry</CardTitle>
          <Badge variant={data.direction === 'BUY' ? 'success' : 'destructive'}>
            {data.direction}
          </Badge>
        </div>
        <CardDescription>
          {data.symbol} · {data.strategy_version}
        </CardDescription>
      </CardHeader>

      <CardContent className="flex flex-col gap-4">
        <dl className="grid grid-cols-2 gap-x-4 gap-y-3 text-[13px]">
          <div>
            <dt className="text-muted-foreground">Harga entry</dt>
            <dd className="text-base font-semibold">{formatNumber(data.entry_price, 5)}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Rasio R:R</dt>
            <dd className="text-base font-semibold">1 : {formatNumber(data.risk_reward, 2)}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Stop Loss</dt>
            <dd className="font-medium">{formatNumber(data.stop_loss, 5)}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Take Profit</dt>
            <dd className="font-medium">{formatNumber(data.take_profit, 5)}</dd>
          </div>
        </dl>

        <p className="text-[11px] text-muted-foreground">
          Berlaku sampai {formatDateTime(data.expires_at)}
        </p>

        {!data.can_confirm && data.blocked_reason && (
          <Alert variant="warning">{data.blocked_reason}</Alert>
        )}
        {actionError && <Alert>{actionError}</Alert>}

        {data.can_confirm && (
          <div className="flex gap-2">
            <Button
              className="flex-1"
              isLoading={confirm.isPending}
              onClick={() => {
                setActionError(null)
                confirm.mutate()
              }}
            >
              Konfirmasi Entry
            </Button>
            <Button
              variant="outline"
              className="flex-1"
              isLoading={decline.isPending}
              onClick={() => {
                setActionError(null)
                decline.mutate()
              }}
            >
              Tolak
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
