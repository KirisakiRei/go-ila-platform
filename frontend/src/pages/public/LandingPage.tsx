import { useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ArrowRight } from 'lucide-react'
import { Link, useLocation } from 'react-router-dom'
import Timeline from '@/components/ui/timeline'
import { publicContentApi } from '@/lib/system-api'
import { HeroTradingGeometry, MetricsCurveSvg } from './components/LandingArtwork'
import { InviteRequestSection } from './components/InviteRequestSection'
import { SignalOverviewGrid } from './components/SignalOverviewGrid'
import { useInView } from './components/use-in-view'

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

/**
 * Jeda kemunculan label metrik, dihitung dari saat kurva mulai menggambar.
 *
 * Tiap label menyala sedikit setelah node-nya tersentuh garis, jadi urutannya
 * mengikuti arah pertumbuhan kurva dari kiri ke kanan.
 */
const LABEL_DELAY = [500, 800, 1100, 1400]

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
    note: 'Akses dibuka bertahap lewat undangan dengan mode simulasi tersedia sebelum menghubungkan akun broker live.',
    anchor: 'Akses Terkurasi & Simulasi',
  },
]

/**
 * LandingPage adalah beranda situs publik Imaginary Liquidity (rekavia.com).
 *
 * Susunan beranda:
 *  1. Hero Utama (`#hero-band`)
 *  2. Overview & Kurva Metrik (`#cara-kerja`)
 *  3. Storyline Evolusi Metode ILA (`#evolusi-ila`)
 *  4. Pantauan Pasar Terkini
 *  5. Eksekusi Cepat & Pilar Kendali
 *  6. Form Pengajuan Akses Undangan (`#ajukan-akses`) + Disclaimer Risiko
 */
export default function LandingPage() {
  const { hash } = useLocation()
  const [kurvaRef, kurvaTerlihat] = useInView<HTMLDivElement>()
  const [metrikRef, metrikTerlihat] = useInView<HTMLDivElement>()
  const { data: status } = useQuery({
    queryKey: ['public', 'status'],
    queryFn: publicContentApi.status,
    staleTime: 30_000,
  })

  useEffect(() => {
    if (hash === '#cara-kerja' || hash === '#ajukan-akses' || hash === '#evolusi-ila') {
      const target = document.getElementById(hash.slice(1))
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
        {/* 1. BAGIAN HERO */}
        <section id="hero-band" className="relative text-white">
          {/* Background geometris tema trading yang clean & rapi */}
          <HeroTradingGeometry
            className="pointer-events-none absolute inset-0 h-full w-full"
            active
          />

          <div className="relative mx-auto w-full max-w-6xl px-4 pb-16 pt-28 sm:px-6 sm:pb-24 sm:pt-36">
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
                  Sinyal sweep likuiditas dan eksekusi order, langsung di akun broker Anda.
                </p>
              </div>
              <div className="flex shrink-0 flex-wrap items-center gap-3">
                <a
                  href="#ajukan-akses"
                  className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-5 py-2.5 text-[13px] font-semibold text-[#09090b] shadow-lg shadow-black/30 press hover:bg-zinc-200"
                >
                  Ajukan Akses
                  <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
                </a>
                <Link
                  to="/fitur"
                  className="inline-flex items-center justify-center rounded-lg border border-white/25 bg-black/35 px-4 py-2.5 text-[13px] font-medium text-white press hover:bg-white/10"
                >
                  Pelajari Cara Kerja
                </Link>
              </div>
            </div>
          </div>
        </section>

        {/* 2. OVERVIEW + KURVA NAIK TANPA CARD */}
        <section
          id="cara-kerja"
          className="relative z-10 mx-auto w-full max-w-6xl scroll-mt-20 px-4 pb-20 pt-12 sm:px-6 sm:pb-24 sm:pt-16"
        >
          <div className="max-w-2xl space-y-5">
            <h2 className="text-3xl font-semibold leading-[1.12] tracking-tight text-[#09090b] sm:text-4xl lg:text-[2.75rem]">
              Atur batas risikonya, sistem yang menagih disiplinnya.
            </h2>
            <p className="max-w-lg text-[14px] leading-relaxed text-[#3f3f46]">
              Setiap setup punya jejak: kapan terbit, kapan Anda setujui, di harga berapa masuk,
              dan bagaimana posisi ditutup.
            </p>
            <div className="flex flex-wrap items-center gap-4 pt-1">
              <a
                href="#ajukan-akses"
                className="inline-flex items-center justify-center rounded-lg bg-[#09090b] px-5 py-2.5 text-[13px] font-medium text-white press hover:bg-[#27272a]"
              >
                Ajukan Akses
              </a>
              <Link
                to="/bantuan/zona-waktu"
                className="text-[13px] font-medium text-[#18181b] underline underline-offset-4 hover:text-[#52525b]"
              >
                Cara kerja dua zona waktu
              </Link>
            </div>
          </div>

          {/* Kurva naik melengkung mulus + garis vertikal "|" sejajar tanpa card (desktop).
              Garis dan label baru muncul setelah seksi ini masuk layar, supaya
              pengunjung menyaksikan kurvanya tergambar, bukan menemukannya sudah jadi. */}
          <div ref={kurvaRef} className="relative -mt-6 hidden md:block">
            <MetricsCurveSvg className="h-auto w-full" active={kurvaTerlihat} />
            {CURVE_METRICS.map((item, i) => (
              <div
                key={item.value}
                className={`absolute ${item.posClass} w-[20%] pl-4`}
                style={{
                  opacity: kurvaTerlihat ? 1 : 0,
                  transform: kurvaTerlihat ? 'translateY(0)' : 'translateY(8px)',
                  transition: `opacity var(--motion-enter) var(--ease-out) ${LABEL_DELAY[i]}ms, transform var(--motion-enter) var(--ease-out) ${LABEL_DELAY[i]}ms`,
                }}
              >
                <p className="text-2xl font-semibold leading-none tracking-tight text-[#09090b] tabular-nums lg:text-[2rem]">
                  {item.value}
                </p>
                <p className="mt-2 text-[12px] leading-snug text-[#52525b] lg:text-[13px]">
                  {item.label}
                </p>
              </div>
            ))}
          </div>

          {/* Layout metrik bersih tanpa card untuk layar kecil.
              Di layar kecil tidak ada kurva, jadi yang bergiliran hanya angkanya. */}
          <div ref={metrikRef} className="mt-10 grid grid-cols-2 gap-x-6 gap-y-8 md:hidden">
            {CURVE_METRICS.map((item, i) => (
              <div
                key={item.value}
                className="border-l border-zinc-300 pl-3.5"
                style={{
                  opacity: metrikTerlihat ? 1 : 0,
                  transform: metrikTerlihat ? 'translateY(0)' : 'translateY(8px)',
                  transition: `opacity var(--motion-enter) var(--ease-out) ${i * 60}ms, transform var(--motion-enter) var(--ease-out) ${i * 60}ms`,
                }}
              >
                <p className="text-2xl font-semibold leading-none tracking-tight text-[#09090b] tabular-nums">
                  {item.value}
                </p>
                <p className="mt-2 text-[12px] leading-snug text-[#52525b]">{item.label}</p>
              </div>
            ))}
          </div>
        </section>
      </div>

      {/* 3. STORYLINE EVOLUSI METODE ILA (ILA 1.0 -> ILA 2.0 -> ILA 2.1 -> ILA 3.0) */}
      <Timeline />

      {/* 4. PANTAUAN PASAR TERKINI: tiga instrumen utama dengan arah terakhir. */}
      <section className="bg-[#f5f5f7] pb-4 pt-20 sm:pb-6 sm:pt-28">
        <div className="mx-auto w-full max-w-6xl px-4 sm:px-6">
          <div className="max-w-2xl space-y-3">
            <p className="text-[12px] font-semibold uppercase tracking-[0.14em] text-[#52525b]">
              Pantauan Pasar
            </p>
            <h2 className="text-3xl font-semibold leading-[1.12] tracking-tight sm:text-4xl">
              <span className="text-[#09090b]">Arah pasar terkini,</span>{' '}
              <span className="text-[#52525b]">diperbarui sepanjang sesi</span>
            </h2>
            <p className="text-[14px] leading-relaxed text-[#52525b]">
              Pembacaan arah tren dari struktur likuiditas instrumen utama secara real-time.
            </p>
          </div>

          <div className="mt-8">
            <SignalOverviewGrid />
          </div>

          <div className="mt-8 flex flex-wrap items-center justify-between gap-4 border-t border-zinc-200 pt-6">
            <p className="text-[13px] text-[#71717a]">
              Angka entry, SL, dan TP masuk ke dashboard begitu sinyal Anda terima.
            </p>
            <div className="flex flex-wrap items-center gap-3">
              <a
                href="#ajukan-akses"
                className="inline-flex items-center justify-center rounded-lg bg-[#09090b] px-4 py-2 text-[13px] font-medium text-white press hover:bg-[#27272a]"
              >
                Ajukan Akses
              </a>
              <Link
                to="/fitur"
                className="text-[13px] font-medium text-[#18181b] underline underline-offset-4 hover:text-[#52525b]"
              >
                Pelajari Fitur
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* 5 & 6. ZONA OBSIDIAN BERGRADASI: terang -> gelap (Hero 5 & 6) -> memudar kembali ke footer terang.
          Pita gradasi atas & bawah memakai tinggi lega + vertical edge-feathering agar tidak pernah
          meninggalkan garis potong diagonal di lebar layar berapa pun, sekaligus menjaga teks Hero 5 & 6
          tetap di atas #050507 ber-kontras penuh. */}
      <div className="relative bg-[#050507] text-white">
        {/* Transisi masuk: terang (#f5f5f7) -> obsidian (#050507) mulus senada Hero 1 -> 2 */}
        <div
          aria-hidden="true"
          className="pointer-events-none h-56 w-full sm:h-80 lg:h-[420px]"
          style={{
            background: [
              'linear-gradient(to bottom, #f5f5f7 0%, rgba(245, 245, 247, 0.92) 8%, rgba(245, 245, 247, 0.45) 20%, rgba(245, 245, 247, 0) 38%)',
              'linear-gradient(to top, #050507 0%, rgba(5, 5, 7, 0.92) 8%, rgba(5, 5, 7, 0.45) 20%, rgba(5, 5, 7, 0) 38%)',
              'radial-gradient(65% 58% at 18% 26%, rgba(245, 245, 247, 0.85) 0%, rgba(245, 245, 247, 0) 100%)',
              'radial-gradient(65% 58% at 82% 76%, rgba(5, 5, 7, 0.85) 0%, rgba(5, 5, 7, 0) 100%)',
              'linear-gradient(173deg, #f5f5f7 0%, #f5f5f7 16%, #e4e4e7 28%, #a1a1aa 43%, #52525b 58%, #27272a 71%, #111114 82%, #050507 91%, #050507 100%)',
            ].join(', '),
          }}
        />

        {/* 5. PILAR KEPUTUSAN ("Eksekusi cepat, keputusan tetap di tangan Anda.") */}
        <section className="relative z-10 pb-8 pt-2 sm:pb-12 sm:pt-6">
          <div className="mx-auto w-full max-w-6xl px-4 sm:px-6">
            <div className="grid gap-6 lg:grid-cols-12 lg:items-end">
              <div className="lg:col-span-6">
                <h2 className="text-3xl font-semibold leading-[1.12] tracking-tight sm:text-4xl">
                  <span className="text-white">Eksekusi cepat,</span>{' '}
                  <span className="text-zinc-400">keputusan tetap di tangan Anda.</span>
                </h2>
              </div>
              <div className="flex flex-col items-start justify-between gap-4 sm:flex-row sm:items-center lg:col-span-6">
                <p className="max-w-sm text-[14px] leading-relaxed text-zinc-300">
                  Pilih otomatis tanpa jeda, atau konfirmasi manual sekali ketuk.
                </p>
                <Link
                  to="/fitur"
                  className="inline-flex shrink-0 items-center justify-center rounded-lg bg-white px-4 py-2.5 text-[13px] font-semibold text-[#09090b] press hover:bg-zinc-200"
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
                    className="hover-lift flex min-h-[205px] flex-col justify-between rounded-2xl border border-white/10 bg-white/[0.04] p-6 hover:border-white/25"
                  >
                    <h3 className="text-[16px] font-semibold tracking-tight text-white">
                      {pillar.title}
                    </h3>
                    <p className="mt-6 text-[13px] leading-relaxed text-zinc-400">{pillar.body}</p>
                  </div>
                ) : (
                  <div
                    key={pillar.anchor}
                    className="hover-lift flex min-h-[205px] flex-col justify-between rounded-2xl border border-white/15 bg-white/[0.08] p-6 hover:border-white/30"
                  >
                    <p className="text-[13px] leading-relaxed text-zinc-300">{pillar.note}</p>
                    <p className="mt-6 text-[16px] font-semibold tracking-tight text-white">
                      {pillar.anchor}
                    </p>
                  </div>
                ),
              )}
            </div>
          </div>
        </section>

        {/* 6. CTA INVITE-ONLY: form pengajuan akses + sosmed + disclaimer ringkas */}
        <InviteRequestSection socialLinks={status?.social_links} />

        {/* Transisi keluar: obsidian (#050507) -> terang (#f5f5f7) menyatu mulus ke footer */}
        <div
          aria-hidden="true"
          className="pointer-events-none h-56 w-full sm:h-80 lg:h-[420px]"
          style={{
            background: [
              'linear-gradient(to bottom, #050507 0%, rgba(5, 5, 7, 0.92) 8%, rgba(5, 5, 7, 0.45) 20%, rgba(5, 5, 7, 0) 38%)',
              'linear-gradient(to top, #f5f5f7 0%, rgba(245, 245, 247, 0.92) 8%, rgba(245, 245, 247, 0.45) 20%, rgba(245, 245, 247, 0) 38%)',
              'radial-gradient(65% 58% at 18% 24%, rgba(5, 5, 7, 0.85) 0%, rgba(5, 5, 7, 0) 100%)',
              'radial-gradient(65% 58% at 82% 74%, rgba(245, 245, 247, 0.85) 0%, rgba(245, 245, 247, 0) 100%)',
              'linear-gradient(173deg, #050507 0%, #050507 14%, #111114 25%, #27272a 37%, #52525b 50%, #a1a1aa 64%, #e4e4e7 78%, #f5f5f7 89%, #f5f5f7 100%)',
            ].join(', '),
          }}
        />
      </div>
    </div>
  )
}
