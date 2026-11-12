import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Paperclip } from 'lucide-react'
import { bugApi, type BugDetail, type BugStatus } from '@/lib/bug-api'
import { Can } from '@/features/admin/components/Can'
import { formatDate } from '@/lib/format'
import {
  Alert,
  Badge,
  Button,
  Card,
  EmptyState,
  Label,
  PageHeader,
  Pagination,
  Select,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  usePaginasiLokal,
} from '@/components/ui'

const LABEL_STATUS: Record<BugStatus, string> = {
  baru: 'Baru',
  ditinjau: 'Ditinjau',
  selesai: 'Selesai',
  ditolak: 'Ditolak',
}

const VARIAN_STATUS: Record<BugStatus, 'muted' | 'outline' | 'solid'> = {
  baru: 'solid',
  ditinjau: 'outline',
  selesai: 'muted',
  ditolak: 'muted',
}

/**
 * AdminBugReportsPage menampilkan laporan bug pengguna untuk ditriage.
 *
 * Lampiran dibuka lewat tautan biasa agar cookie sesi admin ikut terkirim;
 * server mengirimnya sebagai unduhan, bukan ditampilkan di peramban.
 */
export default function AdminBugReportsPage() {
  const queryClient = useQueryClient()
  const [status, setStatus] = useState('')
  const [kategori, setKategori] = useState('')
  const [dipilih, setDipilih] = useState<BugDetail | null>(null)
  const [galat, setGalat] = useState<string | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'bug-reports', status, kategori],
    queryFn: () => bugApi.adminList({ status: status || undefined, category: kategori || undefined }),
  })

  const rows = data ?? []
  // Daftar dipecah per halaman agar tabel panjang tidak memaksa menggulir jauh.
  const paginasi = usePaginasiLokal(rows)
  const invalidate = () => void queryClient.invalidateQueries({ queryKey: ['admin', 'bug-reports'] })

  const ubah = useMutation({
    mutationFn: ({ id, next, note }: { id: string; next: BugStatus; note: string }) =>
      bugApi.adminUpdate(id, { status: next, note }),
    onSuccess: () => {
      setDipilih(null)
      invalidate()
    },
    onError: () => setGalat('Gagal memperbarui status laporan.'),
  })

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title="Laporan Bug"
        description="Laporan yang dikirim pengguna, siap ditriage."
      />

      {galat && <Alert>{galat}</Alert>}

      <Card className="p-4">
        <div className="mb-3 flex flex-wrap items-end gap-3">
          <div className="flex flex-col gap-1">
            <Label htmlFor="filter-status">Status</Label>
            <Select
              id="filter-status"
              value={status}
              onChange={(e) => setStatus(e.target.value)}
              className="w-40"
            >
              <option value="">Semua</option>
              <option value="baru">Baru</option>
              <option value="ditinjau">Ditinjau</option>
              <option value="selesai">Selesai</option>
              <option value="ditolak">Ditolak</option>
            </Select>
          </div>
          <div className="flex flex-col gap-1">
            <Label htmlFor="filter-kategori">Kategori</Label>
            <Select
              id="filter-kategori"
              value={kategori}
              onChange={(e) => setKategori(e.target.value)}
              className="w-48"
            >
              <option value="">Semua</option>
              <option value="tampilan">Tampilan</option>
              <option value="sinyal">Sinyal</option>
              <option value="eksekusi">Eksekusi</option>
              <option value="akun-broker">Akun broker</option>
              <option value="login">Login</option>
              <option value="kinerja">Kinerja</option>
              <option value="lainnya">Lainnya</option>
            </Select>
          </div>
        </div>

        {isLoading ? (
          <Skeleton className="h-40 w-full rounded-xl" />
        ) : rows.length === 0 ? (
          <EmptyState title="Belum ada laporan" description="Laporan pengguna akan muncul di sini." />
        ) : (
          <>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Tanggal</TableHead>
                <TableHead>Kategori</TableHead>
                <TableHead>Judul</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Aksi</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {paginasi.baris.map((row) => (
                <TableRow key={row.id}>
                  <TableCell className="text-xs">{formatDate(row.created_at)}</TableCell>
                  <TableCell className="text-xs">{row.category}</TableCell>
                  <TableCell className="max-w-[280px] truncate text-xs font-medium">
                    {row.title}
                    {row.attachments.length > 0 && (
                      <span className="ml-1 inline-flex items-center gap-0.5 text-muted-foreground">
                        <Paperclip className="h-3 w-3" />
                        {row.attachments.length}
                      </span>
                    )}
                  </TableCell>
                  <TableCell>
                    <Badge variant={VARIAN_STATUS[row.status]}>
                      {LABEL_STATUS[row.status] ?? row.status}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Button size="sm" variant="outline" onClick={() => setDipilih(row)}>
                      Buka
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          <div className="border-t border-border px-3 py-2">
            <Pagination
              page={paginasi.page}
              totalPages={paginasi.totalPages}
              shown={paginasi.baris.length}
              total={paginasi.total}
              unit="laporan"
              pageSize={paginasi.pageSize}
              onPageChange={paginasi.setPage}
              onPageSizeChange={paginasi.setPageSize}
            />
          </div>
          </>
        )}
      </Card>

      {dipilih && (
        <Card className="flex flex-col gap-3 p-4">
          <div className="flex flex-wrap items-start justify-between gap-2">
            <div>
              <p className="text-sm font-bold tracking-tight text-foreground">{dipilih.title}</p>
              <p className="text-[11px] text-muted-foreground">
                {dipilih.category} · {formatDate(dipilih.created_at)} · pengguna {dipilih.user_id}
              </p>
            </div>
            <Button size="sm" variant="outline" onClick={() => setDipilih(null)}>
              Tutup
            </Button>
          </div>

          <p className="whitespace-pre-wrap rounded-lg border border-border bg-secondary/30 p-3 text-[13px] text-foreground">
            {dipilih.body}
          </p>

          <div className="grid gap-1 text-[11.5px] text-muted-foreground">
            {dipilih.page_url && <p>Halaman: {dipilih.page_url}</p>}
            {dipilih.user_agent && <p className="truncate">Perangkat: {dipilih.user_agent}</p>}
          </div>

          {dipilih.attachments.length > 0 && (
            <div className="flex flex-col gap-1">
              <p className="text-[11.5px] font-semibold text-foreground">Lampiran</p>
              {dipilih.attachments.map((att) => (
                <a
                  key={att.id}
                  href={bugApi.attachmentUrl(dipilih.id, att.id)}
                  className="inline-flex items-center gap-1.5 text-[12px] text-foreground underline underline-offset-2"
                >
                  <Paperclip className="h-3.5 w-3.5" />
                  {att.file_name} ({Math.ceil(att.size_bytes / 1024)} KB)
                </a>
              ))}
            </div>
          )}

          <TriageForm
            awal={dipilih.status}
            catatanAwal={dipilih.admin_note}
            sedangKirim={ubah.isPending}
            onSubmit={(next, note) => ubah.mutate({ id: dipilih.id, next, note })}
          />
        </Card>
      )}
    </div>
  )
}

/** TriageForm mengubah status dan menambah catatan untuk pelapor. */
function TriageForm({
  awal,
  catatanAwal,
  sedangKirim,
  onSubmit,
}: {
  awal: BugStatus
  catatanAwal: string
  sedangKirim: boolean
  onSubmit: (next: BugStatus, note: string) => void
}) {
  const [next, setNext] = useState<BugStatus>(awal)
  const [note, setNote] = useState(catatanAwal)

  return (
    <Can permission="user.edit">
      <div className="flex flex-col gap-2 border-t border-border pt-3">
        <div className="flex flex-wrap items-end gap-2">
          <div className="flex flex-col gap-1">
            <Label htmlFor="triage-status">Ubah status</Label>
            <Select
              id="triage-status"
              value={next}
              onChange={(e) => setNext(e.target.value as BugStatus)}
              className="w-40"
            >
              <option value="baru">Baru</option>
              <option value="ditinjau">Ditinjau</option>
              <option value="selesai">Selesai</option>
              <option value="ditolak">Ditolak</option>
            </Select>
          </div>
          <Button size="sm" isLoading={sedangKirim} onClick={() => onSubmit(next, note)}>
            Simpan
          </Button>
        </div>
        <div className="flex flex-col gap-1">
          <Label htmlFor="triage-catatan">Catatan untuk pelapor (opsional)</Label>
          <textarea
            id="triage-catatan"
            value={note}
            rows={3}
            maxLength={2000}
            onChange={(e) => setNote(e.target.value)}
            className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          />
        </div>
      </div>
    </Can>
  )
}
