import { useEffect, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { AlertTriangle, Menu, X } from 'lucide-react'
import { Link, NavLink, Outlet, useLocation } from 'react-router-dom'
import { publicContentApi } from '@/lib/system-api'
import { Logo } from '@/components/brand/Logo'
import { SocialLinksRow } from '@/pages/public/components/SocialLinksRow'
import { cn } from '@/lib/utils'


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
/** Navigasi utama situs publik, mengikuti alur evaluasi calon pengguna. */
const NAV = [
  { to: '/', label: 'Beranda' },
  { to: '/harga', label: 'Harga' },
  { to: '/fitur', label: 'Fitur' },
  { to: '/bantuan', label: 'Bantuan' },
  { to: '/tentang', label: 'Tentang' },
]

type HeaderTheme = 'top' | 'hero-scrolled' | 'light'

const WATERMARK_LETTERS = ['I', 'M', 'A', 'G', 'I', 'N', 'A', 'R', 'Y'] as const

/** Opasitas awal tiap huruf sebelum animasi gulir menyentuhnya. */
const BASE_OPACITY = 0.08

/** clampProgress membatasi nilai ke rentang 0..1. */
function clampProgress(value: number): number {
  if (Number.isNaN(value)) return 0
  return Math.max(0, Math.min(1, value))
}

/**
 * Watermark footer IMAGINARY dengan animasi fade per huruf berdasarkan scroll.
 *
 * Progress ditakar terhadap sisa gulir dokumen, bukan terhadap posisi elemen di
 * viewport. Alasannya: watermark menempel di dasar halaman, sehingga jarak
 * pandangnya sangat pendek dan rumus "posisi elemen" berhenti di sekitar 20%
 * ketika gulir sudah mentok — itulah sebabnya huruf terakhir tidak pernah
 * mencapai hitam pekat. Dengan menakar sisa gulir, progress dipastikan tuntas
 * (1.0) tepat saat halaman mencapai dasar, dan memudar kembali saat digulir naik.
 */
function FooterWatermark() {
  const containerRef = useRef<HTMLDivElement>(null)
  const [opacities, setOpacities] = useState<number[]>(() => WATERMARK_LETTERS.map(() => BASE_OPACITY))

  useEffect(() => {
    const handleScroll = () => {
      if (!containerRef.current) return
      const doc = document.documentElement
      const viewport = window.innerHeight
      const maxScroll = doc.scrollHeight - viewport

      // Halaman yang tidak dapat digulir: tampilkan watermark sepenuhnya.
      if (maxScroll <= 0) {
        setOpacities(WATERMARK_LETTERS.map(() => 1))
        return
      }

      // Lintasan animasi selebar satu layar terakhir. Pada titik ini footer
      // sudah mulai terlihat dari bawah, lalu animasi tuntas tepat di dasar.
      const track = Math.min(maxScroll, viewport * 0.85)
      const start = maxScroll - track
      const progress = clampProgress((window.scrollY - start) / track)

      setOpacities(
        WATERMARK_LETTERS.map((_, index) => {
          const from = (index / WATERMARK_LETTERS.length) * 0.75
          const to = from + 0.25
          if (progress <= from) return BASE_OPACITY
          if (progress >= to) return 1
          return BASE_OPACITY + ((1 - BASE_OPACITY) * (progress - from)) / (to - from)
        }),
      )
    }

    handleScroll()
    window.addEventListener('scroll', handleScroll, { passive: true })
    window.addEventListener('resize', handleScroll, { passive: true })
    return () => {
      window.removeEventListener('scroll', handleScroll)
      window.removeEventListener('resize', handleScroll)
    }
  }, [])

  return (
    <div
      ref={containerRef}
      aria-hidden="true"
      className="pointer-events-none relative z-10 w-full select-none overflow-hidden"
    >
      {/* Sedikit digeser turun agar tepi bawah huruf IMAGINARY terpotong tipis & rapi */}
      <svg
        viewBox="0 0 1200 174.6"
        className="block h-auto w-full translate-y-[12%] sm:translate-y-[14%]"
        preserveAspectRatio="none"
        focusable="false"
      >
        <text
          x="600"
          y="171.6"
          textAnchor="middle"
          textLength="1176"
          lengthAdjust="spacing"
          fill="#09090b"
          style={{ fontFamily: 'inherit' }}
          fontSize="218"
          fontWeight="600"
          letterSpacing="-6"
        >
          {WATERMARK_LETTERS.map((char, index) => (
            <tspan
              key={index}
              style={{
                opacity: opacities[index] ?? BASE_OPACITY,
                transition: 'opacity 0.2s cubic-bezier(0.16, 1, 0.3, 1)',
              }}
            >
              {char}
            </tspan>
          ))}
        </text>
      </svg>
    </div>
  )
}

/**
 * PublicLayout adalah kerangka situs publik.
 *
 * Halaman publik memakai bundel terpisah dari dashboard, sehingga pengunjung
 * tanpa akun tidak mengunduh kode panel yang tidak dibutuhkannya (plans/11 J5b).
 */
/**
 * Rute yang benar-benar dilayani situs publik.
 *
 * Dipakai untuk membedakan "halaman yang tidak ada" dari halaman sah. Daftar
 * ini harus ikut diperbarui setiap ada rute publik baru: kalau tertinggal,
 * halaman baru tersebut akan kehilangan footer tanpa ada yang menyadarinya.
 * Pasangannya ada di PublicApp.tsx.
 */
const RUTE_PUBLIK = [
  '/',
  '/fitur',
  '/harga',
  '/tentang',
  '/bantuan',
  '/legal',
  '/lupa-sandi',
  '/reset-password',
  '/verifikasi',
  '/daftar',
] as const

/** ruteDikenal memeriksa pathname terhadap daftar rute publik, termasuk anaknya. */
function ruteDikenal(pathname: string): boolean {
  const bersih = pathname !== '/' && pathname.endsWith('/') ? pathname.slice(0, -1) : pathname
  return RUTE_PUBLIK.some((rute) => bersih === rute || bersih.startsWith(`${rute}/`))
}

export function PublicLayout() {
  const [open, setOpen] = useState(false)
  const { pathname } = useLocation()
  const isHome = pathname === '/'
  // Halaman 404 tidak menampilkan footer. Footer penuh tautan di bawah pesan
  // "tidak ditemukan" mengalihkan perhatian dari satu-satunya tindakan yang
  // berguna, yaitu kembali ke beranda.
  const tampilkanFooter = ruteDikenal(pathname)
  const [homeHeaderTheme, setHomeHeaderTheme] = useState<HeaderTheme>('top')
  const status = useQuery({ queryKey: ['public', 'status'], queryFn: publicContentApi.status })
  // Default fail-closed: harga dan pendaftaran mandiri disembunyikan saat invite-only aktif.
  const inviteOnly = status.data ? status.data.invite_only !== false : true
  const navItems = inviteOnly ? NAV.filter((item) => item.to !== '/harga') : NAV
  const socialLinks = status.data?.social_links

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
        if (rect.height > 0 && rect.bottom > 88) {
          setHomeHeaderTheme('hero-scrolled')
          return
        }
      }
      // Saat memasuki zona obsidian bawah (Hero 5 & 6), navbar kembali gelap;
      // begitu melewati batas bawahnya menuju footer terang, navbar kembali putih.
      const darkBandEl = document.getElementById('dark-cta-band')
      if (darkBandEl) {
        const darkRect = darkBandEl.getBoundingClientRect()
        const edgeMargin = Math.min(120, Math.round(window.innerHeight * 0.14))
        if (
          darkRect.height > 0 &&
          darkRect.top <= edgeMargin &&
          darkRect.bottom >= -edgeMargin
        ) {
          setHomeHeaderTheme('hero-scrolled')
          return
        }
      }
      if (heroEl && heroEl.getBoundingClientRect().height > 0) {
        setHomeHeaderTheme('light')
        return
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
            {/* Logo gelap & terang ditumpuk agar cross-fade mulus saat navbar berubah tema */}
            <span className="relative inline-flex h-8 shrink-0 items-center">
              <Logo
                theme="light"
                className={cn(
                  'h-8 w-auto transition-opacity duration-300 ease-out',
                  isLightHeader ? 'opacity-100' : 'opacity-0',
                )}
              />
              <Logo
                theme="dark"
                className={cn(
                  'pointer-events-none absolute inset-0 h-8 w-auto transition-opacity duration-300 ease-out',
                  isLightHeader ? 'opacity-0' : 'opacity-100',
                )}
              />
            </span>
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
            {navItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.to === '/'}
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
            ))}
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
            {inviteOnly ? (
              <Link
                to="/#ajukan-akses"
                className={cn(
                  'rounded-lg px-4 py-2 text-[13px] font-semibold press',
                  isLightHeader
                    ? 'bg-[#09090b] text-white hover:bg-zinc-800'
                    : 'bg-white text-[#09090b] hover:bg-zinc-200',
                )}
              >
                Ajukan Akses
              </Link>
            ) : (
              <a
                href="/app/register"
                className={cn(
                  'rounded-lg px-4 py-2 text-[13px] font-semibold press',
                  isLightHeader
                    ? 'bg-[#09090b] text-white hover:bg-zinc-800'
                    : 'bg-white text-[#09090b] hover:bg-zinc-200',
                )}
              >
                Mulai Sekarang
              </a>
            )}
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
            {navItems.map((item) => (
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
              {inviteOnly ? (
                <Link
                  to="/#ajukan-akses"
                  className={cn(
                    'rounded-lg px-3 py-2 text-center text-[13px] font-semibold',
                    isLightHeader
                      ? 'bg-[#09090b] text-white hover:bg-zinc-800'
                      : 'bg-white text-[#09090b] hover:bg-zinc-200',
                  )}
                  onClick={() => setOpen(false)}
                >
                  Ajukan Akses
                </Link>
              ) : (
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
              )}
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

      {tampilkanFooter && (
        <footer
          className={cn(
            'relative z-0 flex flex-col justify-between bg-[#f5f5f7] text-[#09090b] lg:min-h-[82vh]',
            !isHome && 'border-t border-zinc-200/80',
          )}
        >
          {/* Konten tautan footer: rata tengah, rapi & proporsional di desktop maupun mobile */}
          <div className="relative z-10 mx-auto w-full max-w-5xl px-4 pb-10 pt-10 text-center sm:px-6 sm:pb-14 sm:pt-14 lg:pt-20">
            {/* Header brand besar rata tengah + tautan sosmed terpusat */}
            <div className="mb-10 flex flex-col items-center gap-5 border-b border-zinc-200/80 pb-8 sm:mb-12 sm:pb-10">
              <div className="flex items-center justify-center gap-3.5">
                <img
                  src="/brand/logo-light-64.png"
                  srcSet="/brand/logo-light-64.png 1x, /brand/logo-light-128.png 2x"
                  alt="Imaginary Liquidity"
                  className="h-10 w-auto object-contain sm:h-12"
                  decoding="async"
                />
                <p className="text-2xl font-semibold tracking-tight text-[#09090b] sm:text-3xl">
                  Imaginary Liquidity
                </p>
              </div>
              <SocialLinksRow links={socialLinks} variant="light" showLabel className="justify-center" />
            </div>

            {/* 3 kolom tautan rata tengah yang seimbang di mobile & desktop */}
            <div className="mx-auto grid max-w-3xl grid-cols-2 gap-x-6 gap-y-9 text-center sm:grid-cols-3 sm:gap-x-12 lg:gap-x-16">
              <div className="space-y-3.5">
                <p className="font-mono text-[11px] font-semibold uppercase tracking-[0.16em] text-[#71717a]">
                  Produk
                </p>
                <ul className="space-y-2.5 text-[13px] font-medium text-[#18181b]">
                  <li>
                    <Link to="/fitur" className="transition-colors hover:text-[#52525b]">
                      Fitur Platform
                    </Link>
                  </li>
                  {!inviteOnly && (
                    <li>
                      <Link to="/harga" className="transition-colors hover:text-[#52525b]">
                        Paket &amp; Harga
                      </Link>
                    </li>
                  )}
                  <li>
                    <Link to="/#cara-kerja" className="transition-colors hover:text-[#52525b]">
                      Cara Kerja
                    </Link>
                  </li>
                  <li>
                    <Link to="/bantuan/zona-waktu" className="transition-colors hover:text-[#52525b]">
                      Sesi &amp; Zona Waktu
                    </Link>
                  </li>
                </ul>
              </div>

              <div className="space-y-3.5">
                <p className="font-mono text-[11px] font-semibold uppercase tracking-[0.16em] text-[#71717a]">
                  Platform
                </p>
                <ul className="space-y-2.5 text-[13px] font-medium text-[#18181b]">
                  <li>
                    {inviteOnly ? (
                      <Link to="/#ajukan-akses" className="transition-colors hover:text-[#52525b]">
                        Ajukan Akses
                      </Link>
                    ) : (
                      <a href="/app/register" className="transition-colors hover:text-[#52525b]">
                        Mulai Sekarang
                      </a>
                    )}
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

              <div className="col-span-2 space-y-3.5 sm:col-span-1">
                <p className="font-mono text-[11px] font-semibold uppercase tracking-[0.16em] text-[#71717a]">
                  Dokumen Legal
                </p>
                <ul className="space-y-2.5 text-[13px] font-medium text-[#18181b]">
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
            </div>
          </div>

          {/* Watermark raksasa IMAGINARY, full-width dengan tepi bawah terpotong rapi */}
          <FooterWatermark />
        </footer>
      )}
    </div>
  )
}
