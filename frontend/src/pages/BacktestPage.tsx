/**
 * BacktestPage menguji metode ILA 2.0 pada data historis.
 *
 * Susunannya mengikuti cara orang benar-benar memakainya: bentuk yang dipilih
 * di atas, hasil mengalir ke bawah. Tidak ada langkah berpindah halaman, karena
 * pengujian biasanya berulang — ganti pair, ganti rentang, jalankan lagi.
 */
import { useMemo, useState } from 'react'
import { Clock, History, Trash2 } from 'lucide-react'
import {
  Button,
  EmptyState,
  PageHeader,
  Section,
  Skeleton,
} from '@/components/ui'
import {
  useBacktestHistory,
  useBacktestRun,
  useBacktestTrades,
  useDeleteBacktest,
  usePairCatalog,
} from '@/lib/queries'
import { cn } from '@/lib/utils'
import type { BacktestStatus } from '@/types/api'
import { BacktestForm } from '@/features/backtest/components/BacktestForm'
import { BacktestSummary } from '@/features/backtest/components/BacktestSummary'
import { BacktestTradeChart } from '@/features/backtest/components/BacktestTradeChart'
import { BacktestTradeTable } from '@/features/backtest/components/BacktestTradeTable'
import { formatTanggalPanjang, formatWaktu } from '@/features/backtest/lib/tanggal'

const STATUS_LABEL: Record<BacktestStatus, { teks: string; nada: 'netral' | 'aktif' | 'gagal' }> = {
  queued: { teks: 'Menunggu antrean', nada: 'netral' },
  running: { teks: 'Sedang dijalankan', nada: 'aktif' },
  done: { teks: 'Selesai', nada: 'aktif' },
  failed: { teks: 'Gagal', nada: 'gagal' },
}

export default function BacktestPage() {
  const { data: katalog } = usePairCatalog()
  const symbols = katalog?.symbols ?? []

  const [symbol, setSymbol] = useState('')
  const [runID, setRunID] = useState<string | null>(null)
  const [seqTerpilih, setSeqTerpilih] = useState<number | null>(null)

  // Pair pertama dipakai selama pengguna belum memilih. Diturunkan, bukan
  // diset lewat effect, supaya tidak ada render ganda saat katalog tiba.
  const symbolAktif = symbol || symbols[0] || ''

  const run = useBacktestRun(runID)
  const selesai = run.data?.status === 'done'
  const { data: trades } = useBacktestTrades(runID, selesai)

  // Entry pertama ditampilkan selama pengguna belum memilih baris lain. Chart
  // yang langsung terisi lebih berguna daripada ruang kosong yang menunggu klik.
  const seqAktif = useMemo(() => {
    if (seqTerpilih) return seqTerpilih
    return trades?.[0]?.seq ?? null
  }, [seqTerpilih, trades])

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Backtest"
        description="Uji metode ILA 2.0 pada data historis: seluruh tahapan dari radar sampai hasil, beserta statistik dan kurva ekuitasnya."
      />

      <BacktestForm
        symbols={symbols}
        symbolTerpilih={symbolAktif}
        onPilihSymbol={setSymbol}
        onDibuat={(id) => {
          setRunID(id)
          setSeqTerpilih(null)
        }}
      />

      {runID && (
        <Section title="Hasil" description={run.data ? ringkasJudul(run.data.symbol, run.data.htf, run.data.ltf, run.data.dari, run.data.sampai) : undefined}>
          <StatusRun status={run.data?.status} pesanGalat={run.data?.error} />
        </Section>
      )}

      {selesai && run.data && (
        <>
          <BacktestSummary run={run.data} />

          <Section title="Entry" description="Pilih satu baris untuk melihat potongan chart dan tahapannya.">
            {trades ? (
              <div className="grid gap-5 xl:grid-cols-[minmax(0,7fr)_minmax(0,5fr)]">
                <div className="min-w-0">
                  <BacktestTradeTable
                    trades={trades}
                    seqTerpilih={seqAktif}
                    onPilih={setSeqTerpilih}
                  />
                </div>
                <div className="min-w-0">
                  {seqAktif ? (
                    <BacktestTradeChart
                      runID={runID ?? ''}
                      symbol={run.data.symbol}
                      seq={seqAktif}
                    />
                  ) : (
                    <EmptyState
                      title="Belum ada entry dipilih"
                      description="Klik satu baris di tabel untuk melihat chart-nya."
                    />
                  )}
                </div>
              </div>
            ) : (
              <Skeleton className="h-40 w-full" />
            )}
          </Section>
        </>
      )}

      <Riwayat onPilih={(id) => {
        setRunID(id)
        setSeqTerpilih(null)
      }} />
    </div>
  )
}

/**
 * StatusRun menjelaskan keadaan run selagi berjalan.
 *
 * Bagian ini sengaja hanya muncul ketika belum selesai: setelah selesai,
 * statistik di bawahnya sudah menjelaskan dirinya sendiri.
 */
function StatusRun({
  status,
  pesanGalat,
}: {
  status?: BacktestStatus
  pesanGalat?: string
}) {
  if (!status) {
    return <Skeleton className="h-16 w-full" />
  }
  if (status === 'done') {
    return null
  }
  if (status === 'failed') {
    return (
      <div className="border border-destructive/40 bg-destructive/5 p-4 text-[13px] text-destructive">
        {pesanGalat || 'Backtest gagal dijalankan. Coba rentang atau pasangan lain.'}
      </div>
    )
  }

  return (
    <div className="flex items-center gap-2 border border-border bg-secondary/30 p-4 text-[13px]">
      <Clock className="h-4 w-4 animate-pulse" aria-hidden="true" />
      <span>
        {status === 'queued'
          ? 'Menunggu antrean. Backtest lain sedang berjalan.'
          : 'Sedang membaca data historis dan menelusuri setiap candle.'}
      </span>
      <span className="text-[11px] text-muted-foreground">
        biasanya selesai kurang dari satu menit
      </span>
    </div>
  )
}

/** Riwayat menampilkan run sebelumnya agar hasil bisa dibandingkan. */
function Riwayat({ onPilih }: { onPilih: (id: string) => void }) {
  const { data: daftar, isLoading } = useBacktestHistory()
  const hapus = useDeleteBacktest()

  const urut = useMemo(() => daftar ?? [], [daftar])

  return (
    <Section title="Riwayat" description="Hasil lama tetap tersimpan meski potongan candle sudah dibersihkan.">
      {isLoading ? (
        <Skeleton className="h-24 w-full" />
      ) : urut.length === 0 ? (
        <EmptyState
          icon={<History className="h-5 w-5" aria-hidden="true" />}
          title="Belum ada backtest"
          description="Hasil pengujian akan muncul di sini setelah yang pertama selesai."
        />
      ) : (
        <ul className="divide-y divide-border border border-border">
          {urut.map((item) => (
            <li key={item.id} className="flex flex-wrap items-center justify-between gap-3 px-3 py-2.5">
              <button
                type="button"
                onClick={() => onPilih(item.id)}
                className="flex min-w-0 flex-1 flex-wrap items-baseline gap-x-3 gap-y-1 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
              >
                <span className="text-[13px] font-semibold">{item.symbol}</span>
                <span className="text-[12px] text-muted-foreground">
                  {item.htf} ke {item.ltf}
                </span>
                <span className="tabular text-[12px] text-muted-foreground">
                  {formatTanggalPanjang(item.dari)} sampai {formatTanggalPanjang(item.sampai)}
                </span>
                {item.status === 'done' && (
                  <span className="tabular text-[12px]">
                    {item.statistik.total_trade} trade · {(item.statistik.win_rate * 100).toFixed(0)}% ·{' '}
                    {item.statistik.total_r > 0 ? '+' : ''}
                    {item.statistik.total_r.toFixed(2)}R
                  </span>
                )}
                <span className={cn('text-[11px]', item.status === 'failed' && 'text-destructive')}>
                  {STATUS_LABEL[item.status]?.teks ?? item.status}
                </span>
              </button>
              <div className="flex items-center gap-2">
                <span className="hidden text-[11px] text-muted-foreground sm:inline">
                  {formatWaktu(item.dibuat_pada)}
                </span>
                <Button
                  variant="ghost"
                  size="sm"
                  aria-label={`Hapus backtest ${item.symbol} ${formatWaktu(item.dibuat_pada)}`}
                  onClick={() => hapus.mutate(item.id)}
                  disabled={hapus.isPending}
                >
                  <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                </Button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </Section>
  )
}

function ringkasJudul(symbol: string, htf: string, ltf: string, dari: number, sampai: number): string {
  return `${symbol} ${htf} ke ${ltf}, ${formatTanggalPanjang(dari)} sampai ${formatTanggalPanjang(sampai)}`
}
