import { useEffect, useState, type ReactNode } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import {
  BarChart3,
  Bot,
  Bug,
  CreditCard,
  FlaskConical,
  LayoutDashboard,
  LogOut,
  MessageSquareText,
  PanelLeftClose,
  PanelLeftOpen,
  Radio,
  Settings,
  Sparkles,
  Wallet,
} from 'lucide-react'
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
import { ThemeIconButton } from '@/components/layout/ThemeToggle'
import { useApplyUserDashboardTheme } from '@/lib/theme'
import { useInviteOnly } from '@/lib/use-invite-only'
import type { Tier, User } from '@/types/api'

const TIER_LABEL: Record<Tier, string> = { free: 'Free', basic: 'Basic', plus: 'Plus' }

/** Nama halaman untuk ditampilkan di bilah atas. */
const TITLES: Record<string, string> = {
  '/dashboard': 'Dashboard',
  '/': 'Dashboard',
  '/sinyal': 'Signals & Radar',
  '/posisi': 'Positions',
  '/jurnal': 'Jurnal Manual',
  '/akun': 'Broker Accounts',
  '/strategi': 'Watchlist Pair',
  '/asisten': 'ILA AI',
  '/backtest': 'Backtest ILA 2.0',
  '/backtest/riwayat': 'Riwayat backtest',
  '/langganan': 'Subscription & Billing',
  '/langganan/bayar': 'Checkout Langganan',
  '/pengaturan': 'Settings',
  '/pengaturan/keamanan': 'Security Settings',
  '/mulai': 'Onboarding',
  '/laporkan-bug': 'Laporkan Bug',
  '/laporkan-bug/riwayat': 'Riwayat Laporan',
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
      { to: '/jurnal', label: 'Jurnal Manual', short: 'Jurnal', icon: BarChart3 },
      { to: '/akun', label: 'Broker Accounts', short: 'Accounts', icon: Wallet },
      { to: '/strategi', label: 'Watchlist Pair', short: 'Watchlist', icon: Bot },
      { to: '/asisten', label: 'ILA AI', short: 'ILA AI', icon: MessageSquareText },
      { to: '/backtest', label: 'Backtest', short: 'Backtest', icon: FlaskConical },
    ],
  },
  {
    label: 'Akun & Billing',
    items: [
      { to: '/langganan', label: 'Subscription & Billing', short: 'Billing', icon: CreditCard },
      { to: '/pengaturan', label: 'Settings & Security', short: 'Settings', icon: Settings },
    ],
  },
  {
    label: 'Bantuan',
    items: [
      { to: '/laporkan-bug', label: 'Laporkan Bug', short: 'Lapor', icon: Bug },
    ],
  },
]

type AppShellProps = { user: User; children: ReactNode }

/**
 * Kerangka aplikasi pelanggan:
 * - Sidebar desktop fixed di kiri (tidak terikut scroll halaman utama).
 * - Mendukung mode collapse / expand dengan transisi mulus dan penyimpanan preferensi lokal.
 * - Bilah judul desktop tetap sticky di atas.
 * - Pada layar ponsel/tablet (< lg), beralih ke MobileTopBar & MobileBottomNav.
 */
export function AppShell({ user, children }: AppShellProps) {
  useApplyUserDashboardTheme()
  const signOut = useSignOut()
  const queryClient = useQueryClient()
  const { pathname } = useLocation()
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const inviteOnly = useInviteOnly()
  const title = TITLES[pathname] ?? 'ILA Bot'

  // State sidebar collapse / expand (tersimpan di localStorage)
  const [isCollapsed, setIsCollapsed] = useState(() => {
    try {
      return localStorage.getItem('ila_user_sidebar_collapsed') === 'true'
    } catch {
      return false
    }
  })

  useEffect(() => {
    try {
      localStorage.setItem('ila_user_sidebar_collapsed', String(isCollapsed))
    } catch {
      // Abaikan bila storage dicegah oleh izin peramban
    }
  }, [isCollapsed])

  const toggleSidebar = () => setIsCollapsed((prev) => !prev)

  // Saat Service Worker menerima Web Push di latar belakang, segarkan data
  // sinyal dan posisi seketika tanpa menunggu jeda polling.
  useEffect(() => {
    if (typeof navigator === 'undefined' || !('serviceWorker' in navigator)) return
    const onMessage = (event: MessageEvent) => {
      if (event.data && typeof event.data === 'object' && event.data.type === 'ILA_PUSH_EVENT') {
        void queryClient.invalidateQueries({ queryKey: ['signals'] })
        void queryClient.invalidateQueries({ queryKey: ['positions'] })
        void queryClient.invalidateQueries({ queryKey: ['journal'] })
      }
    }
    navigator.serviceWorker.addEventListener?.('message', onMessage)
    return () => {
      navigator.serviceWorker.removeEventListener?.('message', onMessage)
    }
  }, [queryClient])

  // Menu billing hanya berguna bila ada yang bisa dibeli. Saat invite only,
  // kelompoknya hilang sama sekali supaya tidak ada pintu mati di sidebar.
  const navGroups = inviteOnly
    ? NAV_GROUPS.map((group) =>
        group.label === 'Akun & Billing'
          ? { ...group, label: 'Akun', items: group.items.filter((item) => item.to !== '/langganan') }
          : group,
      ).filter((group) => group.items.length > 0)
    : NAV_GROUPS

  return (
    <div className="flex min-h-screen w-full flex-col overflow-x-clip lg:flex-row">
      <LegalReconsentGate />
      {!pathname.startsWith('/approve') && <OnboardingModal />}

      {/* Top Bar khusus tampilan Mobile (< lg) */}
      <MobileTopBar user={user} onOpenMenu={() => setMobileMenuOpen(true)} />

      {/* Sidebar tetap di kiri untuk layar Desktop (>= lg) — Fixed & tidak terikut scroll */}
      <aside
        className={cn(
          'hidden lg:fixed lg:inset-y-0 lg:left-0 lg:z-30 lg:flex lg:h-screen lg:flex-col lg:border-r lg:border-border lg:bg-card transition-[width] duration-200 ease-in-out',
          isCollapsed ? 'lg:w-[68px]' : 'lg:w-64',
        )}
      >
        <div
          className={cn(
            'flex h-14 shrink-0 items-center border-b border-border px-3.5',
            isCollapsed ? 'justify-center px-2' : 'justify-between',
          )}
        >
          {isCollapsed ? (
            <button
              type="button"
              onClick={toggleSidebar}
              className="group relative flex h-9 w-9 items-center justify-center rounded-lg border border-border/80 bg-background hover:bg-secondary text-muted-foreground hover:text-foreground transition-all shadow-xs"
              title="Perluas bilah navigasi"
              aria-label="Perluas bilah navigasi"
            >
              <Logo className="h-6 w-auto transition-all group-hover:opacity-20 group-hover:scale-90" />
              <PanelLeftOpen className="absolute h-4 w-4 opacity-0 transition-all group-hover:opacity-100 text-foreground" />
            </button>
          ) : (
            <>
              <div className="flex items-center gap-2.5 overflow-hidden">
                <Logo className="h-7 w-auto shrink-0" />
                <div className="flex items-baseline gap-1">
                  <span className="text-sm font-bold tracking-tight">ILA</span>
                  <span className="rounded bg-secondary px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">
                    Platform
                  </span>
                </div>
              </div>
              <div className="flex items-center gap-1.5">
                <Badge variant="muted" className="inline-flex text-[10px]">
                  {TIER_LABEL[user.tier]}
                </Badge>
                <button
                  type="button"
                  onClick={toggleSidebar}
                  className="flex h-8 w-8 items-center justify-center rounded-md border border-border text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
                  title="Ciutkan bilah navigasi"
                  aria-label="Ciutkan bilah navigasi"
                >
                  <PanelLeftClose className="h-4 w-4" aria-hidden="true" />
                </button>
              </div>
            </>
          )}
        </div>

        <nav
          aria-label="Navigasi utama"
          className="flex flex-1 flex-col overflow-y-auto p-2.5 scrollbar-thin"
        >
          {navGroups.map((group, groupIndex) => (
            <div
              key={group.label}
              className={cn('flex flex-col', groupIndex > 0 && (isCollapsed ? 'mt-2' : 'mt-3'))}
            >
              {isCollapsed ? (
                groupIndex > 0 && <div className="my-1.5 border-t border-border/60 mx-1.5" />
              ) : (
                <p className="px-2.5 pb-1 pt-1.5 text-[10px] font-semibold uppercase tracking-[0.14em] text-muted-foreground/70">
                  {group.label}
                </p>
              )}
              <div className="flex flex-col gap-1">
                {group.items.map(({ to, label, icon: Icon, end }) => (
                  <NavLink
                    key={to}
                    to={to}
                    end={end}
                    title={isCollapsed ? label : undefined}
                    aria-current={pathname === to ? 'page' : undefined}
                    className={({ isActive }) =>
                      cn(
                        'flex items-center gap-2.5 rounded-lg text-[13px] transition-colors',
                        isCollapsed ? 'justify-center p-2.5' : 'px-2.5 py-2',
                        isActive
                          ? 'bg-secondary font-semibold text-foreground shadow-xs'
                          : 'text-muted-foreground hover:bg-secondary/60 hover:text-foreground',
                      )
                    }
                  >
                    <Icon className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden="true" />
                    <span className={cn('truncate', isCollapsed && 'hidden')}>{label}</span>
                  </NavLink>
                ))}
              </div>
            </div>
          ))}
        </nav>

        <div className="shrink-0 border-t border-border p-2.5">
          {isCollapsed ? (
            <div className="flex flex-col items-center gap-2">
              <button
                type="button"
                onClick={signOut}
                className="flex h-9 w-9 items-center justify-center rounded-md border border-border text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
                title={`Keluar (${user.name || user.email})`}
                aria-label="Keluar dari akun"
              >
                <LogOut className="h-4 w-4" aria-hidden="true" />
              </button>
            </div>
          ) : (
            <div className="flex flex-col gap-2.5 p-1">
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
          )}
        </div>
      </aside>

      {/* Konten Utama Desktop (Diberi padding kiri sesuai lebar sidebar fixed) */}
      <div
        className={cn(
          'flex min-w-0 flex-1 flex-col transition-[padding] duration-200 ease-in-out',
          isCollapsed ? 'lg:pl-[68px]' : 'lg:pl-64',
        )}
      >
        {/* Desktop Header Bar (>= lg) — Sticky di puncak konten */}
        <div className="sticky top-0 z-20 hidden h-14 items-center justify-between gap-3 border-b border-border bg-background/95 backdrop-blur-xs px-6 lg:flex">
          <h2 className="truncate text-[13px] font-semibold">{title}</h2>
          <div className="flex items-center gap-2">
            <span className="text-[12px] text-muted-foreground">{user.email}</span>
            <Badge variant="muted">{TIER_LABEL[user.tier]}</Badge>
            <ThemeIconButton className="border border-border/80 bg-card/60" />
            <Button variant="ghost" size="sm" onClick={signOut}>
              Keluar
            </Button>
          </div>
        </div>

        <main className="min-w-0 flex-1 px-3.5 py-3.5 pb-[calc(4.75rem+env(safe-area-inset-bottom))] sm:px-4 sm:py-4 lg:px-6 lg:py-6 lg:pb-6">
          <div className="mx-auto flex w-full min-w-0 max-w-6xl flex-col gap-3.5 lg:gap-4">
            <UserBanners />
            <div className="flex min-w-0 flex-col gap-4 lg:gap-6">{children}</div>
          </div>
        </main>
      </div>

      {/* Navigasi Bawah Sticky khusus Mobile */}
      <MobileBottomNav inviteOnly={inviteOnly} />

      {/* Drawer menu penuh ketika hamburger atau menu Lainnya dibuka */}
      <MobileMoreDrawer
        open={mobileMenuOpen}
        onClose={() => setMobileMenuOpen(false)}
        user={user}
        inviteOnly={inviteOnly}
      />
    </div>
  )
}
