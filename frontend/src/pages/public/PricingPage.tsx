import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Alert, Badge, Card, CardContent, CardDescription, CardHeader, CardTitle, Skeleton } from '@/components/ui'
import { publicPlanApi } from '@/lib/admin-api'
import { formatIDR } from '@/features/admin/lib/format'

/**
 * PricingPage menampilkan katalog paket dari database.
 *
 * Harga tidak ditulis di kode: perubahan dari panel admin langsung terlihat
 * di sini, sehingga tidak ada dua sumber harga yang bisa berbeda (plans/11 J4).
 */
export default function PricingPage() {
  const { data, isLoading } = useQuery({ queryKey: ['public', 'plans'], queryFn: publicPlanApi.list })
  const plans = data ?? []

  return (
    <div className="mx-auto w-full max-w-6xl px-4 py-12">
      <h1 className="text-2xl font-bold tracking-tight">Harga</h1>
      <p className="mt-2 max-w-2xl text-[13px] text-muted-foreground">
        Semua paket memakai akun broker Anda sendiri. Pembayaran bulanan; masa tenggang tetap memberi
        akses baca dan pemantauan posisi.
      </p>

      {isLoading && (
        <div className="mt-6 grid gap-3 sm:grid-cols-3">
          <Skeleton className="h-56 w-full" />
          <Skeleton className="h-56 w-full" />
          <Skeleton className="h-56 w-full" />
        </div>
      )}

      <div className="mt-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {plans.map((plan) => (
          <Card key={plan.code}>
            <CardHeader>
              <div className="flex items-center justify-between gap-2">
                <CardTitle className="text-sm">{plan.name}</CardTitle>
                {plan.code === 'plus' && <Badge variant="solid">Favorit</Badge>}
              </div>
              <CardDescription className="text-xs">{plan.description || '—'}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <div>
                <p className="text-2xl font-semibold tabular-nums">{formatIDR(plan.price_idr)}</p>
                <p className="text-[11px] text-muted-foreground">per bulan</p>
              </div>
              <ul className="space-y-1 text-[12px]">
                {Object.entries(plan.features ?? {}).map(([key, value]) => (
                  <li key={key} className="flex justify-between gap-2 border-b border-border pb-1 last:border-0">
                    <span className="text-muted-foreground">{key}</span>
                    <span className="text-right font-medium">{value}</span>
                  </li>
                ))}
              </ul>
              {plan.ila3_coming_soon && <Badge variant="default">ILA 3.0 belum tersedia</Badge>}
              <Link
                to="/app/register"
                className="block rounded-md bg-foreground px-3 py-2 text-center text-[13px] font-medium text-background"
              >
                Pilih {plan.name}
              </Link>
            </CardContent>
          </Card>
        ))}
      </div>

      <Alert variant="warning" className="mt-8">
        Hasil trading bergantung pada kondisi pasar dan keputusan Anda. Tidak ada jaminan keuntungan;
        modal dapat berkurang sebagian atau seluruhnya.
      </Alert>
    </div>
  )
}
