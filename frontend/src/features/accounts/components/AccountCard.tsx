import { useState } from 'react'
import { Link } from 'react-router-dom'
import { ArrowRight, Settings } from 'lucide-react'
import { Badge, Button, Input, Label, Modal, StatusDot } from '@/components/ui'
import { formatDateTime, formatMoney, formatNumber } from '@/lib/format'
import { useDeleteAccount, useUpdateAccount } from '@/lib/queries'
import type { BrokerAccount } from '@/types/api'

function Field({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <dt className="truncate text-[11px] uppercase tracking-wide text-muted-foreground">{label}</dt>
      <dd className="truncate tabular text-[13px] font-medium">{value}</dd>
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
    <article className="overflow-hidden rounded-2xl border border-border bg-card shadow-xs sm:rounded-xl">
      <header className="flex flex-wrap items-start justify-between gap-2 border-b border-border px-4 py-3">
        <div className="flex min-w-0 flex-col gap-0.5">
          <div className="flex min-w-0 items-center gap-2">
            <StatusDot state={status} label={statusText} />
            <h3 className="truncate text-[14px] font-semibold">{account.label}</h3>
          </div>
          <p className="text-[12px] text-muted-foreground">
            {account.broker_type} · {account.external_account_id || 'nomor belum terbaca'} ·{' '}
            {statusText}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-1.5">
          <Badge variant={account.is_dry_run ? 'muted' : 'solid'}>
            {account.is_dry_run ? 'Demo' : 'Live'}
          </Badge>
          <Badge variant="outline">{account.mode === 'auto' ? 'Auto' : 'Manual'}</Badge>
          {!account.is_active && <Badge variant="muted">Nonaktif</Badge>}
        </div>
      </header>

      <div className="flex flex-col gap-3 px-4 py-3">
        <dl className="grid grid-cols-2 gap-x-4 gap-y-3 sm:max-w-xl sm:grid-cols-4 sm:gap-x-6">
          <Field label="Balance" value={formatMoney(account.balance)} />
          <Field label="Equity" value={formatMoney(account.equity)} />
          <Field label="Free Margin" value={formatMoney(account.free_margin)} />
          <Field
            label="Risk / Trade"
            value={
              account.risk_mode === 'lot'
                ? `${formatNumber(account.lot_size ?? 0.01, 2)} lot`
                : `${formatNumber(account.risk_percent, 2)}%`
            }
          />
        </dl>

        {account.last_error && (
          <p className="break-words border-l-2 border-foreground pl-2.5 text-[12px]">
            {account.last_error}
          </p>
        )}

        <div className="flex flex-wrap items-center justify-between gap-2 border-t border-border pt-3">
          <div className="flex w-full flex-wrap items-center gap-2 sm:w-auto">
            <Link
              to={`/akun/${account.id}`}
              className="inline-flex w-full items-center justify-center gap-1.5 rounded-lg border border-border bg-foreground px-3 py-2 text-xs font-semibold text-background transition-colors hover:bg-foreground/90 sm:w-auto sm:py-1.5"
            >
              <Settings className="h-3.5 w-3.5" />
              <span>Detail & Pengaturan</span>
              <ArrowRight className="h-3 w-3" />
            </Link>

            {account.broker_type === 'matchtrader' && (
              <Button size="sm" variant="outline" onClick={() => setGantiToken(true)}>
                Ganti token
              </Button>
            )}

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
          </div>

          <Button
            size="sm"
            variant="ghost"
            className="text-destructive hover:bg-destructive/10"
            onClick={() => setKonfirmasiHapus(true)}
          >
            Hapus Akun
          </Button>
        </div>

        {/* Modal Ganti Token */}
        <Modal
          open={gantiToken}
          onClose={() => {
            setToken('')
            setGantiToken(false)
          }}
          title="Ganti Token Akses Match-Trader"
          description={`Perbarui access token untuk akun ${account.label}. Token lama langsung digantikan seketika.`}
          footer={
            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  setToken('')
                  setGantiToken(false)
                }}
              >
                Batal
              </Button>
              <Button size="sm" isLoading={update.isPending} onClick={simpanToken} disabled={!token.trim()}>
                Simpan Token Baru
              </Button>
            </div>
          }
        >
          <div className="space-y-3 py-2">
            <div className="space-y-1.5">
              <Label htmlFor={`token-${account.id}`}>Access Token Baru</Label>
              <Input
                id={`token-${account.id}`}
                type="password"
                autoComplete="off"
                placeholder="Tempel token Match-Trader di sini"
                value={token}
                onChange={(event) => setToken(event.target.value)}
                className="font-mono text-xs"
              />
            </div>
            <p className="text-[12px] text-muted-foreground leading-relaxed">
              Token ini dienkripsi di server dan hanya digunakan untuk eksekusi sinyal atas nama Anda.
              Jangan bagikan token ini kepada siapa pun.
            </p>
          </div>
        </Modal>

        {/* Modal Konfirmasi Hapus Akun */}
        <Modal
          open={konfirmasiHapus}
          onClose={() => setKonfirmasiHapus(false)}
          title="Hapus Akun Broker?"
          description={`Anda akan menghapus akun ${account.label} (${account.external_account_id || account.broker_type}) secara permanen.`}
          footer={
            <div className="flex justify-end gap-2">
              <Button variant="outline" size="sm" onClick={() => setKonfirmasiHapus(false)}>
                Batal
              </Button>
              <Button
                variant="destructive"
                size="sm"
                isLoading={remove.isPending}
                onClick={() => {
                  remove.mutate(account.id, {
                    onSuccess: () => setKonfirmasiHapus(false),
                  })
                }}
              >
                Hapus Permanen
              </Button>
            </div>
          }
        >
          <div className="space-y-2 py-2 text-[13px] text-muted-foreground leading-relaxed">
            <p>
              Seluruh kredensial token broker yang tersimpan di sistem akan langsung dihapus. Sinyal trading
              tidak lagi dapat diteruskan ke akun ini.
            </p>
          </div>
        </Modal>

        <div className="flex items-center justify-between border-t border-border pt-2 text-[11px] text-muted-foreground">
          <span>Sinkron terakhir {formatDateTime(account.last_synced_at)}</span>
        </div>
      </div>
    </article>
  )
}
