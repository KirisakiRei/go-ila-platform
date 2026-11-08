import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Menu, X } from 'lucide-react'
import { Link, NavLink, Outlet } from 'react-router-dom'
import { publicContentApi } from '@/lib/system-api'
import { cn } from '@/lib/utils'
import { Alert } from '@/components/ui'

/** Navigasi situs publik. Urutannya mengikuti perjalanan calon pengguna. */
const NAV = [
  { to: '/fitur', label: 'Fitur' },
  { to: '/harga', label: 'Harga' },
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
    <div className="flex min-h-screen flex-col">
      <header className="sticky top-0 z-10 border-b border-border bg-background/90 backdrop-blur">
        <div className="mx-auto flex h-14 w-full max-w-6xl items-center justify-between px-4">
          <Link to="/" className="flex items-baseline gap-2">
            <span className="text-sm font-bold tracking-tight">Imaginary Liquidity</span>
            <span className="hidden text-[10px] uppercase tracking-[0.2em] text-muted-foreground sm:inline">
              ILA Bot
            </span>
          </Link>

          <nav aria-label="Navigasi situs" className="hidden items-center gap-1 md:flex">
            {NAV.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                className={({ isActive }) =>
                  cn(
                    'rounded-md px-2.5 py-1.5 text-[13px]',
                    isActive ? 'bg-secondary font-medium' : 'text-muted-foreground hover:text-foreground',
                  )
                }
              >
                {item.label}
              </NavLink>
            ))}
          </nav>

          <div className="hidden items-center gap-2 md:flex">
            <Link to="/app/login" className="rounded-md px-2.5 py-1.5 text-[13px] text-muted-foreground hover:text-foreground">
              Masuk
            </Link>
            <Link to="/app/register" className="rounded-md bg-foreground px-3 py-1.5 text-[13px] font-medium text-background">
              Daftar
            </Link>
          </div>

          <button
            type="button"
            className="rounded-md p-1.5 md:hidden"
            aria-label={open ? 'Tutup menu' : 'Buka menu'}
            onClick={() => setOpen((value) => !value)}
          >
            {open ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
          </button>
        </div>

        {open && (
          <nav aria-label="Navigasi situs" className="flex flex-col gap-1 border-t border-border p-3 md:hidden">
            {NAV.map((item) => (
              <Link key={item.to} to={item.to} className="rounded-md px-2 py-1.5 text-[13px]" onClick={() => setOpen(false)}>
                {item.label}
              </Link>
            ))}
            <Link to="/app/login" className="rounded-md px-2 py-1.5 text-[13px]" onClick={() => setOpen(false)}>
              Masuk
            </Link>
            <Link
              to="/app/register"
              className="rounded-md bg-foreground px-2 py-1.5 text-center text-[13px] font-medium text-background"
              onClick={() => setOpen(false)}
            >
              Daftar
            </Link>
          </nav>
        )}
      </header>

      {status.data && status.data.mode !== 'normal' && (
        <div className="mx-auto w-full max-w-6xl px-4 pt-4">
          <Alert variant="warning">{status.data.message}</Alert>
        </div>
      )}

      <main className="flex-1">
        <Outlet />
      </main>

      <footer className="border-t border-border">
        <div className="mx-auto flex w-full max-w-6xl flex-col gap-4 px-4 py-8 text-[12px] text-muted-foreground sm:flex-row sm:justify-between">
          <div className="max-w-md space-y-2">
            <p className="font-medium text-foreground">Imaginary Liquidity</p>
            <p>
              Perangkat lunak analisis dan eksekusi trading. Bukan nasihat keuangan. Trading memiliki
              risiko kehilangan modal; lakukan riset sendiri.
            </p>
          </div>
          <div className="flex flex-col gap-1.5">
            <Link to="/legal/kebijakan-privasi" className="hover:text-foreground">
              Kebijakan Privasi
            </Link>
            <Link to="/legal/syarat-layanan" className="hover:text-foreground">
              Syarat Layanan
            </Link>
            <Link to="/legal/disclaimer-risiko" className="hover:text-foreground">
              Disclaimer Risiko
            </Link>
            <Link to="/bantuan" className="hover:text-foreground">
              Pusat Bantuan
            </Link>
          </div>
        </div>
      </footer>
    </div>
  )
}
