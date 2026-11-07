import { SignalCard } from '@/features/signals/components/SignalCard'
import { useSignals } from '@/lib/queries'
import { Alert, Skeleton } from '@/components/ui'

export default function SignalsPage() {
  const { data, isLoading, isError, error } = useSignals()

  return (
    <div className="flex flex-col gap-4">
      <header className="flex flex-col gap-1">
        <h1 className="text-xl font-semibold tracking-tight">Sinyal</h1>
        <p className="text-[13px] text-muted-foreground">
          Setup terbaru dari engine. Hanya sinyal siap entry yang bisa ditindaklanjuti.
        </p>
      </header>

      {isLoading && (
        <div className="flex flex-col gap-3" aria-busy="true">
          <Skeleton className="h-40 w-full" />
          <Skeleton className="h-40 w-full" />
        </div>
      )}

      {isError && <Alert>{(error as Error).message}</Alert>}

      {data?.length === 0 && (
        <p className="rounded-lg border border-dashed border-border p-8 text-center text-[13px] text-muted-foreground">
          Belum ada sinyal. Sinyal muncul saat pasar buka dan engine menemukan setup.
        </p>
      )}

      <div className="flex flex-col gap-3">
        {data?.map((signal) => (
          <SignalCard key={signal.signal_id} signal={signal} />
        ))}
      </div>
    </div>
  )
}
