/**
 * BacktestHistoryPage adalah daftar semua run backtest milik pengguna.
 *
 * Halaman ini dipisah dari halaman kerja backtest supaya keduanya bisa dipakai
 * sambil lalu: datang untuk melihat apa yang sudah pernah diuji tanpa harus
 * melewati formulir, lalu masuk ke satu run untuk membaca hasilnya.
 *
 * Daftar dimuat berhalaman dari server, bukan seluruhnya sekaligus: riwayat
 * bisa berisi ratusan run, dan memuat semuanya hanya untuk menampilkan dua
 * puluh kartu pertama membuang waktu serta memori.
 */
import { useState } from 'react'
import { Link } from 'react-router-dom'
import { FlaskConical, History } from 'lucide-react'
import { Button, EmptyState, PageHeader, Pagination, Skeleton } from '@/components/ui'
import { useBacktestHistory, useDeleteBacktest } from '@/lib/queries'
import { BacktestHistoryCard } from '@/features/backtest/components/BacktestHistoryCard'

/** Satu halaman berisi dua belas kartu: tiga baris rapat di layar lebar. */
const KARTU_PER_HALAMAN = 12

export default function BacktestHistoryPage() {
  const [halaman, setHalaman] = useState(1)
  const offset = (halaman - 1) * KARTU_PER_HALAMAN

  const { data, isLoading } = useBacktestHistory(KARTU_PER_HALAMAN, offset)
  const hapus = useDeleteBacktest()

  const daftar = data?.daftar ?? []
  const total = data?.total ?? 0
  const totalHalaman = Math.max(1, Math.ceil(total / KARTU_PER_HALAMAN))

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Riwayat backtest"
        description="Setiap run yang pernah dijalankan, terbaru di atas. Buka satu kartu untuk membaca statistik dan seluruh entry-nya."
        actions={
          <Button asChild variant="outline" size="sm">
            <Link to="/backtest">
              <FlaskConical className="h-4 w-4" aria-hidden="true" />
              Backtest baru
            </Link>
          </Button>
        }
      />

      {isLoading ? (
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {Array.from({ length: 6 }, (_, indeks) => (
            <Skeleton key={indeks} className="h-[168px] w-full rounded-lg" />
          ))}
        </div>
      ) : daftar.length === 0 ? (
        <EmptyState
          icon={<History className="h-5 w-5" aria-hidden="true" />}
          title="Belum ada backtest"
          description="Mulai satu pengujian, lalu hasilnya muncul di sini untuk dibandingkan dengan run berikutnya."
          action={
            <Button asChild size="sm">
              <Link to="/backtest">Jalankan backtest pertama</Link>
            </Button>
          }
        />
      ) : (
        <>
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {daftar.map((run) => (
              <BacktestHistoryCard
                key={run.id}
                run={run}
                onHapus={(id) => hapus.mutate(id)}
                sedangDihapus={hapus.isPending}
              />
            ))}
          </div>

          <Pagination
            page={halaman}
            totalPages={totalHalaman}
            shown={daftar.length}
            total={total}
            unit="run"
            onPageChange={setHalaman}
          />
        </>
      )}
    </div>
  )
}
