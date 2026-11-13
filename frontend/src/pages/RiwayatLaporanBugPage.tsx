import { Link } from 'react-router-dom'
import { Card, EmptyState, Skeleton } from '@/components/ui'
import { formatDate } from '@/lib/format'
import { useMyBugReports, type BugStatus } from '@/lib/bug-api'

const LABEL_STATUS: Record<BugStatus, string> = {
  baru: 'Menunggu tinjauan',
  ditinjau: 'Sedang ditinjau',
  selesai: 'Selesai',
  ditolak: 'Tidak dapat ditindaklanjuti',
}

/** Kelas lencana status memakai pola monokrom yang sudah ada, bukan warna baru. */
const GAYA_STATUS: Record<BugStatus, string> = {
  baru: 'border-border bg-secondary text-foreground',
  ditinjau: 'border-border bg-card text-foreground',
  selesai: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400',
  ditolak: 'border-border bg-card text-muted-foreground',
}

/** RiwayatLaporanBugPage menampilkan laporan milik pengguna beserta statusnya. */
export default function RiwayatLaporanBugPage() {
  const { data, isLoading } = useMyBugReports()
  const rows = data ?? []

  return (
    <div className="flex flex-col gap-3.5 pb-2">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div>
          <h1 className="text-lg font-bold tracking-tight text-foreground">Riwayat Laporan</h1>
          <p className="mt-0.5 text-[11.5px] leading-relaxed text-muted-foreground">
            Laporan yang Anda kirim dan status penanganannya.
          </p>
        </div>
        <Link
          to="/laporkan-bug"
          className="inline-flex h-8 items-center rounded-lg border border-border px-3 text-xs font-semibold text-foreground"
        >
          Lapor bug baru
        </Link>
      </div>

      {isLoading ? (
        <Skeleton className="h-40 w-full rounded-2xl" />
      ) : rows.length === 0 ? (
        <EmptyState
          title="Belum ada laporan"
          description="Bila menemukan masalah, kirim laporan lewat halaman Laporkan Bug."
        />
      ) : (
        <div className="flex flex-col gap-2">
          {rows.map((row) => (
            <Card key={row.id} className="flex flex-col gap-1.5 p-3.5">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <p className="text-[13px] font-semibold text-foreground">{row.title}</p>
                <span
                  className={`rounded-md border px-1.5 py-0.5 text-[9.5px] font-semibold uppercase tracking-wider ${GAYA_STATUS[row.status]}`}
                >
                  {LABEL_STATUS[row.status]}
                </span>
              </div>
              <p className="text-[11px] text-muted-foreground">
                {row.category} · {formatDate(row.created_at)}
              </p>
              {row.admin_note && (
                <p className="rounded-lg border border-border bg-secondary/40 p-2 text-[11.5px] text-foreground">
                  Catatan tim: {row.admin_note}
                </p>
              )}
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
