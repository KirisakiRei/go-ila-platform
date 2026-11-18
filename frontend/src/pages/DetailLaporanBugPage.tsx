import { Link, useParams } from 'react-router-dom'
import { ArrowLeft, Check, FileText, Paperclip, X } from 'lucide-react'
import { Card, EmptyState, Skeleton } from '@/components/ui'
import { formatDateTime } from '@/lib/format'
import { bugApi, useBugDetail, type BugAttachmentItem, type BugStatus } from '@/lib/bug-api'

const LABEL_STATUS: Record<BugStatus, string> = {
  baru: 'Menunggu tinjauan',
  ditinjau: 'Sedang ditinjau',
  selesai: 'Selesai',
  ditolak: 'Tidak dapat ditindaklanjuti',
}

/**
 * Tahapan laporan. Urutannya mengikuti alur nyata di panel admin: laporan
 * masuk, lalu ditinjau, lalu diputuskan. `ditolak` bukan tahap keempat, ia
 * cabang dari tahap keputusan, jadi digambar sebagai akhir yang berbeda.
 */
const TAHAP: { kunci: 'baru' | 'ditinjau' | 'selesai'; label: string; keterangan: string }[] = [
  { kunci: 'baru', label: 'Laporan diterima', keterangan: 'Masuk ke antrean tinjauan tim.' },
  { kunci: 'ditinjau', label: 'Sedang ditinjau', keterangan: 'Tim menelusuri penyebabnya.' },
  { kunci: 'selesai', label: 'Diputuskan', keterangan: 'Ada tanggapan dari tim.' },
]

/** posisiTahap memetakan status laporan ke indeks tahap yang sudah dilewati. */
function posisiTahap(status: BugStatus): number {
  switch (status) {
    case 'baru':
      return 0
    case 'ditinjau':
      return 1
    default:
      return 2
  }
}

function labelUkuran(bytes: number): string {
  return bytes >= 1024 * 1024
    ? `${(bytes / (1024 * 1024)).toFixed(1)} MB`
    : `${Math.max(1, Math.ceil(bytes / 1024))} KB`
}

function DaftarLampiran({ judul, items, reportID }: { judul: string; items: BugAttachmentItem[]; reportID: string }) {
  if (items.length === 0) return null
  return (
    <div className="flex min-w-0 flex-col gap-1.5">
      <p className="text-[11.5px] font-semibold text-foreground">{judul}</p>
      {items.map((att) => (
        <a
          key={att.id}
          href={bugApi.attachmentUrl(reportID, att.id)}
          target="_blank"
          rel="noreferrer"
          className="inline-flex max-w-full min-w-0 items-center gap-1.5 self-start rounded-lg border border-border px-2.5 py-1.5 text-[11.5px] text-foreground transition-colors duration-150 hover:bg-secondary"
        >
          <Paperclip className="h-3.5 w-3.5 shrink-0" />
          <span className="min-w-0 flex-1 truncate">{att.file_name}</span>
          <span className="shrink-0 text-muted-foreground">({labelUkuran(att.size_bytes)})</span>
        </a>
      ))}
    </div>
  )
}

export default function DetailLaporanBugPage() {
  const { id } = useParams()
  const { data, isLoading, isError } = useBugDetail(id)

  if (isLoading) return <Skeleton className="h-64 w-full rounded-2xl" />
  if (isError || !data) {
    return (
      <EmptyState
        title="Laporan tidak ditemukan"
        description="Laporan ini tidak ada atau bukan milik akun Anda."
      />
    )
  }

  const tahapAktif = posisiTahap(data.status)
  const ditolak = data.status === 'ditolak'
  const lampiranPelapor = data.attachments.filter((a) => !a.from_admin)
  const lampiranTim = data.attachments.filter((a) => a.from_admin)

  return (
    <div className="flex flex-col gap-3.5 pb-2">
      <div className="flex flex-col gap-2">
        <Link
          to="/laporkan-bug/riwayat"
          className="inline-flex w-fit items-center gap-1 text-[11.5px] font-semibold text-muted-foreground transition-colors duration-150 hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Riwayat laporan
        </Link>
        <div className="flex flex-wrap items-start justify-between gap-2">
          <div className="min-w-0 flex-1">
            <h1 className="break-words text-lg font-bold tracking-tight text-foreground">
              {data.title}
            </h1>
            <p className="mt-0.5 text-[11.5px] text-muted-foreground">
              {data.category} · dikirim {formatDateTime(data.created_at)}
            </p>
          </div>
          <span className="shrink-0 rounded-md border border-border bg-secondary px-2 py-1 text-[11px] font-semibold text-foreground">
            {LABEL_STATUS[data.status]}
          </span>
        </div>
      </div>

      <Card className="p-4">
        <ol className="flex flex-col gap-0">
          {TAHAP.map((tahap, i) => {
            const lewat = i <= tahapAktif
            const aktif = i === tahapAktif
            const terakhir = i === TAHAP.length - 1
            const gagal = terakhir && ditolak
            return (
              <li key={tahap.kunci} className="flex gap-3">
                <div className="flex flex-col items-center">
                  <span
                    className={`flex h-6 w-6 shrink-0 items-center justify-center rounded-full border text-[10px] font-semibold ${
                      gagal && aktif
                        ? 'border-border bg-card text-muted-foreground'
                        : lewat
                          ? 'border-foreground bg-foreground text-background'
                          : 'border-border bg-card text-muted-foreground'
                    }`}
                  >
                    {gagal && aktif ? (
                      <X className="h-3 w-3" />
                    ) : lewat && !aktif ? (
                      <Check className="h-3 w-3" />
                    ) : (
                      i + 1
                    )}
                  </span>
                  {!terakhir && (
                    <span className={`w-px flex-1 ${i < tahapAktif ? 'bg-foreground/30' : 'bg-border'}`} />
                  )}
                </div>
                <div className={`min-w-0 ${terakhir ? 'pb-0' : 'pb-4'}`}>
                  <p
                    className={`text-[12.5px] font-semibold ${lewat ? 'text-foreground' : 'text-muted-foreground'}`}
                  >
                    {gagal ? 'Tidak dapat ditindaklanjuti' : tahap.label}
                  </p>
                  <p className="text-[11px] leading-relaxed text-muted-foreground">
                    {aktif
                      ? gagal
                        ? 'Tim memutuskan laporan ini tidak dilanjutkan.'
                        : tahap.keterangan
                      : lewat
                        ? tahap.keterangan
                        : 'Menunggu tahap sebelumnya.'}
                  </p>
                  {terakhir && lewat && data.updated_at && (
                    <p className="mt-0.5 text-[10.5px] text-muted-foreground">
                      {formatDateTime(data.updated_at)}
                    </p>
                  )}
                </div>
              </li>
            )
          })}
        </ol>
      </Card>

      <Card className="flex min-w-0 flex-col gap-2.5 p-4">
        <p className="text-[11.5px] font-semibold text-foreground">Yang Anda laporkan</p>
        <p className="whitespace-pre-wrap break-words text-[12.5px] leading-relaxed text-foreground">
          {data.body}
        </p>
        {data.page_url && (
          <p className="break-all text-[10.5px] text-muted-foreground">Halaman: {data.page_url}</p>
        )}
        <DaftarLampiran judul="Lampiran Anda" items={lampiranPelapor} reportID={data.id} />
      </Card>

      {data.admin_note && (
        <Card className="flex min-w-0 flex-col gap-2.5 border-foreground/20 p-4">
          <div className="flex items-center gap-1.5">
            <FileText className="h-3.5 w-3.5 text-muted-foreground" />
            <p className="text-[11.5px] font-semibold text-foreground">Tanggapan tim</p>
          </div>
          <p className="whitespace-pre-wrap break-words text-[12.5px] leading-relaxed text-foreground">
            {data.admin_note}
          </p>
          <DaftarLampiran judul="Lampiran tim" items={lampiranTim} reportID={data.id} />
        </Card>
      )}

      <Link
        to="/laporkan-bug"
        className="inline-flex w-fit items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-[11.5px] font-semibold text-foreground transition-colors duration-150 hover:bg-secondary active:scale-[0.98]"
      >
        Lapor bug lain
      </Link>
    </div>
  )
}
