import type { ReactNode } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import { BarChart3, Bot, CreditCard, LayoutDashboard, Radio, Settings, Sparkles, Wallet } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Badge, Button } from '@/components/ui'
import { useSignOut } from '@/features/auth/hooks/use-auth'
import { UserBanners } from '@/components/layout/UserBanners'
import { LegalReconsentGate } from '@/features/auth/components/LegalReconsentGate'
import { OnboardingModal } from '@/features/onboarding'
import type { Tier, User } from '@/types/api'

const TIER_LABEL: Record<Tier, string> = { free: 'Free', basic: 'Basic', plus: 'Plus' }

/** Nama halaman untuk ditampilkan di bilah atas. */
const TITLES: Record<string, string> = {
  '/dashboard': 'Ringkasan',
  '/': 'Ringkasan',
  '/sinyal': 'Sinyal',
  '/posisi': 'Posisi',
  '/akun': 'Akun Broker',
  '/strategi': 'Pair Pantauan',
  '/langganan': 'Langganan',
  '/langganan/bayar': 'Pembayaran Langganan',
  '/pengaturan': 'Pengaturan',
  '/pengaturan/keamanan': 'Keamanan',
  '/mulai': 'Mulai',
}

type NavItem = {
  to: string
  label: string
  short: string
  icon: typeof LayoutDashboard
  end?: boolean
}

type NavGroup = {
  label: string
  items: NavItem[]
}

const NAV_GROUPS: NavGroup[] = [
  {
    label: 'Utama',
    items: [
      { to: '/dashboard', label: 'Ringkasan', short: 'Ringkasan', icon: LayoutDashboard, end: true },
      { to: '/mulai', label: 'Penyiapan Akun', short: 'Mulai', icon: Sparkles },
    ],
  },
  {
    label: 'Trading & Pantauan',
    items: [
      { to: '/sinyal', label: 'Sinyal Per Pair', short: 'Sinyal', icon: Radio },
      { to: '/posisi', label: 'Riwayat Posisi', short: 'Posisi', icon: BarChart3 },
      { to: '/akun', label: 'Akun Broker', short: 'Akun', icon: Wallet },
      { to: '/strategi', label: 'Pair Pantauan', short: 'Pair', icon: Bot },
    ],
  },
  {
    label: 'Akun & Billing',
    items: [
      { to: '/langganan', label: 'Paket & Langganan', short: 'Langganan', icon: CreditCard },
      { to: '/pengaturan', label: 'Pengaturan & Keamanan', short: 'Setelan', icon: Settings },
    ],
  },
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
      <LegalReconsentGate />
      <OnboardingModal />
      <aside className="shrink-0 border-b border-border lg:sticky lg:top-0 lg:flex lg:h-screen lg:w-64 lg:flex-col lg:border-b-0 lg:border-r">
        <div className="flex h-14 items-center justify-between border-b border-border px-4">
          <div className="flex items-center gap-2.5">
            {/* logo-light.png = garis hitam untuk latar terang; logo-dark.png = garis putih untuk latar gelap. */}
            <img src="/logo-light.png" alt="ILA" className="h-7 w-auto object-contain dark:hidden" />
            <img src="/logo-dark.png" alt="ILA" className="h-7 w-auto object-contain hidden dark:block" />
            <div className="flex items-baseline gap-1">
              <span className="text-sm font-bold tracking-tight">ILA</span>
              <span className="rounded bg-secondary px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">
                Platform
              </span>
            </div>
          </div>
          <Badge variant="muted" className="hidden lg:inline-flex text-[10px]">
            {TIER_LABEL[user.tier]}
          </Badge>
        </div>

        <nav
          aria-label="Navigasi utama"
          className="flex gap-1 overflow-x-auto scrollbar-thin p-2 lg:flex-1 lg:flex-col lg:overflow-y-auto lg:p-3"
        >
          {NAV_GROUPS.map((group, groupIndex) => (
            <div key={group.label} className={cn('flex gap-1 lg:flex-col', groupIndex > 0 && 'lg:mt-3')}>
              <p className="hidden px-2.5 pb-1 pt-1.5 text-[10px] font-semibold uppercase tracking-[0.14em] text-muted-foreground/70 lg:block">
                {group.label}
              </p>
              {group.items.map(({ to, label, short, icon: Icon, end }) => (
                <NavLink
                  key={to}
                  to={to}
                  end={end}
                  aria-current={pathname === to ? 'page' : undefined}
                  className={({ isActive }) =>
                    cn(
                      'flex shrink-0 items-center gap-2.5 whitespace-nowrap rounded-lg px-2.5 py-2 text-[13px] transition-colors',
                      isActive
                        ? 'bg-secondary font-semibold text-foreground shadow-sm'
                        : 'text-muted-foreground hover:bg-secondary/60 hover:text-foreground',
                    )
                  }
                >
                  <Icon className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden="true" />
                  <span className="lg:hidden">{short}</span>
                  <span className="hidden lg:inline">{label}</span>
                </NavLink>
              ))}
            </div>
          ))}
        </nav>

        <div className="hidden flex-col gap-2.5 border-t border-border p-3.5 lg:flex">
          <div className="flex items-center justify-between gap-2">
            <div className="min-w-0">
              <p className="truncate text-[12.5px] font-semibold">{user.name}</p>
              <p className="truncate text-[11px] text-muted-foreground">{user.email}</p>
            </div>
            <Badge variant="muted">{TIER_LABEL[user.tier]}</Badge>
          </div>
          <Button variant="outline" size="sm" className="w-full text-xs" onClick={signOut}>
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
          <div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
            <UserBanners />
            <div className="flex flex-col gap-6">{children}</div>
          </div>
        </main>
      </div>
    </div>
  )
}
