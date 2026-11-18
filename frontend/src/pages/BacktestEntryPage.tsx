/**
 * BacktestEntryPage menampilkan satu entry backtest secara utuh.
 *
 * Urutannya mengikuti cara entry itu terjadi: konteks pair dan run, lalu chart
 * acuan dan eksekusi selebar konten, lalu angka posisinya. Chart mendahului
 * tabel angka karena bentuk pergerakan harga lebih dulu menjawab pertanyaan
 * "kenapa di sini", baru angkanya menjelaskan "berapa".
 *
 * Halaman ini sengaja penuh lebar, tanpa panel samping: chart harga dengan dua
 * panel bertumpuk butuh lebar untuk menyebar waktu. Berpindah entry dilakukan
 * lewat tautan sebelumnya/berikutnya di atas, supaya pembaca tidak perlu
 * kembali ke tabel setiap kali.
 */
import { Link, useParams } from 'react-router-dom'
import { ArrowLeft, ArrowRight } from 'lucide-react'
import { Button, EmptyState, Skeleton, StatusPill } from '@/components/ui'
import { useBacktestRun, useBacktestTrades } from '@/lib/queries'
import { BacktestTradeChart } from '@/features/backtest/components/BacktestTradeChart'

export default function BacktestEntryPage() {
  const { id = '', seq = '' } = useParams()
  const nomorEntry = Number(seq)

  const run = useBacktestRun(id || null)
  const selesai = run.data?.status === 'done'
  const { data: trades } = useBacktestTrades(id || null, selesai)

  if (run.isLoading) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton className="h-10 w-64" />
        <Skeleton className="h-[360px] w-full" />
        <Skeleton className="h-[460px] w-full" />
      </div>
    )
  }

  if (!run.data) {
    return (
      <EmptyState
        title="Backtest tidak ditemukan"
        description="Run ini mungkin sudah dihapus, atau bukan milik akun ini."
        action={
          <Button asChild size="sm" variant="outline">
            <Link to="/backtest/riwayat">Kembali ke riwayat</Link>
          </Button>
        }
      />
    )
  }

  if (!Number.isInteger(nomorEntry) || nomorEntry <= 0) {
    return (
      <EmptyState
        title="Entry tidak dikenali"
        description="Alamat halaman ini tidak menunjuk entry mana pun."
        action={
          <Button asChild size="sm" variant="outline">
            <Link to={`/backtest/${encodeURIComponent(id)}`}>Kembali ke run</Link>
          </Button>
        }
      />
    )
  }

  const data = run.data
  // Tetangga entry dihitung dari daftar yang sudah dimuat halaman run. Bila
  // daftar belum tiba, tautan navigasi disembunyikan alih-alih digambar
  // dengan tujuan yang salah.
  const daftar = trades ?? []
  const posisi = daftar.findIndex((satu) => satu.seq === nomorEntry)
  const sebelumnya = posisi > 0 ? daftar[posisi - 1] : undefined
  const berikutnya = posisi >= 0 && posisi < daftar.length - 1 ? daftar[posisi + 1] : undefined

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-col gap-3">
        <Link
          to={`/backtest/${encodeURIComponent(data.id)}`}
          className="inline-flex w-fit items-center gap-1.5 text-[12px] text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5" aria-hidden="true" />
          {data.symbol} {data.htf} ke {data.ltf}
        </Link>

        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border pb-3">
          <div className="flex min-w-0 flex-col gap-0.5">
            <h1 className="text-lg font-semibold tracking-tight">
              Entry {nomorEntry}
              <span className="ml-2 text-[13px] font-normal text-muted-foreground">
                dari {daftar.length || '—'} entry
              </span>
            </h1>
            <p className="text-[12.5px] text-muted-foreground">
              Satu kejadian lengkap: timeframe acuan untuk struktur, timeframe eksekusi untuk
              titik masuk.
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <StatusPill tone="muted">{data.versi || data.strategi}</StatusPill>
            <Button asChild variant="outline" size="sm" disabled={!sebelumnya}>
              <Link
                to={sebelumnya ? `/backtest/${encodeURIComponent(data.id)}/${sebelumnya.seq}` : '#'}
                aria-disabled={!sebelumnya}
                tabIndex={sebelumnya ? undefined : -1}
              >
                <ArrowLeft className="h-3.5 w-3.5" aria-hidden="true" />
                Sebelumnya
              </Link>
            </Button>
            <Button asChild variant="outline" size="sm" disabled={!berikutnya}>
              <Link
                to={berikutnya ? `/backtest/${encodeURIComponent(data.id)}/${berikutnya.seq}` : '#'}
                aria-disabled={!berikutnya}
                tabIndex={berikutnya ? undefined : -1}
              >
                Berikutnya
                <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
              </Link>
            </Button>
          </div>
        </div>
      </div>

      {/* Wadah selebar konten dengan lebar minimum nol: chart harga yang lebar
          tidak boleh mendorong layout lebih lebar dari layar. */}
      <div className="w-full min-w-0">
        <BacktestTradeChart runID={data.id} symbol={data.symbol} seq={nomorEntry} />
      </div>
    </div>
  )
}
