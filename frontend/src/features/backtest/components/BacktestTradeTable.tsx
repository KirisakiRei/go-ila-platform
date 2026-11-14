/**
 * BacktestTradeTable menampilkan seluruh entry sebuah run.
 *
 * Tabel, bukan kartu per entry: satu run bisa berisi puluhan entry, dan
 * menggambar puluhan chart sekaligus membuat halaman berat serta mata tidak
 * bisa membandingkan apa pun.
 *
 * Setiap baris adalah tautan sungguhan, sehingga bisa dibuka di tab baru dan
 * bisa disalin alamatnya. Klik di sini membuka halaman detail entry, bukan
 * menyisipkan chart di bawah tabel.
 */
import { Link } from 'react-router-dom'
import {
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
  const paginasi = usePaginasiLokal(trades, 20)

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
                  {/* Tautan yang membentang memenuhi baris: seluruh baris bisa
                      diklik, tetapi yang menerima fokus hanya satu elemen dan
                      pembaca layar mendengar judul entry, bukan lima sel. */}
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

      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-[12px] text-muted-foreground">
          {menang} dari {trades.length} entry mencapai target. Pilih satu baris untuk melihat
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

/**
 * HasilLabel menulis cara posisi berakhir.
 *
 * Candle ambigu ditandai terpisah karena angkanya bisa berubah bila data tick
 * tersedia: pembaca perlu tahu hasil itu tidak sekuat labelnya.
 */
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
