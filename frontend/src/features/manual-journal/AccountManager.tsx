import { useState, type FormEvent } from 'react'
import { useIsMutating, useQuery } from '@tanstack/react-query'
import { Alert, Button, Input, Select } from '@/components/ui'
import { journalApi } from './api'
import { journalKey, useJournalMutation } from './hooks'
import { splitList, validateDecimal, wibDay } from './model'
import { formatAmount } from './model'
import { AccountBalance } from './AccountBalance'
import type { AccountInput, CapitalInput, CapitalMovement, JournalAccount } from './types'

export function AccountManager({ accounts }: { accounts: JournalAccount[] }) {
  const busy = useIsMutating({ mutationKey: journalKey }) > 0
  const [editing, setEditing] = useState<JournalAccount | null>(null)
  const blank: AccountInput = {
    name: '',
    broker: '',
    currency: 'USD',
    initial_capital: '0',
    initial_date: wibDay(new Date().toISOString()),
    notes: '',
    archived: false,
    checklist: [],
  }
  const [form, setForm] = useState<AccountInput>(blank)
  const [checklist, setChecklist] = useState('')
  const [error, setError] = useState('')
  const save = useJournalMutation(({ input, id }: { input: AccountInput; id?: string }) =>
    journalApi.saveAccount(input, id),
  )
  const archive = useJournalMutation(journalApi.archiveAccount)
  const [capitalAccount, setCapitalAccount] = useState('')
  const selected = accounts.find((a) => a.id === capitalAccount)
  function reset() {
    setEditing(null)
    setForm(blank)
    setChecklist('')
    setError('')
  }
  async function submit(e: FormEvent) {
    e.preventDefault()
    const labels =
      editing && checklist === editing.checklist.join('\n')
        ? editing.checklist
        : splitList(checklist)
    if (
      !form.name.trim() ||
      !/^[A-Z]{3}$/.test(form.currency) ||
      !validateDecimal(form.initial_capital) ||
      labels.length > 30
    ) {
      setError(
        'Isi nama, mata uang 3 huruf, modal nonnegatif maksimal 4 desimal dan checklist maksimal 30 item.',
      )
      return
    }
    try {
      await save.mutateAsync({
        input: { ...form, checklist: labels },
        id: editing?.id,
      })
      reset()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Gagal menyimpan akun.')
    }
  }
  return (
    <div className="space-y-6">
      <p className="text-sm text-muted-foreground">
        Akun ini independen dari koneksi broker. Saldo terealisasi dihitung dari catatan jurnal,
        bukan equity broker. Modal awal dan deposit/withdrawal tidak dihitung sebagai profit
        trading.
      </p>
      {error && <Alert variant="destructive">{error}</Alert>}
      <ul className="divide-y divide-border">
        {accounts.map((account) => (
          <li key={account.id} className="flex flex-wrap items-center justify-between gap-2 py-3">
            <div className="min-w-0">
              <p className="break-words text-sm font-semibold">
                {account.name} {account.archived && '(diarsipkan)'}
              </p>
              <p className="text-xs text-muted-foreground">
                {account.broker || 'Tanpa broker'} · {account.currency} · modal awal{' '}
                {formatAmount(account.initial_capital, account.currency)}
              </p>
              <AccountBalance account={account} />
            </div>
            <div className="flex gap-2">
              <Button
                size="sm"
                variant="outline"
                disabled={busy}
                onClick={() => {
                  setEditing(account)
                  setForm({ ...account })
                  setChecklist(account.checklist.join('\n'))
                  setError('')
                }}
              >
                Edit
              </Button>
              <Button
                size="sm"
                variant="ghost"
                disabled={busy}
                onClick={() => setCapitalAccount(account.id)}
              >
                Modal
              </Button>
              {!account.archived && (
                <Button
                  size="sm"
                  variant="ghost"
                  disabled={busy}
                  onClick={async () => {
                    if (!window.confirm(`Arsipkan ${account.name}? Trade tetap tersimpan.`)) return
                    try {
                      await archive.mutateAsync(account.id)
                    } catch (err) {
                      setError(err instanceof Error ? err.message : 'Gagal mengarsipkan akun.')
                    }
                  }}
                >
                  Arsip
                </Button>
              )}
            </div>
          </li>
        ))}
      </ul>
      <form onSubmit={submit} className="space-y-4 border-t border-border pt-4">
        <h3 className="font-semibold">{editing ? `Edit ${editing.name}` : 'Tambah akun jurnal'}</h3>
        <fieldset disabled={busy} className="grid gap-3 sm:grid-cols-2">
          <label className="space-y-1 text-sm">
            Nama akun
            <Input
              required
              maxLength={100}
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
            />
          </label>
          <label className="space-y-1 text-sm">
            Broker (opsional)
            <Input
              maxLength={100}
              value={form.broker}
              onChange={(e) => setForm({ ...form, broker: e.target.value })}
            />
          </label>
          <label className="space-y-1 text-sm">
            Mata uang
            <Input
              required
              pattern="[A-Z]{3}"
              maxLength={3}
              disabled={!!editing}
              value={form.currency}
              onChange={(e) => setForm({ ...form, currency: e.target.value.toUpperCase() })}
            />
          </label>
          <label className="space-y-1 text-sm">
            Modal awal
            <Input
              required
              inputMode="decimal"
              value={form.initial_capital}
              onChange={(e) => setForm({ ...form, initial_capital: e.target.value })}
            />
          </label>
          <label className="space-y-1 text-sm">
            Tanggal modal awal (WIB)
            <Input
              required
              type="date"
              value={form.initial_date}
              onChange={(e) => setForm({ ...form, initial_date: e.target.value })}
            />
          </label>
          <label className="space-y-1 text-sm">
            Checklist akun (satu per baris)
            <textarea
              className="min-h-24 w-full rounded-md border border-input bg-background p-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              value={checklist}
              onChange={(e) => setChecklist(e.target.value)}
            />
          </label>
          <label className="space-y-1 text-sm sm:col-span-2">
            Catatan
            <Input
              maxLength={1000}
              value={form.notes}
              onChange={(e) => setForm({ ...form, notes: e.target.value })}
            />
          </label>
          {editing && (
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={form.archived}
                onChange={(e) => setForm({ ...form, archived: e.target.checked })}
              />
              Arsipkan akun
            </label>
          )}
        </fieldset>
        <div className="flex justify-end gap-2">
          {editing && (
            <Button type="button" variant="outline" disabled={busy} onClick={reset}>
              Batal edit
            </Button>
          )}
          <Button type="submit" disabled={busy} isLoading={save.isPending}>
            {editing ? 'Simpan akun' : 'Tambah akun'}
          </Button>
        </div>
      </form>
      <section className="space-y-3 border-t border-border pt-4">
        <h3 className="font-semibold">Pergerakan modal</h3>
        <label className="block space-y-1 text-sm">
          Akun modal
          <Select
            disabled={busy}
            value={capitalAccount}
            onChange={(e) => setCapitalAccount(e.target.value)}
          >
            <option value="">Pilih akun</option>
            {accounts.map((a) => (
              <option key={a.id} value={a.id}>
                {a.name} · {a.currency}
              </option>
            ))}
          </Select>
        </label>
        {selected && <CapitalManager key={selected.id} account={selected} />}
      </section>
    </div>
  )
}

function CapitalManager({ account }: { account: JournalAccount }) {
  const busy = useIsMutating({ mutationKey: journalKey }) > 0
  const blank: CapitalInput = {
    account_id: account.id,
    kind: 'deposit',
    amount: '',
    date: wibDay(new Date().toISOString()),
    notes: '',
  }
  const [form, setForm] = useState(blank)
  const [editing, setEditing] = useState<CapitalMovement | null>(null)
  const [error, setError] = useState('')
  const query = useQuery({
    queryKey: [...journalKey, 'capital', account.id],
    queryFn: ({ signal }) => journalApi.capital(account.id, signal),
  })
  const save = useJournalMutation(({ input, id }: { input: CapitalInput; id?: string }) =>
    journalApi.saveCapital(input, id),
  )
  const remove = useJournalMutation(journalApi.deleteCapital)
  async function submit(e: FormEvent) {
    e.preventDefault()
    if (!validateDecimal(form.amount) || Number(form.amount) <= 0) {
      setError('Jumlah harus lebih dari nol, maksimal 4 desimal dan 1 miliar.')
      return
    }
    try {
      await save.mutateAsync({ input: form, id: editing?.id })
      setEditing(null)
      setForm(blank)
      setError('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Gagal menyimpan modal.')
    }
  }
  return (
    <div className="space-y-3">
      {query.isLoading && (
        <p role="status" className="text-sm">
          Memuat modal…
        </p>
      )}
      {(error || query.error) && (
        <Alert variant="destructive">{error || query.error?.message}</Alert>
      )}
      <ul className="max-h-52 divide-y divide-border overflow-y-auto">
        {query.data?.data.map((item) => (
          <li
            key={item.id}
            className="flex flex-wrap items-center justify-between gap-2 py-2 text-xs"
          >
            <span>
              {item.date} · {item.kind} · {formatAmount(item.amount, account.currency)}
              <span className="block text-muted-foreground">{item.notes}</span>
            </span>
            <div className="flex gap-2">
              <Button
                size="sm"
                variant="ghost"
                disabled={busy || account.archived}
                onClick={() => {
                  setEditing(item)
                  setForm(item)
                }}
              >
                Edit
              </Button>
              <Button
                size="sm"
                variant="ghost"
                disabled={busy || account.archived}
                onClick={async () => {
                  if (!window.confirm('Hapus pergerakan modal ini?')) return
                  try {
                    await remove.mutateAsync(item.id)
                  } catch (err) {
                    setError(err instanceof Error ? err.message : 'Gagal menghapus modal.')
                  }
                }}
              >
                Hapus
              </Button>
            </div>
          </li>
        ))}
      </ul>
      {query.data && !query.data.data.length && (
        <p className="text-xs text-muted-foreground">Belum ada deposit atau withdrawal.</p>
      )}
      <form onSubmit={submit}>
        <fieldset disabled={busy || account.archived} className="grid gap-3 sm:grid-cols-2">
          <label className="space-y-1 text-sm">
            Jenis
            <Select
              value={form.kind}
              onChange={(e) => setForm({ ...form, kind: e.target.value as CapitalInput['kind'] })}
            >
              <option value="deposit">Deposit</option>
              <option value="withdrawal">Withdrawal</option>
            </Select>
          </label>
          <label className="space-y-1 text-sm">
            Jumlah ({account.currency})
            <Input
              inputMode="decimal"
              required
              value={form.amount}
              onChange={(e) => setForm({ ...form, amount: e.target.value })}
            />
          </label>
          <label className="space-y-1 text-sm">
            Tanggal (WIB)
            <Input
              type="date"
              required
              value={form.date}
              onChange={(e) => setForm({ ...form, date: e.target.value })}
            />
          </label>
          <label className="space-y-1 text-sm">
            Catatan
            <Input
              maxLength={1000}
              value={form.notes}
              onChange={(e) => setForm({ ...form, notes: e.target.value })}
            />
          </label>
          <div className="flex gap-2 sm:col-span-2">
            {editing && (
              <Button
                type="button"
                variant="outline"
                disabled={busy || account.archived}
                onClick={() => {
                  setEditing(null)
                  setForm(blank)
                }}
              >
                Batal edit
              </Button>
            )}
            <Button type="submit" isLoading={save.isPending}>
              {editing ? 'Simpan perubahan modal' : 'Catat modal'}
            </Button>
          </div>
        </fieldset>
      </form>
    </div>
  )
}
