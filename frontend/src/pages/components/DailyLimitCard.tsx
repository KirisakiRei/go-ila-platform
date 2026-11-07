import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Skeleton,
} from '@/components/ui'
import { useSystemStatus } from '@/lib/queries'

/**
 * DailyLimitCard menjelaskan apakah sistem masih boleh membuka trade hari ini.
 * Aturannya: maksimal dua trade per hari, berhenti begitu satu trade menang.
 */
export function DailyLimitCard() {
  const { data, isLoading, isError, error } = useSystemStatus()

  if (isLoading) return <Skeleton className="h-44 w-full" />
  if (isError) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Batas harian</CardTitle>
          <CardDescription>{(error as Error).message}</CardDescription>
        </CardHeader>
      </Card>
    )
  }
  if (!data) return null

  const { daily } = data

  return (
    <Card>
      <CardHeader>
        <CardTitle>Batas harian</CardTitle>
        <CardDescription>
          Maksimal {daily.max_trades} trade per hari, dihitung dalam zona waktu{' '}
          {data.daily_timezone}.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <div className="flex items-baseline justify-between text-[13px]">
          <span className="text-muted-foreground">Terpakai</span>
          <span className="tabular font-semibold">
            {daily.total_trades} dari {daily.max_trades}
          </span>
        </div>

        {/* Bilah kemajuan monokrom: terisi = sudah terpakai. */}
        <div className="h-1.5 w-full overflow-hidden rounded-full bg-secondary" role="presentation">
          <div
            className="h-full bg-foreground"
            style={{
              width: `${Math.min(100, (daily.total_trades / Math.max(1, daily.max_trades)) * 100)}%`,
            }}
          />
        </div>

        {daily.locked ? (
          <p className="border-l-2 border-foreground pl-2.5 text-[13px] font-medium">
            {daily.lock_reason || 'Entry baru ditahan sampai besok.'}
          </p>
        ) : (
          <p className="text-[13px] text-muted-foreground">
            Masih dapat menerima setup berikutnya hari ini.
          </p>
        )}
      </CardContent>
    </Card>
  )
}
