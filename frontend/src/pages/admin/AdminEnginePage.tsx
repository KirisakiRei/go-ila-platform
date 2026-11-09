import { useQuery } from '@tanstack/react-query'
import { Alert, Badge, Card, CardContent, CardDescription, CardHeader, CardTitle, EmptyState, PageHeader, Skeleton, StatusDot } from '@/components/ui'
import { adminSystemApi } from '@/lib/system-api'
import { formatDateTime } from '@/lib/format'

/**
 * AdminEnginePage memantau engine yang mencari setup.
 *
 * Engine hidup di sistem terpisah dan hanya mengirim sinyal. Bila semua
 * engine berhenti melapor, tidak ada sinyal baru — tetapi posisi yang sudah
 * terbuka tetap dikelola Runner dan broker (plans/11 I5).
 */
export default function AdminEnginePage() {
  const engines = useQuery({
    queryKey: ['admin', 'system', 'engine'],
    queryFn: adminSystemApi.engines,
    refetchInterval: 30_000,
  })

  const rows = engines.data ?? []

  return (
    <>
      <PageHeader
        title="Analysis Engines"
        description="Status laporan telemetri engine. Status Stale menandakan tidak ada heartbeat lebih dari 3 menit."
      />

      {engines.isError && <Alert variant="destructive">Gagal memuat status engine.</Alert>}
      {engines.isLoading && <Skeleton className="h-40 w-full" />}

      {!engines.isLoading && rows.length === 0 && (
        <EmptyState
          title="Belum ada engine yang melapor"
          description="Engine akan muncul di sini setelah koneksi pertamanya ke Runner."
        />
      )}

      <div className="grid gap-4 sm:grid-cols-2">
        {rows.map((engine) => (
          <Card key={engine.strategy}>
            <CardHeader>
              <div className="flex items-start justify-between gap-2">
                <div>
                  <CardTitle className="text-sm uppercase">{engine.strategy}</CardTitle>
                  <CardDescription className="mt-1 text-xs">
                    {engine.engine_version ? `Versi ${engine.engine_version}` : 'Versi belum dilaporkan'}
                  </CardDescription>
                </div>
                <Badge variant={engine.connected ? 'solid' : 'outline'}>
                  {engine.connected ? 'Online' : engine.stale ? 'Stale' : 'Offline'}
                </Badge>
              </div>
            </CardHeader>
            <CardContent className="space-y-2 text-[13px]">
              <div className="flex items-center justify-between gap-3">
                <span className="text-muted-foreground">Sesi pasar</span>
                <span className="flex items-center gap-2 font-medium">
                  <StatusDot
                    state={engine.market_state === 'OPEN' ? 'online' : 'idle'}
                    label={engine.market_state ?? 'tidak diketahui'}
                  />
                  {engine.market_state || '—'}
                </span>
              </div>
              <div className="flex items-center justify-between gap-3">
                <span className="text-muted-foreground">Terakhir terlihat</span>
                <span className="font-medium">
                  {engine.last_seen_at ? formatDateTime(engine.last_seen_at) : 'Belum pernah'}
                </span>
              </div>
              {engine.last_error && (
                <p className="rounded-md border border-border p-2 text-[12px] text-muted-foreground">
                  Error terakhir: {engine.last_error}
                </p>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      <Alert variant="info">
        Engine berhenti bukan alasan menutup posisi. Stop loss dan take profit yang sudah terpasang tetap
        dijaga broker.
      </Alert>
    </>
  )
}
