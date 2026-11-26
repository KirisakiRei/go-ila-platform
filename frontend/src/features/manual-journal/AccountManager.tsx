import { useState, type FormEvent } from 'react'
import { ArrowLeft, Plus, RotateCcw } from 'lucide-react'
import { useIsMutating, useQuery } from '@tanstack/react-query'
import { Alert, Button, Input, Select } from '@/components/ui'
import { cn } from '@/lib/utils'
import { journalApi } from './api'
import { journalKey, useJournalMutation } from './hooks'
import {
  formatAmount,
  friendlyJournalError,
  mergeChipDraft,
  normalizeDecimalInput,
  validateDecimal,
  wibDay,
} from './model'
import { AccountBalance } from './AccountBalance'
import type { AccountInput, CapitalInput, CapitalMovement, JournalAccount } from './types'
import { ChipInput, ConfirmButton, SegmentedControl } from './ui'

const CURRENCY_PRESETS = ['USD', 'IDR', 'EUR', 'GBP'] as const
const CAPITAL_KINDS = [
  { value: 'deposit', label: '+ Deposit', tone: 'buy' as const },
  { value: 'withdrawal', label: '− Withdrawal', tone: 'sell' as const },
] as const

function blankAccount(): AccountInput {
  return {
    name: '',
    broker: '',
    currency: 'USD',
    initial_capital: '0',
    initial_date: wibDay(new Date().toISOString()),
    notes: '',
    archived: false,
    checklist: [],
  }
}

export function AccountsPanel({
  accounts,
  onSelectCapital,
}: {
  accounts: JournalAccount[]
  onSelectCapital?: (accountId: string) => void
}) {
  const busy = useIsMutating({ mutationKey: journalKey }) > 0
  const [mode, setMode] = useState<'list' | 'form'>(accounts.length === 0 ? 'form' : 'list')
  const [editing, setEditing] = useState<JournalAccount | null>(null)
  const [form, setForm] = useState<AccountInput>(blankAccount)
  const [checklistDraft, setChecklistDraft] = useState('')
  const [error, setError] = useState('')
  const [fieldErrors, setFieldErrors] = useState<
    Partial<Record<'name' | 'currency' | 'initial_capital' | 'initial_date' | 'checklist', string>>
  >({})

  const save = useJournalMutation(({ input, id }: { input: AccountInput; id?: string }) =>
    journalApi.saveAccount(input, id),
  )
  const archive = useJournalMutation(journalApi.archiveAccount)
  const remove = useJournalMutation(journalApi.deleteAccount)

  function openNew() {
    setEditing(null)
    setForm(blankAccount())
    setChecklistDraft('')
    setFieldErrors({})
    setError('')
    setMode('form')
  }

  function openEdit(account: JournalAccount) {
    setEditing(account)
    setForm({
      name: account.name,
      broker: account.broker,
      currency: account.currency,
      initial_capital: account.initial_capital,
      initial_date: account.initial_date,
      notes: account.notes,
      archived: account.archived,
      checklist: [...account.checklist],
    })
    setChecklistDraft('')
    setFieldErrors({})
    setError('')
    setMode('form')
  }

  function backToList() {
    setEditing(null)
    setForm(blankAccount())
    setChecklistDraft('')
    setFieldErrors({})
    setError('')
    setMode('list')
  }

  async function submit(e: FormEvent) {
    e.preventDefault()
    const checklist = mergeChipDraft(form.checklist, checklistDraft, 31)
    if (checklist !== form.checklist) {
      setForm((prev) => ({ ...prev, checklist }))
    }
    if (checklistDraft.trim()) setChecklistDraft('')
    const normalizedCapital = normalizeDecimalInput(form.initial_capital) ?? ''
    const payload: AccountInput = {
      ...form,
      name: form.name.trim(),
      currency: form.currency.trim().toUpperCase(),
      initial_capital: normalizedCapital,
      checklist,
    }
    const nextErrors: Partial<
      Record<'name' | 'currency' | 'initial_capital' | 'initial_date' | 'checklist', string>
    > = {}
    if (!payload.name) nextErrors.name = 'Nama jurnal wajib diisi.'
    if (!/^[A-Z]{3}$/.test(payload.currency)) {
      nextErrors.currency = 'Gunakan 3 huruf mata uang (mis. USD, IDR).'
    }
    if (!payload.initial_capital || !validateDecimal(payload.initial_capital)) {
      nextErrors.initial_capital = payload.initial_capital.startsWith('-')
        ? 'Tidak boleh negatif.'
        : 'Gunakan angka valid (maks 4 desimal, contoh: 1000.00).'
    }
    if (!payload.initial_date) nextErrors.initial_date = 'Pilih tanggal mulai.'
    if (payload.checklist.length > 30) nextErrors.checklist = 'Maksimal 30 item checklist.'

    if (Object.keys(nextErrors).length > 0) {
      setFieldErrors(nextErrors)
      setError('Periksa kolom yang ditandai merah.')
      return
    }
    setFieldErrors({})
    setError('')
    try {
      await save.mutateAsync({ input: payload, id: editing?.id })
      backToList()
    } catch (err) {
      setError(friendlyJournalError(err, 'Gagal menyimpan jurnal.'))
    }
  }

  if (mode === 'form') {
    const errCls = (key: keyof typeof fieldErrors) =>
      fieldErrors[key] ? 'border-destructive focus-visible:ring-destructive' : ''
    return (
      <form onSubmit={submit} noValidate className="space-y-4">
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            {accounts.length > 0 && (
              <Button
                type="button"
                variant="ghost"
                size="sm"
                disabled={busy}
                onClick={backToList}
                className="h-8 px-2"
              >
                <ArrowLeft className="h-4 w-4" aria-hidden="true" />
                Kembali
              </Button>
            )}
            <h3 className="text-sm font-semibold">
              {editing ? `Edit ${editing.name}` : 'Jurnal baru'}
            </h3>
          </div>
        </div>

        {error && <Alert variant="destructive">{error}</Alert>}

        <fieldset disabled={busy} className="space-y-4">
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="space-y-1">
              <label className="block space-y-1.5 text-xs font-medium text-muted-foreground">
                Nama jurnal
                <Input
                  maxLength={100}
                  aria-invalid={Boolean(fieldErrors.name) || undefined}
                  className={errCls('name')}
                  value={form.name}
                  onChange={(e) => {
                    setForm({ ...form, name: e.target.value })
                    setFieldErrors((p) => ({ ...p, name: undefined }))
                  }}
                  placeholder="Mis. XAUUSD H1, Swing Forex, Prop Firm"
                />
              </label>
              {fieldErrors.name && (
                <span role="alert" className="block text-[11px] text-destructive">
                  {fieldErrors.name}
                </span>
              )}
            </div>
            <label className="space-y-1.5 text-xs font-medium text-muted-foreground">
              Broker / keterangan (opsional)
              <Input
                maxLength={100}
                value={form.broker}
                onChange={(e) => setForm({ ...form, broker: e.target.value })}
                placeholder="Mis. Exness, Demo, Backtest"
              />
            </label>
          </div>

          <div className="grid gap-3 sm:grid-cols-3">
            <div className="space-y-1.5">
              <label className="block space-y-1.5 text-xs font-medium text-muted-foreground">
                Mata uang
                <Input
                  maxLength={3}
                  disabled={!!editing}
                  aria-invalid={Boolean(fieldErrors.currency) || undefined}
                  className={errCls('currency')}
                  value={form.currency}
                  onChange={(e) => {
                    setForm({ ...form, currency: e.target.value.toUpperCase() })
                    setFieldErrors((p) => ({ ...p, currency: undefined }))
                  }}
                />
              </label>
              {fieldErrors.currency && (
                <span role="alert" className="block text-[11px] text-destructive">
                  {fieldErrors.currency}
                </span>
              )}
              {!editing && (
                <div className="flex flex-wrap gap-1">
                  {CURRENCY_PRESETS.map((cur) => (
                    <button
                      key={cur}
                      type="button"
                      onClick={() => setForm({ ...form, currency: cur })}
                      className={cn(
                        'rounded border px-1.5 py-0.5 text-[11px] font-medium transition-colors',
                        form.currency === cur
                          ? 'border-foreground/40 bg-foreground text-background'
                          : 'border-border text-muted-foreground hover:bg-secondary',
                      )}
                    >
                      {cur}
                    </button>
                  ))}
                </div>
              )}
            </div>

            <div className="space-y-1">
              <label className="block space-y-1.5 text-xs font-medium text-muted-foreground">
                Modal awal
                <Input
                  inputMode="decimal"
                  aria-invalid={Boolean(fieldErrors.initial_capital) || undefined}
                  className={cn('tabular-nums', errCls('initial_capital'))}
                  value={form.initial_capital}
                  onChange={(e) => {
                    setForm({ ...form, initial_capital: e.target.value })
                    setFieldErrors((p) => ({ ...p, initial_capital: undefined }))
                  }}
                />
              </label>
              {fieldErrors.initial_capital && (
                <span role="alert" className="block text-[11px] text-destructive">
                  {fieldErrors.initial_capital}
                </span>
              )}
            </div>

            <div className="space-y-1">
              <label className="block space-y-1.5 text-xs font-medium text-muted-foreground">
                Tanggal mulai (WIB)
                <Input
                  type="date"
                  aria-invalid={Boolean(fieldErrors.initial_date) || undefined}
                  className={errCls('initial_date')}
                  value={form.initial_date}
                  onChange={(e) => {
                    setForm({ ...form, initial_date: e.target.value })
                    setFieldErrors((p) => ({ ...p, initial_date: undefined }))
                  }}
                />
              </label>
              {fieldErrors.initial_date && (
                <span role="alert" className="block text-[11px] text-destructive">
                  {fieldErrors.initial_date}
                </span>
              )}
            </div>
          </div>

          <ChipInput
            label="Template pre-trade checklist"
            values={form.checklist}
            onChange={(checklist) => {
              setForm({ ...form, checklist })
              setFieldErrors((p) => ({ ...p, checklist: undefined }))
            }}
            draftValue={checklistDraft}
            onDraftChange={setChecklistDraft}
            placeholder="Ketik aturan lalu Enter (mis. Risiko <= 1%)"
            disabled={busy}
            error={fieldErrors.checklist}
          />

          <label className="block space-y-1.5 text-xs font-medium text-muted-foreground">
            Aturan risiko / catatan (opsional)
            <Input
              maxLength={1000}
              value={form.notes}
              onChange={(e) => setForm({ ...form, notes: e.target.value })}
              placeholder="Mis. Max loss harian 2%, stop setelah 2 SL"
            />
          </label>
        </fieldset>

        <div className="flex justify-end gap-2 border-t border-border pt-4">
          {accounts.length > 0 && (
            <Button type="button" variant="outline" disabled={busy} onClick={backToList}>
              Batal
            </Button>
          )}
          <Button type="submit" disabled={busy} isLoading={save.isPending}>
            {editing ? 'Simpan perubahan' : 'Simpan jurnal'}
          </Button>
        </div>
      </form>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-xs text-muted-foreground">
          Pisahkan jurnal sesuai strategi, pair, atau target modal.
        </p>
        <Button size="sm" disabled={busy} onClick={openNew}>
          <Plus className="h-3.5 w-3.5" aria-hidden="true" />
          Tambah jurnal
        </Button>
      </div>

      {error && <Alert variant="destructive">{error}</Alert>}

      <ul className="space-y-2.5">
        {accounts.map((account) => (
          <li
            key={account.id}
            className="flex flex-wrap items-start justify-between gap-3 rounded-xl border border-border/80 bg-secondary/20 p-3.5"
          >
            <div className="min-w-0 space-y-1">
              <div className="flex flex-wrap items-center gap-2">
                <p className="truncate text-sm font-semibold">{account.name}</p>
                <span className="rounded bg-secondary px-1.5 py-0.5 text-[11px] font-medium text-muted-foreground">
                  {account.currency}
                </span>
                {account.archived && (
                  <span className="rounded border border-border px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
                    Diarsipkan
                  </span>
                )}
              </div>
              <p className="text-xs text-muted-foreground">
                {account.broker || 'Mandiri'} · Modal awal{' '}
                {formatAmount(account.initial_capital, account.currency)} · {account.checklist.length}{' '}
                checklist
              </p>
              <AccountBalance account={account} />
            </div>

            <div className="flex flex-wrap items-center gap-1.5">
              <Button size="sm" variant="outline" disabled={busy} onClick={() => openEdit(account)}>
                Edit
              </Button>
              {onSelectCapital && (
                <Button
                  size="sm"
                  variant="ghost"
                  disabled={busy}
                  onClick={() => onSelectCapital(account.id)}
                >
                  Modal
                </Button>
              )}
              {account.archived ? (
                <Button
                  size="sm"
                  variant="outline"
                  disabled={busy}
                  onClick={async () => {
                    setError('')
                    try {
                      await save.mutateAsync({
                        input: {
                          name: account.name,
                          broker: account.broker,
                          currency: account.currency,
                          initial_capital: account.initial_capital,
                          initial_date: account.initial_date,
                          notes: account.notes,
                          archived: false,
                          checklist: account.checklist,
                        },
                        id: account.id,
                      })
                    } catch (err) {
                      setError(err instanceof Error ? err.message : 'Gagal memulihkan akun.')
                    }
                  }}
                >
                  <RotateCcw className="h-3.5 w-3.5" aria-hidden="true" />
                  Pulihkan
                </Button>
              ) : (
                <ConfirmButton
                  label="Arsipkan"
                  confirmLabel="Yakin arsip?"
                  variant="ghost"
                  size="sm"
                  disabled={busy}
                  onConfirm={async () => {
                    setError('')
                    try {
                      await archive.mutateAsync(account.id)
                    } catch (err) {
                      setError(err instanceof Error ? err.message : 'Gagal mengarsipkan jurnal.')
                    }
                  }}
                />
              )}
              <ConfirmButton
                label="Hapus"
                confirmLabel="Hapus permanen?"
                variant="ghost"
                size="sm"
                disabled={busy}
                className="text-rose-600 hover:text-rose-700 dark:text-rose-400"
                onConfirm={async () => {
                  setError('')
                  try {
                    await remove.mutateAsync(account.id)
                  } catch (err) {
                    setError(err instanceof Error ? err.message : 'Gagal menghapus jurnal.')
                  }
                }}
              />
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}

export function CapitalPanel({
  accounts,
  initialAccountId,
}: {
  accounts: JournalAccount[]
  initialAccountId?: string
}) {
  const busy = useIsMutating({ mutationKey: journalKey }) > 0
  const [accountId, setAccountId] = useState(
    initialAccountId || accounts.find((a) => !a.archived)?.id || accounts[0]?.id || '',
  )
  const selected = accounts.find((a) => a.id === accountId) ?? accounts[0]

  if (!accounts.length) {
    return (
      <p className="py-8 text-center text-sm text-muted-foreground">
        Buat jurnal dulu di tab Jurnal.
      </p>
    )
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <label className="min-w-[200px] flex-1 space-y-1.5 text-xs font-medium text-muted-foreground">
          Jurnal
          <Select
            disabled={busy}
            value={selected?.id ?? ''}
            onChange={(e) => setAccountId(e.target.value)}
          >
            {accounts.map((a) => (
              <option key={a.id} value={a.id}>
                {a.name} · {a.currency}
                {a.archived ? ' (arsip)' : ''}
              </option>
            ))}
          </Select>
        </label>
        {selected && <AccountBalance account={selected} compact />}
      </div>
      {selected && <CapitalManager key={selected.id} account={selected} />}
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
  const [form, setForm] = useState<CapitalInput>(blank)
  const [editing, setEditing] = useState<CapitalMovement | null>(null)
  const [error, setError] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Partial<Record<'amount' | 'date', string>>>({})
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
    const normalizedAmount = normalizeDecimalInput(form.amount) ?? ''
    const nextErrors: Partial<Record<'amount' | 'date', string>> = {}
    if (!normalizedAmount || !validateDecimal(normalizedAmount) || Number(normalizedAmount) <= 0) {
      nextErrors.amount = normalizedAmount.startsWith('-')
        ? 'Tidak boleh negatif.'
        : 'Isi jumlah lebih dari 0 (maks 4 desimal, contoh: 100.00).'
    }
    if (!form.date) {
      nextErrors.date = 'Pilih tanggal transaksi.'
    }
    if (Object.keys(nextErrors).length > 0) {
      setFieldErrors(nextErrors)
      setError('Periksa kolom yang ditandai merah.')
      return
    }
    setFieldErrors({})
    setError('')
    try {
      await save.mutateAsync({
        input: { ...form, amount: normalizedAmount, account_id: account.id },
        id: editing?.id,
      })
      setEditing(null)
      setForm(blank)
    } catch (err) {
      setError(friendlyJournalError(err, 'Gagal menyimpan pergerakan modal.'))
    }
  }

  return (
    <div className="space-y-4">
      {(error || query.error) && (
        <Alert variant="destructive">{error || query.error?.message}</Alert>
      )}

      <form
        onSubmit={submit}
        noValidate
        className="space-y-3 rounded-xl border border-border/80 bg-secondary/25 p-3.5"
      >
        <div className="flex items-center justify-between gap-2">
          <span className="text-xs font-semibold">
            {editing ? 'Edit pergerakan modal' : 'Catat deposit / withdrawal'}
          </span>
          {editing && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-7 px-2 text-xs"
              disabled={busy}
              onClick={() => {
                setEditing(null)
                setForm(blank)
                setError('')
              }}
            >
              Batal edit
            </Button>
          )}
        </div>

        <fieldset disabled={busy || account.archived} className="space-y-3">
          <SegmentedControl
            ariaLabel="Jenis transaksi modal"
            value={form.kind}
            onChange={(kind) => setForm({ ...form, kind })}
            options={CAPITAL_KINDS}
            fullWidth
            disabled={busy || account.archived}
          />
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="space-y-1">
              <label className="block space-y-1 text-xs text-muted-foreground">
                Jumlah ({account.currency})
                <Input
                  inputMode="decimal"
                  aria-invalid={Boolean(fieldErrors.amount) || undefined}
                  className={cn(
                    'tabular-nums',
                    fieldErrors.amount && 'border-destructive focus-visible:ring-destructive',
                  )}
                  value={form.amount}
                  onChange={(e) => {
                    setForm({ ...form, amount: e.target.value })
                    setFieldErrors((p) => ({ ...p, amount: undefined }))
                  }}
                  placeholder="100.00"
                />
              </label>
              {fieldErrors.amount && (
                <span role="alert" className="block text-[11px] text-destructive">
                  {fieldErrors.amount}
                </span>
              )}
            </div>
            <div className="space-y-1">
              <label className="block space-y-1 text-xs text-muted-foreground">
                Tanggal (WIB)
                <Input
                  type="date"
                  aria-invalid={Boolean(fieldErrors.date) || undefined}
                  className={fieldErrors.date ? 'border-destructive focus-visible:ring-destructive' : ''}
                  value={form.date}
                  onChange={(e) => {
                    setForm({ ...form, date: e.target.value })
                    setFieldErrors((p) => ({ ...p, date: undefined }))
                  }}
                />
              </label>
              {fieldErrors.date && (
                <span role="alert" className="block text-[11px] text-destructive">
                  {fieldErrors.date}
                </span>
              )}
            </div>
          </div>
          <label className="block space-y-1 text-xs text-muted-foreground">
            Catatan (opsional)
            <Input
              maxLength={1000}
              value={form.notes}
              onChange={(e) => setForm({ ...form, notes: e.target.value })}
              placeholder="Mis. Top up bulanan"
            />
          </label>
          <div className="flex justify-end">
            <Button type="submit" size="sm" isLoading={save.isPending}>
              {editing ? 'Simpan perubahan' : 'Tambah transaksi'}
            </Button>
          </div>
        </fieldset>
      </form>

      <div className="space-y-2">
        <p className="text-xs font-medium text-muted-foreground">Riwayat pergerakan modal</p>
        {query.isLoading ? (
          <p role="status" className="text-xs text-muted-foreground">
            Memuat riwayat…
          </p>
        ) : !query.data?.data.length ? (
          <p className="rounded-lg border border-dashed border-border py-5 text-center text-xs text-muted-foreground">
            Belum ada riwayat deposit atau withdrawal
          </p>
        ) : (
          <ul className="max-h-52 divide-y divide-border overflow-y-auto rounded-lg border border-border scrollbar-thin">
            {query.data.data.map((item) => (
              <li
                key={item.id}
                className="flex flex-wrap items-center justify-between gap-2 px-3 py-2.5 text-xs"
              >
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <span
                      className={cn(
                        'font-semibold tabular-nums',
                        item.kind === 'deposit'
                          ? 'text-emerald-600 dark:text-emerald-400'
                          : 'text-rose-600 dark:text-rose-400',
                      )}
                    >
                      {item.kind === 'deposit' ? '+' : '−'}
                      {formatAmount(item.amount, account.currency)}
                    </span>
                    <span className="text-muted-foreground">{item.date} WIB</span>
                  </div>
                  {item.notes && <p className="mt-0.5 truncate text-muted-foreground">{item.notes}</p>}
                </div>
                <div className="flex items-center gap-1">
                  <Button
                    size="sm"
                    variant="ghost"
                    className="h-7 px-2 text-xs"
                    disabled={busy || account.archived}
                    onClick={() => {
                      setEditing(item)
                      setForm(item)
                      setError('')
                    }}
                  >
                    Edit
                  </Button>
                  <ConfirmButton
                    label="Hapus"
                    confirmLabel="Yakin?"
                    variant="ghost"
                    size="sm"
                    className="h-7 px-2 text-xs"
                    disabled={busy || account.archived}
                    onConfirm={async () => {
                      setError('')
                      try {
                        await remove.mutateAsync(item.id)
                        if (editing?.id === item.id) {
                          setEditing(null)
                          setForm(blank)
                        }
                      } catch (err) {
                        setError(err instanceof Error ? err.message : 'Gagal menghapus.')
                      }
                    }}
                  />
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}

/** Wrapper gabungan untuk kompatibilitas pemanggil tunggal. */
export function AccountManager({ accounts }: { accounts: JournalAccount[] }) {
  const busy = useIsMutating({ mutationKey: journalKey }) > 0
  const [subTab, setSubTab] = useState<'accounts' | 'capital'>('accounts')
  const [capitalAccountId, setCapitalAccountId] = useState<string | undefined>()
  return (
    <div className="space-y-4">
      <SegmentedControl
        ariaLabel="Kelola jurnal dan modal"
        value={subTab}
        onChange={setSubTab}
        disabled={busy}
        fullWidth
        options={[
          { value: 'accounts', label: 'Jurnal' },
          { value: 'capital', label: 'Deposit & Withdrawal' },
        ]}
      />
      {subTab === 'accounts' ? (
        <AccountsPanel
          accounts={accounts}
          onSelectCapital={(id) => {
            setCapitalAccountId(id)
            setSubTab('capital')
          }}
        />
      ) : (
        <CapitalPanel accounts={accounts} initialAccountId={capitalAccountId} />
      )}
    </div>
  )
}
