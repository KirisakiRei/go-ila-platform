import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ArrowRight, ShieldCheck } from 'lucide-react'
import { Link, useLocation } from 'react-router-dom'
import { formatIDR } from '@/features/admin/lib/format'
import { publicPlanApi } from '@/lib/admin-api'
import {
  CtaInsetPreview,
  CtaPrismArtwork,
  HeroFinsPreview,
  HeroObsidianRings,
  MetricsCurveSvg,
  PurpleOrbTile,
  PurpleRibbonTile,
  PurpleSilkWaves,
  SecurityChainArtwork,
} from './components/LandingArtwork'

const METRICS = [
  {
    value: '2 Engine',
    label: 'ILA 2.0 aktif untuk semua paket; ILA 3.0 dibuka bertahap.',
    offsetClass: 'lg:mt-36',
  },
  {
    value: '< 50ms',
    label: 'Distribusi sinyal dari Engine ke Runner melalui kanal gRPC mTLS.',
    offsetClass: 'lg:mt-24',
  },
  {
    value: '2 Zona',
    label: 'Penilaian sesi mengikuti waktu New York; kuota harian memakai WIB.',
    offsetClass: 'lg:mt-12',
  },
  {
    value: '0 Dana',
    label: 'Saldo tetap di akun broker Anda; kredensial terenkripsi AES-256.',
    offsetClass: 'lg:mt-0',
  },
]

const PILLARS = [
  {
    kind: 'standard' as const,
    title: 'Mode Manual & Otomatis',
    body: 'Pilih tautan persetujuan sekali ketuk lewat Telegram atau biarkan sistem mengeksekusi aturan risiko yang sudah Anda setujui.',
  },
  {
    kind: 'accent' as const,
    note: 'Setiap setup disaring terpisah terhadap kuota harian, batas kerugian, dan jam buka sesi.',
    anchor: 'Saringan Runner per Akun',
  },
  {
    kind: 'standard' as const,
    title: 'Pemantauan Berbasis Likuiditas',
    body: 'Mengamati sapuan likuiditas, perubahan struktur pasar, dan menunggu konfirmasi alih-alih menebak arah harga.',
  },
  {
    kind: 'accent' as const,
    note: 'Coba seluruh alur dalam mode simulasi selama 7 hari tanpa kartu kredit maupun komitmen.',
    anchor: 'Mulai Tanpa Risiko Biaya',
  },
]

const BENTO_STATS = [
  {
    value: 'AES-256',
    detail: 'Enkripsi kredensial broker dengan kunci master terpisah dari database.',
  },
  {
    value: 'mTLS',
    detail: 'Autentikasi dua arah untuk seluruh jalur komunikasi antar Engine dan Runner.',
  },
  {
    value: '7 Hari',
    detail: 'Masa uji coba fitur setara Plus dalam mode simulasi sebelum berlangganan.',
  },
]

/**
 * LandingPage adalah beranda situs publik Imaginary Liquidity (rekavia.com).
 *
 * Seluruh bagian menjelaskan arsitektur Engine vs Runner serta prinsip kerja
 * tanpa menjanjikan keuntungan, tanpa membocorkan parameter rahasia ILA, dan
 * memuat disclaimer risiko yang jelas sesuai `plans/10_LANDING_PAGE.md`.
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
    <div className="bg-[#f3f3f6] text-[#0f0b1a]">
      {/* 1. HERO GELAP -> AURORA DARK PURPLE -> FADE KE #f3f3f6 */}
      <section className="relative overflow-hidden bg-[#06040b] text-white">
        {/* Glow radial atmosfer dark purple */}
        <div
          aria-hidden="true"
          className="pointer-events-none absolute inset-0"
          style={{
            background:
              'radial-gradient(62% 55% at 50% 42%, rgba(109, 40, 217, 0.28) 0%, rgba(59, 7, 100, 0.16) 48%, rgba(6, 4, 11, 0) 100%)',
          }}
        />

        <div className="relative mx-auto w-full max-w-6xl px-4 pt-8 pb-24 sm:px-6 sm:pt-12 sm:pb-32">
          {/* Baris atas: caption kiri + floating glass preview card kanan */}
          <div className="flex flex-col justify-between gap-6 sm:flex-row sm:items-start">
            <p className="max-w-xs text-[13px] leading-relaxed text-white/75">
              Lapisan analisis terstruktur — berjalan di atas akun broker milik Anda sendiri.
            </p>

            <div className="w-full max-w-[240px] self-end rounded-2xl border border-white/15 bg-white/[0.04] p-2.5 shadow-2xl shadow-purple-950/60 backdrop-blur-md sm:self-auto">
              <HeroFinsPreview className="h-auto w-full rounded-xl" />
              <div className="mt-2 flex items-center justify-between px-1 text-[11px] text-white/80">
                <span>Pemindai Struktur ILA</span>
                <span className="rounded bg-[#5b21b6]/70 px-1.5 py-0.5 text-[10px] font-medium text-purple-100">
                  2.0 · 3.0
                </span>
              </div>
            </div>
          </div>

          {/* Tengah: Cincin obsidian kaca + display H1 dua-nada */}
          <div className="relative mt-6 flex min-h-[240px] flex-col items-center justify-center py-6 sm:mt-4 sm:min-h-[340px]">
            <HeroObsidianRings className="pointer-events-none absolute left-1/2 top-1/2 w-[540px] max-w-none -translate-x-1/2 -translate-y-1/2 opacity-90 sm:w-[780px] lg:w-[900px]" />

            <div className="relative z-10 w-full max-w-4xl text-left sm:pl-12 lg:pl-20">
              <h1 className="text-4xl font-semibold leading-[1.06] tracking-tight sm:text-6xl lg:text-[4.4rem]">
                <span className="text-white">Eksekusi Trading</span>
                <span className="text-white/60"> Terkendali</span>
                <br />
                <span className="text-white/80">Berbasis Likuiditas</span>
              </h1>
            </div>
          </div>

          {/* Sub-bar horison di atas pita cahaya ungu gelap */}
          <div className="relative z-10 mt-10 flex flex-col justify-between gap-6 border-t border-white/10 pt-8 md:mt-14 md:flex-row md:items-end">
            <div className="max-w-xl space-y-2">
              <p className="text-[15px] leading-relaxed text-white/85">
                Imaginary Liquidity mengubah aturan trading yang tertulis menjadi proses yang
                konsisten: setup dikenali dengan metode ILA, risiko dibatasi per akun, dan setiap
                keputusan tercatat rapi.
              </p>
              <p className="text-[12px] text-white/65">
                Bukan jaminan keuntungan &amp; bukan nasihat investasi · Akun tetap milik Anda di
                broker Anda.
              </p>
            </div>

            <div className="flex flex-wrap items-center gap-3 shrink-0">
              {/* Anchor biasa: bundel panel terpisah di /app/*. */}
              <a
                href="/app/register"
                className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-5 py-2.5 text-[13px] font-semibold text-[#160a2c] shadow-lg shadow-purple-950/30 transition-colors hover:bg-purple-50"
              >
                Mulai Uji Coba 7 Hari
                <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
              </a>
              <Link
                to="/fitur"
                className="inline-flex items-center justify-center rounded-lg border border-white/20 bg-white/[0.06] px-4 py-2.5 text-[13px] font-medium text-white transition-colors hover:bg-white/10"
              >
                Lihat Cara Kerja
              </Link>
            </div>
          </div>
        </div>

        {/* Pita horison dark-purple yang meluruh mulus ke kanvas terang #f3f3f6 */}
        <div
          aria-hidden="true"
          className="h-24 w-full sm:h-32"
          style={{
            background:
              'linear-gradient(180deg, rgba(6,4,11,0) 0%, rgba(91,33,182,0.52) 38%, rgba(221,214,254,0.72) 76%, #f3f3f6 100%)',
          }}
        />
      </section>

      {/* 2. OVERVIEW + GARIS LENGKUNG METRIK FAKTUAL (#f3f3f6) */}
      <section
        id="cara-kerja"
        className="mx-auto w-full max-w-6xl scroll-mt-20 px-4 pt-12 pb-16 sm:px-6 sm:pt-16 sm:pb-24"
      >
        <div className="grid gap-8 lg:grid-cols-12 lg:items-end">
          <div className="lg:col-span-7">
            <h2 className="text-3xl font-semibold leading-[1.12] tracking-tight sm:text-4xl lg:text-[2.75rem]">
              <span className="text-[#0f0b1a]">Aturan Trading Tertulis,</span>{' '}
              <span className="text-[#615a73]">Berjalan Konsisten</span>
            </h2>
          </div>

          <div className="space-y-4 lg:col-span-5">
            <p className="text-[14px] leading-relaxed text-[#4a435c]">
              Engine memantau pasar dan mengenali setup menggunakan aturan seragam. Runner menilai
              kelayakan setiap akun secara terpisah berdasarkan kuota, tier, dan batas risiko Anda.
            </p>
            <div className="flex flex-wrap items-center gap-3">
              <a
                href="/app/register"
                className="inline-flex items-center justify-center rounded-lg bg-[#4c1d95] px-5 py-2.5 text-[13px] font-medium text-white shadow-sm shadow-purple-950/25 transition-colors hover:bg-[#5b21b6]"
              >
                Mulai Uji Coba 7 Hari
              </a>
              <Link
                to="/bantuan/zona-waktu"
                className="text-[13px] font-medium text-[#4c1d95] underline underline-offset-4 hover:text-[#2e1065]"
              >
                Cara kerja dua zona waktu
              </Link>
            </div>
          </div>
        </div>

        {/* Visual kurva menanjak + 4 metrik berjenjang (desktop) */}
        <div className="relative mt-14 hidden lg:block">
          <MetricsCurveSvg className="w-full" />
          <div className="relative -mt-20 grid grid-cols-4 gap-6 px-4">
            {METRICS.map((item) => (
              <div
                key={item.value}
                className={`rounded-2xl border border-[#e3dff0] bg-white/85 p-5 shadow-sm backdrop-blur-sm ${item.offsetClass}`}
              >
                <p className="text-2xl font-bold tracking-tight text-[#3b0764]">{item.value}</p>
                <p className="mt-2 text-[13px] leading-relaxed text-[#4a435c]">{item.label}</p>
              </div>
            ))}
          </div>
        </div>

        {/* Grid metrik responsif untuk mobile & tablet */}
        <div className="mt-10 grid gap-3 sm:grid-cols-2 lg:hidden">
          {METRICS.map((item) => (
            <div
              key={item.value}
              className="rounded-2xl border border-[#e3dff0] bg-white p-5 shadow-sm"
            >
              <div className="mb-2 h-1.5 w-8 rounded-full bg-[#5b21b6]" aria-hidden="true" />
              <p className="text-xl font-bold tracking-tight text-[#3b0764]">{item.value}</p>
              <p className="mt-1.5 text-[13px] leading-relaxed text-[#4a435c]">{item.label}</p>
            </div>
          ))}
        </div>
      </section>

      {/* 3. SECTION 4-KOLOM ("Kami buat keputusan ini mudah untuk Anda") */}
      <section className="mx-auto w-full max-w-6xl px-4 pb-20 sm:px-6 sm:pb-28">
        <div className="grid gap-6 border-t border-[#e4e0ef] pt-14 lg:grid-cols-12 lg:items-end">
          <div className="lg:col-span-6">
            <h2 className="text-3xl font-semibold leading-[1.12] tracking-tight sm:text-4xl">
              <span className="text-[#0f0b1a]">Kami buat eksekusi</span>{' '}
              <span className="text-[#615a73]">mudah dikendalikan</span>
            </h2>
          </div>
          <div className="flex flex-col items-start justify-between gap-4 sm:flex-row sm:items-center lg:col-span-6">
            <p className="max-w-sm text-[14px] leading-relaxed text-[#4a435c]">
              Dari pembacaan struktur pasar hingga persetujuan sekali ketuk, semua langkah tetap
              berada di bawah kendali penuh Anda.
            </p>
            <Link
              to="/fitur"
              className="inline-flex shrink-0 items-center justify-center rounded-lg bg-[#160a2c] px-4 py-2.5 text-[13px] font-medium text-white transition-colors hover:bg-[#3b0764]"
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
                className="flex min-h-[210px] flex-col justify-between rounded-2xl border border-[#e3dff0] bg-white p-6"
              >
                <h3 className="text-[16px] font-semibold tracking-tight text-[#0f0b1a]">
                  {pillar.title}
                </h3>
                <p className="mt-6 text-[13px] leading-relaxed text-[#4a435c]">{pillar.body}</p>
              </div>
            ) : (
              <div
                key={pillar.anchor}
                className="flex min-h-[210px] flex-col justify-between rounded-2xl border border-[#dcd4f2] bg-[#ece8f8] p-6"
              >
                <p className="text-[13px] leading-relaxed text-[#4a435c]">{pillar.note}</p>
                <p className="mt-6 text-[16px] font-semibold tracking-tight text-[#2e1065]">
                  {pillar.anchor}
                </p>
              </div>
            ),
          )}
        </div>
      </section>

      {/* 4. SPOTLIGHT KEAMANAN (KANVAS #ffffff KONTRAS) */}
      <section className="bg-white py-20 sm:py-28">
        <div className="mx-auto w-full max-w-6xl px-4 sm:px-6">
          <div className="grid gap-10 lg:grid-cols-12 lg:items-center">
            <div className="space-y-5 lg:col-span-4">
              <div className="inline-flex items-center gap-2 rounded-full border border-[#dcd4f2] bg-[#f5f3ff] px-3 py-1 text-[12px] font-medium text-[#4c1d95]">
                <ShieldCheck className="h-3.5 w-3.5" aria-hidden="true" />
                <span>Arsitektur Non-Kustodial</span>
              </div>
              <p className="text-[15px] leading-relaxed text-[#4a435c]">
                Kredensial broker Anda dienkripsi terpisah dari database utama dan tidak pernah
                ditampilkan kembali, bahkan kepada admin. Posisi aktif tetap dijaga oleh stop loss
                dan take profit di sisi broker.
              </p>
              <Link
                to="/harga"
                className="inline-flex items-center justify-center rounded-lg bg-[#160a2c] px-5 py-2.5 text-[13px] font-medium text-white transition-colors hover:bg-[#3b0764]"
              >
                Lihat Paket &amp; Harga
              </Link>
            </div>

            <div className="relative lg:col-span-8">
              <SecurityChainArtwork className="mx-auto h-auto w-full max-w-xl" />
              <div className="mt-4 max-w-xs rounded-xl border border-[#e3dff0] bg-[#f8f7fc] p-4 text-[13px] leading-relaxed text-[#4a435c] lg:absolute lg:right-0 lg:top-1/2 lg:mt-0 lg:-translate-y-1/2 lg:bg-white/90 lg:shadow-md lg:backdrop-blur-sm">
                Saat mode pemeliharaan aktif, sistem hanya menahan entry baru. Posisi yang sudah
                terbuka tidak pernah ditutup sepihak.
              </div>
            </div>
          </div>

          <h2 className="mt-12 max-w-3xl text-3xl font-semibold leading-[1.1] tracking-tight sm:text-5xl">
            <span className="text-[#0f0b1a]">Keamanan berlapis menjaga</span>
            <br />
            <span className="text-[#615a73]">akun Anda tetap terlindungi</span>
          </h2>
        </div>
      </section>

      {/* 5. ASYMMETRIC 3-COLUMN SHOWCASE (#f3f3f6) */}
      <section className="mx-auto w-full max-w-6xl px-4 py-20 sm:px-6 sm:py-28">
        <div className="grid gap-6 lg:grid-cols-12 lg:items-stretch">
          {/* Kolom Kiri (40%) */}
          <div className="flex flex-col justify-between gap-8 lg:col-span-5">
            <div className="space-y-4">
              <h2 className="text-3xl font-semibold leading-[1.12] tracking-tight sm:text-4xl">
                <span className="text-[#0f0b1a]">Arsitektur terpisah,</span>{' '}
                <span className="text-[#615a73]">kendali di tangan Anda</span>
              </h2>
              <p className="text-[14px] leading-relaxed text-[#4a435c]">
                Engine hanya bertugas membaca struktur pasar dan menyiarkan kandidat setup. Runner
                memproses kelayakan setiap akun secara terisolasi agar kesalahan satu akun tidak
                memengaruhi akun lainnya.
              </p>
            </div>

            <div className="flex flex-wrap items-center gap-3">
              <a
                href="/app/register"
                className="inline-flex items-center justify-center rounded-lg bg-[#4c1d95] px-5 py-2.5 text-[13px] font-medium text-white shadow-sm shadow-purple-950/20 transition-colors hover:bg-[#5b21b6]"
              >
                Daftar Sekarang
              </a>
              <Link
                to="/tentang"
                className="inline-flex items-center justify-center rounded-lg border border-[#d6d0e6] bg-white px-4 py-2.5 text-[13px] font-medium text-[#18112b] transition-colors hover:bg-[#ece8f8]"
              >
                Tentang Sistem
              </Link>
            </div>
          </div>

          {/* Kolom Tengah (35%): Kartu vertikal tinggi gelombang sutra dark-purple */}
          <div className="relative min-h-[340px] overflow-hidden rounded-2xl border border-[#3b0764]/20 shadow-lg shadow-purple-950/15 lg:col-span-4 lg:min-h-[430px]">
            <PurpleSilkWaves className="h-full w-full object-cover" />
            <div className="absolute inset-x-4 bottom-4 rounded-xl border border-white/15 bg-[#090414]/75 px-4 py-3 text-white backdrop-blur-md">
              <p className="text-[11px] uppercase tracking-[0.16em] text-purple-200">
                Pipeline Terisolasi
              </p>
              <p className="mt-0.5 text-[13px] font-medium">ILA 2.0 &amp; ILA 3.0</p>
            </div>
          </div>

          {/* Kolom Kanan (25%): 2 catatan ringkas + kartu lanskap mini */}
          <div className="flex flex-col justify-between gap-4 lg:col-span-3">
            <div className="space-y-4">
              <div className="rounded-2xl border border-[#e3dff0] bg-white p-5">
                <h3 className="text-[14px] font-semibold text-[#0f0b1a]">Metode ILA Terukur</h3>
                <p className="mt-1.5 text-[13px] leading-relaxed text-[#4a435c]">
                  Berfokus pada sapuan likuiditas dan konfirmasi struktur harga, tanpa
                  memperhitungkan berita ekonomi secara otomatis.
                </p>
              </div>
              <div className="rounded-2xl border border-[#e3dff0] bg-white p-5">
                <h3 className="text-[14px] font-semibold text-[#0f0b1a]">
                  Koneksi Broker Langsung
                </h3>
                <p className="mt-1.5 text-[13px] leading-relaxed text-[#4a435c]">
                  Terhubung melalui token akses dari akun broker Anda sendiri dengan pencatatan
                  audit penuh.
                </p>
              </div>
            </div>

            <div className="relative overflow-hidden rounded-2xl border border-[#3b0764]/20">
              <PurpleRibbonTile className="h-32 w-full object-cover" />
              <div className="absolute inset-0 flex items-end bg-gradient-to-t from-[#090414]/85 via-transparent to-transparent p-4">
                <p className="text-[12px] font-medium text-white">Kendali Risiko Mandiri</p>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* 6. BENTO GRID "SIAP BERKEMBANG DI SETIAP FASE" + RINGKAS PAKET */}
      <section className="mx-auto w-full max-w-6xl px-4 pb-20 sm:px-6 sm:pb-24">
        <div className="mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-semibold leading-[1.12] tracking-tight sm:text-4xl">
            <span className="text-[#0f0b1a]">Siap Berkembang</span>{' '}
            <span className="text-[#615a73]">di Setiap Fase</span>
          </h2>
          <p className="mt-3 text-[14px] leading-relaxed text-[#4a435c]">
            Mulai dari mode simulasi gratis, uji coba 7 hari setara Plus, hingga akun live dengan
            pengaturan risiko yang sesuai kebutuhan Anda.
          </p>
        </div>

        <div className="mt-12 grid gap-4 lg:grid-cols-12">
          {/* Kolom Kiri: 1 kartu tinggi */}
          <div className="flex flex-col justify-between gap-6 rounded-2xl border border-[#e3dff0] bg-white p-6 lg:col-span-4">
            <div>
              <p className="text-[12px] font-semibold uppercase tracking-[0.14em] text-[#4c1d95]">
                Jejak Audit
              </p>
              <p className="mt-2 text-[14px] leading-relaxed text-[#4a435c]">
                Setiap sinyal, status persetujuan, dan alasan penolakan runner tercatat agar Anda
                selalu tahu mengapa sebuah entry diambil atau dilewati.
              </p>
            </div>

            <div className="overflow-hidden rounded-xl border border-[#3b0764]/15">
              <PurpleOrbTile className="h-36 w-full object-cover" />
            </div>

            <div>
              <p className="text-[13px] text-[#615a73]">Transparan tanpa kotak hitam.</p>
              <p className="mt-1 text-2xl font-bold tracking-tight text-[#0f0b1a]">
                100% Tercatat di Log
              </p>
            </div>
          </div>

          {/* Kolom Tengah: 2 kartu bertumpuk */}
          <div className="flex flex-col gap-4 lg:col-span-4">
            <div className="rounded-2xl border border-[#e3dff0] bg-white p-6">
              <h3 className="text-[16px] font-semibold tracking-tight text-[#0f0b1a]">
                Notifikasi Telegram Instan
              </h3>
              <p className="mt-2 text-[13px] leading-relaxed text-[#4a435c]">
                Terima peringatan setup dan tautan konfirmasi sekali pakai langsung di Telegram
                tanpa harus selalu membuka dashboard.
              </p>
            </div>

            <div className="flex flex-1 flex-col justify-between rounded-2xl border border-[#dcd4f2] bg-[#ece8f8] p-6">
              <span className="w-fit rounded-full bg-[#4c1d95] px-3 py-1 text-[11px] font-medium text-white">
                Disiplin Eksekusi
              </span>
              <div className="mt-8">
                <h3 className="text-xl font-bold tracking-tight text-[#1e0948]">
                  Batas Harian Terukur per Akun
                </h3>
                <p className="mt-2 text-[13px] leading-relaxed text-[#4a435c]">
                  Atur persentase risiko, batas jumlah trade harian, serta aturan berhenti setelah
                  mencapai target harian untuk mencegah overtrading.
                </p>
              </div>
            </div>
          </div>

          {/* Kolom Kanan: 3 kartu metrik bertumpuk */}
          <div className="flex flex-col gap-4 lg:col-span-4">
            {BENTO_STATS.map((stat) => (
              <div
                key={stat.value}
                className="flex flex-1 flex-col justify-between rounded-2xl border border-[#e3dff0] bg-white p-5"
              >
                <p className="text-2xl font-bold tracking-tight text-[#3b0764]">{stat.value}</p>
                <p className="mt-2 text-[13px] leading-relaxed text-[#4a435c]">{stat.detail}</p>
              </div>
            ))}
          </div>
        </div>

        {/* Ringkas Paket dari Database (harga selalu bersumber dari backend / admin) */}
        <div className="mt-14 rounded-2xl border border-[#e3dff0] bg-white p-6 sm:p-8">
          <div className="flex flex-col justify-between gap-4 border-b border-[#ece8f4] pb-6 sm:flex-row sm:items-end">
            <div>
              <h3 className="text-xl font-semibold tracking-tight text-[#0f0b1a]">
                Paket Berlangganan
              </h3>
              <p className="mt-1 text-[13px] text-[#4a435c]">
                Harga mengikuti katalog aktif di server; detail perbandingan lengkap tersedia di
                halaman Harga.
              </p>
            </div>
            <Link
              to="/harga"
              className="inline-flex items-center gap-1.5 text-[13px] font-semibold text-[#4c1d95] underline underline-offset-4 hover:text-[#2e1065]"
            >
              Bandingkan paket selengkapnya
              <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
            </Link>
          </div>

          {plansLoading ? (
            <div className="mt-6 grid gap-4 sm:grid-cols-3">
              {[0, 1, 2].map((idx) => (
                <div
                  key={idx}
                  className="h-44 animate-pulse rounded-xl border border-[#ece8f4] bg-[#f7f6fb]"
                />
              ))}
            </div>
          ) : plans && plans.length > 0 ? (
            <div className="mt-6 grid gap-4 sm:grid-cols-3">
              {plans.map((plan) => {
                const isHighlight = plan.code === 'plus'
                const featureEntries = Object.entries(plan.features ?? {}).slice(0, 5)
                return (
                  <div
                    key={plan.code}
                    className={
                      isHighlight
                        ? 'flex flex-col justify-between rounded-xl border-2 border-[#5b21b6] bg-[#faf8ff] p-5'
                        : 'flex flex-col justify-between rounded-xl border border-[#e3dff0] bg-white p-5'
                    }
                  >
                    <div>
                      <div className="flex items-center justify-between gap-2">
                        <p className="text-[15px] font-semibold text-[#0f0b1a]">{plan.name}</p>
                        {isHighlight && (
                          <span className="rounded-full bg-[#4c1d95] px-2.5 py-0.5 text-[11px] font-medium text-white">
                            Lengkap
                          </span>
                        )}
                      </div>
                      <p className="mt-1 text-[12px] text-[#4a435c]">{plan.description || '—'}</p>
                      <p className="mt-4 text-2xl font-bold tabular-nums tracking-tight text-[#0f0b1a]">
                        {formatIDR(plan.price_idr)}
                      </p>
                      <p className="text-[11px] text-[#615a73]">per bulan</p>

                      {featureEntries.length > 0 && (
                        <ul className="mt-4 space-y-1.5 border-t border-[#ece8f4] pt-3 text-[12px]">
                          {featureEntries.map(([key, value]) => (
                            <li key={key} className="flex justify-between gap-2">
                              <span className="text-[#615a73]">{key}</span>
                              <span className="text-right font-medium text-[#0f0b1a]">{value}</span>
                            </li>
                          ))}
                        </ul>
                      )}
                    </div>

                    <div className="mt-5">
                      {plan.ila3_coming_soon && (
                        <p className="mb-2 text-[11px] font-medium text-[#4c1d95]">
                          ILA 3.0 dibuka bertahap
                        </p>
                      )}
                      <a
                        href="/app/register"
                        className={
                          isHighlight
                            ? 'block rounded-lg bg-[#4c1d95] px-3 py-2 text-center text-[13px] font-medium text-white transition-colors hover:bg-[#5b21b6]'
                            : 'block rounded-lg border border-[#d6d0e6] bg-[#f8f7fc] px-3 py-2 text-center text-[13px] font-medium text-[#18112b] transition-colors hover:bg-[#ece8f8]'
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
            <p className="mt-6 text-[13px] text-[#4a435c]">
              Daftar paket dapat dilihat langsung pada halaman{' '}
              <Link to="/harga" className="font-medium text-[#4c1d95] underline">
                Harga
              </Link>
              .
            </p>
          )}
        </div>
      </section>

      {/* 7. BANNER CTA GELAP + DISCLAIMER RISIKO JUJUR */}
      <section className="mx-auto w-full max-w-6xl px-4 pb-8 sm:px-6">
        <div className="relative overflow-hidden rounded-3xl bg-[#06040b] px-6 py-12 text-white shadow-2xl shadow-purple-950/20 sm:px-10 sm:py-16 lg:px-14">
          <div
            aria-hidden="true"
            className="pointer-events-none absolute inset-0"
            style={{
              background:
                'radial-gradient(50% 80% at 85% 50%, rgba(109, 40, 217, 0.36) 0%, rgba(59, 7, 100, 0.14) 55%, rgba(6, 4, 11, 0) 100%)',
            }}
          />

          <div className="relative z-10 grid gap-10 lg:grid-cols-12 lg:items-center">
            <div className="space-y-6 lg:col-span-7">
              <h2 className="text-3xl font-semibold leading-[1.12] tracking-tight sm:text-4xl lg:text-[2.6rem]">
                <span className="text-white">Mulai Uji Coba</span>{' '}
                <span className="text-white/65">Tanpa Komitmen</span>
              </h2>

              <div className="rounded-xl border border-purple-400/25 bg-white/[0.04] p-4 text-[13px] leading-relaxed text-white/80">
                <p className="font-semibold text-purple-200">Disclaimer Risiko Trading</p>
                <p className="mt-1">
                  Perdagangan mata uang dan derivatif memiliki risiko kerugian tinggi. Imaginary
                  Liquidity adalah alat bantu analisis dan eksekusi—bukan jaminan keuntungan dan
                  bukan nasihat keuangan. Sistem dapat membuat kesalahan; hasil masa lalu tidak
                  menjamin hasil di masa depan.
                </p>
              </div>

              <div className="flex flex-wrap items-center gap-4">
                <a
                  href="/app/register"
                  className="inline-flex items-center justify-center gap-2 rounded-lg bg-[#5b21b6] px-5 py-2.5 text-[13px] font-semibold text-white shadow-lg shadow-purple-950/40 transition-colors hover:bg-[#6d28d9]"
                >
                  Mulai Uji Coba 7 Hari
                  <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
                </a>
                <Link
                  to="/legal/risk"
                  className="text-[13px] font-medium text-purple-200 underline underline-offset-4 transition-colors hover:text-white"
                >
                  Baca Disclaimer Risiko
                </Link>
              </div>
            </div>

            <div className="relative flex items-center justify-center lg:col-span-5">
              <CtaPrismArtwork className="h-auto w-full max-w-[340px]" />
              <div className="absolute bottom-2 right-0 w-48 rounded-xl border border-white/15 bg-white/[0.06] p-3 backdrop-blur-md sm:right-2">
                <CtaInsetPreview className="h-auto w-full" />
                <p className="mt-1 text-[11px] font-medium text-purple-100">
                  gRPC mTLS · Pemisahan Engine &amp; Runner
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>
  )
}
