import { Link } from 'react-router-dom'
import { ChevronRight } from 'lucide-react'
import {
  Badge,
  Pagination,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  usePaginasiLokal,
} from '@/components/ui'
import { cn } from '@/lib/utils'
import type { BacktestTrade } from '@/types/api'
import { priceDigits } from '@/lib/pricePrecision'
import { useIsTabletOrMobile } from '@/lib/use-mobile-viewport'
import { formatDurasi, formatWaktu } from '../lib/tanggal'

export function BacktestTradeTable({
  trades,
  runID,
  symbol,
}: {
  trades: BacktestTrade[]
  /** Run pemilik entry, dipakai menyusun alamat halaman detail. */
  runID: string
  /** Presisi harga mengikuti pair, bukan dua desimal tetap. */
  symbol: string
}) {
  const isMobileCards = useIsTabletOrMobile()
  const paginasi = usePaginasiLokal(trades, isMobileCards ? 10 : 20)

  if (trades.length === 0) {
    return (
      <div className="border border-border p-6 text-center text-[13px] text-muted-foreground">
        Tidak ada entry pada rentang ini. Coba rentang yang lebih panjang atau pasangan timeframe
        lain.
      </div>
    )
  }

  const menang = trades.filter((trade) => trade.hasil === 'TP').length

  return (
    <div className="flex flex-col gap-3">
      {isMobileCards ? (
        <ul className="space-y-2.5">
          {paginasi.baris.map((trade) => {
            const digit = priceDigits(symbol, trade.entry)
            const rLabel = `${trade.r > 0 ? '+' : ''}${trade.r.toFixed(2)}R`
            return (
              <li key={trade.seq}>
                <Link
                  to={`/backtest/${encodeURIComponent(runID)}/${trade.seq}`}
                  aria-label={`Buka detail entry ${trade.seq}, ${trade.arah} pada ${formatWaktu(trade.waktu_masuk)}`}
                  className="flex flex-col gap-2.5 rounded-2xl border border-border bg-card p-3.5 shadow-xs transition-colors hover:bg-secondary/40 active:bg-secondary/60"
                >
                  <div className="flex items-center justify-between gap-2">
                    <div className="flex min-w-0 items-center gap-2">
                      <span className="font-mono text-xs font-semibold text-muted-foreground">
                        #{trade.seq}
                      </span>
                      <Badge variant={trade.arah === 'BUY' ? 'solid' : 'outline'} className="text-[10px]">
                        {trade.arah}
                      </Badge>
                      <HasilLabel hasil={trade.hasil} ambigu={trade.ambigu} />
                    </div>
                    <div className="flex shrink-0 items-center gap-1">
                      <span
                        className={cn(
                          'font-mono text-xs font-bold tabular',
                          trade.r > 0 && 'text-foreground',
                          trade.r < 0 && 'text-destructive',
                        )}
                      >
                        {rLabel}
                      </span>
                      <ChevronRight className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
                    </div>
                  </div>

                  <div className="grid grid-cols-3 gap-2 rounded-xl border border-border/70 bg-secondary/25 p-2 text-[11px]">
                    <div className="min-w-0">
                      <span className="block text-[9.5px] uppercase text-muted-foreground">Entry</span>
                      <span className="block truncate font-mono font-semibold tabular">
                        {trade.entry.toFixed(digit)}
                      </span>
                    </div>
                    <div className="min-w-0">
                      <span className="block text-[9.5px] uppercase text-muted-foreground">SL</span>
                      <span className="block truncate font-mono tabular text-muted-foreground">
                        {trade.stop_loss.toFixed(digit)}
                      </span>
                    </div>
                    <div className="min-w-0 text-right">
                      <span className="block text-[9.5px] uppercase text-muted-foreground">TP</span>
                      <span className="block truncate font-mono tabular text-muted-foreground">
                        {trade.take_profit.toFixed(digit)}
                      </span>
                    </div>
                  </div>

                  <div className="flex flex-wrap items-center justify-between gap-2 text-[11px] text-muted-foreground">
                    <span className="tabular">{formatWaktu(trade.waktu_masuk)}</span>
                    <span>
                      RR 1:{trade.rr_rencana.toFixed(2)} · {formatDurasi(trade.durasi_detik)}
                    </span>
                  </div>
                </Link>
              </li>
            )
          })}
        </ul>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-12">#</TableHead>
              <TableHead>Masuk</TableHead>
              <TableHead>Arah</TableHead>
              <TableHead className="text-right">Entry</TableHead>
              <TableHead className="text-right">Batas rugi</TableHead>
              <TableHead className="text-right">Target</TableHead>
              <TableHead className="text-right">RR</TableHead>
              <TableHead>Hasil</TableHead>
              <TableHead className="text-right">R</TableHead>
              <TableHead>Lama</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {paginasi.baris.map((trade) => {
              const digit = priceDigits(symbol, trade.entry)
              return (
                <TableRow key={trade.seq} className="relative cursor-pointer">
                  <TableCell className="tabular text-muted-foreground">{trade.seq}</TableCell>
                  <TableCell className="tabular whitespace-nowrap">
                    <Link
                      to={`/backtest/${encodeURIComponent(runID)}/${trade.seq}`}
                      className="absolute inset-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
                      aria-label={`Buka detail entry ${trade.seq}, ${trade.arah} pada ${formatWaktu(trade.waktu_masuk)}`}
                    >
                      <span className="sr-only">Buka detail entry {trade.seq}</span>
                    </Link>
                    {formatWaktu(trade.waktu_masuk)}
                  </TableCell>
                  <TableCell>
                    <span className="text-[12px] font-semibold">{trade.arah}</span>
                  </TableCell>
                  <TableCell className="tabular text-right">{trade.entry.toFixed(digit)}</TableCell>
                  <TableCell className="tabular text-right text-muted-foreground">
                    {trade.stop_loss.toFixed(digit)}
                  </TableCell>
                  <TableCell className="tabular text-right text-muted-foreground">
                    {trade.take_profit.toFixed(digit)}
                  </TableCell>
                  <TableCell className="tabular text-right">{trade.rr_rencana.toFixed(2)}</TableCell>
                  <TableCell>
                    <HasilLabel hasil={trade.hasil} ambigu={trade.ambigu} />
                  </TableCell>
                  <TableCell
                    className={cn(
                      'tabular text-right font-medium',
                      trade.r > 0 && 'text-foreground',
                      trade.r < 0 && 'text-destructive',
                    )}
                  >
                    {trade.r > 0 ? '+' : ''}
                    {trade.r.toFixed(2)}
                  </TableCell>
                  <TableCell className="whitespace-nowrap text-muted-foreground">
                    {formatDurasi(trade.durasi_detik)}
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      )}

      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-[12px] text-muted-foreground">
          {menang} dari {trades.length} entry mencapai target. Pilih satu entry untuk melihat
          potongan chart dan tahapannya.
        </p>
        <Pagination
          page={paginasi.page}
          totalPages={paginasi.totalPages}
          shown={paginasi.baris.length}
          total={paginasi.total}
          unit="entry"
          onPageChange={paginasi.setPage}
        />
      </div>
    </div>
  )
}

function HasilLabel({ hasil, ambigu }: { hasil: BacktestTrade['hasil']; ambigu: boolean }) {
  const teks = hasil === 'TP' ? 'Target' : hasil === 'SL' ? 'Batas rugi' : 'Belum tuntas'
  return (
    <span className="flex items-center gap-1.5 whitespace-nowrap text-[12px]">
      <span
        className={cn(
          'font-medium',
          hasil === 'TP' && 'text-foreground',
          hasil === 'SL' && 'text-destructive',
          hasil === 'EXPIRED' && 'text-muted-foreground',
        )}
      >
        {teks}
      </span>
      {ambigu && (
        <span
          className="text-[11px] text-muted-foreground"
          title="TP dan SL tersentuh pada candle yang sama"
        >
          ambigu
        </span>
      )}
    </span>
  )
}
