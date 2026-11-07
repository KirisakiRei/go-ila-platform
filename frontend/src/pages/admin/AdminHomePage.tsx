import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { adminReportApi, adminUserApi } from '@/lib/admin-api'
import { formatIDR } from '@/features/admin/lib/format'
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  PageHeader,
  Skeleton,
  Stat,
  StatGrid,
} from '@/components/ui'

/** AdminHomePage memberi gambaran cepat keadaan platform hari ini. */
export default function AdminHomePage() {
  const revenue = useQuery({ queryKey: ['admin', 'report', 'revenue'], queryFn: () => adminReportApi.revenue() })
  const customers = useQuery({
    queryKey: ['admin', 'report', 'customers'],
    queryFn: () => adminReportApi.customers(),
  })
  const users = useQuery({
    queryKey: ['admin', 'users', 'recent'],
    queryFn: () => adminUserApi.list({ page: 1, per_page: 5 }),
  })

  return (
    <>
      <PageHeader
        title="Ringkasan"
        description="Keadaan platform dalam 30 hari terakhir. Angka rupiah hanya menghitung invoice lunas."
      />

      {revenue.isLoading ? (
        <Skeleton className="h-24 w-full" />
      ) : (
        <StatGrid>
          <Stat label="Pendapatan bersih" value={formatIDR(revenue.data?.net_idr ?? 0)} />
          <Stat label="Pelanggan membayar" value={customers.data?.paid_users ?? 0} />
          <Stat label="Pelanggan baru" value={customers.data?.new_users ?? 0} />
          <Stat label="Mulai trial" value={customers.data?.trial_started ?? 0} />
        </StatGrid>
      )}

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Pelanggan terbaru</CardTitle>
            <CardDescription>Lima akun yang paling baru mendaftar.</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-2">
            {(users.data?.data ?? []).map((user) => (
              <Link
                key={user.id}
                to={`/admin/pengguna/${user.id}`}
                className="flex items-center justify-between rounded-md border border-border px-3 py-2 text-[12px] hover:bg-muted/50"
              >
                <span className="flex flex-col">
                  <span className="font-medium">{user.name}</span>
                  <span className="text-[11px] text-muted-foreground">{user.email}</span>
                </span>
                <span className="text-[11px] text-muted-foreground">{user.tier}</span>
              </Link>
            ))}
            {users.data && users.data.data.length === 0 && (
              <p className="text-[12px] text-muted-foreground">Belum ada pelanggan.</p>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Pintasan</CardTitle>
            <CardDescription>Tugas yang sering dikerjakan staf.</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-wrap gap-2">
            <Button variant="outline" size="sm" asChild>
              <Link to="/admin/pengguna">Cari pelanggan</Link>
            </Button>
            <Button variant="outline" size="sm" asChild>
              <Link to="/admin/laporan">Buka laporan</Link>
            </Button>
            <Button variant="outline" size="sm" asChild>
              <Link to="/admin/audit">Periksa audit</Link>
            </Button>
          </CardContent>
        </Card>
      </div>
    </>
  )
}
