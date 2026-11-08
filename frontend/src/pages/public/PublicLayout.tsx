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
    <div className="flex min-h-screen flex-col bg-[#f3f3f6] text-[#0f0b1a]">
      <header className="sticky top-0 z-40 border-b border-white/[0.08] bg-[#06040b]/95 text-white backdrop-blur-md">
        <div className="mx-auto flex h-16 w-full max-w-6xl items-center justify-between px-4 sm:px-6">
          <Link
            to="/"
            className="flex items-center gap-2.5 rounded-md focus-visible:ring-purple-400"
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
              className="rounded-lg bg-[#5b21b6] px-4 py-2 text-[13px] font-medium text-white shadow-sm shadow-purple-950/50 transition-colors hover:bg-[#6d28d9]"
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
            className="flex flex-col gap-1 border-t border-white/10 bg-[#080510] px-4 py-4 md:hidden"
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
                className="rounded-lg bg-[#5b21b6] px-3 py-2 text-center text-[13px] font-medium text-white hover:bg-[#6d28d9]"
                onClick={() => setOpen(false)}
              >
                Mulai Sekarang
              </a>
            </div>
          </nav>
        )}
      </header>

      {status.data && status.data.mode !== 'normal' && (
        <div className="border-b border-amber-500/30 bg-[#1a0d2e] text-amber-100">
          <div className="mx-auto flex w-full max-w-6xl items-center gap-2.5 px-4 py-2.5 text-[13px] sm:px-6">
            <AlertTriangle className="h-4 w-4 shrink-0 text-amber-300" aria-hidden="true" />
            <span>{status.data.message}</span>
          </div>
        </div>
      )}

      <main className="flex-1">
        <Outlet />
      </main>

      <footer className="bg-[#f3f3f6] text-[#18112b]">
        {/* Bagian atas footer: tautan & CTA tetap di latar terang agar kontras > 7:1 */}
        <div className="mx-auto w-full max-w-6xl px-4 pt-16 pb-10 sm:px-6">
          <div className="grid gap-10 sm:grid-cols-2 lg:grid-cols-12">
            <div className="space-y-3 lg:col-span-2">
              <p className="text-[12px] font-semibold uppercase tracking-[0.14em] text-[#4c1d95]">
                Produk
              </p>
              <ul className="space-y-2 text-[13px] text-[#3f3756]">
                <li>
                  <Link to="/fitur" className="transition-colors hover:text-[#0f0b1a]">
                    Fitur Sistem
                  </Link>
                </li>
                <li>
                  <Link to="/harga" className="transition-colors hover:text-[#0f0b1a]">
                    Paket &amp; Harga
                  </Link>
                </li>
                <li>
                  <Link to="/#cara-kerja" className="transition-colors hover:text-[#0f0b1a]">
                    Cara Kerja ILA
                  </Link>
                </li>
                <li>
                  <Link to="/bantuan/zona-waktu" className="transition-colors hover:text-[#0f0b1a]">
                    Sesi &amp; Zona Waktu
                  </Link>
                </li>
              </ul>
            </div>

            <div className="space-y-3 lg:col-span-2">
              <p className="text-[12px] font-semibold uppercase tracking-[0.14em] text-[#4c1d95]">
                Platform
              </p>
              <ul className="space-y-2 text-[13px] text-[#3f3756]">
                <li>
                  <a href="/app/register" className="transition-colors hover:text-[#0f0b1a]">
                    Daftar Uji Coba
                  </a>
                </li>
                <li>
                  <a href="/app/login" className="transition-colors hover:text-[#0f0b1a]">
                    Masuk Dashboard
                  </a>
                </li>
                <li>
                  <Link to="/bantuan" className="transition-colors hover:text-[#0f0b1a]">
                    Pusat Bantuan
                  </Link>
                </li>
                <li>
                  <Link to="/tentang" className="transition-colors hover:text-[#0f0b1a]">
                    Tentang Kami
                  </Link>
                </li>
              </ul>
            </div>

            <div className="space-y-3 lg:col-span-3">
              <p className="text-[12px] font-semibold uppercase tracking-[0.14em] text-[#4c1d95]">
                Dokumen Legal
              </p>
              <ul className="space-y-2 text-[13px] text-[#3f3756]">
                <li>
                  <Link to="/legal/tos" className="transition-colors hover:text-[#0f0b1a]">
                    Syarat &amp; Ketentuan
                  </Link>
                </li>
                <li>
                  <Link to="/legal/privacy" className="transition-colors hover:text-[#0f0b1a]">
                    Kebijakan Privasi
                  </Link>
                </li>
                <li>
                  <Link to="/legal/risk" className="transition-colors hover:text-[#0f0b1a]">
                    Disclaimer Risiko Trading
                  </Link>
                </li>
                <li>
                  <Link to="/legal/refund" className="transition-colors hover:text-[#0f0b1a]">
                    Kebijakan Pengembalian Dana
                  </Link>
                </li>
                <li>
                  <Link to="/legal/fair_use" className="transition-colors hover:text-[#0f0b1a]">
                    Kebijakan Penggunaan Wajar
                  </Link>
                </li>
              </ul>
            </div>

            <div className="space-y-4 lg:col-span-5 lg:pl-6">
              <p className="text-[14px] leading-relaxed text-[#2d2542]">
                Perangkat lunak analisis likuiditas &amp; eksekusi trading di atas akun broker milik
                Anda sendiri. Bukan nasihat keuangan; perdagangan derivatif mengandung risiko
                kehilangan modal.
              </p>
              <div className="flex flex-wrap items-center gap-3">
                <a
                  href="/app/register"
                  className="inline-flex items-center justify-center rounded-lg bg-[#160a2c] px-5 py-2.5 text-[13px] font-medium text-white transition-colors hover:bg-[#3b0764]"
                >
                  Mulai Sekarang
                </a>
                <Link
                  to="/legal/risk"
                  className="text-[13px] font-medium text-[#4c1d95] underline underline-offset-4 hover:text-[#2e1065]"
                >
                  Disclaimer Risiko
                </Link>
              </div>
            </div>
          </div>
        </div>

        {/* Aurora dark-purple transisi ke watermark raksasa terpotong di tepi bawah */}
        <div
          className="relative overflow-hidden pt-10"
          style={{
            background:
              'linear-gradient(180deg, #f3f3f6 0%, #ddd2f7 26%, #5b21b6 62%, #3b0764 84%, #190536 100%)',
          }}
        >
          <div className="mx-auto flex w-full max-w-6xl flex-col items-start justify-between gap-2 px-4 pb-4 text-[12px] text-purple-100/90 sm:flex-row sm:items-center sm:px-6">
            <span>© {new Date().getFullYear()} Imaginary Liquidity · rekavia.com</span>
            <span>Akun tetap di broker Anda · Bukan nasihat keuangan</span>
          </div>

          <div
            aria-hidden="true"
            className="pointer-events-none select-none overflow-hidden text-center font-bold tracking-tighter text-white/[0.16]"
            style={{
              fontSize: 'clamp(4.5rem, 17vw, 13rem)',
              lineHeight: 0.78,
              transform: 'translateY(14%)',
            }}
          >
            Imaginary
          </div>
        </div>
      </footer>
    </div>
  )
}
