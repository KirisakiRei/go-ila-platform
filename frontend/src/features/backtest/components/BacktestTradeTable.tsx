/**
 * BacktestTradeTable menampilkan seluruh entry sebuah run.
 *
 * Tabel, bukan kartu per entry: satu run bisa berisi puluhan entry, dan
 * menggambar puluhan chart sekaligus membuat halaman berat serta mata tidak
 * bisa membandingkan apa pun. Detail satu entry dimuat hanya ketika dipilih.
 */
import { useMemo } from 'react'
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
import { formatDurasi, formatWaktu } from '../lib/tanggal'

export function BacktestTradeTable({
  trades,
  seqTerpilih,
  onPilih,
}: {
  trades: BacktestTrade[]
  seqTerpilih: number | null
  onPilih: (seq: number) => void
}) {
  const paginasi = usePaginasiLokal(trades, 20)

  const ringkas = useMemo(() => {
    const menang = trades.filter((trade) => trade.hasil === 'TP').length
    return { total: trades.length, menang }
  }, [trades])

  if (trades.length === 0) {
    return (
      <div className="border border-border p-6 text-center text-[13px] text-muted-foreground">
        Tidak ada entry pada rentang ini. Coba rentang yang lebih panjang atau pasangan timeframe
        lain.
      </div>
    )
  }

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
            const terpilih = trade.seq === seqTerpilih
            return (
              <TableRow
                key={trade.seq}
                onClick={() => onPilih(trade.seq)}
                tabIndex={0}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault()
                    onPilih(trade.seq)
                  }
                }}
                aria-selected={terpilih}
                className={cn(
                  'cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring',
                  terpilih && 'bg-secondary/60 hover:bg-secondary/60',
                )}
              >
                <TableCell className="tabular text-muted-foreground">{trade.seq}</TableCell>
                <TableCell className="tabular whitespace-nowrap">{formatWaktu(trade.waktu_masuk)}</TableCell>
                <TableCell>
                  <span className="text-[12px] font-semibold">{trade.arah}</span>
                </TableCell>
                <TableCell className="tabular text-right">{trade.entry.toFixed(2)}</TableCell>
                <TableCell className="tabular text-right text-muted-foreground">
                  {trade.stop_loss.toFixed(2)}
                </TableCell>
                <TableCell className="tabular text-right text-muted-foreground">
                  {trade.take_profit.toFixed(2)}
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
          {ringkas.menang} dari {ringkas.total} entry mencapai target. Pilih satu baris untuk melihat
          potongan chartnya.
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
  const teks =
    hasil === 'TP' ? 'Target' : hasil === 'SL' ? 'Batas rugi' : 'Belum tuntas'
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
        <span className="text-[11px] text-muted-foreground" title="TP dan SL tersentuh pada candle yang sama">
          ambigu
        </span>
      )}
    </span>
  )
}
