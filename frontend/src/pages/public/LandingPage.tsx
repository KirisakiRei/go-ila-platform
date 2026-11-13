import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ArrowRight, Check } from 'lucide-react'
import { Link, useLocation } from 'react-router-dom'
import { Skeleton } from '@/components/ui/skeleton'
import { formatIDR } from '@/features/admin/lib/format'
import { publicPlanApi } from '@/lib/admin-api'
import {
  HeroTradingGeometry,
  MetricsCurveSvg,
  MonochromeHorizonTile,
  MonochromeLiquidityWaves,
} from './components/LandingArtwork'
import { SignalOverviewGrid } from './components/SignalOverviewGrid'
import { formatPlanBenefits, getPlanAudience } from './lib/plan-benefits'

const CURVE_METRICS = [
  {
    value: '2 Strategi',
    label: 'ILA 2.0 & ILA 3.0',
    posClass: 'left-[8%] top-[62.5%]',
  },
  {
    value: '8 Pair',
    label: 'Instrumen Likuid Utama',
    posClass: 'left-[33%] top-[56.8%]',
  },
  {
    value: '2 Mode',
    label: 'Manual & Otomatis',
    posClass: 'left-[58%] top-[44.2%]',
  },
  {
    value: '2 Zona',
    label: 'Sesi NY & Batas WIB',
    posClass: 'left-[83%] top-[24.5%]',
  },
]

const PILLARS = [
  {
    kind: 'standard' as const,
    title: 'Mode Manual & Otomatis',
    body: 'Setujui sinyal lewat satu ketukan di Telegram, atau aktifkan eksekusi otomatis sesuai batas risiko yang Anda tetapkan.',
  },
  {
    kind: 'accent' as const,
    note: 'Setiap peluang disaring terhadap kuota harian, batas kerugian, dan jadwal sesi aktif akun Anda.',
    anchor: 'Disiplin Aturan per Akun',
  },
  {
    kind: 'standard' as const,
    title: 'Pendekatan Berbasis Likuiditas',
    body: 'Membaca liquidity sweep dan pergeseran struktur pasar, menunggu konfirmasi candle valid tanpa spekulasi.',
  },
  {
    kind: 'accent' as const,
    note: 'Coba seluruh alur dalam mode simulasi selama 7 hari tanpa komitmen pembayaran.',
    anchor: 'Mulai Uji Coba Simulasi',
  },
]

const FEATURE_STATS = [
  {
    value: '2 Mode',
    detail: 'Pilih konfirmasi manual per sinyal atau eksekusi otomatis sesuai aturan Anda.',
  },
  {
    value: '7 Hari',
    detail: 'Masa uji coba fitur paket Basic dalam mode simulasi sebelum memilih paket.',
  },
  {
    value: '100%',
    detail: 'Setiap sinyal, persetujuan, dan riwayat order tercatat rapi untuk evaluasi.',
  },
]

/**
 * LandingPage adalah beranda situs publik Imaginary Liquidity (rekavia.com).
 *
 * Seluruh bagian memakai palet monokrom hitam-putih netral, menjelaskan cara
 * kerja dan batasan produk secara jujur tanpa janji keuntungan maupun bocoran
 * parameter internal ILA (`plans/10_LANDING_PAGE.md`).
 */
export default function LandingPage() {
  const { hash } = useLocation()
  const { data: plans, isLoading: plansLoading } = useQuery({
    queryKey: ['public', 'plans'],
    queryFn: publicPlanApi.list,
  })

  useEffect(() => {
    if (hash === '#cara-kerja') {
      const target = document.getElementById('cara-kerja')
      if (target) target.scrollIntoView()
    }
  }, [hash])

  return (
    <div className="bg-[#f5f5f7] text-[#09090b]">
      {/* 1 & 2. HERO + OVERVIEW + KURVA: transisi diagonal "/" (kiri-atas gelap -> kanan-bawah lebih cepat terang) */}
      <div
        className="relative overflow-hidden"
        style={{
          background: [
            'radial-gradient(62% 40% at 22% 72%, rgba(245, 245, 247, 0.92) 0%, rgba(245, 245, 247, 0) 100%)',
            'linear-gradient(166deg, #050507 0%, #050507 28%, #18181b 39%, #3f3f46 47%, #71717a 53%, #d4d4d8 61%, #f5f5f7 70%, #f5f5f7 100%)',
          ].join(', '),
        }}
      >
        {/* BAGIAN HERO */}
        <section id="hero-band" className="relative text-white">
          {/* Selubung gelap di belakang teks: gradasi halaman berangsur
              terang, jadi judul putih butuh latar yang menjamin kontras. */}
          <div
            className="pointer-events-none absolute inset-x-0 top-0 h-[520px] bg-gradient-to-b from-[#050507]/90 via-[#050507]/70 to-transparent"
            aria-hidden="true"
          />
          {/* Background geometris tema trading yang clean & rapi */}
          <HeroTradingGeometry className="pointer-events-none absolute inset-0 h-full w-full" />

          <div className="relative mx-auto w-full max-w-6xl px-4 pt-28 pb-16 sm:px-6 sm:pt-36 sm:pb-24">
            {/* Judul utama hero */}
            <div className="relative mt-6 flex min-h-[190px] flex-col justify-center py-4 sm:mt-10 sm:min-h-[240px]">
              <div className="relative z-10 w-full max-w-4xl text-left">
                <h1 className="text-4xl font-semibold leading-[1.06] tracking-tight sm:text-6xl lg:text-[4.5rem]">
                  <span className="text-white">Imaginary Liquidity</span>{' '}
                  <br />
                  <span className="text-white/60">Algorithm</span>
                </h1>
              </div>
            </div>

            {/* Sub-bar horison tipis dengan tombol putih bersih */}
            <div className="relative z-10 mt-12 flex flex-col justify-between gap-6 border-t border-white/15 pt-8 md:mt-16 md:flex-row md:items-end">
              <div className="max-w-xl">
                <p className="text-[15px] leading-relaxed text-zinc-200">
                  Analisis struktur likuiditas real-time dan eksekusi presisi langsung di akun broker Anda.
                </p>
              </div>

              <div className="flex shrink-0 flex-wrap items-center gap-3">
                {/* Anchor biasa: bundel panel terpisah di /app/*. */}
                <a
                  href="/app/register"
                  className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-5 py-2.5 text-[13px] font-semibold text-[#09090b] shadow-lg shadow-black/30 transition-colors hover:bg-zinc-200"
                >
                  Mulai Uji Coba 7 Hari
                  <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
                </a>
                <Link
                  to="/fitur"
                  className="inline-flex items-center justify-center rounded-lg border border-white/25 bg-black/35 px-4 py-2.5 text-[13px] font-medium text-white transition-colors hover:bg-white/10"
                >
                  Pelajari Cara Kerja
                </Link>
              </div>
            </div>
          </div>
        </section>

        {/* OVERVIEW + KURVA NAIK TANPA CARD */}
        <section
          id="cara-kerja"
          className="relative z-10 mx-auto w-full max-w-6xl scroll-mt-20 px-4 pt-12 pb-20 sm:px-6 sm:pt-16 sm:pb-24"
        >
          <div className="max-w-2xl space-y-5">
            <h2 className="text-3xl font-semibold leading-[1.12] tracking-tight text-[#09090b] sm:text-4xl lg:text-[2.75rem]">
              Trading lebih sistematis dengan aturan risiko terukur
            </h2>
            <p className="max-w-lg text-[14px] leading-relaxed text-[#3f3f46]">
              Dari analisis struktur chart hingga eksekusi posisi, semua keputusan tercatat
              transparan di akun trading Anda.
            </p>
            <div className="flex flex-wrap items-center gap-4 pt-1">
              <a
                href="/app/register"
                className="inline-flex items-center justify-center rounded-lg bg-[#09090b] px-5 py-2.5 text-[13px] font-medium text-white transition-colors hover:bg-[#27272a]"
              >
                Mulai Uji Coba 7 Hari
              </a>
              <Link
                to="/bantuan/zona-waktu"
                className="text-[13px] font-medium text-[#18181b] underline underline-offset-4 hover:text-[#52525b]"
              >
                Cara kerja dua zona waktu
              </Link>
            </div>
          </div>

          {/* Kurva naik melengkung mulus + garis vertikal "|" sejajar tanpa card (desktop) */}
          <div className="relative -mt-6 hidden md:block">
            <MetricsCurveSvg className="h-auto w-full" />
            {CURVE_METRICS.map((item) => (
              <div key={item.value} className={`absolute ${item.posClass} w-[20%] pl-4`}>
                <p className="text-2xl font-semibold leading-none tracking-tight text-[#09090b] tabular-nums lg:text-[2rem]">
                  {item.value}
                </p>
                <p className="mt-2 text-[12px] leading-snug text-[#52525b] lg:text-[13px]">
                  {item.label}
                </p>
              </div>
            ))}
          </div>

          {/* Layout metrik bersih tanpa card untuk layar kecil */}
          <div className="mt-10 grid grid-cols-2 gap-x-6 gap-y-8 md:hidden">
            {CURVE_METRICS.map((item) => (
              <div key={item.value} className="border-l border-zinc-300 pl-3.5">
                <p className="text-2xl font-semibold leading-none tracking-tight text-[#09090b] tabular-nums">
                  {item.value}
                </p>
                <p className="mt-2 text-[12px] leading-snug text-[#52525b]">{item.label}</p>
              </div>
            ))}
          </div>
        </section>
      </div>

      {/* 3. PILAR KEPUTUSAN ("Kami buat eksekusi mudah dikendalikan") */}
      <section className="bg-[#f5f5f7] pb-20 sm:pb-28">
        <div className="mx-auto w-full max-w-6xl px-4 sm:px-6">
          <div className="grid gap-6 border-t border-zinc-200 pt-14 lg:grid-cols-12 lg:items-end">
            <div className="lg:col-span-6">
              <h2 className="text-3xl font-semibold leading-[1.12] tracking-tight sm:text-4xl">
                <span className="text-[#09090b]">Kami buat eksekusi</span>{' '}
                <span className="text-[#52525b]">mudah dikendalikan</span>
              </h2>
            </div>
            <div className="flex flex-col items-start justify-between gap-4 sm:flex-row sm:items-center lg:col-span-6">
              <p className="max-w-sm text-[14px] leading-relaxed text-[#52525b]">
                Dari pembacaan struktur pasar hingga persetujuan sekali ketuk, semua langkah tetap
                berada di bawah kendali penuh Anda.
              </p>
              <Link
                to="/fitur"
                className="inline-flex shrink-0 items-center justify-center rounded-lg bg-[#09090b] px-4 py-2.5 text-[13px] font-medium text-white transition-colors hover:bg-[#27272a]"
              >
                Pelajari Fitur
              </Link>
            </div>
          </div>

          <div className="mt-10 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {PILLARS.map((pillar) =>
              pillar.kind === 'standard' ? (
                <div
                  key={pillar.title}
                  className="flex min-h-[205px] flex-col justify-between rounded-2xl border border-zinc-200/80 bg-white p-6"
                >
                  <h3 className="text-[16px] font-semibold tracking-tight text-[#09090b]">
                    {pillar.title}
                  </h3>
                  <p className="mt-6 text-[13px] leading-relaxed text-[#52525b]">{pillar.body}</p>
                </div>
              ) : (
                <div
                  key={pillar.anchor}
                  className="flex min-h-[205px] flex-col justify-between rounded-2xl border border-zinc-200 bg-[#ececef] p-6"
                >
                  <p className="text-[13px] leading-relaxed text-[#52525b]">{pillar.note}</p>
                  <p className="mt-6 text-[16px] font-semibold tracking-tight text-[#09090b]">
                    {pillar.anchor}
                  </p>
                </div>
              ),
            )}
          </div>
        </div>
      </section>

      {/* 4. PANTAUAN PASAR TERKINI: tiga instrumen utama dengan arah terakhir. */}
      <section className="bg-[#f5f5f7] pt-16 pb-20 sm:pt-20 sm:pb-28">
        <div className="mx-auto w-full max-w-6xl px-4 sm:px-6">
          <div className="max-w-2xl space-y-3">
            <h2 className="text-3xl font-semibold leading-[1.12] tracking-tight sm:text-4xl">
              <span className="text-[#09090b]">Arah pasar terkini,</span>{' '}
              <span className="text-[#52525b]">diperbarui tiap sesi</span>
            </h2>
          </div>

          <div className="mt-8">
            <SignalOverviewGrid />
          </div>

          <div className="mt-8 flex flex-wrap items-center justify-between gap-4 border-t border-zinc-200 pt-6">
            <p className="text-[13px] text-[#6b6b74]">
              Parameter lengkap entry price, stop loss, dan take profit tersedia di dashboard member.
            </p>
            <div className="flex flex-wrap items-center gap-3">
              <Link
                to="/harga"
                className="inline-flex items-center justify-center rounded-lg bg-[#09090b] px-4 py-2 text-[13px] font-medium text-white transition-colors hover:bg-[#27272a]"
              >
                Lihat Paket &amp; Harga
              </Link>
              <Link
                to="/fitur"
                className="text-[13px] font-medium text-[#18181b] underline underline-offset-4 hover:text-[#52525b]"
              >
                Pelajari Fitur
              </Link>
            </div>
          </div>

          <h2 className="mt-20 max-w-3xl text-3xl font-semibold leading-[1.1] tracking-tight sm:text-5xl">
            <span className="text-[#09090b]">Satu tempat untuk pantau,</span>
            <br />
            <span className="text-[#52525b]">setujui, dan jaga risiko</span>
          </h2>
        </div>
      </section>

      {/* 5. ILA 2.0 & ILA 3.0 (menggantikan bagian gelombang ungu) */}
      <section className="bg-[#f5f5f7] py-20 sm:py-28">
        <div className="mx-auto w-full max-w-6xl px-4 sm:px-6">
          <div className="grid gap-6 lg:grid-cols-12 lg:items-stretch">
            <div className="flex flex-col justify-between gap-8 lg:col-span-5">
              <div className="space-y-4">
                <h2 className="text-3xl font-semibold leading-[1.12] tracking-tight sm:text-4xl">
                  <span className="text-[#09090b]">Dua metode analisis,</span>{' '}
                  <span className="text-[#52525b]">satu disiplin risiko</span>
                </h2>
                <p className="max-w-[60ch] text-[14px] leading-relaxed text-[#52525b]">
                  Metode ILA memetakan area likuiditas dan menunggu konfirmasi pergerakan harga
                  sebelum mengirim setup — tanpa spekulasi berlebihan.
                </p>
              </div>

              <div className="flex flex-wrap items-center gap-3">
                <a
                  href="/app/register"
                  className="inline-flex items-center justify-center rounded-lg bg-[#09090b] px-5 py-2.5 text-[13px] font-medium text-white transition-colors hover:bg-[#27272a]"
                >
                  Mulai Uji Coba
                </a>
                <Link
                  to="/fitur"
                  className="inline-flex items-center justify-center rounded-lg border border-zinc-300 bg-white px-4 py-2.5 text-[13px] font-medium text-[#09090b] transition-colors hover:bg-zinc-100"
                >
                  Pelajari Fitur
                </Link>
              </div>
            </div>

            <div className="relative min-h-[340px] overflow-hidden rounded-2xl border border-zinc-800/30 shadow-lg shadow-black/10 lg:col-span-4 lg:min-h-[420px]">
              <MonochromeLiquidityWaves className="h-full w-full object-cover" />
              <div className="absolute inset-x-4 bottom-4 rounded-xl border border-white/15 bg-black/75 px-4 py-3 text-white backdrop-blur-md">
                <p className="text-[11px] uppercase tracking-[0.16em] text-zinc-300">
                  Pemetaan Struktur Harga
                </p>
                <p className="mt-0.5 text-[13px] font-medium">ILA 2.0 &amp; ILA 3.0</p>
              </div>
            </div>

            <div className="flex flex-col justify-between gap-4 lg:col-span-3">
              <div className="space-y-4">
                <div className="rounded-2xl border border-zinc-200/80 bg-white p-5">
                  <h3 className="text-[14px] font-semibold text-[#09090b]">
                    ILA 2.0 · Tersedia Sekarang
                  </h3>
                  <p className="mt-1.5 text-[13px] leading-relaxed text-[#52525b]">
                    Menyaring liquidity sweep dan konfirmasi struktur pasar secara konsisten pada
                    seluruh paket langganan.
                  </p>
                </div>
                <div className="rounded-2xl border border-zinc-200/80 bg-white p-5">
                  <h3 className="text-[14px] font-semibold text-[#09090b]">
                    ILA 3.0 · Hadir Bertahap
                  </h3>
                  <p className="mt-1.5 text-[13px] leading-relaxed text-[#52525b]">
                    Penyaringan struktur multi-frame lanjutan yang disiapkan bertahap sebagai pilihan
                    tambahan di paket Plus.
                  </p>
                </div>
              </div>

              <div className="relative overflow-hidden rounded-2xl border border-zinc-800/20">
                <MonochromeHorizonTile className="h-32 w-full object-cover" />
                <div className="absolute inset-0 flex items-end bg-gradient-to-t from-black/90 via-black/55 to-black/15 p-4">
                  <p className="text-[12px] font-medium text-white">Fokus pada Konfirmasi</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* 6. FITUR & CAKUPAN PEMANTAUAN (menggantikan Bento jargon internal) */}
      <section className="bg-[#f5f5f7] pt-16 pb-20 sm:pt-20 sm:pb-24">
        <div className="mx-auto w-full max-w-6xl px-4 sm:px-6">
          <div className="mx-auto max-w-2xl text-center">
            <h2 className="text-3xl font-semibold leading-[1.12] tracking-tight sm:text-4xl">
              <span className="text-[#09090b]">Dirancang untuk</span>{' '}
              <span className="text-[#52525b]">trading yang terukur</span>
            </h2>
          </div>

          <div className="mt-12 grid gap-4 lg:grid-cols-12">
            <div className="flex flex-col justify-between gap-6 rounded-2xl border border-zinc-200/80 bg-white p-6 lg:col-span-4">
              <div>
                <p className="text-[12px] font-semibold uppercase tracking-[0.14em] text-[#52525b]">
                  Pemindaian Sesi Aktif
                </p>
                <p className="mt-2 text-[14px] leading-relaxed text-[#3f3f46]">
                  Memantau instrumen pilihan secara terjadwal selama jam pasar dibuka dan
                  mencatat setiap sinyal yang lolos maupun yang dilewati agar mudah Anda tinjau.
                </p>
              </div>

              <div className="border-t border-zinc-200 pt-5">
                <p className="text-[13px] text-[#52525b]">Cakupan instrumen likuid</p>
                <p className="mt-1 text-2xl font-bold tracking-tight text-[#09090b]">
                  8 Pair Utama
                </p>
              </div>
            </div>

            <div className="flex flex-col gap-4 lg:col-span-4">
              <div className="rounded-2xl border border-zinc-200/80 bg-white p-6">
                <h3 className="text-[16px] font-semibold tracking-tight text-[#09090b]">
                  Peringatan &amp; Persetujuan Telegram
                </h3>
                <p className="mt-2 text-[13px] leading-relaxed text-[#52525b]">
                  Terima notifikasi sinyal real-time dan setujui eksekusi lewat satu ketukan tanpa
                  perlu menatap layar terus-menerus.
                </p>
              </div>

              <div className="flex flex-1 flex-col justify-between rounded-2xl border border-zinc-200 bg-[#e6e6ea] p-6">
                <span className="w-fit rounded-full bg-[#09090b] px-3 py-1 text-[11px] font-medium text-white">
                  Pengendalian Risiko
                </span>
                <div className="mt-8">
                  <h3 className="text-xl font-bold tracking-tight text-[#09090b]">
                    Batas Harian Terukur per Akun
                  </h3>
                  <p className="mt-2 text-[13px] leading-relaxed text-[#3f3f46]">
                    Atur persentase risiko per posisi, batas jumlah trade harian, serta aturan
                    berhenti otomatis untuk mencegah overtrading.
                  </p>
                </div>
              </div>
            </div>

            <div className="flex flex-col gap-4 lg:col-span-4">
              {FEATURE_STATS.map((stat) => (
                <div
                  key={stat.value}
                  className="flex flex-1 flex-col justify-between rounded-2xl border border-zinc-200/80 bg-white p-5"
                >
                  <p className="text-2xl font-bold tracking-tight text-[#09090b]">{stat.value}</p>
                  <p className="mt-2 text-[13px] leading-relaxed text-[#52525b]">{stat.detail}</p>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      {/* 7. OVERVIEW HARGA (TANPA CARD KOTAK, KOLOM TERBUKA + BENEFIT MANUSIAWI) */}
      <section className="relative z-10 bg-[#f5f5f7] py-20 sm:py-24">
        <div className="mx-auto w-full max-w-6xl px-4 sm:px-6">
          <div className="flex flex-col justify-between gap-4 pb-10 sm:flex-row sm:items-end">
            <div className="space-y-2">
              <h2 className="text-3xl font-semibold leading-[1.12] tracking-tight sm:text-4xl">
                <span className="text-[#09090b]">Pilihan Paket</span>{' '}
                <span className="text-[#52525b]">Sesuai Kebutuhan</span>
              </h2>
              <p className="max-w-xl text-[14px] text-[#52525b]">
                Harga mengikuti katalog aktif di server. Mulai dari simulasi gratis atau langsung
                coba uji coba 7 hari dengan fitur paket Basic.
              </p>
            </div>
            <Link
              to="/harga"
              className="inline-flex shrink-0 items-center gap-1.5 text-[13px] font-semibold text-[#09090b] underline underline-offset-4 hover:text-[#52525b]"
            >
              Bandingkan paket selengkapnya
              <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
            </Link>
          </div>

          {plansLoading ? (
            <div className="grid gap-8 border-t border-zinc-200 pt-8 sm:grid-cols-3">
              {[0, 1, 2].map((idx) => (
                <div key={idx} className="flex flex-col justify-between border-t border-zinc-200 pt-6 space-y-6">
                  <div className="space-y-4">
                    <Skeleton className="h-6 w-28 bg-zinc-200" />
                    <Skeleton className="h-4 w-4/5 bg-zinc-200" />
                    <Skeleton className="h-8 w-36 bg-zinc-200" />
                    <div className="space-y-2 pt-2">
                      <Skeleton className="h-4 w-full bg-zinc-200" />
                      <Skeleton className="h-4 w-5/6 bg-zinc-200" />
                      <Skeleton className="h-4 w-3/4 bg-zinc-200" />
                    </div>
                  </div>
                  <Skeleton className="h-10 w-full rounded-lg bg-zinc-200" />
                </div>
              ))}
            </div>
          ) : plans && plans.length > 0 ? (
            <div className="grid gap-10 sm:grid-cols-3 sm:gap-8">
              {plans.map((plan) => {
                const isHighlight = plan.code === 'plus'
                const benefits = formatPlanBenefits(plan)
                return (
                  <div
                    key={plan.code}
                    className={
                      isHighlight
                        ? 'flex flex-col justify-between border-t-2 border-[#09090b] pt-6'
                        : 'flex flex-col justify-between border-t border-zinc-300 pt-6'
                    }
                  >
                    <div>
                      <div className="flex items-center justify-between gap-2">
                        <h3 className="text-lg font-semibold text-[#09090b]">{plan.name}</h3>
                        {isHighlight && (
                          <span className="rounded-full bg-[#09090b] px-2.5 py-0.5 text-[11px] font-medium text-white">
                            Paling Lengkap
                          </span>
                        )}
                      </div>

                      <p className="mt-2 min-h-[40px] text-[13px] leading-relaxed text-[#52525b]">
                        {getPlanAudience(plan)}
                      </p>

                      <div className="mt-5 flex items-baseline gap-1.5 border-b border-zinc-200 pb-5">
                        <span className="text-3xl font-bold tracking-tight text-[#09090b] tabular-nums">
                          {formatIDR(plan.price_idr)}
                        </span>
                        <span className="text-[12px] text-[#52525b]">/ bulan</span>
                      </div>

                      <ul className="mt-5 space-y-3 text-[13px] text-[#27272a]">
                        {benefits.map((benefit) => (
                          <li key={benefit.label} className="flex items-start gap-2.5">
                            <Check
                              className="mt-0.5 h-4 w-4 shrink-0 text-[#09090b]"
                              aria-hidden="true"
                            />
                            <span>{benefit.label}</span>
                          </li>
                        ))}
                      </ul>
                    </div>

                    <div className="mt-8">
                      <a
                        href="/app/register"
                        className={
                          isHighlight
                            ? 'block rounded-lg bg-[#09090b] px-4 py-2.5 text-center text-[13px] font-medium text-white transition-colors hover:bg-[#27272a]'
                            : 'block rounded-lg border border-zinc-300 bg-transparent px-4 py-2.5 text-center text-[13px] font-medium text-[#09090b] transition-colors hover:bg-zinc-100'
                        }
                      >
                        Pilih {plan.name}
                      </a>
                    </div>
                  </div>
                )
              })}
            </div>
          ) : (
            <p className="border-t border-zinc-200 pt-6 max-w-[62ch] text-[13px] leading-relaxed text-[#52525b]">
              Daftar paket dapat dilihat langsung pada halaman{' '}
              <Link to="/harga" className="font-medium text-[#09090b] underline">
                Harga
              </Link>
              .
            </p>
          )}
        </div>
      </section>

      {/* 8. BANNER CTA GET STARTED (transparan di luar agar gradasi kontinu footer menyatu dari belakang kartu) */}
      <section className="relative z-10 bg-transparent px-4 pt-10 pb-2 sm:px-6 sm:pt-12 sm:pb-4">
        <div className="mx-auto w-full max-w-6xl">
          <div className="rounded-3xl bg-[#050507] px-6 py-12 text-white shadow-2xl shadow-black/15 sm:px-12 sm:py-16">
            <div className="max-w-2xl space-y-6">
              <h2 className="text-3xl font-semibold leading-[1.12] tracking-tight sm:text-4xl lg:text-[2.6rem]">
                <span className="text-white">Siap menjalankan trading</span>{' '}
                <span className="text-zinc-400">dengan aturan lebih jelas?</span>
              </h2>

              <p className="max-w-[60ch] text-[14px] leading-relaxed text-zinc-300">
                Mulai uji coba simulasi 7 hari tanpa kartu kredit. Hubungkan akun broker Anda saat
                sudah siap menjalankan trading.
              </p>

              <p className="border-l-2 border-zinc-700 pl-4 max-w-[68ch] text-[12px] leading-relaxed text-zinc-300">
                <strong className="font-semibold text-white">Disclaimer Risiko Trading:</strong>{' '}
                Perdagangan mata uang dan derivatif memiliki risiko kerugian tinggi. Platform ini
                adalah alat bantu analisis dan eksekusi—bukan jaminan keuntungan dan bukan nasihat
                keuangan.
              </p>

              <div className="flex flex-wrap items-center gap-4 pt-2">
                <a
                  href="/app/register"
                  className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-5 py-2.5 text-[13px] font-semibold text-[#09090b] transition-colors hover:bg-zinc-200"
                >
                  Mulai Uji Coba 7 Hari
                  <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
                </a>
                <Link
                  to="/legal/risk"
                  className="text-[13px] font-medium text-zinc-200 underline underline-offset-4 transition-colors hover:text-white"
                >
                  Baca Disclaimer Risiko
                </Link>
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>
  )
}
