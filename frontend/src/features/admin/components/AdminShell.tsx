import type { ReactNode } from 'react'
import { NavLink } from 'react-router-dom'
import {
  Activity,
  BarChart3,
  FileClock,
  FileText,
  LayoutDashboard,
  Percent,
  Settings2,
  ShieldAlert,
  ShieldCheck,
  Users,
  Wallet,
} from 'lucide-react'
import { cn } from '@/lib/utils'

/** Menu panel admin. Urutannya mengikuti alur kerja: orang → uang → jejak. */
const NAV = [
  { to: '/admin', label: 'Ringkasan', icon: LayoutDashboard, end: true },
  { to: '/admin/pengguna', label: 'Pengguna', icon: Users, end: false },
  { to: '/admin/paket', label: 'Paket & Harga', icon: Wallet, end: false },
  { to: '/admin/kupon', label: 'Kupon', icon: Percent, end: false },
  { to: '/admin/laporan', label: 'Laporan', icon: BarChart3, end: false },
  { to: '/admin/audit', label: 'Jejak Audit', icon: FileClock, end: false },
  { to: '/admin/konten', label: 'Konten', icon: FileText, end: false },
  { to: '/admin/kesehatan', label: 'Kesehatan', icon: Activity, end: false },
  { to: '/admin/pengaturan-sistem', label: 'Flag Sistem', icon: Settings2, end: false },
  { to: '/admin/pemeliharaan', label: 'Pemeliharaan', icon: ShieldAlert, end: false },
]

/**
 * AdminShell adalah kerangka panel admin. Terpisah dari AppShell pengguna
 * karena sesi admin memakai pintu sendiri dan tidak saling bergantian.
 */
export function AdminShell({ children, actorEmail }: { children: ReactNode; actorEmail?: string }) {
  return (
    <div className="flex min-h-full flex-col lg:flex-row">
      <aside className="shrink-0 border-b border-border lg:sticky lg:top-0 lg:h-screen lg:w-60 lg:border-b-0 lg:border-r">
        <div className="flex h-14 items-center gap-2 border-b border-border px-4">
          <ShieldCheck className="h-4 w-4" aria-hidden />
          <span className="text-sm font-bold tracking-tight">ILA</span>
          <span className="text-[10px] uppercase tracking-[0.2em] text-muted-foreground">Admin</span>
        </div>

        <nav aria-label="Navigasi admin" className="flex gap-1 overflow-x-auto p-2 lg:flex-col">
          {NAV.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                cn(
                  'flex items-center gap-2 whitespace-nowrap rounded-md px-3 py-2 text-[13px] transition-colors',
                  isActive
                    ? 'bg-muted font-medium text-foreground'
                    : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground',
                )
              }
            >
              <item.icon className="h-4 w-4 shrink-0" aria-hidden />
              {item.label}
            </NavLink>
          ))}
        </nav>

        {actorEmail && (
          <div className="hidden border-t border-border p-3 lg:block">
            <p className="truncate text-[11px] text-muted-foreground">{actorEmail}</p>
          </div>
        )}
      </aside>

      <main className="min-w-0 flex-1">
        <div className="mx-auto flex max-w-6xl flex-col gap-5 p-4 sm:p-6">{children}</div>
      </main>
    </div>
  )
}
