import { useEffect, useId, useMemo, useRef, useState, type FormEvent } from 'react'
import { Check, Clock, ImagePlus, Plus, Trash2, X } from 'lucide-react'
import { Alert, Button, Input, Select } from '@/components/ui'
import { cn } from '@/lib/utils'
import { computeTradePreview, fromWib, localWib, mergeChipDraft, pnlTone, validateTrade } from './model'
import { Screenshots } from './Screenshots'
import type { JournalAccount, JournalStatus, TradeInput } from './types'
import { ChipInput, SegmentedControl } from './ui'

const STATUS_OPTIONS = [
  { value: 'closed', label: 'Closed' },
  { value: 'open', label: 'Open' },
  { value: 'draft', label: 'Draft' },
] as const

const SIDE_OPTIONS = [
  { value: 'long', label: 'Buy / Long', tone: 'buy' as const },
  { value: 'short', label: 'Sell / Short', tone: 'sell' as const },
] as const

const SIZE_UNITS = ['lot', 'unit', 'kontrak'] as const
const SESSIONS = ['Asia', 'London', 'New York'] as const
const EMOTION_PRESETS = ['Tenang', 'Disiplin', 'Percaya diri', 'FOMO', 'Revenge', 'Ragu', 'Serakah'] as const
const MISTAKE_PRESETS = [
  'Entry telat',
  'Overtrade',
  'Geser SL',
  'Tanpa rencana',
  'Lot kebesaran',
  'Close terlalu cepat',
] as const

export function TradeForm({
  initial,
  accounts,
  recentAssets = [],
  pending,
  onSave,
  onCancel,
  editing = false,
  entryId,
}: {
  initial: TradeInput
  accounts: JournalAccount[]
  recentAssets?: string[]
  pending: boolean
  editing?: boolean
  entryId?: string
  onSave: (trade: TradeInput, queuedFiles?: File[]) => Promise<void>
  onCancel: () => void
}) {
  const [trade, setTrade] = useState<TradeInput>(initial)
  const [tab, setTab] = useState<'core' | 'review'>('core')
  const [error, setError] = useState('')
  const [tvDraft, setTvDraft] = useState('')
  const [emotionDraft, setEmotionDraft] = useState('')
  const [mistakeDraft, setMistakeDraft] = useState('')
  const [tagDraft, setTagDraft] = useState('')
  const [queuedFiles, setQueuedFiles] = useState<File[]>([])
  const fileInputId = useId()
  const errorRef = useRef<HTMLDivElement>(null)

  const previews = useMemo(
    () => queuedFiles.map((file) => ({ file, url: URL.createObjectURL(file) })),
    [queuedFiles],
  )

  useEffect(() => {
    return () => {
      for (const item of previews) URL.revokeObjectURL(item.url)
    }
  }, [previews])

  const update = <K extends keyof TradeInput>(key: K, value: TradeInput[K]) =>
    setTrade((t) => ({ ...t, [key]: value }))

  function handleStatusChange(next: JournalStatus) {
    setTrade((t) => {
      const updated: TradeInput = { ...t, status: next }
      if (next !== 'closed') {
        updated.closed_at = null
        updated.exit_price = null
        updated.net_pnl = null
      }
      if (next !== 'draft' && !updated.opened_at) {
        updated.opened_at = new Date().toISOString()
      }
      if (next !== 'draft' && !updated.side) {
        updated.side = 'long'
      }
      if (next === 'closed' && !updated.closed_at) {
        updated.closed_at = new Date().toISOString()
      }
      return updated
    })
  }

  function flushDraftInputs(current: TradeInput): TradeInput {
    const cleanTv = tvDraft.trim()
    const urls = cleanTv
      ? [...new Set([...current.tradingview_urls, cleanTv])]
      : current.tradingview_urls
    const emotions = mergeChipDraft(current.emotions, emotionDraft, 31)
    const mistakes = mergeChipDraft(current.mistakes, mistakeDraft, 31)
    const tags = mergeChipDraft(current.tags, tagDraft, 31)
    if (cleanTv) setTvDraft('')
    if (emotionDraft.trim()) setEmotionDraft('')
    if (mistakeDraft.trim()) setMistakeDraft('')
    if (tagDraft.trim()) setTagDraft('')
    if (
      urls !== current.tradingview_urls ||
      emotions !== current.emotions ||
      mistakes !== current.mistakes ||
      tags !== current.tags
    ) {
      const merged: TradeInput = {
        ...current,
        tradingview_urls: urls,
        emotions,
        mistakes,
        tags,
      }
      setTrade(merged)
      return merged
    }
    return current
  }

  function handleTabChange(next: 'core' | 'review') {
    if (next !== tab && tab === 'review') {
      flushDraftInputs(trade)
    }
    setTab(next)
  }

  function addTvUrl() {
    const clean = tvDraft.trim()
    if (!clean) return
    if (!trade.tradingview_urls.includes(clean)) {
      update('tradingview_urls', [...trade.tradingview_urls, clean])
    }
    setTvDraft('')
  }

  async function submit(event: FormEvent) {
    event.preventDefault()
    const flushed = flushDraftInputs(trade)
    const cleanedPnl =
      flushed.net_pnl !== null ? flushed.net_pnl.trim().replace(/^\+/, '') : null
    const input: TradeInput = {
      ...flushed,
      asset: flushed.asset.trim().toUpperCase(),
      net_pnl: cleanedPnl === '' ? null : cleanedPnl,
    }
    const message = validateTrade(input)
    if (message) {
      const reviewFieldError =
        message.startsWith('risk_amount') ||
        message.startsWith('capital_snapshot') ||
        message.includes('TradingView') ||
        message.includes('30 item')
      setTab(reviewFieldError ? 'review' : 'core')
      setError(message)
      requestAnimationFrame(() => {
        if (typeof errorRef.current?.scrollIntoView === 'function') {
          errorRef.current.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
        }
      })
      return
    }
    setError('')
    try {
      await onSave(input, queuedFiles)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Gagal menyimpan trade. Coba lagi.')
    }
  }

  const account = accounts.find((a) => a.id === trade.account_id)
  const currency = account?.currency ?? 'USD'
  const calc = useMemo(() => computeTradePreview(trade), [trade])
  const hasCalc =
    calc.plannedRR || calc.realizedR || calc.holdDuration || calc.riskPercent || calc.gainPercent
  const checkedCount = trade.checklist.filter((c) => c.checked).length
  const reviewCount =
    (trade.setup ? 1 : 0) +
    (trade.session ? 1 : 0) +
    trade.tags.length +
    trade.emotions.length +
    trade.mistakes.length +
    (trade.notes ? 1 : 0) +
    trade.tradingview_urls.length +
    queuedFiles.length

  const disabled = pending || Boolean(account?.archived)
  const isClosed = trade.status === 'closed'

  return (
    <form onSubmit={submit} className="space-y-4">
      {error && (
        <div ref={errorRef}>
          <Alert variant="destructive">{error}</Alert>
        </div>
      )}

      {/* Baris kendali cepat: Status + Arah Buy/Sell */}
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="space-y-1.5">
          <span className="block text-xs font-medium text-muted-foreground">Status trade</span>
          <SegmentedControl
            ariaLabel="Status trade"
            value={trade.status}
            onChange={handleStatusChange}
            options={STATUS_OPTIONS}
            fullWidth
            disabled={disabled}
          />
        </div>
        <div className="space-y-1.5">
          <span className="block text-xs font-medium text-muted-foreground">Arah posisi</span>
          <SegmentedControl
            ariaLabel="Arah posisi"
            value={trade.side}
            onChange={(v) => update('side', v)}
            options={SIDE_OPTIONS}
            fullWidth
            disabled={disabled}
          />
        </div>
      </div>

      {/* Tab form: Inti Trade vs Evaluasi & Bukti */}
      <SegmentedControl
        ariaLabel="Bagian form trade"
        value={tab}
        onChange={handleTabChange}
        fullWidth
        options={[
          { value: 'core', label: 'Inti Trade' },
          { value: 'review', label: 'Evaluasi & Bukti', badge: reviewCount || undefined },
        ]}
      />

      {tab === 'core' && (
        <fieldset disabled={disabled} className="space-y-4">
          {/* Identitas: Akun & Pair */}
          <div className="grid gap-3 sm:grid-cols-2">
            <label className="space-y-1.5 text-xs font-medium text-muted-foreground">
              Akun
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

            <div className="space-y-1.5">
              <label className="block space-y-1.5 text-xs font-medium text-muted-foreground">
                Pair / Aset
                <Input
                  value={trade.asset}
                  maxLength={40}
                  onChange={(e) => update('asset', e.target.value.toUpperCase())}
                  placeholder="XAUUSD, EURUSD, BTCUSDT"
                  className="font-medium uppercase"
                />
              </label>
              {recentAssets.length > 0 && (
                <div className="flex flex-wrap gap-1">
                  {recentAssets.slice(0, 5).map((sym) => (
                    <button
                      key={sym}
                      type="button"
                      onClick={() => update('asset', sym)}
                      className={cn(
                        'rounded border px-1.5 py-0.5 text-[11px] font-medium transition-colors',
                        trade.asset === sym
                          ? 'border-foreground/40 bg-foreground text-background'
                          : 'border-border text-muted-foreground hover:bg-secondary hover:text-foreground',
                      )}
                    >
                      {sym}
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* Waktu WIB */}
          <div className={cn('grid gap-3', isClosed ? 'sm:grid-cols-2' : 'grid-cols-1')}>
            <div className="space-y-1.5">
              <div className="flex items-center justify-between">
                <label
                  htmlFor="trade-opened-at"
                  className="text-xs font-medium text-muted-foreground"
                >
                  Waktu masuk (WIB)
                </label>
                <button
                  type="button"
                  onClick={() => update('opened_at', new Date().toISOString())}
                  className="inline-flex items-center gap-1 text-[11px] text-muted-foreground hover:text-foreground"
                >
                  <Clock className="h-3 w-3" aria-hidden="true" />
                  Sekarang
                </button>
              </div>
              <Input
                id="trade-opened-at"
                type="datetime-local"
                value={localWib(trade.opened_at)}
                onChange={(e) => update('opened_at', fromWib(e.target.value))}
              />
            </div>

            {isClosed && (
              <div className="space-y-1.5">
                <div className="flex items-center justify-between">
                  <label
                    htmlFor="trade-closed-at"
                    className="text-xs font-medium text-muted-foreground"
                  >
                    Waktu keluar (WIB)
                  </label>
                  <button
                    type="button"
                    onClick={() => update('closed_at', new Date().toISOString())}
                    className="inline-flex items-center gap-1 text-[11px] text-muted-foreground hover:text-foreground"
                  >
                    <Clock className="h-3 w-3" aria-hidden="true" />
                    Sekarang
                  </button>
                </div>
                <Input
                  id="trade-closed-at"
                  type="datetime-local"
                  value={localWib(trade.closed_at)}
                  onChange={(e) => update('closed_at', fromWib(e.target.value))}
                />
              </div>
            )}
          </div>

          {/* Blok Harga */}
          <div className="rounded-xl border border-border/80 bg-secondary/25 p-3.5">
            <p className="mb-2.5 text-xs font-medium text-muted-foreground">Level harga</p>
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
              {(
                [
                  ['entry_price', 'Entry'],
                  ['stop_loss', 'Stop loss'],
                  ['take_profit', 'Take profit'],
                  ...(isClosed ? ([['exit_price', 'Exit aktual']] as const) : []),
                ] as const
              ).map(([key, label]) => (
                <label key={key} className="space-y-1 text-xs text-muted-foreground">
                  {label}
                  <Input
                    inputMode="decimal"
                    className="tabular-nums"
                    value={trade[key] ?? ''}
                    onChange={(e) => update(key, e.target.value === '' ? null : e.target.value)}
                    placeholder="0.00"
                  />
                </label>
              ))}
            </div>
          </div>

          {/* Blok Ukuran & Hasil */}
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="space-y-1.5">
              <span className="block text-xs font-medium text-muted-foreground">Ukuran posisi</span>
              <div className="flex gap-2">
                <Input
                  aria-label="Size"
                  inputMode="decimal"
                  className="tabular-nums"
                  value={trade.size ?? ''}
                  onChange={(e) => update('size', e.target.value === '' ? null : e.target.value)}
                  placeholder="0.10"
                />
                <Input
                  aria-label="Satuan size"
                  className="w-24 shrink-0"
                  value={trade.size_unit}
                  maxLength={40}
                  onChange={(e) => update('size_unit', e.target.value)}
                  placeholder="lot"
                />
              </div>
              <div className="flex flex-wrap gap-1">
                {SIZE_UNITS.map((unit) => (
                  <button
                    key={unit}
                    type="button"
                    onClick={() => update('size_unit', unit)}
                    className={cn(
                      'rounded border px-2 py-0.5 text-[11px] font-medium transition-colors',
                      trade.size_unit === unit
                        ? 'border-foreground/40 bg-foreground text-background'
                        : 'border-border text-muted-foreground hover:bg-secondary hover:text-foreground',
                    )}
                  >
                    {unit}
                  </button>
                ))}
              </div>
            </div>

            {isClosed ? (
              <div className="space-y-1.5">
                <label className="block space-y-1.5 text-xs font-medium text-muted-foreground">
                  Net P&amp;L aktual ({currency})
                  <Input
                    inputMode="decimal"
                    className={cn(
                      'text-base font-semibold tabular-nums',
                      trade.net_pnl !== null && trade.net_pnl !== '' && pnlTone(trade.net_pnl),
                    )}
                    value={trade.net_pnl ?? ''}
                    onChange={(e) => {
                      const raw = e.target.value.replace(/^\+/, '')
                      update('net_pnl', raw === '' ? null : raw)
                    }}
                    placeholder="120.50 atau -45.00"
                  />
                </label>
                <span className="block text-[11px] font-normal text-muted-foreground">
                  Hasil bersih setelah komisi/swap
                </span>
              </div>
            ) : (
              <div className="flex items-center rounded-lg border border-dashed border-border px-3.5 py-2.5 text-xs text-muted-foreground">
                {trade.status === 'draft'
                  ? 'Mode Draft: simpan sebagian dulu dan lengkapi nanti.'
                  : 'Mode Open: waktu keluar & Net P&L diisi saat posisi ditutup.'}
              </div>
            )}
          </div>

          {/* Live Auto-Calculation Strip */}
          {hasCalc && (
            <div
              aria-label="Preview kalkulasi otomatis"
              className="flex flex-wrap items-center gap-x-4 gap-y-1.5 rounded-lg border border-border/80 bg-secondary/40 px-3.5 py-2 text-xs"
            >
              {calc.plannedRR && (
                <span>
                  <span className="text-muted-foreground">Plan RR</span>{' '}
                  <strong className="font-semibold tabular-nums">{calc.plannedRR}</strong>
                </span>
              )}
              {calc.realizedR && (
                <span>
                  <span className="text-muted-foreground">Realized</span>{' '}
                  <strong className={cn('font-semibold tabular-nums', pnlTone(trade.net_pnl))}>
                    {calc.realizedR}
                  </strong>
                </span>
              )}
              {calc.holdDuration && (
                <span>
                  <span className="text-muted-foreground">Durasi</span>{' '}
                  <strong className="font-semibold tabular-nums">{calc.holdDuration}</strong>
                </span>
              )}
              {calc.riskPercent && (
                <span>
                  <span className="text-muted-foreground">Risiko</span>{' '}
                  <strong className="font-semibold tabular-nums">{calc.riskPercent}</strong>
                </span>
              )}
              {calc.gainPercent && (
                <span>
                  <span className="text-muted-foreground">Gain</span>{' '}
                  <strong className={cn('font-semibold tabular-nums', pnlTone(trade.net_pnl))}>
                    {calc.gainPercent}
                  </strong>
                </span>
              )}
            </div>
          )}
        </fieldset>
      )}

      {tab === 'review' && (
        <fieldset disabled={disabled} className="space-y-4">
          {/* Setup & Sesi */}
          <div className="grid gap-3 sm:grid-cols-2">
            <label className="space-y-1.5 text-xs font-medium text-muted-foreground">
              Setup / Strategi
              <Input
                value={trade.setup}
                maxLength={100}
                onChange={(e) => update('setup', e.target.value)}
                placeholder="Mis. Liquidity sweep, Breakout"
              />
            </label>

            <div className="space-y-1.5">
              <span className="block text-xs font-medium text-muted-foreground">Sesi trading</span>
              <div className="flex flex-wrap gap-1.5">
                {SESSIONS.map((s) => {
                  const active = trade.session === s
                  return (
                    <button
                      key={s}
                      type="button"
                      aria-pressed={active}
                      onClick={() => update('session', active ? '' : s)}
                      className={cn(
                        'h-9 rounded-md border px-3 text-xs font-medium transition-[color,background-color,transform] duration-150 active:scale-[0.97]',
                        active
                          ? 'border-foreground/40 bg-foreground text-background'
                          : 'border-input bg-background text-muted-foreground hover:bg-secondary hover:text-foreground',
                      )}
                    >
                      {s}
                    </button>
                  )
                })}
                <Input
                  aria-label="Sesi kustom"
                  value={SESSIONS.includes(trade.session as (typeof SESSIONS)[number]) ? '' : trade.session}
                  maxLength={100}
                  onChange={(e) => update('session', e.target.value)}
                  placeholder="Lainnya…"
                  className="h-9 min-w-[90px] flex-1 text-xs"
                />
              </div>
            </div>
          </div>

          {/* Risiko & Modal acuan opsional */}
          <div className="grid gap-3 sm:grid-cols-2">
            <label className="space-y-1.5 text-xs font-medium text-muted-foreground">
              Risiko awal ({currency})
              <Input
                inputMode="decimal"
                className="tabular-nums"
                value={trade.risk_amount ?? ''}
                onChange={(e) =>
                  update('risk_amount', e.target.value === '' ? null : e.target.value)
                }
                placeholder="Opsional · untuk hitung R"
              />
            </label>
            <label className="space-y-1.5 text-xs font-medium text-muted-foreground">
              Modal saat entry ({currency})
              <Input
                inputMode="decimal"
                className="tabular-nums"
                value={trade.capital_snapshot ?? ''}
                onChange={(e) =>
                  update('capital_snapshot', e.target.value === '' ? null : e.target.value)
                }
                placeholder="Opsional · untuk hitung %"
              />
            </label>
          </div>

          {/* Chip inputs: Emosi, Kesalahan, Tag */}
          <ChipInput
            label="Kondisi emosi"
            values={trade.emotions}
            onChange={(next) => update('emotions', next)}
            draftValue={emotionDraft}
            onDraftChange={setEmotionDraft}
            placeholder="Ketik lalu Enter"
            suggestions={EMOTION_PRESETS}
            disabled={disabled}
          />

          <ChipInput
            label="Kesalahan (mistake)"
            values={trade.mistakes}
            onChange={(next) => update('mistakes', next)}
            draftValue={mistakeDraft}
            onDraftChange={setMistakeDraft}
            placeholder="Ketik lalu Enter"
            suggestions={MISTAKE_PRESETS}
            disabled={disabled}
          />

          <ChipInput
            label="Tag kustom"
            values={trade.tags}
            onChange={(next) => update('tags', next)}
            draftValue={tagDraft}
            onDraftChange={setTagDraft}
            placeholder="Mis. H1, news, retest (Enter)"
            disabled={disabled}
          />

          {/* Checklist */}
          <div className="space-y-2 rounded-xl border border-border/80 bg-secondary/25 p-3.5">
            <div className="flex items-center justify-between gap-2">
              <span className="text-xs font-medium text-muted-foreground">Pre-trade checklist</span>
              {trade.checklist.length > 0 && (
                <span className="text-[11px] font-medium tabular-nums text-muted-foreground">
                  {checkedCount}/{trade.checklist.length} terpenuhi
                </span>
              )}
            </div>
            {!trade.checklist.length ? (
              <p className="text-xs text-muted-foreground">
                Belum ada checklist. Tambahkan template di Pengaturan → Akun.
              </p>
            ) : (
              <div className="space-y-1.5">
                {trade.checklist.map((item, i) => (
                  <label
                    key={item.label}
                    className={cn(
                      'flex cursor-pointer items-center gap-2.5 rounded-lg border px-3 py-2 text-sm transition-colors focus-within:ring-2 focus-within:ring-ring focus-within:ring-offset-1',
                      item.checked
                        ? 'border-emerald-500/30 bg-emerald-500/[0.06] text-foreground'
                        : 'border-border/70 bg-background text-muted-foreground',
                    )}
                  >
                    <input
                      type="checkbox"
                      checked={item.checked}
                      className="sr-only"
                      onChange={(e) =>
                        update(
                          'checklist',
                          trade.checklist.map((c, index) =>
                            index === i ? { ...c, checked: e.target.checked } : c,
                          ),
                        )
                      }
                    />
                    <span
                      className={cn(
                        'inline-flex h-4 w-4 shrink-0 items-center justify-center rounded border transition-colors',
                        item.checked
                          ? 'border-emerald-600 bg-emerald-600 text-white dark:border-emerald-500 dark:bg-emerald-500'
                          : 'border-input bg-background',
                      )}
                      aria-hidden="true"
                    >
                      {item.checked && <Check className="h-3 w-3" />}
                    </span>
                    <span className="min-w-0 flex-1 text-xs sm:text-sm">{item.label}</span>
                  </label>
                ))}
              </div>
            )}
          </div>

          {/* Catatan refleksi */}
          <label className="block space-y-1.5 text-xs font-medium text-muted-foreground">
            Catatan &amp; refleksi
            <textarea
              className="min-h-24 w-full rounded-lg border border-input bg-background p-3 text-sm text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              value={trade.notes}
              maxLength={10000}
              onChange={(e) => update('notes', e.target.value)}
              placeholder="Apa yang berjalan sesuai rencana, apa yang perlu diperbaiki?"
            />
          </label>

          {/* Link TradingView */}
          <div className="space-y-2">
            <span className="block text-xs font-medium text-muted-foreground">
              Tautan TradingView
            </span>
            <div className="flex gap-2">
              <Input
                type="text"
                inputMode="url"
                value={tvDraft}
                onChange={(e) => setTvDraft(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault()
                    addTvUrl()
                  }
                }}
                placeholder="https://www.tradingview.com/x/..."
              />
              <Button type="button" variant="outline" size="sm" className="h-10 shrink-0" onClick={addTvUrl}>
                <Plus className="h-4 w-4" aria-hidden="true" />
                Tambah
              </Button>
            </div>
            {trade.tradingview_urls.length > 0 && (
              <ul className="space-y-1.5">
                {trade.tradingview_urls.map((url) => (
                  <li
                    key={url}
                    className="flex items-center justify-between gap-2 rounded-lg border border-border bg-secondary/30 px-3 py-1.5 text-xs"
                  >
                    <span className="min-w-0 truncate text-foreground">{url}</span>
                    <button
                      type="button"
                      aria-label={`Hapus ${url}`}
                      onClick={() =>
                        update(
                          'tradingview_urls',
                          trade.tradingview_urls.filter((u) => u !== url),
                        )
                      }
                      className="rounded p-1 text-muted-foreground hover:text-foreground"
                    >
                      <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>

          {/* Screenshot: langsung tersedia saat edit maupun tambah baru */}
          {editing && entryId ? (
            <Screenshots entryId={entryId} readOnly={Boolean(account?.archived)} />
          ) : (
            <div className="space-y-2.5 border-t border-border pt-4">
              <div className="flex items-center justify-between gap-2">
                <div>
                  <span className="block text-xs font-medium text-muted-foreground">
                    Screenshot chart (opsional)
                  </span>
                  <span className="text-[11px] text-muted-foreground">
                    Diunggah otomatis saat trade disimpan · maks 5 MiB
                  </span>
                </div>
                <label
                  htmlFor={fileInputId}
                  className="inline-flex cursor-pointer items-center gap-1.5 rounded-md border border-input px-3 py-1.5 text-xs font-medium transition-colors hover:bg-secondary focus-within:ring-2 focus-within:ring-ring focus-within:ring-offset-1"
                >
                  <ImagePlus className="h-3.5 w-3.5" aria-hidden="true" />
                  Pilih gambar
                  <input
                    id={fileInputId}
                    type="file"
                    multiple
                    accept="image/png,image/jpeg,image/webp,image/gif"
                    className="sr-only"
                    onChange={(e) => {
                      const next = Array.from(e.target.files ?? [])
                      if (next.length) setQueuedFiles((prev) => [...prev, ...next].slice(0, 6))
                      e.target.value = ''
                    }}
                  />
                </label>
              </div>
              {previews.length > 0 && (
                <ul className="grid grid-cols-2 gap-2 sm:grid-cols-3">
                  {previews.map((item, idx) => (
                    <li
                      key={`${item.file.name}-${idx}`}
                      className="relative overflow-hidden rounded-lg border border-border bg-secondary/30"
                    >
                      <img
                        src={item.url}
                        alt={item.file.name}
                        className="aspect-video w-full object-cover"
                      />
                      <button
                        type="button"
                        aria-label={`Hapus ${item.file.name}`}
                        onClick={() =>
                          setQueuedFiles((prev) => prev.filter((_, i) => i !== idx))
                        }
                        className="absolute right-1.5 top-1.5 inline-flex h-6 w-6 items-center justify-center rounded-full bg-background/85 text-foreground backdrop-blur-sm"
                      >
                        <X className="h-3.5 w-3.5" aria-hidden="true" />
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}
        </fieldset>
      )}

      {/* Footer aksi */}
      <div className="flex items-center justify-between gap-2 border-t border-border pt-4">
        <span className="text-xs text-muted-foreground">
          {tab === 'core' ? 'Langkah 1/2 · Inti' : 'Langkah 2/2 · Evaluasi'}
        </span>
        <div className="flex items-center gap-2">
          <Button type="button" variant="outline" onClick={onCancel} disabled={pending}>
            Batal
          </Button>
          <Button type="submit" disabled={Boolean(account?.archived)} isLoading={pending}>
            Simpan trade
          </Button>
        </div>
      </div>
    </form>
  )
}
