import { useState } from 'react'
import { Badge, Button, Input, Label, StatusDot } from '@/components/ui'
import { formatDateTime, formatMoney, formatNumber } from '@/lib/format'
import { useDeleteAccount, useUpdateAccount } from '@/lib/queries'
import type { BrokerAccount } from '@/types/api'

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-0.5">
      <dt className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</dt>
      <dd className="tabular text-[13px] font-medium">{value}</dd>
    </div>
  )
}

/** AccountCard menampilkan satu akun broker beserta aksi yang tersedia. */
export function AccountCard({ account }: { account: BrokerAccount }) {
  const update = useUpdateAccount()
  const remove = useDeleteAccount()
  const [konfirmasiHapus, setKonfirmasiHapus] = useState(false)
  const [gantiToken, setGantiToken] = useState(false)
  const [token, setToken] = useState('')

  const simpanToken = () => {
    if (!token.trim()) return
    update.mutate(
      { id: account.id, payload: { token: token.trim() } },
      {
        onSuccess: () => {
          // Token hanya dikirim sekali; jangan disimpan di state lebih lama.
          setToken('')
          setGantiToken(false)
        },
      },
    )
  }

  const status = account.connected ? 'online' : account.last_error ? 'offline' : 'idle'
  const statusText = account.connected
    ? 'Sesi aktif'
    : account.last_error
      ? 'Bermasalah'
      : 'Belum tersambung'

  return (
    <article className="border border-border">
      <header className="flex flex-wrap items-start justify-between gap-2 border-b border-border px-4 py-3">
        <div className="flex flex-col gap-0.5">
          <div className="flex items-center gap-2">
            <StatusDot state={status} label={statusText} />
            <h3 className="text-[14px] font-semibold">{account.label}</h3>
          </div>
          <p className="text-[12px] text-muted-foreground">
            {account.broker_type} · {account.external_account_id || 'nomor belum terbaca'} ·{' '}
            {statusText}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-1.5">
          <Badge variant={account.is_dry_run ? 'muted' : 'solid'}>
            {account.is_dry_run ? 'Simulasi' : 'Nyata'}
          </Badge>
          <Badge variant="outline">{account.mode === 'auto' ? 'Otomatis' : 'Manual'}</Badge>
          {!account.is_active && <Badge variant="muted">Nonaktif</Badge>}
        </div>
      </header>

      <div className="flex flex-col gap-3 px-4 py-3">
        <dl className="grid grid-cols-2 gap-x-6 gap-y-3 sm:grid-cols-4 sm:max-w-xl">
          <Field label="Saldo" value={formatMoney(account.balance)} />
          <Field label="Ekuitas" value={formatMoney(account.equity)} />
          <Field label="Margin bebas" value={formatMoney(account.free_margin)} />
          <Field label="Risiko" value={`${formatNumber(account.risk_percent, 2)}%`} />
        </dl>

        {account.last_error && (
          <p className="border-l-2 border-foreground pl-2.5 text-[12px]">{account.last_error}</p>
        )}

        {gantiToken && (
          <div className="flex flex-col gap-2 border-t border-border pt-3">
            <Label htmlFor={`token-${account.id}`}>Access token baru</Label>
            <p className="text-[11px] text-muted-foreground">
              Tempel token dari aplikasi Match-Trader. Sesi lama langsung diganti setelah disimpan.
              Setiap akun memakai token berbeda. Jangan tampilkan token di luar panel ini.
            </p>
            <div className="flex flex-wrap gap-2">
              <Input
                id={`token-${account.id}`}
                type="password"
                autoComplete="off"
                value={token}
                onChange={(event) => setToken(event.target.value)}
                className="max-w-md font-mono"
              />
              <Button size="sm" isLoading={update.isPending} onClick={simpanToken}>
                Simpan token
              </Button>
              <Button
                size="sm"
                variant="ghost"
                onClick={() => {
                  setToken('')
                  setGantiToken(false)
                }}
              >
                Batal
              </Button>
            </div>
          </div>
        )}

        <div className="flex flex-wrap items-center gap-2 border-t border-border pt-3">
          <Button size="sm" variant="outline" onClick={() => setGantiToken(!gantiToken)}>
            {gantiToken ? 'Tutup ganti token' : 'Ganti token'}
          </Button>
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

          {!konfirmasiHapus ? (
            <Button size="sm" variant="ghost" onClick={() => setKonfirmasiHapus(true)}>
              Hapus akun
            </Button>
          ) : (
            <span className="flex flex-wrap items-center gap-1.5 text-[12px]">
              <span className="text-muted-foreground">Hapus kredensial permanen?</span>
              <Button
                size="sm"
                variant="destructive"
                isLoading={remove.isPending}
                onClick={() => remove.mutate(account.id)}
              >
                Ya, hapus
              </Button>
              <Button size="sm" variant="ghost" onClick={() => setKonfirmasiHapus(false)}>
                Batal
              </Button>
            </span>
          )}

          <span className="ml-auto text-[11px] text-muted-foreground">
            Sinkron terakhir {formatDateTime(account.last_synced_at)}
          </span>
        </div>
      </div>
    </article>
  )
}
