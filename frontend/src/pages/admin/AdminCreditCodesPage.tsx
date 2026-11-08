import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { adminCreditCodeApi } from '@/lib/admin-api'
import { formatIDR, formatInvoice } from '@/features/admin/lib/format'
import {
  Alert,
  Badge,
  Button,
  Card,
  EmptyState,
  Input,
  Label,
  Modal,
  PageHeader,
  Select,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui'

const STATUS_TEXT: Record<string, string> = {
  active: 'Berjalan',
  scheduled: 'Terjadwal',
  expired: 'Kedaluwarsa',
  exhausted: 'Habis',
  disabled: 'Dihentikan',
}

/**
 * AdminCreditCodesPage mengelola voucher saldo.
 *
 * Voucher menambah saldo yang bisa dipakai memotong tagihan berikutnya.
 * Kode tidak pernah dihapus, hanya dihentikan, agar riwayat penukaran utuh.
 */
export default function AdminCreditCodesPage() {
  const queryClient = useQueryClient()
  const [status, setStatus] = useState('')
  const [search, setSearch] = useState('')
  const [notice, setNotice] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'credit-codes', status, search],
    queryFn: () => adminCreditCodeApi.list({ status, q: search }),
  })

  const rows = data ?? []
  const invalidate = () => void queryClient.invalidateQueries({ queryKey: ['admin', 'credit-codes'] })

  const disable = useMutation({
    mutationFn: (id: string) => adminCreditCodeApi.disable(id, 'Dihentikan dari panel admin'),
    onSuccess: () => {
      setNotice('Kode dihentikan. Saldo yang sudah ditukar tetap milik pengguna.')
      invalidate()
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal menghentikan kode.'),
  })

  return (
    <>
      <PageHeader
        title="Kode Kredit"
        description="Voucher saldo untuk kompensasi atau kampanye; pengguna menukarnya sendiri dari dashboard."
        actions={
          <Button size="sm" onClick={() => setCreating(true)}>
            Kode baru
          </Button>
        }
      />

      {notice && <Alert>{notice}</Alert>}

      <Modal
        open={creating}
        onClose={() => setCreating(false)}
        title="Kode Kredit Baru"
        description="Voucher saldo untuk kompensasi atau kampanye; pengguna menukarnya dari dashboard."
        size="lg"
      >
        <CreateCreditCodeForm
          onDone={() => {
            setCreating(false)
            setNotice('Kode kredit dibuat.')
            invalidate()
          }}
          onCancel={() => setCreating(false)}
          onError={setNotice}
        />
      </Modal>

      <Card className="p-3">
        <div className="grid gap-2 sm:grid-cols-2">
          <Input
            placeholder="Cari kode"
            aria-label="Cari kode kredit"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
          />
          <Select
            aria-label="Filter status kode kredit"
            value={status}
            onChange={(event) => setStatus(event.target.value)}
          >
            <option value="">Semua status</option>
            <option value="active">Berjalan</option>
            <option value="disabled">Dihentikan</option>
          </Select>
        </div>
      </Card>

      {isLoading && <Skeleton className="h-40 w-full" />}

      {!isLoading && rows.length === 0 && (
        <EmptyState title="Belum ada kode kredit" description="Buat kode untuk kompensasi pengguna." />
      )}

      {rows.length > 0 && (
        <Card className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Kode</TableHead>
                <TableHead className="text-right">Nilai</TableHead>
                <TableHead>Periode</TableHead>
                <TableHead className="text-right">Dipakai</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Tindakan</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => (
                <TableRow key={row.id}>
                  <TableCell className="text-[13px] font-medium">{row.code}</TableCell>
                  <TableCell className="text-right text-[12px] tabular-nums">
                    {formatIDR(row.amount_idr)}
                  </TableCell>
                  <TableCell className="text-[11px] text-muted-foreground">
                    {formatInvoice(row.starts_at.wib)} → {formatInvoice(row.ends_at.wib)}
                  </TableCell>
                  <TableCell className="text-right text-[12px] tabular-nums">
                    {row.redeemed}
                    {row.max_redemptions > 0 ? ` / ${row.max_redemptions}` : ''}
                  </TableCell>
                  <TableCell>
                    <Badge variant={row.status === 'active' ? 'solid' : 'default'}>
                      {STATUS_TEXT[row.status] ?? row.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={row.status === 'disabled' || disable.isPending}
                      onClick={() => disable.mutate(row.id)}
                    >
                      Hentikan
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>
      )}
    </>
  )
}

function CreateCreditCodeForm({
  onDone,
  onCancel,
  onError,
}: {
  onDone: () => void
  onCancel?: () => void
  onError: (message: string) => void
}) {
  const [code, setCode] = useState('')
  const [amount, setAmount] = useState('')
  const [startsAt, setStartsAt] = useState('')
  const [endsAt, setEndsAt] = useState('')
  const [maxRedemptions, setMaxRedemptions] = useState('1')
  const [maxPerUser, setMaxPerUser] = useState('1')

  const create = useMutation({
    mutationFn: () =>
      adminCreditCodeApi.create({
        code: code.toUpperCase(),
        amount_idr: Number(amount.replace(/[^\d]/g, '')),
        max_redemptions: Number(maxRedemptions) || 1,
        max_per_user: Number(maxPerUser) || 1,
        starts_at: new Date(startsAt).toISOString(),
        ends_at: new Date(endsAt).toISOString(),
        reason: 'Kode kredit dibuat dari panel admin',
      }),
    onSuccess: onDone,
    onError: (error) => onError(error instanceof Error ? error.message : 'Gagal membuat kode.'),
  })

  const ready = code.trim() !== '' && amount !== '' && startsAt !== '' && endsAt !== ''

  return (
    <div className="space-y-4">
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-1">
          <Label htmlFor="credit-code">Kode</Label>
          <Input
            id="credit-code"
            value={code}
            onChange={(event) => setCode(event.target.value.toUpperCase())}
          />
        </div>
        <div className="flex flex-col gap-1">
          <Label htmlFor="credit-amount">Nilai (Rp)</Label>
          <Input
            id="credit-amount"
            value={amount}
            onChange={(event) => setAmount(event.target.value.replace(/[^\d]/g, ''))}
          />
        </div>
        <div className="flex flex-col gap-1">
          <Label htmlFor="credit-max">Batas penukaran</Label>
          <Input
            id="credit-max"
            value={maxRedemptions}
            onChange={(event) => setMaxRedemptions(event.target.value)}
          />
        </div>
        <div className="flex flex-col gap-1">
          <Label htmlFor="credit-per-user">Maksimal per pengguna</Label>
          <Input
            id="credit-per-user"
            value={maxPerUser}
            onChange={(event) => setMaxPerUser(event.target.value)}
          />
        </div>
        <div className="flex flex-col gap-1">
          <Label htmlFor="credit-start">Mulai</Label>
          <Input
            id="credit-start"
            type="date"
            value={startsAt}
            onChange={(event) => setStartsAt(event.target.value)}
          />
        </div>
        <div className="flex flex-col gap-1">
          <Label htmlFor="credit-end">Berakhir</Label>
          <Input
            id="credit-end"
            type="date"
            value={endsAt}
            onChange={(event) => setEndsAt(event.target.value)}
          />
        </div>
      </div>
      <div className="flex justify-end gap-2 border-t border-border pt-4">
        {onCancel && (
          <Button variant="outline" size="sm" type="button" onClick={onCancel}>
            Batal
          </Button>
        )}
        <Button size="sm" disabled={!ready || create.isPending} onClick={() => create.mutate()}>
          {create.isPending ? 'Menyimpan…' : 'Simpan kode'}
        </Button>
      </div>
    </div>
  )
}
