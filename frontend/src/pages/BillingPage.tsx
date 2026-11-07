import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { request } from '@/lib/http'
import { publicPlanApi } from '@/lib/admin-api'
import { formatIDR, formatInvoice } from '@/features/admin/lib/format'
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  EmptyState,
  PageHeader,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui'
import type { TimePair } from '@/types/api'

type Invoice = {
  id: string
  number: string
  plan_code: string
  total_idr: number
  status: string
  created_at: TimePair
  paid_at?: TimePair
}

const STATUS_TEXT: Record<string, string> = {
  pending: 'Menunggu pembayaran',
  paid: 'Lunas',
  expired: 'Kedaluwarsa',
  cancelled: 'Dibatalkan',
  refunded: 'Dikembalikan',
}

/**
 * BillingPage menampilkan paket dan riwayat tagihan pelanggan.
 *
 * Halaman ini tidak mengubah tier. Kenaikan tier hanya terjadi setelah
 * webhook pembayaran bertanda tangan sah diterima server.
 */
export default function BillingPage() {
  const plans = useQuery({ queryKey: ['public', 'plans'], queryFn: publicPlanApi.list })
  const invoices = useQuery({
    queryKey: ['billing', 'invoices'],
    queryFn: () => request<Invoice[]>('/billing/invoices'),
  })
  const subscription = useQuery({
    queryKey: ['billing', 'subscription'],
    queryFn: () => request<{ plan_code: string; period_end: TimePair; status: string } | null>(
      '/billing/subscription',
    ),
  })

  return (
    <>
      <PageHeader
        title="Langganan"
        description="Pembayaran diproses lewat gateway. Status lunas hanya berubah setelah server menerima konfirmasi resmi."
      />

      {subscription.data && (
        <Card>
          <CardHeader>
            <CardTitle>Paket aktif: {subscription.data.plan_code}</CardTitle>
            <CardDescription>
              Berlaku sampai {formatInvoice(subscription.data.period_end.wib)} WIB.
            </CardDescription>
          </CardHeader>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Pilihan paket</CardTitle>
          <CardDescription>Harga sudah termasuk pajak sesuai ketentuan yang berlaku.</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 sm:grid-cols-3">
          {plans.isLoading && <Skeleton className="h-32 w-full sm:col-span-3" />}
          {(plans.data ?? []).map((plan) => (
            <div key={plan.code} className="rounded-md border border-border p-3">
              <div className="flex items-center justify-between">
                <p className="text-[13px] font-semibold">{plan.name}</p>
                {plan.ila3_coming_soon && <Badge variant="default">ILA 3.0 segera</Badge>}
              </div>
              <p className="mt-1 text-[15px] font-semibold tabular-nums">{formatIDR(plan.price_idr)}</p>
              <p className="mt-1 text-[11px] leading-relaxed text-muted-foreground">
                {plan.description}
              </p>
              {plan.code !== 'free' && (
                <Button className="mt-3 w-full" size="sm" variant="outline" asChild>
                  <Link to={`/langganan/bayar?paket=${plan.code}`}>Pilih paket</Link>
                </Button>
              )}
            </div>
          ))}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Riwayat tagihan</CardTitle>
          <CardDescription>Invoice lama tidak berubah walau harga paket berubah.</CardDescription>
        </CardHeader>
        <CardContent>
          {invoices.isLoading && <Skeleton className="h-24 w-full" />}
          {invoices.data && invoices.data.length === 0 && (
            <EmptyState title="Belum ada tagihan" description="Tagihan muncul setelah Anda memilih paket." />
          )}
          {invoices.data && invoices.data.length > 0 && (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Nomor</TableHead>
                  <TableHead>Paket</TableHead>
                  <TableHead className="text-right">Total</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Terbit (WIB)</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {invoices.data.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell className="text-[12px]">{row.number}</TableCell>
                    <TableCell className="text-[12px]">{row.plan_code}</TableCell>
                    <TableCell className="text-right text-[12px] tabular-nums">
                      {formatIDR(row.total_idr)}
                    </TableCell>
                    <TableCell className="text-[12px]">{STATUS_TEXT[row.status] ?? row.status}</TableCell>
                    <TableCell className="text-[12px] text-muted-foreground">
                      {formatInvoice(row.created_at.wib)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </>
  )
}
