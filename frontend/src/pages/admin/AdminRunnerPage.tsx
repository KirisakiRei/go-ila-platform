import { useQuery } from '@tanstack/react-query'
import { Alert, Badge, Card, CardContent, CardDescription, CardHeader, CardTitle, PageHeader, Skeleton, StatusDot } from '@/components/ui'
import { adminSystemApi } from '@/lib/system-api'
import { formatDateTime } from '@/lib/format'

/**
 * AdminRunnerPage memantau proses Runner.
 *
 * Runner yang mati berarti tidak ada eksekusi baru dan tidak ada pemantauan
 * posisi dari sisi platform. Peringatan dikirim otomatis setelah dua menit
 * tanpa laporan (plans/18 §2).
 */
export default function AdminRunnerPage() {
  const runner = useQuery({
    queryKey: ['admin', 'system', 'runner'],
    queryFn: adminSystemApi.runner,
    refetchInterval: 30_000,
  })

  const data = runner.data

  return (
    <>
      <PageHeader
        title="Runner"
        description="Proses yang menilai sinyal per akun, mengeksekusi order, dan memantau posisi."
      />

      {runner.isError && <Alert variant="destructive">Gagal memuat status Runner.</Alert>}
      {runner.isLoading && <Skeleton className="h-32 w-full" />}

      {data && (
        <Card>
          <CardHeader>
            <div className="flex items-start justify-between gap-2">
              <div>
                <CardTitle className="text-sm">Status</CardTitle>
                <CardDescription className="text-xs">
                  Disimpulkan dari laporan engine terakhir yang diterima.
                </CardDescription>
              </div>
              <Badge variant={data.connected ? 'solid' : 'outline'}>
                {data.connected ? 'Aktif' : 'Tidak ada laporan'}
              </Badge>
            </div>
          </CardHeader>
          <CardContent className="space-y-2 text-[13px]">
            <div className="flex items-center justify-between gap-3">
              <span className="text-muted-foreground">Koneksi</span>
              <span className="flex items-center gap-2 font-medium">
                <StatusDot
                  state={data.connected ? 'online' : 'offline'}
                  label={data.connected ? 'aktif' : 'tidak ada laporan'}
                />
                {data.connected ? 'Mengirim laporan' : 'Berhenti melapor'}
              </span>
            </div>
            <div className="flex items-center justify-between gap-3">
              <span className="text-muted-foreground">Terakhir terlihat</span>
              <span className="font-medium">
                {data.last_seen ? formatDateTime(data.last_seen) : 'Belum pernah'}
              </span>
            </div>
          </CardContent>
        </Card>
      )}

      <Alert variant="warning">
        Bila Runner berhenti lebih dari beberapa menit, periksa container di VPS 2 lalu nyalakan ulang.
        Rekonsiliasi posisi berjalan otomatis setelah Runner hidup kembali.
      </Alert>
    </>
  )
}
