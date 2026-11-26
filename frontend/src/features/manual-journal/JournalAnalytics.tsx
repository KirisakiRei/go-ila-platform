import { useState } from 'react'
import { Button, Card } from '@/components/ui'
import type { JournalInsight, JournalStats } from './types'
import { formatAmount } from './model'

export function JournalAnalytics({
  stats,
  insights,
}: {
  stats: JournalStats
  insights: JournalInsight[]
}) {
  const [hover, setHover] = useState<string | null>(null)
  const [breakdown, setBreakdown] = useState('setup')
  const currency = stats.currency || 'USD'
  const points = stats.curve ?? []
  const values = [0, ...points.map((p) => Number(p.cumulative))]
  const min = Math.min(...values),
    max = Math.max(...values)
  const y = (n: number) => 150 - ((n - min) / (max - min || 1)) * 125
  const x = (i: number) => 20 + ((i + 1) / Math.max(points.length, 1)) * 640
  const groups = Object.entries(stats.breakdowns?.[breakdown] ?? {})
  const scale = Math.max(...groups.map(([, g]) => Math.abs(Number(g.net_pnl))), 1)
  return (
    <>
      <dl className="grid grid-cols-2 gap-x-6 gap-y-4 border-b border-border py-2 pb-5 lg:grid-cols-4">
        {[
          [
            'Net P&L terealisasi',
            formatAmount(stats.net_pnl, currency),
            'Jumlah net P&L trade closed yang diketahui setelah biaya, sesuai filter aktif. Tidak termasuk modal awal, deposit, withdrawal, atau floating P&L.',
          ],
          [
            'Win rate',
            stats.trades ? `${Number(stats.win_rate).toFixed(1)}%` : '—',
            'Win ÷ (win + loss) × 100%. Breakeven tidak masuk penyebut; backend mengembalikan 0% bila tidak ada win/loss.',
          ],
          [
            'Profit factor',
            stats.profit_factor === null ? '—' : Number(stats.profit_factor).toFixed(2),
            'Total net P&L positif ÷ nilai absolut total net P&L negatif. Tanda — berarti belum ada kerugian sebagai pembagi, bukan nol.',
          ],
          [
            'Trade closed',
            String(stats.trades),
            'Jumlah trade closed dengan net P&L diketahui dalam filter aktif. Draft, open, dan closed tanpa net P&L tidak dihitung.',
          ],
          [
            'Rata-rata profit',
            formatAmount(stats.average_win, currency),
            'Total net P&L positif dibagi jumlah trade win; nol bila belum ada win.',
          ],
          [
            'Rata-rata loss',
            formatAmount(stats.average_loss, currency),
            'Total kerugian net dibagi jumlah trade loss, ditampilkan sebagai besaran positif; nol bila belum ada loss.',
          ],
          [
            'Max drawdown',
            formatAmount(stats.max_drawdown, currency),
            'Penurunan nominal terbesar dari puncak ke lembah akumulasi net P&L closed secara kronologis dalam filter aktif. Bukan drawdown equity broker.',
          ],
          [
            'Win / loss / BE',
            `${stats.wins} / ${stats.losses} / ${stats.breakeven}`,
            'Jumlah trade closed dengan net P&L positif / negatif / tepat nol dalam filter aktif.',
          ],
        ].map(([label, value, definition]) => (
          <div key={label}>
            <dt className="text-xs text-muted-foreground">
              <details>
                <summary
                  className="w-fit cursor-pointer rounded underline decoration-dotted underline-offset-4 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  aria-label={`Definisi ${label}`}
                >
                  {label}
                </summary>
                <p className="mt-2 max-w-prose text-xs leading-relaxed">{definition}</p>
              </details>
            </dt>
            <dd className="mt-1 text-lg font-semibold tabular-nums sm:text-xl">{value}</dd>
          </div>
        ))}
      </dl>
      <div className="grid gap-5 lg:grid-cols-2">
        <Card className="min-w-0 p-4">
          <h2 className="font-semibold">Kurva hasil terealisasi</h2>
          <p className="mt-1 text-xs text-muted-foreground">
            Akumulasi net P&L · WIB. Bukan saldo, deposit, atau floating P&L.
          </p>
          {!points.length ? (
            <p className="py-12 text-center text-sm text-muted-foreground">
              Kurva muncul setelah ada trade closed.
            </p>
          ) : (
            <>
              <svg
                viewBox="0 0 680 180"
                role="img"
                aria-label={`Kurva net P&L terealisasi ${currency}`}
                className="mt-4 w-full text-primary"
              >
                <line x1="20" y1={y(0)} x2="660" y2={y(0)} className="stroke-border" />
                <polyline
                  points={`20,${y(0)} ${points.map((p, i) => `${x(i)},${y(Number(p.cumulative))}`).join(' ')}`}
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                />
                {points.map((p, i) => (
                  <circle
                    key={p.id}
                    cx={x(i)}
                    cy={y(Number(p.cumulative))}
                    r="3"
                    fill="currentColor"
                  >
                    <title>
                      {p.date} WIB: {formatAmount(p.cumulative, currency)}
                    </title>
                  </circle>
                ))}
              </svg>
              <div className="flex justify-between gap-3 text-xs text-muted-foreground">
                <span>{points[0]?.date.slice(0, 10)}</span>
                <span>{formatAmount(stats.net_pnl, currency)}</span>
                <span>{points.at(-1)?.date.slice(0, 10)}</span>
              </div>
              <details className="mt-3 text-xs">
                <summary className="cursor-pointer rounded focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
                  Data kurva
                </summary>
                <ul className="mt-2 max-h-40 overflow-y-auto space-y-1">
                  {points.map((p) => (
                    <li key={p.id} className="flex justify-between gap-2">
                      <span>{p.date} WIB</span>
                      <span>{formatAmount(p.cumulative, currency)}</span>
                    </li>
                  ))}
                </ul>
              </details>
            </>
          )}
        </Card>
        <Card className="min-w-0 p-4">
          <h2 className="font-semibold">Breakdown hasil</h2>
          <div className="mt-2 flex flex-wrap gap-1">
            {Object.entries({
              asset: 'Aset',
              setup: 'Setup',
              session: 'Sesi',
              tag: 'Tag',
              emotion: 'Emosi',
              mistake: 'Kesalahan',
              side: 'Arah',
            }).map(([key, label]) => (
              <Button
                key={key}
                size="sm"
                variant={breakdown === key ? 'secondary' : 'ghost'}
                aria-pressed={breakdown === key}
                onClick={() => setBreakdown(key)}
              >
                {label}
              </Button>
            ))}
          </div>
          {!groups.length ? (
            <p className="py-10 text-center text-sm text-muted-foreground">
              Belum ada hasil untuk dikelompokkan.
            </p>
          ) : (
            <ul className="mt-4 max-h-60 space-y-3 overflow-y-auto">
              {groups
                .sort((a, b) => Number(b[1].net_pnl) - Number(a[1].net_pnl))
                .map(([label, group]) => (
                  <li key={label}>
                    <button
                      type="button"
                      className="w-full rounded text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                      onMouseEnter={() => setHover(label)}
                      onMouseLeave={() => setHover(null)}
                      onFocus={() => setHover(label)}
                      onBlur={() => setHover(null)}
                      onClick={() => setHover(hover === label ? null : label)}
                      aria-label={`${label}: ${group.trades} trade, ${formatAmount(group.net_pnl, currency)}`}
                    >
                      <span className="flex justify-between gap-3 text-xs">
                        <span className="truncate">{label || 'Belum diisi'}</span>
                        <span className="shrink-0 tabular-nums">
                          {formatAmount(group.net_pnl, currency)}
                        </span>
                      </span>
                      <span className="mt-1 block h-2 rounded bg-secondary">
                        <span
                          className={`block h-2 rounded ${Number(group.net_pnl) < 0 ? 'bg-destructive/70' : 'bg-primary/70'}`}
                          style={{
                            width: `${Math.max(1, (Math.abs(Number(group.net_pnl)) / scale) * 100)}%`,
                          }}
                        />
                      </span>
                      {hover === label && (
                        <span role="tooltip" className="mt-1 block text-xs text-muted-foreground">
                          {group.trades} trade closed · {group.wins} win
                        </span>
                      )}
                    </button>
                  </li>
                ))}
            </ul>
          )}
          <p className="mt-3 text-xs text-muted-foreground">
            Kelompok multi-label dapat tumpang tindih; tidak dijumlahkan menjadi total akun.
          </p>
        </Card>
      </div>
      <section
        className="rounded-lg border border-border bg-secondary/30 p-4"
        aria-label="Evaluasi jurnal"
      >
        <h2 className="font-semibold">Evaluasi berbasis aturan</h2>
        {!insights.length ? (
          <p className="mt-2 text-sm text-muted-foreground">
            Belum ada pola yang ditandai oleh aturan evaluasi.
          </p>
        ) : (
          <ul className="mt-2 list-inside list-disc space-y-2 text-sm">
            {insights.map((insight, i) => (
              <li key={`${insight.code}-${i}`}>{insight.message}</li>
            ))}
          </ul>
        )}
        <p className="mt-3 text-xs text-muted-foreground">
          Berdasarkan seluruh filter aktif, bukan halaman tabel. Evaluasi ini tidak menggunakan AI
          dan bukan rekomendasi transaksi. Review harian akun ditampilkan terpisah di atas.
        </p>
      </section>
    </>
  )
}
