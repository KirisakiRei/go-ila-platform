import { useQuery } from '@tanstack/react-query'
import { Alert, Card, CardContent, CardDescription, CardHeader, CardTitle, PageHeader, Skeleton, Stat, StatGrid } from '@/components/ui'
import { adminSystemApi } from '@/lib/system-api'

/**
 * AdminBrokerHealthPage merangkum kesehatan akun broker seluruh pelanggan.
 *
 * Akun yang gagal tersambung tidak menghentikan akun lain; satu token yang
 * kedaluwarsa hanya memengaruhi pemiliknya (plans/18 §2).
 */
export default function AdminBrokerHealthPage() {
  const brokers = useQuery({
    queryKey: ['admin', 'system', 'brokers'],
    queryFn: adminSystemApi.brokers,
    refetchInterval: 60_000,
  })

  const data = brokers.data
  const failing = (data?.error_count ?? 0) > 0

  return (
    <>
      <PageHeader
        title="Kesehatan Broker"
        description="Diperbarui tiap menit dari telemetry Runner. Kredensial tidak pernah ditampilkan."
      />

      {brokers.isError && <Alert variant="destructive">Gagal memuat kesehatan broker.</Alert>}
      {brokers.isLoading && <Skeleton className="h-28 w-full" />}

      {data && (
        <>
          <StatGrid>
            <Stat label="Akun aktif" value={String(data.total)} />
            <Stat label="Tersambung" value={String(data.connected)} />
            <Stat label="Gagal" value={String(data.error_count)} />
            <Stat label="Dibekukan" value={String(data.frozen)} />
          </StatGrid>

          {failing && (
            <Alert variant="warning">
              Ada akun yang gagal tersambung. Bila hanya satu atau dua, kemungkinan token broker milik
              pengguna itu kedaluwarsa; minta pengguna menghubungkan ulang. Bila banyak sekaligus,
              pertimbangkan mode hanya-baca sampai gangguan broker mereda.
            </Alert>
          )}

          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Yang perlu diingat</CardTitle>
              <CardDescription className="text-xs leading-relaxed">
                Admin tidak dapat melihat atau mengambil alih kredensial akun. Pembekuan akun hanya
                menghentikan eksekusi baru; posisi yang sudah terbuka tetap dikelola sampai ditutup
                broker.
              </CardDescription>
            </CardHeader>
            <CardContent />
          </Card>
        </>
      )}
    </>
  )
}
