import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { AlertTriangle, Menu, X } from 'lucide-react'
import { Link, NavLink, Outlet, useLocation } from 'react-router-dom'
import { publicContentApi } from '@/lib/system-api'
import { cn } from '@/lib/utils'
import { BrandMark } from './components/LandingArtwork'

/** Navigasi utama situs publik, mengikuti alur evaluasi calon pengguna. */
const NAV = [
  { to: '/fitur', label: 'Fitur' },
  { to: '/harga', label: 'Harga' },
  { to: '/#cara-kerja', label: 'Cara Kerja', hash: true },
  { to: '/bantuan', label: 'Bantuan' },
  { to: '/tentang', label: 'Tentang' },
]

type HeaderTheme = 'top' | 'hero-scrolled' | 'light'

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
        {/* Kanvas gradasi tunggal yang naik ke belakang Banner CTA (Hero 8) agar menyatu tanpa patahan */}
        <div
          aria-hidden="true"
          className={cn(
            'pointer-events-none absolute inset-x-0 bottom-0 z-0 overflow-hidden',
            isHome ? '-top-[32rem]' : 'top-0',
          )}
          style={{
            background: [
              'linear-gradient(182deg, rgba(5,5,7,0) 0%, rgba(5,5,7,0) 62%, rgba(24,24,27,0.78) 78%, #050507 90%)',
              'radial-gradient(50% 34% at 12% 48%, rgba(24,24,27,0.3) 0%, rgba(82,82,91,0.14) 48%, rgba(255,255,255,0) 100%)',
              'linear-gradient(194deg, #ffffff 0%, #ffffff 42%, #e4e4e7 56%, #d4d4d8 69%, #52525b 82%, #050507 94%)',
            ].join(', '),
          }}
        />

        {/* Konten tautan footer: berada di zona atas terang dengan kontras tinggi */}
        <div className="relative z-10 mx-auto w-full max-w-6xl px-4 pt-16 pb-24 sm:px-6 sm:pt-20 sm:pb-28">
          <div className="grid gap-10 sm:grid-cols-2 lg:grid-cols-12">
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

          <div className="mt-12 flex flex-col items-start justify-between gap-2 border-t border-zinc-400/35 pt-6 text-[12px] text-[#27272a] sm:flex-row sm:items-center">
            <span>© {new Date().getFullYear()} Imaginary Liquidity · rekavia.com</span>
            <span>Akun tetap di broker Anda · Bukan nasihat keuangan</span>
          </div>
        </div>

        {/* Watermark raksasa di atas dasar hitam paling bawah */}
        <div
          aria-hidden="true"
          className="pointer-events-none relative z-10 select-none overflow-hidden whitespace-nowrap px-2 pt-4 text-center font-semibold tracking-[-0.045em] text-white/90"
          style={{
            fontSize: 'clamp(4.1rem, 19vw, 17rem)',
            lineHeight: 0.78,
            marginBottom: '-0.06em',
          }}
        >
          Imaginary
        </div>
      </footer>
    </div>
  )
}
