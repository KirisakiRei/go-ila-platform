import { useState } from 'react'
import { AccountForm } from '@/features/accounts/components/AccountForm'
import { useAccounts, useDeleteAccount, useUpdateAccount } from '@/lib/queries'
import { useAuthContext } from '@/features/auth/hooks/use-auth'
import { formatDateTime, formatMoney, formatNumber } from '@/lib/format'
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
  StatusDot,
} from '@/components/ui'
import type { BrokerAccount } from '@/types/api'

const ACCOUNT_LIMIT: Record<string, number> = { free: 0, basic: 1, plus: 2 }

function AccountItem({ account }: { account: BrokerAccount }) {
  const update = useUpdateAccount()
  const remove = useDeleteAccount()
  const [confirming, setConfirming] = useState(false)

  return (
    <Card>
      <CardHeader className="flex-row items-start justify-between gap-3">
        <div className="flex flex-col gap-1">
          <div className="flex items-center gap-2">
            <CardTitle>{account.label}</CardTitle>
            <StatusDot
              state={account.connected ? 'online' : account.last_error ? 'offline' : 'idle'}
              label={account.connected ? 'tersambung' : 'tidak tersambung'}
            />
          </div>
          <CardDescription>
            {account.broker_type} · {account.external_account_id || 'nomor belum terbaca'}
          </CardDescription>
        </div>
        <div className="flex flex-wrap items-center justify-end gap-1.5">
          <Badge variant={account.is_dry_run ? 'default' : 'primary'}>
            {account.is_dry_run ? 'Simulasi' : 'Nyata'}
          </Badge>
          <Badge variant={account.mode === 'auto' ? 'accent' : 'default'}>
            {account.mode === 'auto' ? 'Otomatis' : 'Manual'}
          </Badge>
          {!account.is_active && <Badge variant="warning">Nonaktif</Badge>}
        </div>
      </CardHeader>

      <CardContent className="flex flex-col gap-3">
        <dl className="grid grid-cols-2 gap-x-4 gap-y-2 text-[13px] sm:grid-cols-4">
          <div>
            <dt className="text-muted-foreground">Saldo</dt>
            <dd className="font-medium">{formatMoney(account.balance)}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Ekuitas</dt>
            <dd className="font-medium">{formatMoney(account.equity)}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Margin bebas</dt>
            <dd className="font-medium">{formatMoney(account.free_margin)}</dd>
          </div>
          <div>
            <dt className="text-muted-foreground">Risiko</dt>
            <dd className="font-medium">{formatNumber(account.risk_percent, 2)}%</dd>
          </div>
        </dl>

        {account.last_error && <Alert>{account.last_error}</Alert>}

        <div className="flex flex-wrap items-center gap-2">
          <Button
            size="sm"
            variant="outline"
            isLoading={update.isPending}
            onClick={() =>
              update.mutate({ id: account.id, payload: { is_active: !account.is_active } })
            }
          >
            {account.is_active ? 'Nonaktifkan' : 'Aktifkan'}
          </Button>

          {!confirming ? (
            <Button size="sm" variant="ghost" onClick={() => setConfirming(true)}>
              Hapus
            </Button>
          ) : (
            <span className="flex items-center gap-1.5 text-[12px]">
              Hapus kredensial permanen?
              <Button
                size="sm"
                variant="destructive"
                isLoading={remove.isPending}
                onClick={() => remove.mutate(account.id)}
              >
                Ya, hapus
              </Button>
              <Button size="sm" variant="ghost" onClick={() => setConfirming(false)}>
                Batal
              </Button>
            </span>
          )}

          <span className="ml-auto text-[11px] text-muted-foreground">
            Sinkron terakhir {formatDateTime(account.last_synced_at)}
          </span>
        </div>
      </CardContent>
    </Card>
  )
}

export default function AccountsPage() {
  const { user } = useAuthContext()
  const { data, isLoading, isError, error } = useAccounts()

  const tier = user?.tier ?? 'free'
  const limit = ACCOUNT_LIMIT[tier] ?? 0
  const canAdd = (data?.length ?? 0) < limit

  return (
    <div className="flex flex-col gap-4">
      <header className="flex flex-col gap-1">
        <h1 className="text-xl font-semibold tracking-tight">Akun Broker</h1>
        <p className="text-[13px] text-muted-foreground">
          {limit === 0
            ? 'Tier Free tidak dapat menghubungkan akun. Tingkatkan ke Basic atau Plus.'
            : `Tier ${tier} dapat menghubungkan ${limit} akun.`}
        </p>
      </header>

      {isLoading && <Skeleton className="h-40 w-full" />}
      {isError && <Alert>{(error as Error).message}</Alert>}

      {data?.map((account) => (
        <AccountItem key={account.id} account={account} />
      ))}

      {canAdd && (
        <Card>
          <CardHeader>
            <CardTitle>Hubungkan akun baru</CardTitle>
            <CardDescription>Hanya Match-Trader yang didukung saat ini.</CardDescription>
          </CardHeader>
          <CardContent>
            <AccountForm tier={tier} />
          </CardContent>
        </Card>
      )}
    </div>
  )
}
