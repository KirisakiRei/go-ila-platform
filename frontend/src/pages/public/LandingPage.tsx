import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Alert, Badge, Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui'
import { publicPlanApi } from '@/lib/admin-api'
import { formatIDR } from '@/features/admin/lib/format'

const STEPS = [
  {
    title: 'Engine membaca pasar',
    body: 'Candle dari data pasar diproses dengan strategi ILA 2.0 dan ILA 3.0. Engine hanya mencari setup; ia tidak memegang akun siapa pun.',
  },
  {
    title: 'Runner menyaring per akun',
    body: 'Setiap setup dinilai terpisah untuk akun Anda: kuota harian, batas kerugian, tier, dan jam pasar.',
  },
  {
    title: 'Anda memutuskan atau otomatis',
    body: 'Mode manual mengirim tautan konfirmasi singkat. Mode otomatis menjalankan aturan yang sudah Anda setujui sendiri.',
  },
]

/**
 * LandingPage adalah halaman pertama situs publik.
 *
 * Sasaran halaman ini: menjelaskan cara kerja tanpa menjanjikan hasil.
 * Tidak ada klaim keuntungan, dan risiko disebutkan sebelum harga (plans/11 J3).
 */
export default function LandingPage() {
  const { data: plans } = useQuery({ queryKey: ['public', 'plans'], queryFn: publicPlanApi.list })

  return (
    <>
      <section className="mx-auto w-full max-w-6xl px-4 py-16 sm:py-24">
        <Badge variant="outline">ILA 2.0 · ILA 3.0</Badge>
        <h1 className="mt-4 max-w-3xl text-3xl font-bold leading-tight tracking-tight sm:text-5xl">
          Satu lapisan analisis di atas akun trading Anda sendiri.
        </h1>
        <p className="mt-4 max-w-2xl text-[15px] text-muted-foreground">
          Imaginary Liquidity mengubah aturan trading yang tertulis menjadi mesin yang konsisten: setup
          dikenali, risiko dibatasi, dan setiap keputusan tercatat. Akun tetap milik Anda, di broker Anda.
        </p>
        <div className="mt-6 flex flex-wrap gap-2">
          <Link to="/app/register" className="rounded-md bg-foreground px-4 py-2 text-[13px] font-medium text-background">
            Mulai uji coba 7 hari
          </Link>
          <Link to="/fitur" className="rounded-md border border-border px-4 py-2 text-[13px] font-medium">
            Lihat cara kerja
          </Link>
        </div>

        <Alert variant="warning" className="mt-8 max-w-2xl">
          Trading margin memiliki risiko. Sistem ini alat bantu disiplin, bukan jaminan keuntungan, dan
          bukan nasihat investasi.
        </Alert>
      </section>

      <section className="border-y border-border bg-secondary/30">
        <div className="mx-auto grid w-full max-w-6xl gap-4 px-4 py-12 sm:grid-cols-3">
          {STEPS.map((step, index) => (
            <div key={step.title}>
              <p className="text-[11px] uppercase tracking-[0.2em] text-muted-foreground">Langkah {index + 1}</p>
              <p className="mt-1 text-[14px] font-semibold">{step.title}</p>
              <p className="mt-1 text-[13px] text-muted-foreground">{step.body}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="mx-auto w-full max-w-6xl px-4 py-12">
        <h2 className="text-lg font-semibold tracking-tight">Paket</h2>
        <p className="mt-1 text-[13px] text-muted-foreground">
          Harga mengikuti katalog yang berlaku; perubahan tercatat di panel admin.
        </p>
        <div className="mt-5 grid gap-3 sm:grid-cols-3">
          {(plans ?? []).map((plan) => (
            <Card key={plan.code}>
              <CardHeader>
                <CardTitle className="text-sm">{plan.name}</CardTitle>
                <CardDescription className="text-xs">{plan.description || '—'}</CardDescription>
              </CardHeader>
              <CardContent>
                <p className="text-xl font-semibold tabular-nums">{formatIDR(plan.price_idr)}</p>
                <p className="text-[11px] text-muted-foreground">per bulan</p>
              </CardContent>
            </Card>
          ))}
        </div>
        <Link to="/harga" className="mt-4 inline-block text-[13px] underline">
          Bandingkan paket selengkapnya
        </Link>
      </section>
    </>
  )
}
