/**
 * BacktestDetailPage menampilkan satu run: titik awalnya, statistiknya, kurva
 * ekuitas, dan seluruh entry-nya.
 *
 * Chart entry tidak digambar di sini. Satu run bisa berisi puluhan entry, dan
 * menggambar semuanya sekaligus membuat halaman berat sekaligus tidak terbaca.
 * Tabel menjadi daftar isi; satu entry dibuka di halaman sendiri.
 *
 * Selama run belum selesai, halaman ini memeriksa ulang sendiri lewat
 * useBacktestRun, sehingga pengguna bisa membuka run yang antre lalu
 * meninggalkannya terbuka.
 */
import { Link, useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Clock, Trash2 } from 'lucide-react'
import {
  Button,
  EmptyState,
  PageHeader,
  Section,
  Skeleton,
  StatusPill,
} from '@/components/ui'
import { useBacktestRun, useBacktestTrades, useDeleteBacktest } from '@/lib/queries'
import type { BacktestRunDetail } from '@/types/api'
import { BacktestSummary } from '@/features/backtest/components/BacktestSummary'
import { BacktestTradeTable } from '@/features/backtest/components/BacktestTradeTable'
import { labelRingkas } from '@/features/backtest/lib/status'
import { statusRun } from '@/features/backtest/lib/status'
import { formatTanggalPanjang, formatWaktu } from '@/features/backtest/lib/tanggal'

export default function BacktestDetailPage() {
  const { id = '' } = useParams()
  const navigate = useNavigate()
  const run = useBacktestRun(id || null)
  const selesai = run.data?.status === 'done'
  const { data: trades, isLoading: tradesLoading } = useBacktestTrades(id || null, selesai)
  const hapus = useDeleteBacktest()

  if (run.isLoading) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-28 w-full" />
        <Skeleton className="h-52 w-full" />
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

  const data = run.data
  const status = statusRun(data.status)

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-3">
        <Link
          to="/backtest/riwayat"
          className="inline-flex w-fit items-center gap-1.5 text-[12px] text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5" aria-hidden="true" />
          Riwayat backtest
        </Link>

        <PageHeader
          title={`${data.symbol} · ${data.htf} ke ${data.ltf}`}
          description={labelRingkas(data)}
          actions={
            <>
              <StatusPill tone={status.nada}>{status.teks}</StatusPill>
              <Button
                variant="ghost"
                size="sm"
                aria-label={`Hapus backtest ${labelRingkas(data)}`}
                onClick={() =>
                  hapus.mutate(data.id, {
                    // Setelah run hilang, halaman detailnya tidak punya isi lagi.
                    // Navigasi lewat aplikasi, bukan muat ulang penuh, supaya
                    // sesi dan cache tidak dibuang hanya untuk pindah halaman.
                    onSuccess: () => navigate('/backtest/riwayat'),
                  })
                }
                disabled={hapus.isPending}
                className="text-muted-foreground hover:text-destructive"
              >
                <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                Hapus
              </Button>
            </>
          }
        />
      </div>

      <TitikAwal data={data} />

      {data.status === 'failed' && (
        <div className="border border-destructive/40 bg-destructive/5 p-4 text-[13px] text-destructive">
          {data.error || 'Backtest ini gagal dijalankan.'}
        </div>
      )}

      {!selesai && data.status !== 'failed' && (
        <div className="flex items-center gap-2 border border-border bg-secondary/30 p-4 text-[13px]">
          <Clock className="h-4 w-4 animate-pulse" aria-hidden="true" />
          <span>
            {data.status === 'queued'
              ? 'Menunggu antrean. Backtest lain sedang berjalan.'
              : 'Sedang membaca data historis dan menelusuri setiap candle.'}
          </span>
          <span className="text-[11px] text-muted-foreground">
            halaman ini menyegarkan sendiri
          </span>
        </div>
      )}

      {selesai && (
        <>
          <BacktestSummary run={data} />

          <Section
            title="Entry"
            description="Urut dari yang pertama. Pilih satu baris untuk membuka chart acuan, chart eksekusi, dan tahapan entry itu."
          >
            {tradesLoading || !trades ? (
              <Skeleton className="h-40 w-full" />
            ) : (
              <BacktestTradeTable trades={trades} runID={data.id} symbol={data.symbol} />
            )}
          </Section>
        </>
      )}
    </div>
  )
}

/**
 * TitikAwal merangkum apa yang diminta saat run dibuat.
 *
 * Angka-angka ini yang membuat hasil bisa dibandingkan: dua run dengan modal
 * atau risiko berbeda menghasilkan kurva yang tidak sebanding.
 */
function TitikAwal({ data }: { data: BacktestRunDetail }) {
  const baris = [
    { label: 'Modal awal', nilai: formatUang(data.equity) },
    { label: 'Risiko per trade', nilai: `${(data.risk * 100).toFixed(2)}%` },
    { label: 'Dibuat', nilai: formatWaktu(data.dibuat_pada) },
    {
      label: 'Selesai',
      nilai: data.selesai_pada ? formatWaktu(data.selesai_pada) : 'belum selesai',
    },
    { label: 'Metode', nilai: data.versi || data.strategi },
    {
      label: 'Rentang',
      nilai: `${formatTanggalPanjang(data.dari)} sampai ${formatTanggalPanjang(data.sampai)}`,
    },
  ]

  return (
    <dl className="grid grid-cols-2 gap-x-4 gap-y-3 rounded-lg border border-border bg-card p-4 sm:grid-cols-3 lg:grid-cols-6">
      {baris.map((satu) => (
        <div key={satu.label} className="flex min-w-0 flex-col gap-0.5">
          <dt className="truncate text-[11px] text-muted-foreground">{satu.label}</dt>
          <dd className="tabular truncate text-[13px]">{satu.nilai}</dd>
        </div>
      ))}
    </dl>
  )
}

function formatUang(nilai: number): string {
  return new Intl.NumberFormat('id-ID', { maximumFractionDigits: 0 }).format(nilai)
}
