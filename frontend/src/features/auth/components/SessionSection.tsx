import { Alert, Badge, Button, Skeleton } from '@/components/ui'
import { useRevokeSession, useSessions } from '@/lib/queries'

/**
 * SessionSection menampilkan perangkat yang sedang masuk. Pengguna bisa
 * mencabut perangkat yang tidak dikenali — tanda adanya penyalahgunaan akun
 * (plans/03 §8).
 */
export function SessionSection() {
  const { data, isLoading, isError, error } = useSessions()
  const revoke = useRevokeSession()

  return (
    <section className="flex flex-col gap-3">
      <header className="flex flex-col gap-1">
        <h2 className="text-sm font-semibold tracking-tight">Perangkat yang masuk</h2>
        <p className="text-[12px] text-muted-foreground">
          Cabut perangkat yang tidak Anda kenali. Mengganti password otomatis mengeluarkan
          semua perangkat.
        </p>
      </header>

      <div className="border border-border">
        {isLoading && (
          <div className="flex flex-col gap-2 p-4">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        )}

        {isError && (
          <div className="p-4">
            <Alert variant="destructive">{(error as Error).message}</Alert>
          </div>
        )}

        {data?.length === 0 && (
          <p className="p-4 text-[13px] text-muted-foreground">Belum ada sesi aktif.</p>
        )}

        <ul>
          {data?.map((item) => (
            <li
              key={item.id}
              className="flex flex-wrap items-center gap-3 border-b border-border px-4 py-3 text-[13px] last:border-b-0"
            >
              <div className="flex min-w-0 flex-col gap-0.5">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-medium">{item.device_label}</span>
                  {item.current && <Badge variant="solid">Perangkat ini</Badge>}
                </div>
                <span className="text-[12px] text-muted-foreground">
                  Terakhir aktif {item.last_seen_pair.wib} WIB
                  <span className="text-muted-foreground/70"> · {item.last_seen_pair.ny} NY</span>
                </span>
              </div>

              <Button
                size="sm"
                variant="outline"
                className="ml-auto"
                disabled={item.current || revoke.isPending}
                title={item.current ? 'Gunakan "Keluar" untuk perangkat ini' : undefined}
                onClick={() => revoke.mutate(item.id)}
              >
                Cabut
              </Button>
            </li>
          ))}
        </ul>
      </div>

      {revoke.isError && (
        <Alert variant="destructive">
          {revoke.error instanceof Error ? revoke.error.message : 'Gagal mencabut sesi.'}
        </Alert>
      )}
    </section>
  )
}
