import { useState } from 'react'
import { Button, Card } from '@/components/ui'
import { formatAmount } from './model'
import { safeLink, tradeDay, wibTime } from './model'
import type { JournalAccount, JournalGroup, JournalTrade } from './types'

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
        return (
          <button
            key={trade.id}
            type="button"
            onClick={() => onSelect(trade)}
            className="flex w-full items-start justify-between gap-3 rounded-lg border border-border bg-card p-4 text-left hover:bg-secondary/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <span className="min-w-0">
              <span className="block font-semibold">
                {trade.asset || 'Draft tanpa simbol'}{' '}
                <span className="text-xs font-normal text-muted-foreground">
                  {trade.side.toUpperCase()} · {trade.status}
                </span>
              </span>
              <span className="mt-1 block truncate text-xs text-muted-foreground">
                {trade.setup || 'Tanpa setup'} · {account?.name ?? 'Akun jurnal'}
              </span>
              <span className="mt-1 block text-xs text-muted-foreground">
                {tradeDay(trade) || 'Belum bertanggal'} WIB
              </span>
            </span>
            <span className="shrink-0 text-right text-sm font-semibold tabular-nums">
              {trade.net_pnl !== null && trade.status === 'closed' && account
                ? formatAmount(trade.net_pnl, account.currency)
                : '—'}
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
  onSelect,
  onMonth,
  calendar,
}: {
  trades: JournalTrade[]
  month: string
  accounts: JournalAccount[]
  onSelect: (trade: JournalTrade) => void
  onMonth: (month: string) => void
  calendar: Record<string, JournalGroup>
}) {
  const [selected, setSelected] = useState('')
  const date = new Date(`${month}-01T00:00:00+07:00`)
  const year = Number(month.slice(0, 4)),
    m = Number(month.slice(5, 7))
  const count = new Date(Date.UTC(year, m, 0)).getUTCDate()
  const offset = (new Date(Date.UTC(year, m - 1, 1)).getUTCDay() + 6) % 7
  const days = Object.entries(calendar).map(([day, value]) => ({
    day,
    net: Number(value.net_pnl),
    count: value.trades,
  }))
  function move(delta: number) {
    const next = new Date(Date.UTC(year, m - 1 + delta, 1)).toISOString().slice(0, 7)
    onMonth(next)
    setSelected('')
  }
  const daily = trades.filter((t) => tradeDay(t) === selected)
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-2">
        <Button variant="outline" size="sm" onClick={() => move(-1)} aria-label="Bulan sebelumnya">
          Sebelumnya
        </Button>
        <h2 className="text-sm font-semibold">
          {new Intl.DateTimeFormat('id-ID', {
            month: 'long',
            year: 'numeric',
            timeZone: 'Asia/Jakarta',
          }).format(date)}
        </h2>
        <Button variant="outline" size="sm" onClick={() => move(1)} aria-label="Bulan berikutnya">
          Berikutnya
        </Button>
      </div>
      <div className="overflow-hidden rounded-lg border border-border bg-card">
        <div className="grid grid-cols-7 border-b border-border">
          {['Sen', 'Sel', 'Rab', 'Kam', 'Jum', 'Sab', 'Min'].map((d) => (
            <div key={d} className="py-2 text-center text-xs text-muted-foreground">
              {d}
            </div>
          ))}
        </div>
        <div className="grid grid-cols-7">
          {Array.from({ length: offset }, (_, i) => (
            <div
              key={`empty-${i}`}
              aria-hidden="true"
              className="min-h-20 bg-secondary/20 sm:min-h-24"
            />
          ))}
          {Array.from({ length: count }, (_, i) => {
            const day = `${month}-${String(i + 1).padStart(2, '0')}`
            const result = days.find((d) => d.day === day)
            const all = trades.filter((t) => tradeDay(t) === day)
            return (
              <button
                key={day}
                type="button"
                aria-pressed={selected === day}
                aria-label={`${day} WIB, ${all.length} trade${result ? `, net P&L ${result.net}` : ''}`}
                onClick={() => setSelected(day)}
                className={`min-h-20 min-w-0 border-t border-border p-1.5 text-left align-top hover:bg-secondary focus-visible:z-10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:min-h-24 sm:p-3 ${selected === day ? 'bg-secondary' : ''}`}
              >
                <span className="block text-xs text-muted-foreground">{i + 1}</span>
                {result && (
                  <span
                    className={`mt-2 block truncate text-[10px] font-semibold tabular-nums sm:text-sm ${result.net < 0 ? 'text-destructive' : 'text-foreground'}`}
                  >
                    {new Intl.NumberFormat('id-ID', {
                      notation: 'compact',
                      maximumFractionDigits: 1,
                    }).format(result.net)}
                  </span>
                )}
                {!!all.length && (
                  <span className="mt-1 block text-[10px] text-muted-foreground sm:text-xs">
                    {all.length} trade
                  </span>
                )}
              </button>
            )
          })}
        </div>
      </div>
      <section aria-live="polite" className="space-y-3">
        <h3 className="text-sm font-semibold">
          {selected ? `Rekap ${selected} · WIB` : 'Pilih tanggal untuk melihat detail'}
        </h3>
        {selected && (
          <>
            <p className="text-xs text-muted-foreground">
              {daily.length} trade · net terealisasi{' '}
              {formatAmount(
                calendar[selected]?.net_pnl ?? '0',
                accounts.find((a) => a.id === daily[0]?.account_id)?.currency ?? 'USD',
              )}{' '}
              · {daily.filter((t) => t.status === 'closed').length} closed ·{' '}
              {daily.filter((t) => t.status === 'open').length} open ·{' '}
              {daily.filter((t) => t.status === 'draft').length} draft
            </p>
            {daily.length ? (
              <TradeCards trades={daily} accounts={accounts} onSelect={onSelect} />
            ) : (
              <p className="py-4 text-sm text-muted-foreground">
                Tidak ada trade pada tanggal ini.
              </p>
            )}
          </>
        )}
      </section>
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
      <div className="hidden overflow-x-auto rounded-lg border border-border sm:block">
        <table className="w-full text-left text-sm">
          <caption className="sr-only">Trade jurnal manual; semua waktu WIB</caption>
          <thead className="bg-secondary/50 text-xs text-muted-foreground">
            <tr>
              {['Trade', 'Waktu WIB', 'Akun', 'Setup', 'Status', 'Net P&L'].map((h) => (
                <th key={h} scope="col" className="whitespace-nowrap px-4 py-3 font-medium">
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {items.map((t) => {
              const account = accounts.find((a) => a.id === t.account_id)
              const timestamp = t.status === 'closed' ? t.closed_at : t.opened_at
              return (
                <tr key={t.id} className="border-t border-border hover:bg-secondary/30">
                  <td className="px-4 py-3">
                    <button
                      type="button"
                      className="rounded font-semibold underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      onClick={() => onSelect(t)}
                    >
                      {t.asset || 'Draft tanpa simbol'}
                    </button>
                    <span className="ml-2 text-xs text-muted-foreground">
                      {t.side.toUpperCase()}
                    </span>
                  </td>
                  <td className="whitespace-nowrap px-4 py-3 text-xs">
                    {timestamp ? wibTime(timestamp) : '—'}
                  </td>
                  <td className="px-4 py-3">{account?.name ?? '—'}</td>
                  <td className="px-4 py-3">{t.setup || '—'}</td>
                  <td className="px-4 py-3">{t.status}</td>
                  <td className="whitespace-nowrap px-4 py-3 text-right tabular-nums">
                    {account && t.status === 'closed' && t.net_pnl !== null
                      ? formatAmount(t.net_pnl, account.currency)
                      : '—'}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
      <div className="mt-3 flex items-center justify-between gap-2 text-xs text-muted-foreground">
        <span>
          {trades.length} trade · halaman {current}/{pages}
        </span>
        <div className="flex gap-2">
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
    </>
  )
}

export function TradeDetail({
  trade,
  account,
  onEdit,
  onDelete,
  deleting,
}: {
  trade: JournalTrade
  account?: JournalAccount
  onEdit: () => void
  onDelete: () => void
  deleting: boolean
}) {
  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 className="text-lg font-semibold">
            {trade.asset || 'Draft'} · {trade.side.toUpperCase()}
          </h3>
          <p className="text-sm text-muted-foreground">
            {account?.name} · {trade.status}
          </p>
        </div>
        <Button variant="outline" onClick={onEdit} disabled={account?.archived}>
          Edit trade
        </Button>
      </div>
      <dl className="grid grid-cols-2 gap-4 text-sm">
        {[
          ['Masuk (WIB)', trade.opened_at ? wibTime(trade.opened_at) : '—'],
          ['Keluar (WIB)', trade.closed_at ? wibTime(trade.closed_at) : '—'],
          ['Entry', trade.entry_price],
          ['Exit', trade.exit_price],
          ['Stop loss', trade.stop_loss],
          ['Take profit', trade.take_profit],
          ['Size', `${trade.size ?? '—'} ${trade.size_unit}`],
          [
            'Net P&L',
            trade.net_pnl !== null && account ? formatAmount(trade.net_pnl, account.currency) : '—',
          ],
          ['Risiko nominal', trade.risk_amount],
          ['Modal saat entry', trade.capital_snapshot],
          [
            'Risiko / modal',
            Number(trade.capital_snapshot) > 0 && trade.risk_amount !== null
              ? `${((Number(trade.risk_amount) / Number(trade.capital_snapshot)) * 100).toFixed(2)}%`
              : '—',
          ],
          ['Setup', trade.setup],
          ['Emosi', trade.emotions.join(', ')],
          ['Tag', trade.tags.join(', ')],
          ['Kesalahan', trade.mistakes.join(', ')],
          [
            'Checklist terpenuhi',
            trade.checklist
              .map((c) => (c.checked ? 'Terpenuhi: ' : 'Belum: ') + c.label)
              .join(', '),
          ],
        ].map(([label, value]) => (
          <div key={label}>
            <dt className="text-xs text-muted-foreground">{label}</dt>
            <dd className="mt-1 break-words tabular-nums">{value ?? '—'}</dd>
          </div>
        ))}
      </dl>
      {trade.notes && (
        <section>
          <h4 className="text-sm font-semibold">Catatan</h4>
          <p className="mt-1 whitespace-pre-wrap break-words text-sm">{trade.notes}</p>
        </section>
      )}
      <Evidence trade={trade} />
      <Card className="p-3">
        <p className="text-xs text-muted-foreground">
          Hapus hanya jika catatan ini memang tidak diperlukan. Konfirmasi diminta sebelum
          penghapusan.
        </p>
        <Button
          variant="destructive"
          size="sm"
          className="mt-2"
          onClick={onDelete}
          isLoading={deleting}
          disabled={account?.archived}
        >
          Hapus trade
        </Button>
      </Card>
    </div>
  )
}

function Evidence({ trade }: { trade: JournalTrade }) {
  const links = trade.tradingview_urls.filter(safeLink)
  return links.length ? (
    <section className="space-y-2">
      <h4 className="text-sm font-semibold">TradingView sebelum / sesudah</h4>
      {links.map((url, i) => (
        <a
          key={url}
          href={url}
          target="_blank"
          rel="noopener noreferrer"
          className="block rounded text-sm text-primary underline underline-offset-4 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          Chart {i + 1} (buka tab baru)
        </a>
      ))}
    </section>
  ) : null
}
