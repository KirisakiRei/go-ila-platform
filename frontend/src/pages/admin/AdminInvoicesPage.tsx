import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { adminInvoiceApi } from '@/lib/admin-api'
import { formatIDR, formatInvoice } from '@/features/admin/lib/format'
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
import type { AdminInvoiceItem } from '@/types/admin'

const STATUS_TEXT: Record<string, string> = {
  pending: 'Pending',
  paid: 'Lunas',
  expired: 'Kedaluwarsa',
  cancelled: 'Dibatalkan',
  refunded: 'Refunded',
}

/**
 * AdminInvoicesPage adalah buku tagihan.
 *
 * Dua kekuasaan ada di sini: mencatat pembayaran manual (transfer bank) dan
 * mencatat refund. Keduanya mengubah uang, jadi keduanya wajib beralasan dan
 * tercatat di jejak audit. Refund tidak menurunkan tier otomatis; mencabut
 * akses adalah keputusan bisnis terpisah.
 */
export default function AdminInvoicesPage() {
  const queryClient = useQueryClient()
  const { can } = usePermissions()
  const [status, setStatus] = useState('')
  const [search, setSearch] = useState('')
  const [notice, setNotice] = useState<string | null>(null)
  const [selected, setSelected] = useState<AdminInvoiceItem | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'invoices', status, search],
    queryFn: () => adminInvoiceApi.list({ status, q: search }),
  })

  const rows = useMemo(() => data ?? [], [data])


  // Daftar dipecah per halaman agar tabel panjang tidak memaksa menggulir jauh.

  const paginasi = usePaginasiLokal(rows)
  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['admin', 'invoices'] })
    void queryClient.invalidateQueries({ queryKey: ['admin', 'reports'] })
  }

  const markPaid = useMutation({
    mutationFn: (payload: { id: string; amount: number; note: string }) =>
      adminInvoiceApi.markPaid(payload.id, { amount_idr: payload.amount, note: payload.note }),
    onSuccess: () => {
      setNotice('Pembayaran manual dicatat. Hak paket diberikan sesuai periode invoice.')
      setSelected(null)
      invalidate()
    },
    onError: (error) =>
      setNotice(error instanceof Error ? error.message : 'Gagal mencatat pembayaran.'),
  })

  const refund = useMutation({
    mutationFn: (payload: { id: string; amount: number; reason: string; paymentId: string }) =>
      adminInvoiceApi.refund(payload.id, {
        payment_id: payload.paymentId,
        amount_idr: payload.amount,
        reason: payload.reason,
      }),
    onSuccess: () => {
      setNotice('Refund dicatat. Tier tidak berubah otomatis.')
      setSelected(null)
      invalidate()
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal mencatat refund.'),
  })

  return (
    <>
      <PageHeader
        title="Tagihan"
        description="Pembayaran manual dan refund dicatat di sini; keduanya meninggalkan jejak audit."
      />

      {notice && <Alert>{notice}</Alert>}

      <Card className="p-3">
        <div className="grid gap-2 sm:grid-cols-2">
          <Input
            placeholder="Cari nomor invoice atau id pengguna"
            aria-label="Cari invoice"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
          />
          <Select
            aria-label="Filter status invoice"
            value={status}
            onChange={(event) => setStatus(event.target.value)}
          >
            <option value="">Semua status</option>
            <option value="pending">Menunggu bayar</option>
            <option value="paid">Lunas</option>
            <option value="expired">Kedaluwarsa</option>
            <option value="refunded">Direfund</option>
          </Select>
        </div>
      </Card>

      {isLoading && <Skeleton className="h-40 w-full" />}

      {!isLoading && rows.length === 0 && (
        <EmptyState title="Belum ada tagihan" description="Invoice muncul setelah pengguna checkout." />
      )}

      {rows.length > 0 && (
        <Card className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nomor</TableHead>
                <TableHead>Paket</TableHead>
                <TableHead className="text-right">Total</TableHead>
                <TableHead>Periode</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Tindakan</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {paginasi.baris.map((row) => (
                <TableRow key={row.id}>
                  <TableCell className="text-[12px] font-medium">{row.number}</TableCell>
                  <TableCell className="text-[12px] uppercase">{row.plan_code}</TableCell>
                  <TableCell className="text-right text-[12px] tabular-nums">
                    {formatIDR(row.total_idr)}
                  </TableCell>
                  <TableCell className="text-[11px] text-muted-foreground">
                    {formatInvoice(row.period_start.wib)} → {formatInvoice(row.period_end.wib)}
                  </TableCell>
                  <TableCell>
                    <Badge variant={row.status === 'paid' ? 'solid' : 'default'}>
                      {STATUS_TEXT[row.status] ?? row.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    {row.status === 'pending' && can('billing.manual') ? (
                      <Button size="sm" variant="outline" onClick={() => setSelected(row)}>
                        Catat pembayaran
                      </Button>
                    ) : row.status === 'paid' && can('billing.refund') ? (
                      <Button size="sm" variant="outline" onClick={() => setSelected(row)}>
                        Refund
                      </Button>
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
                unit="invoice"
                pageSize={paginasi.pageSize}
                onPageChange={paginasi.setPage}
                onPageSizeChange={paginasi.setPageSize}
              />
            </div>
          )}
        </Card>
      )}

      {selected?.status === 'pending' && (
        <MarkPaidForm
          invoice={selected}
          pending={markPaid.isPending}
          onClose={() => setSelected(null)}
          onSubmit={(amount, note) => markPaid.mutate({ id: selected.id, amount, note })}
        />
      )}

      {selected?.status === 'paid' && (
        <RefundForm
          invoice={selected}
          pending={refund.isPending}
          onClose={() => setSelected(null)}
          onSubmit={(amount, reason, paymentId) =>
            refund.mutate({ id: selected.id, amount, reason, paymentId })
          }
        />
      )}
    </>
  )
}

function MarkPaidForm({
  invoice,
  pending,
  onClose,
  onSubmit,
}: {
  invoice: AdminInvoiceItem
  pending: boolean
  onClose: () => void
  onSubmit: (amount: number, note: string) => void
}) {
  const [amount, setAmount] = useState(String(invoice.total_idr))
  const [note, setNote] = useState('')

  return (
    <Card className="mt-3 p-4">
      <h2 className="text-sm font-semibold">Catat pembayaran manual — {invoice.number}</h2>
      <p className="mt-1 text-[12px] text-muted-foreground">
        Hanya untuk transfer yang sudah masuk. Nominal di bawah total tagihan akan ditolak.
      </p>
      <div className="mt-3 grid gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-1">
          <Label htmlFor="manual-amount">Nominal diterima (Rp)</Label>
          <Input
            id="manual-amount"
            value={amount}
            onChange={(event) => setAmount(event.target.value.replace(/[^\d]/g, ''))}
          />
        </div>
        <div className="flex flex-col gap-1">
          <Label htmlFor="manual-note">Catatan / bukti transfer</Label>
          <Input id="manual-note" value={note} onChange={(event) => setNote(event.target.value)} />
        </div>
      </div>
      <div className="mt-3 flex gap-2">
        <Button size="sm" disabled={pending || note.trim().length < 3} onClick={() => onSubmit(Number(amount), note)}>
          Tandai lunas
        </Button>
        <Button size="sm" variant="ghost" onClick={onClose}>
          Batal
        </Button>
      </div>
    </Card>
  )
}

function RefundForm({
  invoice,
  pending,
  onClose,
  onSubmit,
}: {
  invoice: AdminInvoiceItem
  pending: boolean
  onClose: () => void
  onSubmit: (amount: number, reason: string, paymentId: string) => void
}) {
  const [amount, setAmount] = useState(String(invoice.total_idr))
  const [reason, setReason] = useState('')
  const [paymentId, setPaymentId] = useState('')

  return (
    <Card className="mt-3 p-4">
      <h2 className="text-sm font-semibold">Catat refund — {invoice.number}</h2>
      <p className="mt-1 text-[12px] text-muted-foreground">
        Refund tidak menurunkan tier. Total refund tidak boleh melebihi nilai pembayaran.
      </p>
      <div className="mt-3 grid gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-1">
          <Label htmlFor="refund-amount">Nominal refund (Rp)</Label>
          <Input
            id="refund-amount"
            value={amount}
            onChange={(event) => setAmount(event.target.value.replace(/[^\d]/g, ''))}
          />
        </div>
        <div className="flex flex-col gap-1">
          <Label htmlFor="refund-payment">Id pembayaran</Label>
          <Input
            id="refund-payment"
            value={paymentId}
            placeholder="Salin dari detail transaksi"
            onChange={(event) => setPaymentId(event.target.value)}
          />
        </div>
        <div className="flex flex-col gap-1 sm:col-span-2">
          <Label htmlFor="refund-reason">Alasan</Label>
          <Input id="refund-reason" value={reason} onChange={(event) => setReason(event.target.value)} />
        </div>
      </div>
      <div className="mt-3 flex gap-2">
        <Button
          size="sm"
          disabled={pending || reason.trim().length < 5 || paymentId.trim() === ''}
          onClick={() => onSubmit(Number(amount), reason, paymentId.trim())}
        >
          Catat refund
        </Button>
        <Button size="sm" variant="ghost" onClick={onClose}>
          Batal
        </Button>
      </div>
    </Card>
  )
}
