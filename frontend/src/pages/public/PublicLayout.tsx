import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { AlertTriangle, Menu, X } from 'lucide-react'
import { Link, NavLink, Outlet, useLocation } from 'react-router-dom'
import { publicContentApi } from '@/lib/system-api'
import { cn } from '@/lib/utils'
import { BrandMark } from './components/LandingArtwork'

/**
 * Gradasi footer situs publik — satu kanvas kontinu, tanpa patahan.
 *
 * Prinsip "tidak patah": tepi atas kanvas ditarik sampai persis menyentuh puncak
 * kartu Banner CTA (Hero 8), sehingga gradasi tidak mengintip dari baliknya
 * melainkan mengalir: hitam pekat di belakang kartu, memudar ke terang di
 * sepanjang kolom tautan, lalu turun kembali ke #050507 di belakang IMAGINARY.
 *
 * Persentase di bawah ditakar dari tinggi kanvas sebenarnya (bukan tebakan),
 * lihat komentar tiap lapisan:
 * - Desktop 1440px: kartu CTA 0%-47%, zona tautan 49%-77%, watermark 77%+.
 * - Mobile: footer ±3x lebih tinggi, kartu CTA 0%-39%, tautan 40%-96%.
 * Dua profil ini dipilih di sisi klien mengikuti breakpoint `sm`
 * (lihat useFooterGradient).
 */
const DARK_TOP = '#050507'
const DARK_BOTTOM = 'rgb(5, 5, 7)'

const FOOTER_GRADIENT_DESKTOP = [
  // Lumpur cahaya di atas warna dasar, menyala di zona tautan (49%-77% kanvas).
  'linear-gradient(to bottom, rgba(255, 255, 255, 0) 0%, rgba(255, 255, 255, 0) 46.5%, rgba(255, 255, 255, 0.2) 50.5%, rgba(255, 255, 255, 0.46) 55%, rgba(255, 255, 255, 0.4) 66%, rgba(255, 255, 255, 0.12) 75%, rgba(5, 5, 7, 0.46) 79%, rgba(5, 5, 7, 0.97) 82.5%, rgba(5, 5, 7, 1) 84.5%)',
  // Kantung mist lembut supaya zona terang tidak terasa datar.
  'radial-gradient(58% 14% at 18% 58%, rgba(255, 255, 255, 0.34) 0%, rgba(255, 255, 255, 0) 100%)',
  'radial-gradient(48% 11% at 82% 70%, rgba(244, 244, 245, 0.26) 0%, rgba(244, 244, 245, 0) 100%)',
  // Dasar: hitam pekat di area kartu CTA, lalu #050507 di dasar watermark.
  `linear-gradient(to bottom, ${DARK_TOP} 0%, ${DARK_TOP} 46.5%, rgba(5, 5, 7, 0.9) 51%, rgba(5, 5, 7, 0.66) 62%, rgba(5, 5, 7, 0.35) 72%, ${DARK_BOTTOM} 84%, ${DARK_BOTTOM} 100%)`,
].join(', ')

const FOOTER_GRADIENT_COMPACT = [
  // Mobile: kartu CTA 0%-39%, zona tautan 40%-96%. Di mobile footer ±3x lebih
  // tinggi daripada desktop, sehingga campuran putih transparan saja hanya
  // menghasilkan abu gelap. Karena itu dasar kanvas dibuat gelap penuh lebih
  // dulu, lalu pita terangnya ditulis sebagai warna opaque (#f0f0f3) dengan
  // tepi yang di-soft agar tetap terbaca sebagai gradasi.
  'linear-gradient(to bottom, #f0f0f3 0%, #f0f0f3 39%, #f0f0f3 82%, #e6e6ea 86%, #d3d3d9 89%, #a9a9b0 91.5%, #6b6b73 94%, #2f2f34 96.5%, rgb(5, 5, 7) 99%, rgb(5, 5, 7) 100%)',
  'linear-gradient(to bottom, rgba(255, 255, 255, 0) 0%, rgba(255, 255, 255, 0) 39%, rgba(255, 255, 255, 0.95) 41.5%, rgba(255, 255, 255, 0.5) 44%, rgba(255, 255, 255, 0) 47%)',
  'linear-gradient(to bottom, rgba(5, 5, 7, 0) 0%, rgba(5, 5, 7, 0) 82%, rgba(5, 5, 7, 0.12) 87%, rgba(5, 5, 7, 0.45) 92%, rgba(5, 5, 7, 0.88) 96.5%, rgb(5, 5, 7) 99.5%)',
  `radial-gradient(80% 14% at 22% 52%, rgba(255, 255, 255, 0.18) 0%, rgba(255, 255, 255, 0) 100%)`,
  `linear-gradient(to bottom, ${DARK_TOP} 0%, ${DARK_TOP} 39.5%, rgba(5, 5, 7, 0.92) 42%, rgba(5, 5, 7, 0.85) 60%, rgba(5, 5, 7, 0.8) 85%, ${DARK_BOTTOM} 100%)`,
].join(', ')

/** Subhalaman bertumpu pada #f5f5f7, jadi kanvas cukup turun ke gelap di dasar. */
const FOOTER_GRADIENT_SUBPAGE = [
  'linear-gradient(to bottom, rgba(5, 5, 7, 0) 52%, rgba(5, 5, 7, 0.5) 72%, rgba(5, 5, 7, 0.92) 82%, rgb(5, 5, 7) 88%, rgb(5, 5, 7) 100%)',
  'radial-gradient(58% 40% at 16% 34%, rgba(255, 255, 255, 0.45) 0%, rgba(255, 255, 255, 0) 100%)',
  'linear-gradient(196deg, #f5f5f7 0%, #eeeef1 30%, #e2e2e7 52%, #d5d5da 74%, #d0d0d6 100%)',
].join(', ')

/** Navigasi utama situs publik, mengikuti alur evaluasi calon pengguna. */
const NAV = [
  { to: '/fitur', label: 'Fitur' },
  { to: '/harga', label: 'Harga' },
  { to: '/#cara-kerja', label: 'Cara Kerja', hash: true },
  { to: '/bantuan', label: 'Bantuan' },
  { to: '/tentang', label: 'Tentang' },
]

type HeaderTheme = 'top' | 'hero-scrolled' | 'light'

/** Titik henti Tailwind `sm`: di bawah ini footer bertumpuk sehingga jauh lebih tinggi. */
const COMPACT_FOOTER_QUERY = '(max-width: 639px)'

/**
 * useFooterGradient memilih profil gradasi footer sesuai breakpoint.
 *
 * Tidak memakai gradasi breakpoint Tailwind karena proyek ini hanya memuat
 * `tailwindcss-animate` (tanpa plugin arbitrary variant). Perubahan breakpoint
 * dibaca lewat `matchMedia` di dalam event listener, bukan `setState` sinkron di
 * `useEffect`, agar lolos aturan `react-hooks/set-state-in-effect`.
 *
 * `matchMedia` tidak selalu ada (jsdom, browser sangat lama), jadi profil
 * desktop dipakai sebagai nilai aman bawaan.
 */
function useFooterGradient(isHome: boolean) {
  const [isCompact, setIsCompact] = useState(
    () =>
      typeof window !== 'undefined' &&
      typeof window.matchMedia === 'function' &&
      window.matchMedia(COMPACT_FOOTER_QUERY).matches,
  )

  useEffect(() => {
    if (typeof window.matchMedia !== 'function') return

    const query = window.matchMedia(COMPACT_FOOTER_QUERY)
    const onChange = (event: MediaQueryListEvent) => setIsCompact(event.matches)
    query.addEventListener('change', onChange)
    return () => query.removeEventListener('change', onChange)
  }, [])

  if (!isHome) return FOOTER_GRADIENT_SUBPAGE
  return isCompact ? FOOTER_GRADIENT_COMPACT : FOOTER_GRADIENT_DESKTOP
}

/**
 * PublicLayout adalah kerangka situs publik.
 *
 * Halaman publik memakai bundel terpisah dari dashboard, sehingga pengunjung
 * tanpa akun tidak mengunduh kode panel yang tidak dibutuhkannya (plans/11 J5b).
 */
export function PublicLayout() {
  const [open, setOpen] = useState(false)
  const { pathname } = useLocation()
  const isHome = pathname === '/'
  const [homeHeaderTheme, setHomeHeaderTheme] = useState<HeaderTheme>('top')
  const status = useQuery({ queryKey: ['public', 'status'], queryFn: publicContentApi.status })
  const footerGradient = useFooterGradient(isHome)

  useEffect(() => {
    if (!isHome) return

    const updateHeaderTheme = () => {
      if (window.scrollY === 0) {
        setHomeHeaderTheme('top')
        return
      }
      const heroEl = document.getElementById('hero-band')
      if (heroEl) {
        const rect = heroEl.getBoundingClientRect()
        if (rect.height > 0 && rect.bottom <= 88) {
          setHomeHeaderTheme('light')
          return
        }
      }
      if (window.scrollY > 480) {
        setHomeHeaderTheme('light')
        return
      }
      setHomeHeaderTheme('hero-scrolled')
    }

    const frame = window.requestAnimationFrame(updateHeaderTheme)
    window.addEventListener('scroll', updateHeaderTheme, { passive: true })
    window.addEventListener('resize', updateHeaderTheme, { passive: true })
    return () => {
      window.cancelAnimationFrame(frame)
      window.removeEventListener('scroll', updateHeaderTheme)
      window.removeEventListener('resize', updateHeaderTheme)
    }
  }, [isHome])

  const headerTheme: HeaderTheme = isHome ? homeHeaderTheme : 'light'
  const isLightHeader = headerTheme === 'light'

  return (
    <div className="relative flex min-h-screen flex-col overflow-x-clip bg-[#f5f5f7] text-[#09090b]">
      <header
        className={cn(
          'z-40 w-full transition-colors duration-300',
          isHome ? 'fixed inset-x-0 top-0' : 'sticky top-0',
          headerTheme === 'top' && 'border-b border-transparent bg-transparent text-white',
          headerTheme === 'hero-scrolled' &&
            'border-b border-transparent bg-[#050507]/80 text-white backdrop-blur-md',
          headerTheme === 'light' &&
            'border-b border-zinc-200/80 bg-white/95 text-[#09090b] backdrop-blur-md',
        )}
      >
        <div className="mx-auto flex h-16 w-full max-w-6xl items-center justify-between px-4 sm:px-6">
          <Link
            to="/"
            className="flex items-center gap-2.5 rounded-md focus-visible:ring-zinc-400"
            onClick={() => setOpen(false)}
          >
            <BrandMark className="h-7 w-7 shrink-0" />
            <span
              className={cn(
                'text-sm font-semibold tracking-tight transition-colors',
                isLightHeader ? 'text-[#09090b]' : 'text-white',
              )}
            >
              Imaginary Liquidity
            </span>
          </Link>

          <nav aria-label="Navigasi situs" className="hidden items-center gap-1 md:flex">
            {NAV.map((item) =>
              item.hash ? (
                <Link
                  key={item.to}
                  to={item.to}
                  className={cn(
                    'rounded-md px-3 py-1.5 text-[13px] transition-colors',
                    isLightHeader
                      ? 'text-zinc-600 hover:text-[#09090b]'
                      : 'text-white/75 hover:text-white',
                  )}
                >
                  {item.label}
                </Link>
              ) : (
                <NavLink
                  key={item.to}
                  to={item.to}
                  className={({ isActive }) =>
                    cn(
                      'rounded-md px-3 py-1.5 text-[13px] transition-colors',
                      isLightHeader
                        ? isActive
                          ? 'bg-zinc-100 font-medium text-[#09090b]'
                          : 'text-zinc-600 hover:text-[#09090b]'
                        : isActive
                          ? 'bg-white/10 font-medium text-white'
                          : 'text-white/75 hover:text-white',
                    )
                  }
                >
                  {item.label}
                </NavLink>
              ),
            )}
          </nav>

          <div className="hidden items-center gap-3 md:flex">
            {/* Anchor biasa, bukan <Link>: panel pelanggan adalah bundel
                terpisah di /app/*. Navigasi client-side akan mencari rute itu
                di router publik dan berakhir di halaman tidak ditemukan. */}
            <a
              href="/app/login"
              className={cn(
                'rounded-md px-3 py-1.5 text-[13px] font-medium transition-colors',
                isLightHeader
                  ? 'text-zinc-700 hover:text-[#09090b]'
                  : 'text-white/85 hover:text-white',
              )}
            >
              Masuk
            </a>
            <a
              href="/app/register"
              className={cn(
                'rounded-lg px-4 py-2 text-[13px] font-semibold transition-colors',
                isLightHeader
                  ? 'bg-[#09090b] text-white hover:bg-zinc-800'
                  : 'bg-white text-[#09090b] hover:bg-zinc-200',
              )}
            >
              Mulai Sekarang
            </a>
          </div>

          <button
            type="button"
            className={cn(
              'rounded-md p-2 transition-colors md:hidden',
              isLightHeader
                ? 'text-zinc-700 hover:bg-zinc-100 hover:text-[#09090b]'
                : 'text-white/85 hover:bg-white/10 hover:text-white',
            )}
            aria-label={open ? 'Tutup menu' : 'Buka menu'}
            aria-expanded={open}
            onClick={() => setOpen((value) => !value)}
          >
            {open ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
          </button>
        </div>

        {open && (
          <nav
            aria-label="Navigasi seluler"
            className={cn(
              'flex flex-col gap-1 border-t px-4 py-4 md:hidden',
              isLightHeader
                ? 'border-zinc-200 bg-white text-[#09090b]'
                : 'border-white/10 bg-[#09090b] text-white',
            )}
          >
            {NAV.map((item) => (
              <Link
                key={item.to}
                to={item.to}
                className={cn(
                  'rounded-md px-3 py-2 text-[14px]',
                  isLightHeader
                    ? 'text-zinc-700 hover:bg-zinc-100 hover:text-[#09090b]'
                    : 'text-white/85 hover:bg-white/10 hover:text-white',
                )}
                onClick={() => setOpen(false)}
              >
                {item.label}
              </Link>
            ))}
            <div
              className={cn(
                'mt-2 flex flex-col gap-2 border-t pt-3',
                isLightHeader ? 'border-zinc-200' : 'border-white/10',
              )}
            >
              <a
                href="/app/login"
                className={cn(
                  'rounded-lg border px-3 py-2 text-center text-[13px] font-medium',
                  isLightHeader
                    ? 'border-zinc-300 text-[#09090b] hover:bg-zinc-100'
                    : 'border-white/15 text-white/90 hover:bg-white/5',
                )}
                onClick={() => setOpen(false)}
              >
                Masuk
              </a>
              <a
                href="/app/register"
                className={cn(
                  'rounded-lg px-3 py-2 text-center text-[13px] font-semibold',
                  isLightHeader
                    ? 'bg-[#09090b] text-white hover:bg-zinc-800'
                    : 'bg-white text-[#09090b] hover:bg-zinc-200',
                )}
                onClick={() => setOpen(false)}
              >
                Mulai Sekarang
              </a>
            </div>
          </nav>
        )}

        {status.data && status.data.mode !== 'normal' && (
          <div className="border-t border-zinc-800 bg-[#18181b] text-zinc-100">
            <div className="mx-auto flex w-full max-w-6xl items-center gap-2.5 px-4 py-2.5 text-[13px] sm:px-6">
              <AlertTriangle className="h-4 w-4 shrink-0 text-zinc-300" aria-hidden="true" />
              <span>{status.data.message}</span>
            </div>
          </div>
        )}
      </header>

      <main className="relative z-10 flex-1">
        <Outlet />
      </main>

      <footer className="relative z-0 text-[#09090b]">
        {/*
          Kanvas gradasi tunggal situs publik, dipasang di dalam <footer> karena
          `top` di sini relatif terhadap footer (di dalam <main> nilainya mengacu
          ke tinggi main yang ribuan piksel).

          Di beranda kanvas ditarik ke atas sampai menyentuh kaki section
          sebelumnya (padding bawah section 8 + margin kartu), lalu terus mengalir:
          mulai dari #f5f5f7, turun ke hitam pekat di belakang kartu CTA, memudar
          ke terang di zona tautan, dan kembali ke #050507 di belakang IMAGINARY
          — satu aliran tanpa patahan.

          Nilai `top` dipilih agar tepi atas kanvas tepat berimpit dengan latar
          #f5f5f7 di atasnya, sehingga tidak ada garis batas walau kadar gelapnya
          berbeda.
        */}
        <div
          aria-hidden="true"
          className={cn(
            'pointer-events-none absolute inset-x-0 bottom-0 z-0 overflow-hidden',
            isHome ? '-top-[34rem] sm:-top-[28rem]' : 'top-0',
          )}
          style={{ background: footerGradient }}
        />

        {/* Konten tautan footer: lebih rapat dan tetap berada di zona kontras tinggi */}
        <div className="relative z-10 mx-auto w-full max-w-6xl px-4 pt-10 pb-10 sm:px-6 sm:pt-12 sm:pb-12">
          <div className="grid gap-8 sm:grid-cols-2 lg:grid-cols-12">
            <div className="space-y-3 lg:col-span-2">
              <p className="text-[12px] font-semibold uppercase tracking-[0.14em] text-[#3f3f46]">
                Produk
              </p>
              <ul className="space-y-2.5 text-[13px] font-medium text-[#09090b]">
                <li>
                  <Link to="/fitur" className="transition-colors hover:text-[#52525b]">
                    Fitur Sistem
                  </Link>
                </li>
                <li>
                  <Link to="/harga" className="transition-colors hover:text-[#52525b]">
                    Paket &amp; Harga
                  </Link>
                </li>
                <li>
                  <Link to="/#cara-kerja" className="transition-colors hover:text-[#52525b]">
                    Cara Kerja ILA
                  </Link>
                </li>
                <li>
                  <Link to="/bantuan/zona-waktu" className="transition-colors hover:text-[#52525b]">
                    Sesi &amp; Zona Waktu
                  </Link>
                </li>
              </ul>
            </div>

            <div className="space-y-3 lg:col-span-2">
              <p className="text-[12px] font-semibold uppercase tracking-[0.14em] text-[#3f3f46]">
                Platform
              </p>
              <ul className="space-y-2.5 text-[13px] font-medium text-[#09090b]">
                <li>
                  <a href="/app/register" className="transition-colors hover:text-[#52525b]">
                    Daftar Uji Coba
                  </a>
                </li>
                <li>
                  <a href="/app/login" className="transition-colors hover:text-[#52525b]">
                    Masuk Dashboard
                  </a>
                </li>
                <li>
                  <Link to="/bantuan" className="transition-colors hover:text-[#52525b]">
                    Pusat Bantuan
                  </Link>
                </li>
                <li>
                  <Link to="/tentang" className="transition-colors hover:text-[#52525b]">
                    Tentang Kami
                  </Link>
                </li>
              </ul>
            </div>

            <div className="space-y-3 lg:col-span-3">
              <p className="text-[12px] font-semibold uppercase tracking-[0.14em] text-[#3f3f46]">
                Dokumen Legal
              </p>
              <ul className="space-y-2.5 text-[13px] font-medium text-[#09090b]">
                <li>
                  <Link to="/legal/tos" className="transition-colors hover:text-[#52525b]">
                    Syarat &amp; Ketentuan
                  </Link>
                </li>
                <li>
                  <Link to="/legal/privacy" className="transition-colors hover:text-[#52525b]">
                    Kebijakan Privasi
                  </Link>
                </li>
                <li>
                  <Link to="/legal/risk" className="transition-colors hover:text-[#52525b]">
                    Disclaimer Risiko Trading
                  </Link>
                </li>
                <li>
                  <Link to="/legal/refund" className="transition-colors hover:text-[#52525b]">
                    Kebijakan Pengembalian Dana
                  </Link>
                </li>
                <li>
                  <Link to="/legal/fair_use" className="transition-colors hover:text-[#52525b]">
                    Kebijakan Penggunaan Wajar
                  </Link>
                </li>
              </ul>
            </div>

            <div className="space-y-4 lg:col-span-5 lg:pl-6">
              <p className="text-[14px] leading-relaxed text-[#18181b]">
                Perangkat lunak analisis likuiditas &amp; eksekusi trading di atas akun broker milik
                Anda sendiri. Bukan nasihat keuangan; perdagangan derivatif mengandung risiko
                kehilangan modal.
              </p>
              <div className="flex flex-wrap items-center gap-4">
                <a
                  href="/app/register"
                  className="inline-flex items-center justify-center rounded-lg bg-[#09090b] px-5 py-2.5 text-[13px] font-medium text-white transition-colors hover:bg-[#27272a]"
                >
                  Mulai Sekarang
                </a>
                <Link
                  to="/legal/risk"
                  className="text-[13px] font-medium text-[#09090b] underline underline-offset-4 hover:text-[#52525b]"
                >
                  Disclaimer Risiko
                </Link>
              </div>
            </div>
          </div>
        </div>

        {/*
          Watermark raksasa IMAGINARY, full-width dan menempel tepat di tepi
          paling bawah footer tanpa celah. Angka viewBox diukur dari ink box
          Inter pada fontSize 218: ascent 156 (cap-height) + descent 3, lalu
          ditambah ruang napas 15.6 (10% cap-height) di atas huruf.
          preserveAspectRatio "none" membuat tepi kiri-kanan selalu pas penuh
          tanpa letterbox maupun ruang putih.
        */}
        <div
          aria-hidden="true"
          className="pointer-events-none relative z-10 w-full select-none overflow-hidden"
        >
          <svg
            viewBox="0 0 1200 174.6"
            className="block h-auto w-full"
            preserveAspectRatio="none"
            focusable="false"
          >
            <defs>
              <linearGradient id="footer-wordmark-grad" x1="0%" y1="0%" x2="0%" y2="100%">
                <stop offset="0%" stopColor="#f4f4f5" stopOpacity="0.92" />
                <stop offset="100%" stopColor="#a1a1aa" stopOpacity="0.5" />
              </linearGradient>
            </defs>
            <text
              x="600"
              y="171.6"
              textAnchor="middle"
              textLength="1176"
              lengthAdjust="spacing"
              fill="url(#footer-wordmark-grad)"
              style={{ fontFamily: 'inherit' }}
              fontSize="218"
              fontWeight="600"
              letterSpacing="-6"
            >
              IMAGINARY
            </text>
          </svg>
        </div>
      </footer>
    </div>
  )
}
