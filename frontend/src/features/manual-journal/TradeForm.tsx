import { useEffect, useId, useMemo, useRef, useState, type FormEvent } from 'react'
import { Check, Clock, Plus, Trash2, X } from 'lucide-react'
import { Alert, Button, Input, Select } from '@/components/ui'
import { cn } from '@/lib/utils'
import {
  computeTradePreview,
  friendlyJournalError,
  fromWib,
  localWib,
  mergeChipDraft,
  normalizeDecimalInput,
  pnlTone,
  validateTradeFields,
  type TradeFieldKey,
} from './model'
import { validateChartImageInput } from './image'
import { Screenshots } from './Screenshots'
import type { JournalAccount, JournalStatus, TradeInput } from './types'
import { ChipInput, ImageDropzone, SegmentedControl } from './ui'

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
  const [fieldErrors, setFieldErrors] = useState<Partial<Record<TradeFieldKey, string>>>({})
  const [tvDraft, setTvDraft] = useState('')
  const [emotionDraft, setEmotionDraft] = useState('')
  const [mistakeDraft, setMistakeDraft] = useState('')
  const [tagDraft, setTagDraft] = useState('')
  const [queuedFiles, setQueuedFiles] = useState<File[]>([])
  const [fileError, setFileError] = useState('')
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

  const clearFieldError = (key: TradeFieldKey) => {
    setFieldErrors((prev) => {
      if (!prev[key]) return prev
      const next = { ...prev }
      delete next[key]
      return next
    })
  }

  const update = <K extends keyof TradeInput>(key: K, value: TradeInput[K]) => {
    setTrade((t) => ({ ...t, [key]: value }))
    clearFieldError(key as TradeFieldKey)
    if (key === 'side' || key === 'entry_price' || key === 'stop_loss' || key === 'take_profit') {
      clearFieldError('stop_loss')
      clearFieldError('take_profit')
    }
    if (key === 'opened_at') clearFieldError('closed_at')
    if (key === 'capital_snapshot') clearFieldError('risk_amount')
  }

  function handleStatusChange(next: JournalStatus) {
    setFieldErrors({})
    setError('')
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

  function queueFiles(incoming: File[]) {
    const valid: File[] = []
    let firstError = ''
    for (const file of incoming) {
      const err = validateChartImageInput(file)
      if (err) {
        if (!firstError) firstError = err
        continue
      }
      valid.push(file)
    }
    setFileError(firstError)
    if (valid.length) {
      setQueuedFiles((prev) => [...prev, ...valid].slice(0, 6))
    }
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
    const input: TradeInput = {
      ...flushed,
      asset: flushed.asset.trim().toUpperCase(),
      size_unit: flushed.size_unit.trim(),
      entry_price: normalizeDecimalInput(flushed.entry_price),
      stop_loss: normalizeDecimalInput(flushed.stop_loss),
      take_profit: normalizeDecimalInput(flushed.take_profit),
      exit_price: normalizeDecimalInput(flushed.exit_price),
      size: normalizeDecimalInput(flushed.size),
      risk_amount: normalizeDecimalInput(flushed.risk_amount),
      capital_snapshot: normalizeDecimalInput(flushed.capital_snapshot),
      net_pnl: normalizeDecimalInput(flushed.net_pnl, true),
    }
    const validation = validateTradeFields(input)
    if (validation.summary) {
      setFieldErrors(validation.errors)
      setTab(validation.firstTab)
      setError(validation.summary)
      requestAnimationFrame(() => {
        if (typeof errorRef.current?.scrollIntoView === 'function') {
          errorRef.current.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
        }
      })
      return
    }
    setFieldErrors({})
    setError('')
    try {
      await onSave(input, queuedFiles)
    } catch (err) {
      setError(friendlyJournalError(err, 'Gagal menyimpan trade. Coba lagi.'))
      requestAnimationFrame(() => {
        if (typeof errorRef.current?.scrollIntoView === 'function') {
          errorRef.current.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
        }
      })
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
  const errCls = (key: TradeFieldKey) =>
    fieldErrors[key] ? 'border-destructive focus-visible:ring-destructive' : ''
  const coreErrCount = (
    [
      'account_id',
      'asset',
      'side',
      'opened_at',
      'closed_at',
      'entry_price',
      'stop_loss',
      'take_profit',
      'exit_price',
      'size',
      'size_unit',
      'net_pnl',
    ] as const
  ).filter((k) => Boolean(fieldErrors[k])).length
  const reviewErrCount = (
    [
      'risk_amount',
      'capital_snapshot',
      'tradingview_urls',
      'emotions',
      'mistakes',
      'tags',
      'checklist',
    ] as const
  ).filter((k) => Boolean(fieldErrors[k])).length

  return (
    <form onSubmit={submit} noValidate className="space-y-4">
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
            invalid={Boolean(fieldErrors.side)}
            fullWidth
            disabled={disabled}
          />
          {fieldErrors.side && (
            <span role="alert" className="block text-[11px] text-destructive">
              {fieldErrors.side}
            </span>
          )}
        </div>
      </div>

      {/* Tab form: Inti Trade vs Evaluasi & Bukti */}
      <SegmentedControl
        ariaLabel="Bagian form trade"
        value={tab}
        onChange={handleTabChange}
        fullWidth
        options={[
          {
            value: 'core',
            label: 'Inti Trade',
            badge: coreErrCount ? `${coreErrCount} error` : undefined,
          },
          {
            value: 'review',
            label: 'Evaluasi & Bukti',
            badge: reviewErrCount ? `${reviewErrCount} error` : reviewCount || undefined,
          },
        ]}
      />

      {tab === 'core' && (
        <fieldset disabled={disabled} className="space-y-4">
          {/* Identitas: Jurnal & Pair */}
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="space-y-1">
              <label className="block space-y-1.5 text-xs font-medium text-muted-foreground">
                Jurnal
                <Select
                  value={trade.account_id}
                  disabled={editing}
                  aria-invalid={Boolean(fieldErrors.account_id) || undefined}
                  className={errCls('account_id')}
                  onChange={(e) => {
                    clearFieldError('account_id')
                    setTrade((t) => ({
                      ...t,
                      account_id: e.target.value,
                      checklist: (accounts.find((a) => a.id === e.target.value)?.checklist ?? []).map(
                        (label) => ({ label, checked: false }),
                      ),
                    }))
                  }}
                >
                  <option value="">Pilih jurnal</option>
                  {accounts
                    .filter((a) => !a.archived || a.id === initial.account_id)
                    .map((a) => (
                      <option key={a.id} value={a.id}>
                        {a.name} · {a.currency}
                      </option>
                    ))}
                </Select>
              </label>
              {fieldErrors.account_id && (
                <span role="alert" className="block text-[11px] text-destructive">
                  {fieldErrors.account_id}
                </span>
              )}
            </div>

            <div className="space-y-1.5">
              <label className="block space-y-1.5 text-xs font-medium text-muted-foreground">
                Pair / Aset
                <Input
                  value={trade.asset}
                  maxLength={40}
                  aria-invalid={Boolean(fieldErrors.asset) || undefined}
                  onChange={(e) => update('asset', e.target.value.toUpperCase())}
                  placeholder="XAUUSD, EURUSD, BTCUSDT"
                  className={cn('font-medium uppercase', errCls('asset'))}
                />
              </label>
              {fieldErrors.asset && (
                <span role="alert" className="block text-[11px] text-destructive">
                  {fieldErrors.asset}
                </span>
              )}
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
                aria-invalid={Boolean(fieldErrors.opened_at) || undefined}
                className={errCls('opened_at')}
                value={localWib(trade.opened_at)}
                onChange={(e) => update('opened_at', fromWib(e.target.value))}
              />
              {fieldErrors.opened_at && (
                <span role="alert" className="block text-[11px] text-destructive">
                  {fieldErrors.opened_at}
                </span>
              )}
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
                  aria-invalid={Boolean(fieldErrors.closed_at) || undefined}
                  className={errCls('closed_at')}
                  value={localWib(trade.closed_at)}
                  onChange={(e) => update('closed_at', fromWib(e.target.value))}
                />
                {fieldErrors.closed_at && (
                  <span role="alert" className="block text-[11px] text-destructive">
                    {fieldErrors.closed_at}
                  </span>
                )}
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
                <div key={key} className="space-y-1">
                  <label className="block space-y-1 text-xs text-muted-foreground">
                    {label}
                    <Input
                      inputMode="decimal"
                      aria-invalid={Boolean(fieldErrors[key]) || undefined}
                      className={cn('tabular-nums', errCls(key))}
                      value={trade[key] ?? ''}
                      onChange={(e) => update(key, e.target.value === '' ? null : e.target.value)}
                      placeholder="0.00"
                    />
                  </label>
                  {fieldErrors[key] && (
                    <span role="alert" className="block text-[11px] text-destructive">
                      {fieldErrors[key]}
                    </span>
                  )}
                </div>
              ))}
            </div>
          </div>

          {/* Blok Ukuran & Hasil */}
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="space-y-1.5">
              <span className="block text-xs font-medium text-muted-foreground">Lot / Size</span>
              <div className="flex gap-2">
                <Input
                  aria-label="Size"
                  inputMode="decimal"
                  aria-invalid={Boolean(fieldErrors.size) || undefined}
                  className={cn('tabular-nums', errCls('size'))}
                  value={trade.size ?? ''}
                  onChange={(e) => update('size', e.target.value === '' ? null : e.target.value)}
                  placeholder="0.10"
                />
                <Input
                  aria-label="Satuan size"
                  aria-invalid={Boolean(fieldErrors.size_unit) || undefined}
                  className={cn('w-24 shrink-0', errCls('size_unit'))}
                  value={trade.size_unit}
                  maxLength={40}
                  onChange={(e) => update('size_unit', e.target.value)}
                  placeholder="lot"
                />
              </div>
              {(fieldErrors.size || fieldErrors.size_unit) && (
                <span role="alert" className="block text-[11px] text-destructive">
                  {fieldErrors.size || fieldErrors.size_unit}
                </span>
              )}
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
                    aria-invalid={Boolean(fieldErrors.net_pnl) || undefined}
                    className={cn(
                      'text-base font-semibold tabular-nums',
                      trade.net_pnl !== null && trade.net_pnl !== '' && pnlTone(trade.net_pnl),
                      errCls('net_pnl'),
                    )}
                    value={trade.net_pnl ?? ''}
                    onChange={(e) => {
                      const raw = e.target.value.replace(/^\+/, '')
                      update('net_pnl', raw === '' ? null : raw)
                    }}
                    placeholder="120.50 atau -45.00"
                  />
                </label>
                {fieldErrors.net_pnl ? (
                  <span role="alert" className="block text-[11px] text-destructive">
                    {fieldErrors.net_pnl}
                  </span>
                ) : (
                  <span className="block text-[11px] font-normal text-muted-foreground">
                    Sudah termasuk komisi/swap
                  </span>
                )}
              </div>
            ) : (
              <div className="flex items-center rounded-lg border border-dashed border-border px-3.5 py-2.5 text-xs text-muted-foreground">
                {trade.status === 'draft'
                  ? 'Draft bisa dilengkapi nanti.'
                  : 'Isi exit & P&L saat posisi sudah close.'}
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
          <p className="text-xs text-muted-foreground">
            Semua bagian di sini opsional, dipakai untuk breakdown performa dan catatan pola.
          </p>

          {/* Setup & Sesi */}
          <div className="grid gap-3 sm:grid-cols-2">
            <label className="space-y-1.5 text-xs font-medium text-muted-foreground">
              Setup
              <Input
                value={trade.setup}
                maxLength={100}
                onChange={(e) => update('setup', e.target.value)}
                placeholder="Mis. Sweep H1, Breakout"
              />
            </label>

            <div className="space-y-1.5">
              <span className="block text-xs font-medium text-muted-foreground">Sesi</span>
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
            <div className="space-y-1">
              <label className="block space-y-1.5 text-xs font-medium text-muted-foreground">
                Risiko awal ({currency})
                <Input
                  inputMode="decimal"
                  aria-invalid={Boolean(fieldErrors.risk_amount) || undefined}
                  className={cn('tabular-nums', errCls('risk_amount'))}
                  value={trade.risk_amount ?? ''}
                  onChange={(e) =>
                    update('risk_amount', e.target.value === '' ? null : e.target.value)
                  }
                  placeholder="Opsional · contoh 25.00"
                />
              </label>
              {fieldErrors.risk_amount && (
                <span role="alert" className="block text-[11px] text-destructive">
                  {fieldErrors.risk_amount}
                </span>
              )}
            </div>
            <div className="space-y-1">
              <label className="block space-y-1.5 text-xs font-medium text-muted-foreground">
                Modal saat entry ({currency})
                <Input
                  inputMode="decimal"
                  aria-invalid={Boolean(fieldErrors.capital_snapshot) || undefined}
                  className={cn('tabular-nums', errCls('capital_snapshot'))}
                  value={trade.capital_snapshot ?? ''}
                  onChange={(e) =>
                    update('capital_snapshot', e.target.value === '' ? null : e.target.value)
                  }
                  placeholder="Opsional · contoh 1000.00"
                />
              </label>
              {fieldErrors.capital_snapshot && (
                <span role="alert" className="block text-[11px] text-destructive">
                  {fieldErrors.capital_snapshot}
                </span>
              )}
            </div>
          </div>

          {/* Chip inputs: Emosi, Mistake, Tag */}
          <ChipInput
            label="Psikologi / Emosi"
            values={trade.emotions}
            onChange={(next) => update('emotions', next)}
            draftValue={emotionDraft}
            onDraftChange={setEmotionDraft}
            placeholder="Ketik lalu Enter"
            suggestions={EMOTION_PRESETS}
            disabled={disabled}
            error={fieldErrors.emotions}
          />

          <ChipInput
            label="Mistake"
            values={trade.mistakes}
            onChange={(next) => update('mistakes', next)}
            draftValue={mistakeDraft}
            onDraftChange={setMistakeDraft}
            placeholder="Ketik lalu Enter"
            suggestions={MISTAKE_PRESETS}
            disabled={disabled}
            error={fieldErrors.mistakes}
          />

          <ChipInput
            label="Tag"
            values={trade.tags}
            onChange={(next) => update('tags', next)}
            draftValue={tagDraft}
            onDraftChange={setTagDraft}
            placeholder="Mis. H1, news, retest (Enter)"
            disabled={disabled}
            error={fieldErrors.tags}
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
                Belum ada checklist. Atur di Pengaturan → Jurnal.
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
            Catatan
            <textarea
              className="min-h-24 w-full rounded-lg border border-input bg-background p-3 text-sm text-foreground placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              value={trade.notes}
              maxLength={10000}
              onChange={(e) => update('notes', e.target.value)}
              placeholder="Alasan entry, eksekusi, atau hal yang perlu diperbaiki"
            />
          </label>

          {/* Link TradingView */}
          <div className="space-y-2">
            <span className="block text-xs font-medium text-muted-foreground">
              Link TradingView
            </span>
            <div className="flex gap-2">
              <Input
                type="text"
                inputMode="url"
                aria-invalid={Boolean(fieldErrors.tradingview_urls) || undefined}
                className={errCls('tradingview_urls')}
                value={tvDraft}
                onChange={(e) => {
                  setTvDraft(e.target.value)
                  clearFieldError('tradingview_urls')
                }}
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
            {fieldErrors.tradingview_urls && (
              <span role="alert" className="block text-[11px] text-destructive">
                {fieldErrors.tradingview_urls}
              </span>
            )}
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
              <span className="block text-xs font-medium text-muted-foreground">
                Screenshot chart
              </span>
              <ImageDropzone
                id={fileInputId}
                multiple
                disabled={disabled}
                ariaLabel="Pilih gambar"
                hint="Otomatis dikompres & diunggah saat simpan · PNG, JPG, WebP, GIF (maks 6)"
                onFiles={queueFiles}
              />
              {fileError && <Alert variant="destructive">{fileError}</Alert>}
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
