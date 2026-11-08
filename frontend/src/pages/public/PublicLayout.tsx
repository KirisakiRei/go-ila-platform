import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { AlertTriangle, Menu, X } from 'lucide-react'
import { Link, NavLink, Outlet } from 'react-router-dom'
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

/**
 * PublicLayout adalah kerangka situs publik.
 *
 * Halaman publik memakai bundel terpisah dari dashboard, sehingga pengunjung
 * tanpa akun tidak mengunduh kode panel yang tidak dibutuhkannya (plans/11 J5b).
 */
export function PublicLayout() {
  const [open, setOpen] = useState(false)
  const status = useQuery({ queryKey: ['public', 'status'], queryFn: publicContentApi.status })

  return (
    <div className="flex min-h-screen flex-col bg-[#f5f5f7] text-[#09090b]">
      <header className="sticky top-0 z-40 border-b border-white/[0.08] bg-[#050507]/95 text-white backdrop-blur-md">
        <div className="mx-auto flex h-16 w-full max-w-6xl items-center justify-between px-4 sm:px-6">
          <Link
            to="/"
            className="flex items-center gap-2.5 rounded-md focus-visible:ring-zinc-400"
            onClick={() => setOpen(false)}
          >
            <BrandMark className="h-7 w-7 shrink-0" />
            <span className="text-sm font-semibold tracking-tight text-white">
              Imaginary Liquidity
            </span>
          </Link>

          <nav aria-label="Navigasi situs" className="hidden items-center gap-1 md:flex">
            {NAV.map((item) =>
              item.hash ? (
                <Link
                  key={item.to}
                  to={item.to}
                  className="rounded-md px-3 py-1.5 text-[13px] text-white/75 transition-colors hover:text-white"
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
                      isActive
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
              className="rounded-md px-3 py-1.5 text-[13px] font-medium text-white/85 transition-colors hover:text-white"
            >
              Masuk
            </a>
            <a
              href="/app/register"
              className="rounded-lg bg-white px-4 py-2 text-[13px] font-semibold text-[#09090b] transition-colors hover:bg-zinc-200"
            >
              Mulai Sekarang
            </a>
          </div>

          <button
            type="button"
            className="rounded-md p-2 text-white/80 hover:bg-white/10 hover:text-white md:hidden"
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
            className="flex flex-col gap-1 border-t border-white/10 bg-[#09090b] px-4 py-4 md:hidden"
          >
            {NAV.map((item) => (
              <Link
                key={item.to}
                to={item.to}
                className="rounded-md px-3 py-2 text-[14px] text-white/85 hover:bg-white/10 hover:text-white"
                onClick={() => setOpen(false)}
              >
                {item.label}
              </Link>
            ))}
            <div className="mt-2 flex flex-col gap-2 border-t border-white/10 pt-3">
              <a
                href="/app/login"
                className="rounded-lg border border-white/15 px-3 py-2 text-center text-[13px] font-medium text-white/90 hover:bg-white/5"
                onClick={() => setOpen(false)}
              >
                Masuk
              </a>
              <a
                href="/app/register"
                className="rounded-lg bg-white px-3 py-2 text-center text-[13px] font-semibold text-[#09090b] hover:bg-zinc-200"
                onClick={() => setOpen(false)}
              >
                Mulai Sekarang
              </a>
            </div>
          </nav>
        )}
      </header>

      {status.data && status.data.mode !== 'normal' && (
        <div className="border-b border-zinc-800 bg-[#18181b] text-zinc-100">
          <div className="mx-auto flex w-full max-w-6xl items-center gap-2.5 px-4 py-2.5 text-[13px] sm:px-6">
            <AlertTriangle className="h-4 w-4 shrink-0 text-zinc-300" aria-hidden="true" />
            <span>{status.data.message}</span>
          </div>
        </div>
      )}

      <main className="flex-1">
        <Outlet />
      </main>

      <footer className="bg-[#f5f5f7] text-[#09090b]">
        {/* Bagian atas footer: tetap berada di kanvas terang agar kontras seluruh tautan > 7:1 */}
        <div
          className="w-full"
          style={{
            background:
              'radial-gradient(60% 90% at 8% 100%, rgba(161, 161, 170, 0.22) 0%, rgba(245, 245, 247, 0) 70%), #f5f5f7',
          }}
        >
          <div className="mx-auto w-full max-w-6xl px-4 pt-20 pb-14 sm:px-6">
            <div className="grid gap-10 sm:grid-cols-2 lg:grid-cols-12">
              <div className="space-y-3 lg:col-span-2">
                <p className="text-[12px] font-semibold uppercase tracking-[0.14em] text-[#52525b]">
                  Produk
                </p>
                <ul className="space-y-2.5 text-[13px] text-[#3f3f46]">
                  <li>
                    <Link to="/fitur" className="transition-colors hover:text-[#09090b]">
                      Fitur Sistem
                    </Link>
                  </li>
                  <li>
                    <Link to="/harga" className="transition-colors hover:text-[#09090b]">
                      Paket &amp; Harga
                    </Link>
                  </li>
                  <li>
                    <Link to="/#cara-kerja" className="transition-colors hover:text-[#09090b]">
                      Cara Kerja ILA
                    </Link>
                  </li>
                  <li>
                    <Link
                      to="/bantuan/zona-waktu"
                      className="transition-colors hover:text-[#09090b]"
                    >
                      Sesi &amp; Zona Waktu
                    </Link>
                  </li>
                </ul>
              </div>

              <div className="space-y-3 lg:col-span-2">
                <p className="text-[12px] font-semibold uppercase tracking-[0.14em] text-[#52525b]">
                  Platform
                </p>
                <ul className="space-y-2.5 text-[13px] text-[#3f3f46]">
                  <li>
                    <a href="/app/register" className="transition-colors hover:text-[#09090b]">
                      Daftar Uji Coba
                    </a>
                  </li>
                  <li>
                    <a href="/app/login" className="transition-colors hover:text-[#09090b]">
                      Masuk Dashboard
                    </a>
                  </li>
                  <li>
                    <Link to="/bantuan" className="transition-colors hover:text-[#09090b]">
                      Pusat Bantuan
                    </Link>
                  </li>
                  <li>
                    <Link to="/tentang" className="transition-colors hover:text-[#09090b]">
                      Tentang Kami
                    </Link>
                  </li>
                </ul>
              </div>

              <div className="space-y-3 lg:col-span-3">
                <p className="text-[12px] font-semibold uppercase tracking-[0.14em] text-[#52525b]">
                  Dokumen Legal
                </p>
                <ul className="space-y-2.5 text-[13px] text-[#3f3f46]">
                  <li>
                    <Link to="/legal/tos" className="transition-colors hover:text-[#09090b]">
                      Syarat &amp; Ketentuan
                    </Link>
                  </li>
                  <li>
                    <Link to="/legal/privacy" className="transition-colors hover:text-[#09090b]">
                      Kebijakan Privasi
                    </Link>
                  </li>
                  <li>
                    <Link to="/legal/risk" className="transition-colors hover:text-[#09090b]">
                      Disclaimer Risiko Trading
                    </Link>
                  </li>
                  <li>
                    <Link to="/legal/refund" className="transition-colors hover:text-[#09090b]">
                      Kebijakan Pengembalian Dana
                    </Link>
                  </li>
                  <li>
                    <Link to="/legal/fair_use" className="transition-colors hover:text-[#09090b]">
                      Kebijakan Penggunaan Wajar
                    </Link>
                  </li>
                </ul>
              </div>

              <div className="space-y-4 lg:col-span-5 lg:pl-6">
                <p className="text-[14px] leading-relaxed text-[#27272a]">
                  Perangkat lunak analisis likuiditas &amp; eksekusi trading di atas akun broker
                  milik Anda sendiri. Bukan nasihat keuangan; perdagangan derivatif mengandung
                  risiko kehilangan modal.
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
                    className="text-[13px] font-medium text-[#18181b] underline underline-offset-4 hover:text-[#52525b]"
                  >
                    Disclaimer Risiko
                  </Link>
                </div>
              </div>
            </div>

            <div className="mt-12 flex flex-col items-start justify-between gap-2 border-t border-zinc-200/80 pt-6 text-[12px] text-[#52525b] sm:flex-row sm:items-center">
              <span>© {new Date().getFullYear()} Imaginary Liquidity · rekavia.com</span>
              <span>Akun tetap di broker Anda · Bukan nasihat keuangan</span>
            </div>
          </div>
        </div>

        {/* Transisi diagonal "/" halus menuju hitam pekat + watermark raksasa rapi */}
        <div
          className="relative overflow-hidden pt-28 sm:pt-40"
          style={{
            background:
              'linear-gradient(190deg, #f5f5f7 0%, #e4e4e7 20%, #71717a 42%, #18181b 60%, #050507 76%)',
          }}
        >
          <div
            aria-hidden="true"
            className="pointer-events-none select-none overflow-hidden whitespace-nowrap px-2 text-center font-semibold tracking-[-0.045em] text-white/90"
            style={{
              fontSize: 'clamp(4.1rem, 19vw, 17rem)',
              lineHeight: 0.78,
              marginBottom: '-0.06em',
            }}
          >
            Imaginary
          </div>
        </div>
      </footer>
    </div>
  )
}
