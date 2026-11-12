import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { adminTrialApi } from '@/lib/admin-api'
import { formatInvoice } from '@/features/admin/lib/format'
import { usePermissions } from '@/features/admin/hooks/use-permissions'
import {
  Alert,
  Badge,
  Button,
  Card,
  EmptyState,
  Input,
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
import type { AdminTrialItem } from '@/types/admin'

const STATUS_TEXT: Record<string, string> = {
  active: 'Berjalan',
  ended: 'Berakhir',
  never: 'Belum dipakai',
}

/**
 * AdminTrialsPage mengelola uji coba 7 hari.
 *
 * Trial hanya sekali per akun dan memberi paket Basic dengan satu akun
 * broker. Admin bisa memperpanjang atau menghentikan, tetapi tidak bisa
 * membuka trial baru untuk akun yang sudah pernah memakainya — itu akan
 * membuka pintu daftar ulang berulang.
 */
export default function AdminTrialsPage() {
  const queryClient = useQueryClient()
  const { can } = usePermissions()
  const [status, setStatus] = useState('')
  const [notice, setNotice] = useState<string | null>(null)
  const [selected, setSelected] = useState<AdminTrialItem | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'trials', status],
    queryFn: () => adminTrialApi.list({ status }),
  })

  const rows = useMemo(() => data ?? [], [data])


  // Daftar dipecah per halaman agar tabel panjang tidak memaksa menggulir jauh.

  const paginasi = usePaginasiLokal(rows)
  const invalidate = () => void queryClient.invalidateQueries({ queryKey: ['admin', 'trials'] })

  const end = useMutation({
    mutationFn: (id: string) => adminTrialApi.end(id, 'Dihentikan dari panel admin'),
    onSuccess: () => {
      setNotice('Uji coba dihentikan. Akun kembali ke paket aslinya.')
      invalidate()
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal menghentikan trial.'),
  })

  return (
    <>
      <PageHeader
        title="Uji Coba"
        description="Trial 7 hari: paket Basic, satu akun broker, seluruh eksekusi simulasi."
      />

      {notice && <Alert>{notice}</Alert>}

      <Card className="p-3">
        <Select
          aria-label="Filter status uji coba"
          value={status}
          onChange={(event) => setStatus(event.target.value)}
        >
          <option value="">Semua</option>
          <option value="active">Berjalan</option>
          <option value="ended">Berakhir</option>
        </Select>
      </Card>

      {isLoading && <Skeleton className="h-40 w-full" />}

      {!isLoading && rows.length === 0 && (
        <EmptyState
          title="Belum ada pengguna trial"
          description="Pengguna muncul di sini setelah mengaktifkan uji coba."
        />
      )}

      {rows.length > 0 && (
        <Card className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Pengguna</TableHead>
                <TableHead>Paket asli</TableHead>
                <TableHead>Mulai</TableHead>
                <TableHead>Berakhir</TableHead>
                <TableHead className="text-right">Sisa hari</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Tindakan</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {paginasi.baris.map((row) => (
                <TableRow key={row.user_id}>
                  <TableCell className="text-[12px]">
                    <span className="font-medium">{row.name}</span>
                    <span className="block text-[11px] text-muted-foreground">{row.email}</span>
                  </TableCell>
                  <TableCell className="text-[12px] uppercase">{row.tier}</TableCell>
                  <TableCell className="text-[11px] text-muted-foreground">
                    {formatInvoice(row.started_at?.wib)}
                  </TableCell>
                  <TableCell className="text-[11px] text-muted-foreground">
                    {formatInvoice(row.expires_at?.wib)}
                  </TableCell>
                  <TableCell className="text-right text-[12px] tabular-nums">
                    {row.status === 'active' ? row.days_remaining : '—'}
                  </TableCell>
                  <TableCell>
                    <Badge variant={row.status === 'active' ? 'solid' : 'default'}>
                      {STATUS_TEXT[row.status] ?? row.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    {row.status === 'active' && can('user.tier') ? (
                      <div className="flex justify-end gap-2">
                        <Button size="sm" variant="outline" onClick={() => setSelected(row)}>
                          Perpanjang
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          disabled={end.isPending}
                          onClick={() => end.mutate(row.user_id)}
                        >
                          Hentikan
                        </Button>
                      </div>
                    ) : (
                      <span className="text-[11px] text-muted-foreground">—</span>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>          </Table>
          {paginasi.total > 0 && (
            <div className="border-t border-border px-3 py-2">
              <Pagination
                page={paginasi.page}
                totalPages={paginasi.totalPages}
                shown={paginasi.baris.length}
                total={paginasi.total}
                unit="uji coba"
                pageSize={paginasi.pageSize}
                onPageChange={paginasi.setPage}
                onPageSizeChange={paginasi.setPageSize}
              />
            </div>
          )}
        </Card>
      )}

      {selected && (
        <ExtendForm
          trial={selected}
          onClose={() => setSelected(null)}
          onDone={(message) => {
            setSelected(null)
            setNotice(message)
            invalidate()
          }}
          onError={setNotice}
        />
      )}
    </>
  )
}

function ExtendForm({
  trial,
  onClose,
  onDone,
  onError,
}: {
  trial: AdminTrialItem
  onClose: () => void
  onDone: (message: string) => void
  onError: (message: string) => void
}) {
  const [days, setDays] = useState('7')
  const [reason, setReason] = useState('')

  const extend = useMutation({
    mutationFn: () => adminTrialApi.extend(trial.user_id, { days: Number(days), reason }),
    onSuccess: () => onDone(`Uji coba ${trial.email} diperpanjang ${days} hari.`),
    onError: (error) => onError(error instanceof Error ? error.message : 'Gagal memperpanjang trial.'),
  })

  return (
    <Card className="mt-3 p-4">
      <h2 className="text-sm font-semibold">Perpanjang uji coba — {trial.email}</h2>
      <p className="mt-1 text-[12px] text-muted-foreground">
        Maksimal 30 hari per perpanjangan. Perpanjangan tetap dalam paket Basic.
      </p>
      <div className="mt-3 grid gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-1">
          <Label htmlFor="extend-days">Tambah hari</Label>
          <Input
            id="extend-days"
            value={days}
            onChange={(event) => setDays(event.target.value.replace(/[^\d]/g, ''))}
          />
        </div>
        <div className="flex flex-col gap-1">
          <Label htmlFor="extend-reason">Alasan</Label>
          <Input
            id="extend-reason"
            value={reason}
            onChange={(event) => setReason(event.target.value)}
          />
        </div>
      </div>
      <div className="mt-3 flex gap-2">
        <Button
          size="sm"
          disabled={extend.isPending || reason.trim().length < 3 || Number(days) < 1}
          onClick={() => extend.mutate()}
        >
          Simpan
        </Button>
        <Button size="sm" variant="ghost" onClick={onClose}>
          Batal
        </Button>
      </div>
    </Card>
  )
}
