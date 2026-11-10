import { useState, type ReactNode } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import { BarChart3, Bot, CreditCard, LayoutDashboard, Radio, Settings, Sparkles, Wallet } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Badge, Button } from '@/components/ui'
import { Logo } from '@/components/brand/Logo'
import { useSignOut } from '@/features/auth/hooks/use-auth'
import { UserBanners } from '@/components/layout/UserBanners'
import { LegalReconsentGate } from '@/features/auth/components/LegalReconsentGate'
import { OnboardingModal } from '@/features/onboarding'
import { MobileTopBar } from '@/components/layout/MobileTopBar'
import { MobileBottomNav } from '@/components/layout/MobileBottomNav'
import { MobileMoreDrawer } from '@/components/layout/MobileMoreDrawer'
import type { Tier, User } from '@/types/api'

const TIER_LABEL: Record<Tier, string> = { free: 'Free', basic: 'Basic', plus: 'Plus' }

/** Nama halaman untuk ditampilkan di bilah atas. */
const TITLES: Record<string, string> = {
  '/dashboard': 'Dashboard',
  '/': 'Dashboard',
  '/sinyal': 'Signals & Radar',
  '/posisi': 'Positions',
  '/akun': 'Broker Accounts',
  '/strategi': 'Watchlist Pair',
  '/langganan': 'Subscription & Billing',
  '/langganan/bayar': 'Checkout Langganan',
  '/pengaturan': 'Settings',
  '/pengaturan/keamanan': 'Security Settings',
  '/mulai': 'Onboarding',
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
    label: 'Menu Utama',
    items: [
      { to: '/dashboard', label: 'Dashboard', short: 'Dashboard', icon: LayoutDashboard, end: true },
      { to: '/mulai', label: 'Onboarding', short: 'Onboarding', icon: Sparkles },
    ],
  },
  {
    label: 'Trading & Radar',
    items: [
      { to: '/sinyal', label: 'Signals & Radar', short: 'Signals', icon: Radio },
      { to: '/posisi', label: 'Positions', short: 'Positions', icon: BarChart3 },
      { to: '/akun', label: 'Broker Accounts', short: 'Accounts', icon: Wallet },
      { to: '/strategi', label: 'Watchlist Pair', short: 'Watchlist', icon: Bot },
    ],
  },
  {
    label: 'Akun & Billing',
    items: [
      { to: '/langganan', label: 'Subscription & Billing', short: 'Billing', icon: CreditCard },
      { to: '/pengaturan', label: 'Settings & Security', short: 'Settings', icon: Settings },
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
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const title = TITLES[pathname] ?? 'ILA Bot'

  return (
    <div className="flex min-h-full flex-col lg:flex-row">
      <LegalReconsentGate />
      <OnboardingModal />

      {/* Top Bar khusus tampilan Mobile (< lg) */}
      <MobileTopBar user={user} onOpenMenu={() => setMobileMenuOpen(true)} />

      {/* Sidebar tetap di kiri untuk layar Desktop (>= lg) */}
      <aside className="hidden lg:sticky lg:top-0 lg:flex lg:h-screen lg:w-64 lg:shrink-0 lg:flex-col lg:border-r lg:border-border">
        <div className="flex h-14 items-center justify-between border-b border-border px-4">
          <div className="flex items-center gap-2.5">
            {/* Logo bersama memakai turunan 64/128px, bukan berkas 660 KB asli. */}
            <Logo className="h-7 w-auto" />
            <div className="flex items-baseline gap-1">
              <span className="text-sm font-bold tracking-tight">ILA</span>
              <span className="rounded bg-secondary px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">
                Platform
              </span>
            </div>
          </div>
          <Badge variant="muted" className="inline-flex text-[10px]">
            {TIER_LABEL[user.tier]}
          </Badge>
        </div>

        <nav
          aria-label="Navigasi utama"
          className="flex flex-1 flex-col overflow-y-auto p-3 scrollbar-thin"
        >
          {NAV_GROUPS.map((group, groupIndex) => (
            <div key={group.label} className={cn('flex flex-col', groupIndex > 0 && 'mt-3')}>
              <p className="px-2.5 pb-1 pt-1.5 text-[10px] font-semibold uppercase tracking-[0.14em] text-muted-foreground/70">
                {group.label}
              </p>
              {group.items.map(({ to, label, icon: Icon, end }) => (
                <NavLink
                  key={to}
                  to={to}
                  end={end}
                  aria-current={pathname === to ? 'page' : undefined}
                  className={({ isActive }) =>
                    cn(
                      'flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-[13px] transition-colors',
                      isActive
                        ? 'bg-secondary font-semibold text-foreground shadow-xs'
                        : 'text-muted-foreground hover:bg-secondary/60 hover:text-foreground',
                    )
                  }
                >
                  <Icon className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden="true" />
                  <span>{label}</span>
                </NavLink>
              ))}
            </div>
          ))}
        </nav>

        <div className="flex flex-col gap-2.5 border-t border-border p-3.5">
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
        {/* Desktop Header Bar (>= lg) */}
        <div className="hidden h-14 items-center justify-between gap-3 border-b border-border px-6 lg:flex">
          <h2 className="truncate text-[13px] font-semibold">{title}</h2>
          <div className="flex items-center gap-2">
            <span className="text-[12px] text-muted-foreground">{user.email}</span>
            <Badge variant="muted">{TIER_LABEL[user.tier]}</Badge>
            <Button variant="ghost" size="sm" onClick={signOut}>
              Keluar
            </Button>
          </div>
        </div>

        <main className="min-w-0 flex-1 px-4 py-4 pb-24 lg:px-6 lg:py-6 lg:pb-6">
          <div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
            <UserBanners />
            <div className="flex flex-col gap-6">{children}</div>
          </div>
        </main>
      </div>

      {/* Navigasi Bawah Sticky khusus Mobile */}
      <MobileBottomNav />

      {/* Drawer menu penuh ketika hamburger atau menu Lainnya dibuka */}
      <MobileMoreDrawer
        open={mobileMenuOpen}
        onClose={() => setMobileMenuOpen(false)}
        user={user}
      />
    </div>
  )
}
