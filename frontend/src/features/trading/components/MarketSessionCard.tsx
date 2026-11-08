import { useQuery } from '@tanstack/react-query'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui'
import { systemApi } from '@/lib/api'
import { publicContentApi } from '@/lib/system-api'

/**
 * MarketSessionCard menunjukkan dua waktu yang sering membingungkan:
 * sesi pasar mengikuti New York, hari akuntansi mengikuti WIB.
 *
 * Keduanya ditampilkan berdampingan agar pengguna tidak menghitung sendiri,
 * dan alasan perbedaannya dijelaskan singkat (plans/22).
 */
export function MarketSessionCard() {
  const status = useQuery({ queryKey: ['system', 'status'], queryFn: systemApi.status })
  const publicStatus = useQuery({ queryKey: ['public', 'status'], queryFn: publicContentApi.status })

  const market = publicStatus.data?.market
  const daily = status.data?.daily

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Sesi pasar & kuota harian</CardTitle>
        <CardDescription className="text-xs">
          Jam trading memakai New York; hitungan harian memakai WIB.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3 text-[13px]">
        <div className="flex items-start justify-between gap-3">
          <div>
            <p className="font-medium">{market ? market.label : '—'}</p>
            <p className="text-[12px] text-muted-foreground">
              {market
                ? market.next_change
                  ? `${market.next_label} ${market.next_change_pair.wib} WIB · ${market.next_change_pair.ny} NY`
                  : 'Jadwal perubahan belum tersedia'
                : 'Memuat jadwal…'}
            </p>
          </div>
          <div className="text-right text-[12px] text-muted-foreground">
            <p>WIB {publicStatus.data?.time_pair.wib ?? '—'}</p>
            <p>NY {publicStatus.data?.time_pair.ny ?? '—'}</p>
          </div>
        </div>

        {daily && (
          <div className="grid grid-cols-2 gap-2 border-t border-border pt-3 text-[12px] sm:grid-cols-4">
            <div>
              <p className="text-muted-foreground">Hari akuntansi</p>
              <p className="font-medium">{daily.day}</p>
            </div>
            <div>
              <p className="text-muted-foreground">Trade hari ini</p>
              <p className="font-medium tabular-nums">
                {daily.total_trades}/{daily.max_trades}
              </p>
            </div>
            <div>
              <p className="text-muted-foreground">Menang / kalah</p>
              <p className="font-medium tabular-nums">
                {daily.win_count} / {daily.loss_count}
              </p>
            </div>
            <div>
              <p className="text-muted-foreground">Status</p>
              <p className="font-medium">{daily.locked ? 'Berhenti hari ini' : 'Berjalan'}</p>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
