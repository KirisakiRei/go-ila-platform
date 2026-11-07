import { useState } from 'react'
import { PageHeader, Section, Button, Skeleton, EmptyState, Alert } from '@/components/ui'
import { SignalRow } from '@/features/signals/components/SignalRow'
import { SignalDetail } from '@/features/signals/components/SignalDetail'
import { useSignals } from '@/lib/queries'
import type { Signal } from '@/types/api'

/**
 * Halaman sinyal: daftar padat di kiri, rincian di kanan.
 * Pemilihan sinyal memakai state lokal karena tidak perlu dibagikan.
 */
export default function SignalsPage() {
  const { data, isLoading, isError, error, refetch, isFetching } = useSignals()
  const [selected, setSelected] = useState<Signal | null>(null)

  const aktif = selected ?? data?.[0] ?? null

  return (
    <>
      <PageHeader
        title="Sinyal"
        description="Setup terbaru dari engine. Hanya sinyal siap entry yang bisa ditindaklanjuti."
        actions={
          <Button variant="outline" size="sm" onClick={() => void refetch()} isLoading={isFetching}>
            Muat ulang
          </Button>
        }
      />

      {isError && <Alert variant="destructive">{(error as Error).message}</Alert>}

      {isLoading ? (
        <Skeleton className="h-64 w-full" />
      ) : !data || data.length === 0 ? (
        <EmptyState
          title="Belum ada sinyal"
          description="Sinyal muncul saat pasar buka dan engine menemukan setup yang memenuhi aturan."
        />
      ) : (
        <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_22rem]">
          <Section title="Daftar sinyal" description={`${data.length} sinyal terakhir`}>
            <ul className="divide-y divide-border border border-border">
              {data.map((signal) => (
                <SignalRow
                  key={signal.signal_id}
                  signal={signal}
                  selected={aktif?.signal_id === signal.signal_id}
                  onSelect={() => setSelected(signal)}
                />
              ))}
            </ul>
          </Section>

          {aktif && (
            <Section title="Rincian">
              <SignalDetail signal={aktif} />
            </Section>
          )}
        </div>
      )}
    </>
  )
}
