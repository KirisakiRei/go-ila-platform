import type { ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Alert, Badge, Card, CardContent, CardDescription, CardHeader, CardTitle, PageHeader, Skeleton } from '@/components/ui'
import { StatusDot } from '@/components/ui'
import { adminSystemApi } from '@/lib/system-api'
import { formatDateTime } from '@/lib/format'

/**
 * AdminHealthPage menampilkan kondisi Runner, engine, dan akun broker.
 *
 * Semua nilai berasal dari laporan terakhir yang masuk ke database, bukan
 * tebakan: engine yang berhenti melapor ditandai "basi" walau statusnya
 * masih tertulis tersambung (plans/18 §6).
 */
export default function AdminHealthPage() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['admin', 'system', 'overview'],
    queryFn: adminSystemApi.overview,
    refetchInterval: 30_000,
  })

  return (
    <>
      <PageHeader
        title="Kesehatan Sistem"
        description="Diperbarui otomatis tiap 30 detik. Status Stale menandakan telemetri terakhir lebih dari 3 menit."
      />

      {error && <Alert variant="destructive">Gagal memuat kondisi sistem.</Alert>}
      {isLoading && <Skeleton className="h-40 w-full" />}

      {data && (
        <div className="grid gap-4 lg:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Runner</CardTitle>
              <CardDescription className="text-xs">
                Proses yang mengeksekusi order dan memantau posisi.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-2 text-[13px]">
              <Row label="Status">
                <StatusDot state={data.runner.connected ? 'online' : 'offline'} label={data.runner.connected ? 'tersambung' : 'tidak ada laporan'} />
                {data.runner.connected ? 'Tersambung' : 'Tidak ada laporan'}
              </Row>
              <Row label="Terakhir terlihat">
                {data.runner.last_seen ? formatDateTime(data.runner.last_seen) : 'Belum pernah'}
              </Row>
              <Row label="Waktu server">{formatDateTime(data.server_time)}</Row>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-sm">Akun Broker</CardTitle>
              <CardDescription className="text-xs">
                Akun yang gagal tersambung tidak menghentikan akun lain.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-2 text-[13px]">
              <Row label="Total akun aktif">{data.broker_health.total}</Row>
              <Row label="Tersambung">{data.broker_health.connected}</Row>
              <Row label="Gagal">
                {data.broker_health.error_count > 0 ? (
                  <Badge variant="solid">{data.broker_health.error_count}</Badge>
                ) : (
                  '0'
                )}
              </Row>
              <Row label="Dibekukan">{data.broker_health.frozen}</Row>
            </CardContent>
          </Card>

          <Card className="lg:col-span-2">
            <CardHeader>
              <CardTitle className="text-sm">Engine</CardTitle>
              <CardDescription className="text-xs">
                Engine mencari setup; Runner yang mengeksekusi. Bila engine berhenti, posisi tetap dikelola.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              {data.engines.length === 0 && (
                <p className="text-[13px] text-muted-foreground">Belum ada engine yang pernah melapor.</p>
              )}
              {data.engines.map((engine) => (
                <div key={engine.strategy} className="flex flex-wrap items-center justify-between gap-2 border-b border-border pb-2 last:border-0 last:pb-0">
                  <div className="flex items-center gap-2">
                    <StatusDot state={engine.connected ? 'online' : 'offline'} label={engine.connected ? 'tersambung' : 'tidak ada laporan'} />
                    <span className="text-[13px] font-medium uppercase">{engine.strategy}</span>
                    {engine.stale && <Badge variant="default">Stale</Badge>}
                    {engine.engine_version && <Badge variant="outline">{engine.engine_version}</Badge>}
                  </div>
                  <div className="text-[12px] text-muted-foreground">
                    Pasar {engine.market_state || '—'} · terakhir {engine.last_seen_at ? formatDateTime(engine.last_seen_at) : 'belum pernah'}
                  </div>
                  {engine.last_error && (
                    <p className="w-full text-[12px] text-muted-foreground">Error terakhir: {engine.last_error}</p>
                  )}
                </div>
              ))}
            </CardContent>
          </Card>
        </div>
      )}
    </>
  )
}

function Row({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="text-muted-foreground">{label}</span>
      <span className="flex items-center gap-2 font-medium">{children}</span>
    </div>
  )
}
