import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { adminCouponApi } from '@/lib/admin-api'
import { formatIDR, formatInvoice, formatPercent } from '@/features/admin/lib/format'
import { Can } from '@/features/admin/components/Can'
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
import type { CouponItem } from '@/types/admin'

const STATUS_TEXT: Record<string, string> = {
  active: 'Berjalan',
  scheduled: 'Terjadwal',
  expired: 'Kedaluwarsa',
  exhausted: 'Habis',
  disabled: 'Dihentikan',
}

/**
 * AdminCouponsPage mengelola kupon.
 *
 * Kupon yang sudah dipakai tidak dihapus dan nilainya tidak diubah, karena
 * itu akan membuat laporan pemakaian menyesatkan (plans/14 §4.6).
 */
export default function AdminCouponsPage() {
  const queryClient = useQueryClient()
  const [status, setStatus] = useState('')
  const [search, setSearch] = useState('')
  const [notice, setNotice] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'coupons', status, search],
    queryFn: () => adminCouponApi.list({ status, q: search }),
  })

  const rows: CouponItem[] = data ?? []

  const invalidate = () => void queryClient.invalidateQueries({ queryKey: ['admin', 'coupons'] })

  const disable = useMutation({
    mutationFn: (id: string) => adminCouponApi.disable(id, 'Dihentikan dari panel admin'),
    onSuccess: () => {
      setNotice('Kupon dihentikan. Riwayat pemakaian tetap utuh.')
      invalidate()
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal menghentikan kupon.'),
  })

  return (
    <>
      <PageHeader
        title="Kupon"
        description="Potongan hanya berlaku saat checkout; harga lama di invoice tidak berubah."
        actions={
          <Can permission="billing.voucher">
            <Button size="sm" onClick={() => setCreating(true)}>
              Kupon baru
            </Button>
          </Can>
        }
      />

      {notice && <Alert>{notice}</Alert>}

      <Modal
        open={creating}
        onClose={() => setCreating(false)}
        title="Kupon Baru"
        description="Potongan harga berlaku saat checkout; kupon lama di invoice tidak berubah."
        size="lg"
      >
        <CreateCouponForm
          onDone={() => {
            setCreating(false)
            setNotice('Kupon dibuat.')
            invalidate()
          }}
          onCancel={() => setCreating(false)}
          onError={(message) => setNotice(message)}
        />
      </Modal>

      <Card className="p-3">
        <div className="grid gap-2 sm:grid-cols-2">
          <Input
            placeholder="Cari kode"
            value={search}
            aria-label="Cari kupon"
            onChange={(event) => setSearch(event.target.value)}
          />
          <Select
            aria-label="Filter status kupon"
            value={status}
            onChange={(event) => setStatus(event.target.value)}
          >
            <option value="">Semua status</option>
            <option value="active">Berjalan</option>
            <option value="scheduled">Terjadwal</option>
            <option value="expired">Kedaluwarsa</option>
            <option value="exhausted">Habis</option>
            <option value="disabled">Dihentikan</option>
          </Select>
        </div>
      </Card>

      {isLoading && <Skeleton className="h-40 w-full" />}

      {!isLoading && rows.length === 0 && (
        <EmptyState title="Belum ada kupon" description="Buat kupon pertama untuk kampanye berikutnya." />
      )}

      {rows.length > 0 && (
        <Card className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Kode</TableHead>
                <TableHead>Potongan</TableHead>
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
                  <TableCell className="text-[12px]">
                    {row.discount_type === 'percent'
                      ? formatPercent(row.discount_value)
                      : formatIDR(row.discount_value)}
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
                    <Can permission="billing.voucher">
                      <Button
                        variant="outline"
                        size="sm"
                        disabled={row.status === 'disabled' || disable.isPending}
                        onClick={() => disable.mutate(row.id)}
                      >
                        Hentikan
                      </Button>
                    </Can>
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

function CreateCouponForm({
  onDone,
  onCancel,
  onError,
}: {
  onDone: () => void
  onCancel?: () => void
  onError: (message: string) => void
}) {
  const [code, setCode] = useState('')
  const [type, setType] = useState<'percent' | 'fixed'>('percent')
  const [value, setValue] = useState('')
  const [startsAt, setStartsAt] = useState('')
  const [endsAt, setEndsAt] = useState('')
  const [maxPerUser, setMaxPerUser] = useState('1')

  const create = useMutation({
    mutationFn: () =>
      adminCouponApi.create({
        code: code.toUpperCase(),
        description: '',
        discount_type: type,
        discount_value: Number(value.replace(/[^\d]/g, '')),
        starts_at: new Date(startsAt).toISOString(),
        ends_at: new Date(endsAt).toISOString(),
        max_per_user: Number(maxPerUser) || 1,
      }),
    onSuccess: onDone,
    onError: (error) => onError(error instanceof Error ? error.message : 'Gagal membuat kupon.'),
  })

  const ready = code.trim() !== '' && value !== '' && startsAt !== '' && endsAt !== ''

  return (
    <div className="space-y-4">
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-1">
          <Label htmlFor="coupon-code">Kode</Label>
          <Input id="coupon-code" value={code} onChange={(event) => setCode(event.target.value.toUpperCase())} />
        </div>
        <div className="flex flex-col gap-1">
          <Label htmlFor="coupon-type">Jenis potongan</Label>
          <Select
            id="coupon-type"
            value={type}
            onChange={(event) => setType(event.target.value as 'percent' | 'fixed')}
          >
            <option value="percent">Persen</option>
            <option value="fixed">Rupiah</option>
          </Select>
        </div>
        <div className="flex flex-col gap-1">
          <Label htmlFor="coupon-value">Nilai</Label>
          <Input id="coupon-value" value={value} onChange={(event) => setValue(event.target.value)} />
        </div>
        <div className="flex flex-col gap-1">
          <Label htmlFor="coupon-per-user">Maksimal per pengguna</Label>
          <Input
            id="coupon-per-user"
            value={maxPerUser}
            onChange={(event) => setMaxPerUser(event.target.value)}
          />
        </div>
        <div className="flex flex-col gap-1">
          <Label htmlFor="coupon-start">Mulai</Label>
          <Input
            id="coupon-start"
            type="date"
            value={startsAt}
            onChange={(event) => setStartsAt(event.target.value)}
          />
        </div>
        <div className="flex flex-col gap-1">
          <Label htmlFor="coupon-end">Berakhir</Label>
          <Input id="coupon-end" type="date" value={endsAt} onChange={(event) => setEndsAt(event.target.value)} />
        </div>
      </div>
      <div className="flex justify-end gap-2 border-t border-border pt-4">
        {onCancel && (
          <Button variant="outline" size="sm" type="button" onClick={onCancel}>
            Batal
          </Button>
        )}
        <Button size="sm" disabled={!ready || create.isPending} onClick={() => create.mutate()}>
          {create.isPending ? 'Menyimpan…' : 'Simpan kupon'}
        </Button>
      </div>
    </div>
  )
}
