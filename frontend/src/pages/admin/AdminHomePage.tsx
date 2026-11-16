import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { adminReportApi, adminUserApi } from '@/lib/admin-api'
import { formatIDR } from '@/features/admin/lib/format'
import {
  Alert,
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

/**
 * Sumber data ringkasan. Setiap bagian gagal secara terpisah supaya satu
 * endpoint yang bermasalah tidak mengosongkan seluruh halaman.
 */
type SummarySource = 'revenue' | 'customers' | 'users'

const SOURCE_LABEL: Record<SummarySource, string> = {
  revenue: 'Pendapatan',
  customers: 'Pelanggan',
  users: 'Pengguna terbaru',
}

/**
 * AdminHomePage memberi gambaran cepat keadaan platform hari ini.
 *
 * Angka nol hanya ditampilkan bila server benar-benar mengirim nol. Saat
 * permintaan gagal, halaman menampilkan pesan galat dan tombol ulang, bukan
 * angka nol yang menyamarkan masalah.
 */
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

  const failed = [
    revenue.isError && ('revenue' as const),
    customers.isError && ('customers' as const),
    users.isError && ('users' as const),
  ].filter((source): source is SummarySource => Boolean(source))

  const refetchFailed = () => {
    if (revenue.isError) void revenue.refetch()
    if (customers.isError) void customers.refetch()
    if (users.isError) void users.refetch()
  }

  const summaryLoading = revenue.isLoading || customers.isLoading
  const summaryStale = (revenue.isError && revenue.data) || (customers.isError && customers.data)

  return (
    <>
      <PageHeader
        title="Ringkasan"
        description="Keadaan platform dalam 30 hari terakhir. Angka rupiah hanya menghitung invoice lunas."
      />

      {failed.length > 0 && (
        <Alert variant="warning">
          <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
            <span>
              Gagal memuat: {failed.map((source) => SOURCE_LABEL[source]).join(', ')}.
            </span>
            <Button variant="outline" size="sm" onClick={refetchFailed}>
              Coba lagi
            </Button>
          </div>
        </Alert>
      )}

      {summaryStale && (
        <Alert variant="info">Sebagian angka di bawah berasal dari pemuatan sebelumnya dan mungkin belum terbaru.</Alert>
      )}

      {summaryLoading ? (
        <Skeleton className="h-24 w-full" />
      ) : revenue.data && customers.data ? (
        <StatGrid>
          <Stat label="Pendapatan bersih" value={formatIDR(revenue.data.net_idr)} />
          <Stat label="Pengguna berbayar" value={customers.data.paid_users} />
          <Stat label="Pengguna baru" value={customers.data.new_users} />
          <Stat label="Uji coba aktif" value={customers.data.trial_started} />
        </StatGrid>
      ) : null}

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Pengguna Terbaru</CardTitle>
            <CardDescription>Lima akun yang paling baru mendaftar.</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-2">
            {users.isLoading && <Skeleton className="h-16 w-full" />}
            {!users.isLoading &&
              (users.data?.data ?? []).map((user) => (
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
            {!users.isLoading && users.data && users.data.data.length === 0 && (
              <p className="text-[12px] text-muted-foreground">Belum ada pengguna terdaftar.</p>
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
              <Link to="/admin/pengguna">Kelola Pengguna</Link>
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
