import { useState } from 'react'
import {
  CheckCircle2,
  ChevronLeft,
  ChevronRight,
  Circle,
  ExternalLink,
  Pencil,
} from 'lucide-react'
import { Button } from '@/components/ui'
import { cn } from '@/lib/utils'
import {
  computeTradePreview,
  formatAmount,
  formatSignedAmount,
  pnlTone,
  safeLink,
  tradeDay,
  wibDay,
  wibTime,
} from './model'
import type { JournalAccount, JournalGroup, JournalTrade } from './types'
import { ConfirmButton } from './ui'

function sideBadge(side: JournalTrade['side']) {
  if (side === 'long') {
    return {
      label: 'BUY',
      className:
        'bg-emerald-500/12 text-emerald-700 border-emerald-500/25 dark:text-emerald-300',
    }
  }
  if (side === 'short') {
    return {
      label: 'SELL',
      className: 'bg-rose-500/12 text-rose-700 border-rose-500/25 dark:text-rose-300',
    }
  }
  return {
    label: '—',
    className: 'bg-secondary text-muted-foreground border-border',
  }
}

function statusPill(status: JournalTrade['status']) {
  if (status === 'open') {
    return 'bg-sky-500/12 text-sky-700 dark:text-sky-300'
  }
  if (status === 'draft') {
    return 'bg-secondary text-muted-foreground'
  }
  return 'bg-secondary/80 text-foreground'
}

export function TradeCards({
  trades,
  accounts,
  onSelect,
}: {
  trades: JournalTrade[]
  accounts: JournalAccount[]
  onSelect: (trade: JournalTrade) => void
}) {
  return (
    <div className="space-y-2">
      {trades.map((trade) => {
        const account = accounts.find((a) => a.id === trade.account_id)
        const side = sideBadge(trade.side)
        const preview = computeTradePreview(trade)
        const hasPnl = trade.net_pnl !== null && trade.status === 'closed' && account
        return (
          <button
            key={trade.id}
            type="button"
            onClick={() => onSelect(trade)}
            className="hover-surface flex w-full items-start justify-between gap-3 rounded-xl border border-border/80 bg-card p-3.5 text-left transition-[background-color,transform] duration-150 ease-[var(--ease-out)] active:scale-[0.99] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <span className="min-w-0 space-y-1">
              <span className="flex flex-wrap items-center gap-1.5">
                <span className="font-semibold text-foreground">
                  {trade.asset || 'Draft tanpa simbol'}
                </span>
                <span
                  className={cn(
                    'rounded border px-1.5 py-0.5 text-[10px] font-semibold',
                    side.className,
                  )}
                >
                  {side.label}
                </span>
                <span
                  className={cn(
                    'rounded px-1.5 py-0.5 text-[10px] font-medium capitalize',
                    statusPill(trade.status),
                  )}
                >
                  {trade.status}
                </span>
              </span>
              <span className="block truncate text-xs text-muted-foreground">
                {trade.setup || 'Tanpa setup'} · {tradeDay(trade) || 'Belum bertanggal'} WIB
              </span>
            </span>
            <span className="shrink-0 text-right tabular-nums">
              <span
                className={cn(
                  'block text-sm font-semibold',
                  hasPnl ? pnlTone(trade.net_pnl) : 'text-muted-foreground',
                )}
              >
                {hasPnl ? formatSignedAmount(trade.net_pnl!, account.currency) : '—'}
              </span>
              {preview.realizedR && (
                <span className="mt-0.5 block text-[11px] text-muted-foreground">
                  {preview.realizedR}
                </span>
              )}
            </span>
          </button>
        )
      })}
    </div>
  )
}

export function CalendarView({
  trades,
  month,
  accounts,
  currency,
  onSelect,
  onMonth,
  calendar,
}: {
  trades: JournalTrade[]
  month: string
  accounts: JournalAccount[]
  currency?: string
  onSelect: (trade: JournalTrade) => void
  onMonth: (month: string) => void
  calendar: Record<string, JournalGroup>
}) {
  const todayStr = wibDay(new Date().toISOString())
  const currentMonth = todayStr.slice(0, 7)
  const [selected, setSelected] = useState('')

  const date = new Date(`${month}-01T00:00:00+07:00`)
  const year = Number(month.slice(0, 4))
  const m = Number(month.slice(5, 7))
  const count = new Date(Date.UTC(year, m, 0)).getUTCDate()
  const offset = (new Date(Date.UTC(year, m - 1, 1)).getUTCDay() + 6) % 7

  const days = Object.entries(calendar).map(([day, value]) => ({
    day,
    net: Number(value.net_pnl),
    count: value.trades,
  }))

  const monthDays = days.filter((d) => d.day.startsWith(`${month}-`))
  const monthNet = monthDays.reduce((acc, d) => acc + d.net, 0)
  const monthTrades = monthDays.reduce((acc, d) => acc + d.count, 0)
  const resolvedCurrency = currency || accounts[0]?.currency || 'USD'

  function move(delta: number) {
    const next = new Date(Date.UTC(year, m - 1 + delta, 1)).toISOString().slice(0, 7)
    onMonth(next)
    setSelected('')
  }

  const daily = trades.filter((t) => tradeDay(t) === selected)

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-1.5">
          <Button
            variant="outline"
            size="sm"
            className="h-8 w-8 p-0"
            onClick={() => move(-1)}
            aria-label="Bulan sebelumnya"
          >
            <ChevronLeft className="h-4 w-4" aria-hidden="true" />
          </Button>
          <h2 className="min-w-[140px] text-center text-sm font-semibold capitalize">
            {new Intl.DateTimeFormat('id-ID', {
              month: 'long',
              year: 'numeric',
              timeZone: 'Asia/Jakarta',
            }).format(date)}
          </h2>
          <Button
            variant="outline"
            size="sm"
            className="h-8 w-8 p-0"
            onClick={() => move(1)}
            aria-label="Bulan berikutnya"
          >
            <ChevronRight className="h-4 w-4" aria-hidden="true" />
          </Button>
          {month !== currentMonth && (
            <Button
              variant="ghost"
              size="sm"
              className="h-8 px-2.5 text-xs"
              onClick={() => {
                onMonth(currentMonth)
                setSelected('')
              }}
            >
              Bulan ini
            </Button>
          )}
        </div>

        <div className="flex items-center gap-2 text-xs tabular-nums text-muted-foreground">
          <span>{monthTrades} closed</span>
          <span>·</span>
          <span className={cn('font-semibold', monthTrades ? pnlTone(monthNet) : 'text-foreground')}>
            {monthTrades
              ? formatSignedAmount(monthNet, resolvedCurrency)
              : formatAmount(0, resolvedCurrency)}
          </span>
        </div>
      </div>

      <div className="overflow-hidden rounded-xl border border-border/80 bg-card">
        <div className="grid grid-cols-7 border-b border-border bg-secondary/35">
          {['Sen', 'Sel', 'Rab', 'Kam', 'Jum', 'Sab', 'Min'].map((d) => (
            <div key={d} className="py-2 text-center text-[11px] font-medium text-muted-foreground">
              {d}
            </div>
          ))}
        </div>
        <div className="grid grid-cols-7">
          {Array.from({ length: offset }, (_, i) => (
            <div
              key={`empty-${i}`}
              aria-hidden="true"
              className="min-h-[68px] border-t border-border/60 bg-secondary/15 sm:min-h-[88px]"
            />
          ))}
          {Array.from({ length: count }, (_, i) => {
            const day = `${month}-${String(i + 1).padStart(2, '0')}`
            const result = days.find((d) => d.day === day)
            const all = trades.filter((t) => tradeDay(t) === day)
            const isSelected = selected === day
            const isToday = day === todayStr

            const cellTone = result
              ? result.net > 0
                ? 'bg-emerald-500/[0.08] dark:bg-emerald-500/[0.11]'
                : result.net < 0
                  ? 'bg-rose-500/[0.08] dark:bg-rose-500/[0.11]'
                  : 'bg-secondary/35'
              : ''

            const compactNet = result
              ? `${result.net > 0 ? '+' : result.net < 0 ? '−' : ''}${new Intl.NumberFormat(
                  'id-ID',
                  { notation: 'compact', maximumFractionDigits: 1 },
                ).format(Math.abs(result.net))}`
              : ''

            return (
              <button
                key={day}
                type="button"
                aria-pressed={isSelected}
                aria-label={`${day} WIB, ${all.length} trade${result ? `, net P&L ${result.net}` : ''}`}
                onClick={() => setSelected(day)}
                className={cn(
                  'hover-surface relative flex min-h-[68px] min-w-0 flex-col justify-between border-t border-r border-border/60 p-1.5 text-left transition-[background-color,box-shadow] duration-150 last:border-r-0 focus-visible:z-10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:min-h-[88px] sm:p-2.5',
                  cellTone,
                  isSelected && 'z-10 ring-2 ring-inset ring-foreground/60',
                )}
              >
                <span className="flex items-center justify-between gap-1">
                  <span
                    className={cn(
                      'inline-flex h-5 w-5 items-center justify-center rounded-full text-[11px] tabular-nums',
                      isToday
                        ? 'bg-foreground font-semibold text-background'
                        : 'text-muted-foreground',
                    )}
                  >
                    {i + 1}
                  </span>
                  {all.length > 0 && (
                    <span className="hidden text-[10px] tabular-nums text-muted-foreground sm:inline">
                      {all.length}t
                    </span>
                  )}
                </span>

                {result ? (
                  <span
                    className={cn(
                      'mt-1 block truncate text-[11px] font-semibold tabular-nums sm:text-xs',
                      pnlTone(result.net),
                    )}
                  >
                    {compactNet}
                  </span>
                ) : all.length > 0 ? (
                  <span className="mt-1 block truncate text-[10px] text-muted-foreground">
                    {all.length} aktif
                  </span>
                ) : null}
              </button>
            )
          })}
        </div>
      </div>

      {selected && (
        <section
          aria-live="polite"
          className="space-y-3 rounded-xl border border-border/80 bg-secondary/20 p-3.5 sm:p-4"
        >
          <div className="flex flex-wrap items-baseline justify-between gap-2">
            <h3 className="text-sm font-semibold">Rekap {selected} · WIB</h3>
            <p className="text-xs tabular-nums text-muted-foreground">
              {daily.length} trade · net{' '}
              <span className={cn('font-semibold', pnlTone(calendar[selected]?.net_pnl ?? 0))}>
                {formatSignedAmount(
                  calendar[selected]?.net_pnl ?? '0',
                  accounts.find((a) => a.id === daily[0]?.account_id)?.currency ?? resolvedCurrency,
                )}
              </span>
            </p>
          </div>
          {daily.length ? (
            <TradeCards trades={daily} accounts={accounts} onSelect={onSelect} />
          ) : (
            <p className="py-3 text-xs text-muted-foreground">
              Tidak ada trade pada tanggal ini.
            </p>
          )}
        </section>
      )}
    </div>
  )
}

export function JournalTable({
  trades,
  accounts,
  onSelect,
}: {
  trades: JournalTrade[]
  accounts: JournalAccount[]
  onSelect: (trade: JournalTrade) => void
}) {
  const [page, setPage] = useState(1)
  const pages = Math.max(1, Math.ceil(trades.length / 20))
  const current = Math.min(page, pages)
  const items = trades.slice((current - 1) * 20, current * 20)

  return (
    <>
      <div className="sm:hidden">
        <TradeCards trades={items} accounts={accounts} onSelect={onSelect} />
      </div>
      <div className="hidden overflow-x-auto rounded-xl border border-border/80 bg-card sm:block">
        <table className="w-full text-left text-sm">
          <caption className="sr-only">Daftar trade jurnal; semua waktu WIB</caption>
          <thead className="border-b border-border bg-secondary/40 text-xs text-muted-foreground">
            <tr>
              {['Trade', 'Waktu WIB', 'Setup', 'Status', 'R', 'Net P&L'].map((h, i) => (
                <th
                  key={h}
                  scope="col"
                  className={cn(
                    'whitespace-nowrap px-4 py-2.5 font-medium',
                    i >= 4 && 'text-right',
                  )}
                >
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {items.map((t) => {
              const account = accounts.find((a) => a.id === t.account_id)
              const timestamp = t.status === 'closed' ? t.closed_at : t.opened_at
              const side = sideBadge(t.side)
              const preview = computeTradePreview(t)
              const hasPnl = account && t.status === 'closed' && t.net_pnl !== null
              return (
                <tr
                  key={t.id}
                  onClick={() => onSelect(t)}
                  className="hover-surface cursor-pointer border-t border-border/70 transition-colors duration-150"
                >
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <button
                        type="button"
                        className="rounded font-semibold text-foreground underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                        onClick={(e) => {
                          e.stopPropagation()
                          onSelect(t)
                        }}
                      >
                        {t.asset || 'Draft tanpa simbol'}
                      </button>
                      <span
                        className={cn(
                          'rounded border px-1.5 py-0.5 text-[10px] font-semibold',
                          side.className,
                        )}
                      >
                        {side.label}
                      </span>
                    </div>
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 text-xs tabular-nums text-muted-foreground">
                    {timestamp ? wibTime(timestamp) : '—'}
                  </td>
                  <td className="max-w-[180px] truncate px-4 py-3 text-xs text-foreground">
                    {t.setup || <span className="text-muted-foreground">—</span>}
                  </td>
                  <td className="px-4 py-3">
                    <span
                      className={cn(
                        'inline-flex rounded px-2 py-0.5 text-[11px] font-medium capitalize',
                        statusPill(t.status),
                      )}
                    >
                      {t.status}
                    </span>
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 text-right text-xs tabular-nums text-muted-foreground">
                    {preview.realizedR ?? '—'}
                  </td>
                  <td
                    className={cn(
                      'whitespace-nowrap px-4 py-3 text-right text-sm font-semibold tabular-nums',
                      hasPnl ? pnlTone(t.net_pnl) : 'text-muted-foreground',
                    )}
                  >
                    {hasPnl ? formatSignedAmount(t.net_pnl!, account.currency) : '—'}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
      {pages > 1 && (
        <div className="mt-3 flex items-center justify-between gap-2 text-xs text-muted-foreground">
          <span className="tabular-nums">
            {trades.length} trade · halaman {current}/{pages}
          </span>
          <div className="flex gap-1.5">
            <Button
              size="sm"
              variant="outline"
              disabled={current === 1}
              onClick={() => setPage(current - 1)}
            >
              Sebelumnya
            </Button>
            <Button
              size="sm"
              variant="outline"
              disabled={current === pages}
              onClick={() => setPage(current + 1)}
            >
              Berikutnya
            </Button>
          </div>
        </div>
      )}
    </>
  )
}

export function TradeDetail({
  trade,
  account,
  onEdit,
  onDelete,
  deleting,
  children,
}: {
  trade: JournalTrade
  account?: JournalAccount
  onEdit: () => void
  onDelete: () => void
  deleting: boolean
  children?: React.ReactNode
}) {
  const side = sideBadge(trade.side)
  const calc = computeTradePreview(trade)
  const hasPnl = trade.net_pnl !== null && account
  const checkedCount = trade.checklist.filter((c) => c.checked).length

  return (
    <div className="space-y-5">
      {/* Hero ringkasan */}
      <div className="flex flex-wrap items-start justify-between gap-3 rounded-xl border border-border/80 bg-secondary/25 p-4">
        <div className="space-y-1.5">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="text-lg font-semibold tracking-tight">
              {trade.asset || 'Draft'}
            </h3>
            <span
              className={cn(
                'rounded border px-2 py-0.5 text-[11px] font-semibold',
                side.className,
              )}
            >
              {side.label}
            </span>
            <span
              className={cn(
                'rounded px-2 py-0.5 text-[11px] font-medium capitalize',
                statusPill(trade.status),
              )}
            >
              {trade.status}
            </span>
          </div>
          <p className="text-xs text-muted-foreground">
            {account?.name ?? 'Jurnal'}
            {trade.setup ? ` · ${trade.setup}` : ''}
            {trade.session ? ` · Sesi ${trade.session}` : ''}
          </p>
        </div>

        <div className="text-right tabular-nums">
          <p
            className={cn(
              'text-xl font-semibold',
              hasPnl ? pnlTone(trade.net_pnl) : 'text-muted-foreground',
            )}
          >
            {hasPnl ? formatSignedAmount(trade.net_pnl!, account.currency) : '—'}
          </p>
          <p className="mt-0.5 text-xs text-muted-foreground">
            {[calc.realizedR, calc.gainPercent, calc.holdDuration].filter(Boolean).join(' · ') ||
              'Belum ada metrik hasil'}
          </p>
        </div>
      </div>

      {/* Grid eksekusi & harga */}
      <dl className="grid grid-cols-2 gap-3 rounded-xl border border-border/80 bg-card p-3.5 text-xs sm:grid-cols-4">
        {[
          ['Entry', trade.entry_price ?? '—'],
          ['Stop loss', trade.stop_loss ?? '—'],
          ['Take profit', trade.take_profit ?? '—'],
          ['Exit', trade.exit_price ?? '—'],
          ['Size', trade.size ? `${trade.size} ${trade.size_unit}`.trim() : '—'],
          ['Plan RR', calc.plannedRR ?? '—'],
          ['Risiko', trade.risk_amount && account ? formatAmount(trade.risk_amount, account.currency) : '—'],
          ['Risiko / modal', calc.riskPercent ?? '—'],
        ].map(([label, value]) => (
          <div key={label}>
            <dt className="text-[11px] text-muted-foreground">{label}</dt>
            <dd className="mt-0.5 font-semibold tabular-nums text-foreground">{value}</dd>
          </div>
        ))}
      </dl>

      {/* Waktu WIB */}
      {(trade.opened_at || trade.closed_at) && (
        <div className="flex flex-wrap gap-x-6 gap-y-1 text-xs text-muted-foreground tabular-nums">
          {trade.opened_at && (
            <span>
              Masuk: <strong className="font-medium text-foreground">{wibTime(trade.opened_at)} WIB</strong>
            </span>
          )}
          {trade.closed_at && (
            <span>
              Keluar: <strong className="font-medium text-foreground">{wibTime(trade.closed_at)} WIB</strong>
            </span>
          )}
        </div>
      )}

      {/* Psikologi & tag */}
      {(trade.emotions.length > 0 || trade.mistakes.length > 0 || trade.tags.length > 0) && (
        <div className="space-y-2.5">
          {trade.emotions.length > 0 && (
            <div className="flex flex-wrap items-center gap-1.5">
              <span className="text-xs text-muted-foreground">Emosi:</span>
              {trade.emotions.map((item) => (
                <span
                  key={item}
                  className="rounded-md border border-border bg-secondary/60 px-2 py-0.5 text-xs"
                >
                  {item}
                </span>
              ))}
            </div>
          )}
          {trade.mistakes.length > 0 && (
            <div className="flex flex-wrap items-center gap-1.5">
              <span className="text-xs text-muted-foreground">Mistake:</span>
              {trade.mistakes.map((item) => (
                <span
                  key={item}
                  className="rounded-md border border-rose-500/30 bg-rose-500/[0.08] px-2 py-0.5 text-xs text-rose-700 dark:text-rose-300"
                >
                  {item}
                </span>
              ))}
            </div>
          )}
          {trade.tags.length > 0 && (
            <div className="flex flex-wrap items-center gap-1.5">
              <span className="text-xs text-muted-foreground">Tag:</span>
              {trade.tags.map((item) => (
                <span
                  key={item}
                  className="rounded-md border border-border px-2 py-0.5 text-xs text-muted-foreground"
                >
                  #{item}
                </span>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Checklist visual */}
      {trade.checklist.length > 0 && (
        <section className="space-y-2">
          <div className="flex items-center justify-between">
            <h4 className="text-xs font-medium text-muted-foreground">
              Checklist
            </h4>
            <span className="text-xs tabular-nums text-muted-foreground">
              {checkedCount}/{trade.checklist.length}
            </span>
          </div>
          <ul className="space-y-1.5">
            {trade.checklist.map((item) => (
              <li
                key={item.label}
                className="flex items-center gap-2 text-xs text-foreground"
              >
                {item.checked ? (
                  <CheckCircle2 className="h-4 w-4 shrink-0 text-emerald-600 dark:text-emerald-400" aria-hidden="true" />
                ) : (
                  <Circle className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden="true" />
                )}
                <span className={cn(!item.checked && 'text-muted-foreground')}>{item.label}</span>
              </li>
            ))}
          </ul>
        </section>
      )}

      {/* Catatan refleksi */}
      {trade.notes && (
        <section className="space-y-1.5">
          <h4 className="text-xs font-medium text-muted-foreground">
            Catatan
          </h4>
          <p className="whitespace-pre-wrap break-words rounded-lg border border-border/70 bg-secondary/20 p-3 text-sm leading-relaxed">
            {trade.notes}
          </p>
        </section>
      )}

      <Evidence trade={trade} />

      {children}

      {/* Footer aksi rapi */}
      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-border pt-4">
        <ConfirmButton
          label="Hapus trade"
          confirmLabel="Yakin hapus?"
          variant="ghost"
          size="sm"
          disabled={account?.archived}
          isLoading={deleting}
          onConfirm={onDelete}
          className="text-rose-600 hover:text-rose-700 dark:text-rose-400"
        />
        <Button size="sm" onClick={onEdit} disabled={account?.archived}>
          <Pencil className="h-3.5 w-3.5" aria-hidden="true" />
          Edit trade
        </Button>
      </div>
    </div>
  )
}

function Evidence({ trade }: { trade: JournalTrade }) {
  const links = trade.tradingview_urls.filter(safeLink)
  return links.length ? (
    <section className="space-y-1.5">
      <h4 className="text-xs font-medium text-muted-foreground">
        Link TradingView
      </h4>
      <div className="flex flex-wrap gap-2">
        {links.map((url, i) => (
          <a
            key={url}
            href={url}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1.5 rounded-lg border border-border bg-secondary/40 px-3 py-1.5 text-xs font-medium text-foreground transition-colors hover:bg-secondary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <ExternalLink className="h-3.5 w-3.5 text-muted-foreground" aria-hidden="true" />
            Chart {i + 1}
          </a>
        ))}
      </div>
    </section>
  ) : null
}
