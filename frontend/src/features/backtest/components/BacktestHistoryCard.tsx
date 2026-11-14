/**
 * Label status dan kartu satu run untuk halaman riwayat.
 *
 * Kartu ini dibaca sekali lewat: pair, rentang, lalu hasil. Karena itu angka
 * hasil diletakkan sebagai tiga kolom tetap (trade, win rate, R) sehingga mata
 * bisa membandingkan banyak kartu tanpa membaca kalimat.
 */
import { Link } from 'react-router-dom'
import { Trash2 } from 'lucide-react'
import { Button, StatusPill } from '@/components/ui'
import { cn } from '@/lib/utils'
import type { BacktestRun } from '@/types/api'
import { formatTanggalPanjang, formatWaktu, hitungBulan } from '../lib/tanggal'
import { labelRingkas, statusRun } from '../lib/status'

/** BacktestHistoryCard adalah satu run di halaman riwayat: satu kartu, satu pintu. */
export function BacktestHistoryCard({
  run,
  onHapus,
  sedangDihapus,
}: {
  run: BacktestRun
  onHapus: (id: string) => void
  sedangDihapus: boolean
}) {
  const status = statusRun(run.status)
  const selesai = run.status === 'done'
  const statistik = run.statistik
  const bulan = hitungBulan(run.dari, run.sampai)

  return (
    <article className="group relative flex min-w-0 flex-col gap-3 rounded-lg border border-border bg-card p-4 transition-colors duration-150 ease-out hover:border-foreground/25 focus-within:ring-2 focus-within:ring-ring focus-within:ring-offset-2 focus-within:ring-offset-background">
      <div className="flex min-w-0 items-start justify-between gap-3">
        <div className="flex min-w-0 flex-col gap-1">
          <h3 className="truncate text-[15px] font-semibold leading-tight">
            {/* Tautan menutupi seluruh kartu supaya satu kartu terasa seperti
                satu pintu, bukan sekadar judul yang bisa diklik. */}
            <Link
              to={`/backtest/${encodeURIComponent(run.id)}`}
              className="focus-visible:outline-none after:absolute after:inset-0 after:content-['']"
            >
              {run.symbol}
            </Link>
            <span className="ml-2 text-[13px] font-normal text-muted-foreground">
              {run.htf} ke {run.ltf}
            </span>
          </h3>
          <p className="tabular text-[12px] text-muted-foreground">
            {formatTanggalPanjang(run.dari)} sampai {formatTanggalPanjang(run.sampai)}
            <span className="text-muted-foreground/70"> · {bulan} bulan</span>
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          <StatusPill tone={status.nada}>{status.teks}</StatusPill>
          <Button
            variant="ghost"
            size="sm"
            aria-label={`Hapus backtest ${labelRingkas(run)}`}
            onClick={() => onHapus(run.id)}
            disabled={sedangDihapus}
            className="relative z-10 text-muted-foreground hover:text-destructive"
          >
            <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
          </Button>
        </div>
      </div>

      {selesai ? (
        <dl className="grid grid-cols-3 gap-2 border-t border-border pt-3">
          <Angka label="Trade" nilai={statistik.total_trade} />
          <Angka label="Win rate" nilai={`${(statistik.win_rate * 100).toFixed(0)}%`} />
          <Angka
            label="Total R"
            nilai={`${statistik.total_r > 0 ? '+' : ''}${statistik.total_r.toFixed(2)}`}
            nada={statistik.total_r > 0 ? 'naik' : statistik.total_r < 0 ? 'turun' : 'netral'}
          />
        </dl>
      ) : (
        <p
          className={cn(
            'border-t border-border pt-3 text-[12px]',
            run.status === 'failed' ? 'text-destructive' : 'text-muted-foreground',
          )}
        >
          {run.status === 'failed'
            ? run.error || 'Backtest gagal dijalankan. Buka untuk melihat apakah data bisa dipakai.'
            : run.status === 'queued'
              ? 'Menunggu antrean. Hasil muncul di sini setelah selesai.'
              : 'Sedang menelusuri candle. Halaman ini bisa ditinggal.'}
        </p>
      )}

      <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 text-[11px] text-muted-foreground">
        <span className="truncate">{run.versi || run.strategi}</span>
        <span className="tabular">{formatWaktu(run.dibuat_pada)}</span>
      </div>
    </article>
  )
}

function Angka({
  label,
  nilai,
  nada = 'netral',
}: {
  label: string
  nilai: string | number
  nada?: 'naik' | 'turun' | 'netral'
}) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <dt className="truncate text-[11px] text-muted-foreground">{label}</dt>
      <dd
        className={cn(
          'tabular text-[15px] font-semibold leading-none',
          nada === 'turun' && 'text-destructive',
        )}
      >
        {nilai}
      </dd>
    </div>
  )
}
