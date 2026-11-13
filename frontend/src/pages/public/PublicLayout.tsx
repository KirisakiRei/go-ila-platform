import { useEffect, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { AlertTriangle, Menu, X } from 'lucide-react'
import { Link, NavLink, Outlet, useLocation } from 'react-router-dom'
import { publicContentApi } from '@/lib/system-api'
import { Logo } from '@/components/brand/Logo'
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
      <svg
        viewBox="0 0 1200 174.6"
        className="block h-auto w-full"
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
            {/* Logo bersama: berkas aslinya 660 KB, diganti turunan 64/128px
                agar tidak diunduh ulang sebagai gambar raksasa setiap halaman. */}
            <Logo
              className={cn(
                'h-8 w-auto transition-opacity duration-300',
                isLightHeader ? 'opacity-100' : 'opacity-0',
              )}
            />
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
            {NAV.map((item) => (
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

      {tampilkanFooter && (
        <footer className="relative z-0 border-t border-zinc-200/80 bg-[#f5f5f7] text-[#09090b]">
        {/* Konten tautan footer: lebih rapat dan tetap berada di zona kontras tinggi */}
        <div className="relative z-10 mx-auto w-full max-w-6xl px-4 pt-10 pb-10 sm:px-6 sm:pt-12 sm:pb-12">
          {/* Logo & Brand Header Footer */}
          <div className="mb-8 flex items-center gap-3.5 border-b border-zinc-200/80 pb-6">
            <img
              src="/brand/logo-light-64.png"
              srcSet="/brand/logo-light-64.png 1x, /brand/logo-light-128.png 2x"
              alt="Imaginary Liquidity"
              className="h-10 w-auto object-contain"
              decoding="async"
            />
            <div>
              <p className="text-base font-bold tracking-tight text-[#09090b]">Imaginary Liquidity</p>
              <p className="text-xs text-zinc-500">Analisis Likuiditas &amp; Eksekusi Trading Presisi</p>
            </div>
          </div>

          <div className="grid gap-8 sm:grid-cols-2 lg:grid-cols-12">
            <div className="space-y-3 lg:col-span-2">
              <p className="text-[12px] font-semibold uppercase tracking-[0.14em] text-[#3f3f46]">
                Produk
              </p>
              <ul className="space-y-2.5 text-[13px] font-medium text-[#09090b]">
                <li>
                  <Link to="/fitur" className="transition-colors hover:text-[#52525b]">
                    Fitur Platform
                  </Link>
                </li>
                <li>
                  <Link to="/harga" className="transition-colors hover:text-[#52525b]">
                    Paket &amp; Harga
                  </Link>
                </li>
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

            <div className="space-y-3 lg:col-span-2">
              <p className="text-[12px] font-semibold uppercase tracking-[0.14em] text-[#3f3f46]">
                Platform
              </p>
              <ul className="space-y-2.5 text-[13px] font-medium text-[#09090b]">
                <li>
                  <a href="/app/register" className="transition-colors hover:text-[#52525b]">
                    Mulai Trial
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
                Platform analisis likuiditas dan eksekusi trading langsung di broker Anda. Bukan penasihat keuangan; instrumen derivatif memiliki risiko modal.
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

        {/* Watermark raksasa IMAGINARY, full-width tanpa celah dengan animasi scroll */}
        <FooterWatermark />
        </footer>
      )}
    </div>
  )
}
