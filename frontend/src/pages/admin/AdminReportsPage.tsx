import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { adminReportApi } from '@/lib/admin-api'
import { formatIDR } from '@/features/admin/lib/format'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  EmptyState,
  Input,
  Label,
  PageHeader,
  Skeleton,
  Stat,
  StatGrid,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui'

/** AdminReportsPage merangkum angka bisnis dan operasional. */
export default function AdminReportsPage() {
  const today = new Date()
  const monthAgo = new Date(today.getTime() - 29 * 24 * 60 * 60 * 1000)

  const [from, setFrom] = useState(isoDate(monthAgo))
  const [to, setTo] = useState(isoDate(today))

  const revenue = useQuery({
    queryKey: ['admin', 'report', 'revenue', from, to],
    queryFn: () => adminReportApi.revenue(from, to),
  })
  const customers = useQuery({
    queryKey: ['admin', 'report', 'customers', from, to],
    queryFn: () => adminReportApi.customers(from, to),
  })
  const coupons = useQuery({
    queryKey: ['admin', 'report', 'coupons', from, to],
    queryFn: () => adminReportApi.coupons(from, to),
  })
  const operations = useQuery({
    queryKey: ['admin', 'report', 'operations', from, to],
    queryFn: () => adminReportApi.operations(from, to),
  })

  return (
    <>
      <PageHeader
        title="Laporan"
        description="Pendapatan hanya menghitung invoice lunas. Refund dikurangkan terpisah."
      />

      <Card className="p-3">
        <div className="grid gap-2 sm:grid-cols-2">
          <div className="flex flex-col gap-1">
            <Label htmlFor="report-from">Dari (WIB)</Label>
            <Input id="report-from" type="date" value={from} onChange={(event) => setFrom(event.target.value)} />
          </div>
          <div className="flex flex-col gap-1">
            <Label htmlFor="report-to">Sampai (WIB)</Label>
            <Input id="report-to" type="date" value={to} onChange={(event) => setTo(event.target.value)} />
          </div>
        </div>
      </Card>

      {revenue.isLoading && <Skeleton className="h-24 w-full" />}

      {revenue.data && (
        <StatGrid>
          <Stat label="Pendapatan kotor" value={formatIDR(revenue.data.gross_idr)} />
          <Stat label="Refund" value={formatIDR(revenue.data.refund_idr)} />
          <Stat label="Bersih" value={formatIDR(revenue.data.net_idr)} />
        </StatGrid>
      )}

      {customers.data && (
        <StatGrid>
          <Stat label="Pelanggan baru" value={customers.data.new_users} />
          <Stat label="Pelanggan membayar" value={customers.data.paid_users} />
          <Stat label="Berhenti" value={customers.data.canceled} />
          <Stat label="Mulai trial" value={customers.data.trial_started} />
        </StatGrid>
      )}

      <Card>
        <CardHeader>
          <CardTitle>Pendapatan harian</CardTitle>
          <CardDescription>Hari tanpa transaksi tidak ditampilkan.</CardDescription>
        </CardHeader>
        <CardContent>
          {revenue.data && revenue.data.daily.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Tanggal</TableHead>
                  <TableHead className="text-right">Invoice</TableHead>
                  <TableHead className="text-right">Dibayar</TableHead>
                  <TableHead className="text-right">Refund</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {revenue.data.daily.map((row) => (
                  <TableRow key={row.date}>
                    <TableCell className="text-[12px]">{row.date}</TableCell>
                    <TableCell className="text-right text-[12px] tabular-nums">{row.count}</TableCell>
                    <TableCell className="text-right text-[12px] tabular-nums">
                      {formatIDR(row.paid_idr)}
                    </TableCell>
                    <TableCell className="text-right text-[12px] tabular-nums">
                      {formatIDR(row.refund_idr)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <EmptyState title="Belum ada pendapatan" description="Tidak ada invoice lunas pada rentang ini." />
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Pemakaian kupon</CardTitle>
          <CardDescription>Total potongan yang sudah diberikan.</CardDescription>
        </CardHeader>
        <CardContent>
          {coupons.data && coupons.data.items.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Kode</TableHead>
                  <TableHead className="text-right">Dipakai</TableHead>
                  <TableHead className="text-right">Total potongan</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {coupons.data.items.map((row) => (
                  <TableRow key={row.code}>
                    <TableCell className="text-[12px]">{row.code}</TableCell>
                    <TableCell className="text-right text-[12px] tabular-nums">{row.uses}</TableCell>
                    <TableCell className="text-right text-[12px] tabular-nums">
                      {formatIDR(row.discount_idr)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <EmptyState title="Belum ada kupon dipakai" description="Kupon yang dipakai akan muncul di sini." />
          )}
        </CardContent>
      </Card>

      {operations.data && (
        <Card>
          <CardHeader>
            <CardTitle>Operasional</CardTitle>
            <CardDescription>Angka eksekusi tanpa PnL lintas pengguna.</CardDescription>
          </CardHeader>
          <CardContent>
            <StatGrid>
              <Stat label="Posisi dibuka" value={operations.data.positions_opened} />
              <Stat label="Posisi ditutup" value={operations.data.positions_closed} />
              <Stat label="Simulasi (dry run)" value={operations.data.positions_dry_run} />
              <Stat label="Notifikasi terkirim" value={operations.data.notifications_sent} />
              <Stat label="Notifikasi gagal" value={operations.data.notifications_failed} />
            </StatGrid>
          </CardContent>
        </Card>
      )}
    </>
  )
}

function isoDate(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}
