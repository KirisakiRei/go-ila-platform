/**
 * BacktestPage menguji metode ILA 2.0 pada data historis.
 *
 * Halaman ini adalah ruang kerja: bentuk yang dipilih di atas, hasil run yang
 * baru dijalankan mengalir ke bawah. Tidak ada daftar riwayat di sini — daftar
 * itu punya halamannya sendiri, dan menaruhnya di bawah membuat halaman kerja
 * makin panjang setiap kali ada run baru.
 *
 * Selama belum ada run yang dipilih, halaman berisi form saja. Setelah run
 * selesai, ringkasan dan tabel entry muncul; satu baris membuka halaman detail
 * entry.
 */
import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { Clock, History } from 'lucide-react'
import { Button, EmptyState, PageHeader, Section, Skeleton } from '@/components/ui'
import { useBacktestRun, useBacktestTrades, usePairCatalog } from '@/lib/queries'
import type { BacktestStatus } from '@/types/api'
import { BacktestForm } from '@/features/backtest/components/BacktestForm'
import { BacktestSummary } from '@/features/backtest/components/BacktestSummary'
import { BacktestTradeTable } from '@/features/backtest/components/BacktestTradeTable'
import { labelRingkas } from '@/features/backtest/lib/status'

export default function BacktestPage() {
  const { data: katalog } = usePairCatalog()
  const symbols = katalog?.symbols ?? []

  const [symbol, setSymbol] = useState('')
  const [runID, setRunID] = useState<string | null>(null)

  // Pair pertama dipakai selama pengguna belum memilih. Diturunkan, bukan
  // diset lewat effect, supaya tidak ada render ganda saat katalog tiba.
  const symbolAktif = symbol || symbols[0] || ''

  const run = useBacktestRun(runID)
  const selesai = run.data?.status === 'done'
  const { data: trades } = useBacktestTrades(runID, selesai)

  const judulHasil = useMemo(() => (run.data ? labelRingkas(run.data) : undefined), [run.data])

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Backtest"
        description="Uji metode ILA 2.0 pada data historis: seluruh tahapan dari radar sampai hasil, beserta statistik dan kurva ekuitasnya."
        actions={
          <Button asChild variant="outline" size="sm">
            <Link to="/backtest/riwayat">
              <History className="h-4 w-4" aria-hidden="true" />
              Riwayat
            </Link>
          </Button>
        }
      />

      {/* Formulir dikunci ke pair terpilih; run yang sedang ditampilkan tidak
          ikut berubah bila pengguna mengganti pilihan sebelum menekan tombol. */}
      <BacktestForm
        symbols={symbols}
        symbolTerpilih={symbolAktif}
        onPilihSymbol={setSymbol}
        onDibuat={setRunID}
      />

      {runID && (
        <Section title="Hasil" description={judulHasil}>
          <StatusRun status={run.data?.status} pesanGalat={run.data?.error} />
        </Section>
      )}

      {selesai && run.data && (
        <>
          <BacktestSummary run={run.data} />

          <Section
            title="Entry"
            description="Pilih satu baris untuk membuka chart acuan, chart eksekusi, dan tahapan entry itu."
            actions={
              <Button asChild variant="ghost" size="sm">
                <Link to={`/backtest/${encodeURIComponent(run.data.id)}`}>
                  Buka halaman run
                </Link>
              </Button>
            }
          >
            {trades ? (
              <BacktestTradeTable trades={trades} runID={run.data.id} symbol={run.data.symbol} />
            ) : (
              <Skeleton className="h-40 w-full" />
            )}
          </Section>
        </>
      )}

      {!runID && (
        <EmptyState
          icon={<History className="h-5 w-5" aria-hidden="true" />}
          title="Belum ada hasil di halaman ini"
          description="Hasil run baru muncul di sini setelah dijalankan. Run yang sudah pernah dibuat ada di riwayat."
          action={
            <Button asChild variant="outline" size="sm">
              <Link to="/backtest/riwayat">Lihat riwayat backtest</Link>
            </Button>
          }
        />
      )}
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
    <div className="flex flex-wrap items-center gap-2 rounded-xl border border-border bg-secondary/30 p-3.5 text-[13px] sm:p-4">
      <Clock className="h-4 w-4 shrink-0 animate-pulse" aria-hidden="true" />
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
