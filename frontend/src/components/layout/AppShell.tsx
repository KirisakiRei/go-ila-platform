import type { ReactNode } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import { BarChart3, Bot, LayoutDashboard, Radio, Settings, Wallet } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Badge, Button } from '@/components/ui'
import { useSignOut } from '@/features/auth/hooks/use-auth'
import type { Tier, User } from '@/types/api'

const TIER_LABEL: Record<Tier, string> = { free: 'Free', basic: 'Basic', plus: 'Plus' }

/** Nama halaman untuk ditampilkan di bilah atas. */
const TITLES: Record<string, string> = {
  '/': 'Ringkasan',
  '/sinyal': 'Sinyal',
  '/posisi': 'Posisi',
  '/akun': 'Akun Broker',
  '/strategi': 'Strategi',
  '/pengaturan': 'Pengaturan',
}

const NAV = [
  { to: '/', label: 'Ringkasan', short: 'Ringkasan', icon: LayoutDashboard, end: true },
  { to: '/sinyal', label: 'Sinyal', short: 'Sinyal', icon: Radio, end: false },
  { to: '/posisi', label: 'Posisi', short: 'Posisi', icon: BarChart3, end: false },
  { to: '/akun', label: 'Akun Broker', short: 'Akun', icon: Wallet, end: false },
  { to: '/strategi', label: 'Strategi', short: 'Strategi', icon: Bot, end: false },
  { to: '/pengaturan', label: 'Pengaturan', short: 'Setelan', icon: Settings, end: false },
]

type AppShellProps = { user: User; children: ReactNode }

/**
 * Kerangka aplikasi: sidebar tetap di kiri, bilah judul di atas, konten
 * terpusat. Sidebar runtuh menjadi bilah atas pada layar sempit.
 */
export function AppShell({ user, children }: AppShellProps) {
  const signOut = useSignOut()
  const { pathname } = useLocation()
  const title = TITLES[pathname] ?? 'ILA Bot'

  return (
    <div className="flex min-h-full flex-col lg:flex-row">
      <aside className="shrink-0 border-b border-border lg:sticky lg:top-0 lg:h-screen lg:w-60 lg:border-b-0 lg:border-r">
        <div className="flex h-14 items-center gap-2 border-b border-border px-4">
          <span className="text-sm font-bold tracking-tight">ILA</span>
          <span className="text-[10px] uppercase tracking-[0.2em] text-muted-foreground">Bot</span>
        </div>

        <nav
          aria-label="Navigasi utama"
          className="flex gap-1 overflow-x-auto scrollbar-thin p-2 lg:flex-col lg:overflow-visible"
        >
          {NAV.map(({ to, label, short, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              aria-current={pathname === to ? 'page' : undefined}
              className={({ isActive }) =>
                cn(
                  'flex shrink-0 items-center gap-2 whitespace-nowrap rounded-md px-2.5 py-2 text-[13px] transition-colors',
                  isActive
                    ? 'bg-secondary font-semibold text-foreground'
                    : 'text-muted-foreground hover:bg-secondary/60 hover:text-foreground',
                )
              }
            >
              <Icon className="h-4 w-4 shrink-0" aria-hidden="true" />
              <span className="lg:hidden">{short}</span>
              <span className="hidden lg:inline">{label}</span>
            </NavLink>
          ))}
        </nav>

        <div className="hidden flex-col gap-2 border-t border-border p-3 lg:flex">
          <div className="flex items-center justify-between gap-2">
            <span className="truncate text-[12px] font-medium">{user.name}</span>
            <Badge variant="muted">{TIER_LABEL[user.tier]}</Badge>
          </div>
          <Button variant="outline" size="sm" className="w-full" onClick={signOut}>
            Keluar
          </Button>
        </div>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <div className="flex h-14 items-center justify-between gap-3 border-b border-border px-4 lg:px-6">
          <h2 className="truncate text-[13px] font-semibold">{title}</h2>
          <div className="flex items-center gap-2">
            <span className="hidden text-[12px] text-muted-foreground sm:inline">{user.email}</span>
            <Badge variant="muted" className="sm:hidden">
              {TIER_LABEL[user.tier]}
            </Badge>
            <Button variant="ghost" size="sm" className="lg:hidden" onClick={signOut}>
              Keluar
            </Button>
          </div>
        </div>

        <main className="min-w-0 flex-1 px-4 py-5 lg:px-6 lg:py-6">
          <div className="mx-auto flex w-full max-w-6xl flex-col gap-6">{children}</div>
        </main>
      </div>
    </div>
  )
}
