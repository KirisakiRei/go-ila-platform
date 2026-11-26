import { useState, type FormEvent } from 'react'
import { Alert, Button, Input, Select } from '@/components/ui'
import { fromWib, localWib, splitList, validateTrade } from './model'
import type { JournalAccount, TradeInput } from './types'

const numericFields = [
  ['entry_price', 'Harga entry'],
  ['exit_price', 'Harga exit'],
  ['stop_loss', 'Stop loss'],
  ['take_profit', 'Take profit'],
  ['size', 'Size'],
  ['net_pnl', 'Net P&L aktual'],
  ['risk_amount', 'Risiko nominal'],
  ['capital_snapshot', 'Modal saat entry'],
] as const

export function TradeForm({
  initial,
  accounts,
  pending,
  onSave,
  onCancel,
  editing = false,
}: {
  initial: TradeInput
  accounts: JournalAccount[]
  pending: boolean
  editing?: boolean
  onSave: (trade: TradeInput) => Promise<void>
  onCancel: () => void
}) {
  const [trade, setTrade] = useState(initial)
  const [error, setError] = useState('')
  const [advanced, setAdvanced] = useState(false)
  const [lists, setLists] = useState({
    tags: initial.tags.join('\n'),
    emotions: initial.emotions.join('\n'),
    mistakes: initial.mistakes.join('\n'),
    tradingview_urls: initial.tradingview_urls.join('\n'),
  })
  const update = <K extends keyof TradeInput>(key: K, value: TradeInput[K]) =>
    setTrade((t) => ({ ...t, [key]: value }))
  async function submit(event: FormEvent) {
    event.preventDefault()
    const input = {
      ...trade,
      tags: lists.tags === initial.tags.join('\n') ? initial.tags : splitList(lists.tags),
      emotions:
        lists.emotions === initial.emotions.join('\n')
          ? initial.emotions
          : splitList(lists.emotions),
      mistakes:
        lists.mistakes === initial.mistakes.join('\n')
          ? initial.mistakes
          : splitList(lists.mistakes),
      tradingview_urls: [
        ...new Set(
          lists.tradingview_urls
            .split('\n')
            .map((s) => s.trim())
            .filter(Boolean),
        ),
      ],
    }
    const message = validateTrade(input)
    if (message) {
      setError(message)
      return
    }
    setError('')
    try {
      await onSave({ ...input, asset: input.asset.trim().toUpperCase() })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Gagal menyimpan trade. Coba lagi.')
    }
  }
  const account = accounts.find((a) => a.id === trade.account_id)
  return (
    <form onSubmit={submit} className="space-y-5">
      <p className="text-sm text-muted-foreground">
        Draft boleh belum lengkap. Net P&L diisi dari hasil aktual setelah biaya; tidak dihitung
        otomatis dari lot.
      </p>
      {error && <Alert variant="destructive">{error}</Alert>}
      <fieldset disabled={pending || account?.archived} className="grid gap-4 sm:grid-cols-2">
        <label className="space-y-1 text-sm">
          Akun jurnal
          <Select
            value={trade.account_id}
            disabled={editing}
            onChange={(e) =>
              setTrade((t) => ({
                ...t,
                account_id: e.target.value,
                checklist: (accounts.find((a) => a.id === e.target.value)?.checklist ?? []).map(
                  (label) => ({ label, checked: false }),
                ),
              }))
            }
            required
          >
            <option value="">Pilih akun</option>
            {accounts
              .filter((a) => !a.archived || a.id === initial.account_id)
              .map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name} · {a.currency}
                </option>
              ))}
          </Select>
        </label>
        <label className="space-y-1 text-sm">
          Status
          <Select
            value={trade.status}
            onChange={(e) => update('status', e.target.value as TradeInput['status'])}
          >
            <option value="draft">Draft</option>
            <option value="open">Open</option>
            <option value="closed">Closed</option>
          </Select>
        </label>
        <label className="space-y-1 text-sm">
          Simbol
          <Input
            value={trade.asset}
            maxLength={40}
            onChange={(e) => update('asset', e.target.value)}
            placeholder="XAUUSD"
          />
        </label>
        <label className="space-y-1 text-sm">
          Arah
          <Select
            value={trade.side}
            onChange={(e) => update('side', e.target.value as TradeInput['side'])}
          >
            <option value="">Belum ditentukan</option>
            <option value="long">Buy / Long</option>
            <option value="short">Sell / Short</option>
          </Select>
        </label>
        {(['opened_at', 'closed_at'] as const).map((key) => (
          <label key={key} className="space-y-1 text-sm">
            {key === 'opened_at' ? 'Waktu masuk' : 'Waktu keluar'} (WIB)
            <Input
              type="datetime-local"
              value={localWib(trade[key])}
              onChange={(e) => update(key, fromWib(e.target.value))}
            />
          </label>
        ))}
        {numericFields.map(([key, label]) => (
          <label key={key} className="space-y-1 text-sm">
            {label}
            {['net_pnl', 'risk_amount', 'capital_snapshot'].includes(key) && account
              ? ` (${account.currency})`
              : ''}
            <Input
              inputMode="decimal"
              value={trade[key] ?? ''}
              onChange={(e) => update(key, e.target.value === '' ? null : e.target.value)}
              placeholder="0.00"
            />
          </label>
        ))}
        <label className="space-y-1 text-sm">
          Satuan size
          <Input
            value={trade.size_unit}
            maxLength={40}
            onChange={(e) => update('size_unit', e.target.value)}
            placeholder="lot / unit / kontrak"
          />
        </label>
        <label className="space-y-1 text-sm">
          Setup
          <Input
            value={trade.setup}
            maxLength={100}
            onChange={(e) => update('setup', e.target.value)}
            placeholder="Nama setup sendiri"
          />
        </label>
        <label className="space-y-1 text-sm">
          Tag (satu per baris)
          <textarea
            className="min-h-20 w-full rounded-md border border-input bg-background p-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            value={lists.tags}
            onChange={(e) => setLists({ ...lists, tags: e.target.value })}
            placeholder="London, breakout"
          />
        </label>
        <label className="space-y-1 text-sm">
          Sesi
          <Input
            value={trade.session}
            maxLength={100}
            onChange={(e) => update('session', e.target.value)}
            placeholder="London"
          />
        </label>
      </fieldset>
      <Button
        type="button"
        variant="outline"
        aria-expanded={advanced}
        aria-controls="journal-advanced"
        onClick={() => setAdvanced(!advanced)}
      >
        {advanced ? 'Sembunyikan advanced' : 'Advanced: evaluasi & bukti'}
      </Button>
      <fieldset
        id="journal-advanced"
        hidden={!advanced}
        disabled={pending || account?.archived}
        className="space-y-4"
      >
        <div className="grid gap-4 sm:grid-cols-2">
          <label className="space-y-1 text-sm">
            Emosi (satu per baris)
            <textarea
              className="min-h-20 w-full rounded-md border border-input bg-background p-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              value={lists.emotions}
              onChange={(e) => setLists({ ...lists, emotions: e.target.value })}
              placeholder="Tenang, FOMO"
            />
          </label>
          <label className="space-y-1 text-sm">
            Kesalahan (satu per baris)
            <textarea
              className="min-h-20 w-full rounded-md border border-input bg-background p-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              value={lists.mistakes}
              onChange={(e) => setLists({ ...lists, mistakes: e.target.value })}
            />
          </label>
        </div>
        <fieldset className="space-y-2">
          <legend className="mb-2 text-sm font-medium">Checklist akun (snapshot trade)</legend>
          {!trade.checklist.length && (
            <p className="text-xs text-muted-foreground">
              Atur template checklist pada akun jurnal.
            </p>
          )}
          {trade.checklist.map((item, i) => (
            <label key={item.label} className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={item.checked}
                className="h-4 w-4 accent-primary"
                onChange={(e) =>
                  update(
                    'checklist',
                    trade.checklist.map((c, index) =>
                      index === i ? { ...c, checked: e.target.checked } : c,
                    ),
                  )
                }
              />
              {item.label}
            </label>
          ))}
        </fieldset>
        <label className="block space-y-1 text-sm">
          Catatan
          <textarea
            className="min-h-28 w-full rounded-md border border-input bg-background p-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            value={trade.notes}
            maxLength={10000}
            onChange={(e) => update('notes', e.target.value)}
          />
        </label>
        <label className="block space-y-1 text-sm">
          TradingView sebelum / sesudah (satu URL per baris)
          <textarea
            className="min-h-20 w-full rounded-md border border-input bg-background p-3 text-sm focus-visible:ring-2 focus-visible:ring-ring"
            value={lists.tradingview_urls}
            onChange={(e) => setLists({ ...lists, tradingview_urls: e.target.value })}
            placeholder="https://www.tradingview.com/x/..."
          />
        </label>
        <p className="text-xs text-muted-foreground">
          Screenshot privat dapat diunggah pada detail trade setelah disimpan.
        </p>
      </fieldset>
      <div className="flex flex-wrap justify-end gap-2 border-t border-border pt-4">
        <Button type="button" variant="outline" onClick={onCancel} disabled={pending}>
          Batal
        </Button>
        <Button type="submit" disabled={account?.archived} isLoading={pending}>
          Simpan trade
        </Button>
      </div>
    </form>
  )
}
