import type { ReactNode } from 'react'
import { useEffect, useMemo, useState } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import {
  Activity,
  BarChart3,
  ChevronDown,
  ChevronRight,
  Coins,
  Cpu,
  FileClock,
  FileText,
  Gauge,
  LayoutDashboard,
  LifeBuoy,
  LogOut,
  PanelLeftClose,
  PanelLeftOpen,
  Percent,
  Radio,
  Receipt,
  Settings2,
  ShieldAlert,
  ShieldCheck,
  Ticket,
  Timer,
  Users,
  Wallet,
} from 'lucide-react'
import { useAdminAuth } from '@/features/admin/admin-auth-context'
import { usePermissions } from '@/features/admin/hooks/use-permissions'
import { Logo } from '@/components/brand/Logo'
import { cn } from '@/lib/utils'

type NavItem = {
  to: string
  label: string
  icon: typeof LayoutDashboard
  end?: boolean
  /**
   * Permission minimum untuk melihat menu. Kosong berarti selalu tampil:
   * Ringkasan adalah halaman pendarat yang tidak boleh hilang.
   */
  permission?: string
}

type NavGroup = {
  category: string
  items: NavItem[]
}

/**
 * Menu panel admin yang dikelompokkan berdasarkan kategori kerja.
 *
 * Permission di sini hanya menyembunyikan menu; server tetap menolak dengan
 * 403 bila route dibuka langsung. Kelompok yang seluruh menunya tersembunyi
 * ikut hilang dari sidebar.
 */
const NAV_GROUPS: NavGroup[] = [
  {
    category: 'Utama',
    items: [
      { to: '/admin', label: 'Dashboard', icon: LayoutDashboard, end: true },
      { to: '/admin/pengguna', label: 'Users & Accounts', icon: Users, permission: 'user.view' },
      { to: '/admin/dukungan', label: 'Customer Support', icon: LifeBuoy, permission: 'user.view' },
      { to: '/admin/sinyal', label: 'Signals & Monitor', icon: Radio, permission: 'signal.view' },
    ],
  },
  {
    category: 'Katalog & Layanan',
    items: [
      { to: '/admin/pairs', label: 'Pair Catalog', icon: Coins, permission: 'subscription.edit' },
    ],
  },
  {
    category: 'Billing & Monetisasi',
    items: [
      { to: '/admin/paket', label: 'Plans & Pricing', icon: Wallet, permission: 'billing.plan' },
      { to: '/admin/tagihan', label: 'Invoices & Billing', icon: Receipt, permission: 'billing.view' },
      { to: '/admin/kupon', label: 'Promo Coupons', icon: Percent, permission: 'billing.voucher' },
      { to: '/admin/kode-kredit', label: 'Credit Vouchers', icon: Ticket, permission: 'billing.voucher' },
      { to: '/admin/trial', label: 'Free Trials', icon: Timer, permission: 'user.tier' },
    ],
  },
  {
    category: 'Konten & CMS',
    items: [
      {
        to: '/admin/konten/halaman',
        label: 'Public Pages',
        icon: FileText,
        permission: 'cms.view',
      },
      { to: '/admin/konten/legal', label: 'Legal Documents', icon: ShieldCheck, permission: 'cms.view' },
      { to: '/admin/konten', label: 'Content Blocks', icon: FileText, end: true, permission: 'cms.view' },
    ],
  },
  {
    category: 'Sistem & Infrastruktur',
    items: [
      { to: '/admin/engine', label: 'Analysis Engines', icon: Radio, permission: 'system.view' },
      { to: '/admin/runner', label: 'Execution Runner', icon: Cpu, permission: 'system.view' },
      { to: '/admin/kesehatan', label: 'System Health', icon: Gauge, permission: 'system.view' },
      { to: '/admin/kesehatan-broker', label: 'Broker Telemetry', icon: Activity, permission: 'system.view' },
      { to: '/admin/pengaturan-sistem', label: 'System Flags', icon: Settings2, permission: 'system.settings' },
      { to: '/admin/pemeliharaan', label: 'Maintenance Mode', icon: ShieldAlert, permission: 'system.maintenance' },
    ],
  },
  {
    category: 'Kepatuhan & Laporan',
    items: [
      { to: '/admin/laporan', label: 'Business Reports', icon: BarChart3, permission: 'billing.view' },
      { to: '/admin/audit', label: 'Audit Trails', icon: FileClock, permission: 'audit.view' },
      { to: '/admin/rbac', label: 'Roles & Permissions', icon: ShieldCheck, permission: 'role.view' },
    ],
  },
]

/**
 * AdminShell adalah kerangka panel admin. Terpisah dari AppShell pengguna
 * karena sesi admin memakai pintu sendiri dan tidak saling bergantian.
 */
export function AdminShell({ children, actorEmail }: { children: ReactNode; actorEmail?: string }) {
  const { admin, signOut } = useAdminAuth()
  const { can } = usePermissions()
  const location = useLocation()
  const email = admin?.email ?? actorEmail

  // Sidebar mini/full collapse state
  const [isCollapsed, setIsCollapsed] = useState(() => {
    try {
      return localStorage.getItem('ila_admin_sidebar_collapsed') === 'true'
    } catch {
      return false
    }
  })

  // Group accordion collapse state (key = group category)
  const [collapsedGroups, setCollapsedGroups] = useState<Record<string, boolean>>(() => {
    try {
      const saved = localStorage.getItem('ila_admin_groups_collapsed')
      return saved ? JSON.parse(saved) : {}
    } catch {
      return {}
    }
  })

  useEffect(() => {
    try {
      localStorage.setItem('ila_admin_sidebar_collapsed', String(isCollapsed))
    } catch {
      // Ignore storage errors
    }
  }, [isCollapsed])

  useEffect(() => {
    try {
      localStorage.setItem('ila_admin_groups_collapsed', JSON.stringify(collapsedGroups))
    } catch {
      // Ignore storage errors
    }
  }, [collapsedGroups])

  const toggleSidebar = () => setIsCollapsed((prev) => !prev)

  const toggleGroup = (category: string) => {
    setCollapsedGroups((prev) => ({
      ...prev,
      [category]: !prev[category],
    }))
  }

  // Menu tanpa permission tetap tampil (Ringkasan); kelompok yang seluruh
  // menunya tersaring ikut dibuang agar sidebar tidak berisi judul kosong.
  const navGroups = useMemo(
    () =>
      NAV_GROUPS.map((group) => ({
        ...group,
        items: group.items.filter((item) => can(item.permission)),
      })).filter((group) => group.items.length > 0),
    [can],
  )

  return (
    <div className="flex min-h-full flex-col lg:flex-row">
      <aside
        className={cn(
          'shrink-0 border-b border-border transition-all duration-200 lg:sticky lg:top-0 lg:flex lg:h-screen lg:flex-col lg:border-b-0 lg:border-r',
          isCollapsed ? 'lg:w-[68px]' : 'lg:w-64',
        )}
      >
        <div
          className={cn(
            'flex h-14 shrink-0 items-center border-b border-border px-3.5',
            isCollapsed ? 'lg:justify-center lg:px-2' : 'justify-between',
          )}
        >
          {/* Logo bersama */}
          <div className="flex items-center gap-2.5 overflow-hidden">
            <Logo className="h-7 w-auto shrink-0" />
            <div className={cn('flex items-baseline gap-1', isCollapsed && 'lg:hidden')}>
              <span className="text-sm font-bold tracking-tight">ILA</span>
              <span className="text-[10px] uppercase tracking-[0.2em] text-muted-foreground">
                Admin
              </span>
            </div>
          </div>

          {/* Tombol collapse / expand sidebar desktop */}
          <button
            type="button"
            onClick={toggleSidebar}
            className={cn(
              'hidden h-8 w-8 items-center justify-center rounded-md border border-border text-muted-foreground transition-colors hover:bg-muted hover:text-foreground lg:flex',
              isCollapsed && 'mt-1',
            )}
            title={isCollapsed ? 'Perluas bilah navigasi' : 'Ciutkan bilah navigasi'}
            aria-label={isCollapsed ? 'Perluas bilah navigasi' : 'Ciutkan bilah navigasi'}
          >
            {isCollapsed ? (
              <PanelLeftOpen className="h-4 w-4" aria-hidden="true" />
            ) : (
              <PanelLeftClose className="h-4 w-4" aria-hidden="true" />
            )}
          </button>

          {/* Pada layar sempit sidebar menjadi bilah atas; tombol keluar ikut pindah ke sini. */}
          <button
            type="button"
            onClick={signOut}
            className="ml-auto inline-flex items-center gap-1.5 rounded-md border border-border px-2 py-1.5 text-[12px] text-muted-foreground transition-colors hover:bg-muted hover:text-foreground lg:hidden"
            aria-label="Keluar dari panel admin"
          >
            <LogOut className="h-3.5 w-3.5" aria-hidden="true" />
            Keluar
          </button>
        </div>

        <nav
          aria-label="Navigasi admin"
          className="flex gap-1 overflow-x-auto p-2 lg:flex-1 lg:flex-col lg:overflow-y-auto"
        >
          {navGroups.map((group) => {
            const isGroupCollapsed = !isCollapsed && Boolean(collapsedGroups[group.category])
            const hasActiveItem = group.items.some((item) =>
              item.end
                ? location.pathname === item.to
                : location.pathname.startsWith(item.to),
            )

            return (
              <div key={group.category} className="mb-2 last:mb-0">
                {/* Header grup kategori */}
                {!isCollapsed && (
                  <button
                    type="button"
                    onClick={() => toggleGroup(group.category)}
                    className="group hidden w-full items-center justify-between px-3 pt-2 pb-1 text-left text-[10px] font-semibold uppercase tracking-wider text-muted-foreground transition-colors hover:text-foreground lg:flex"
                    aria-expanded={!isGroupCollapsed}
                  >
                    <span className="flex items-center gap-1.5">
                      {group.category}
                      {hasActiveItem && isGroupCollapsed && (
                        <span className="h-1.5 w-1.5 rounded-full bg-primary" />
                      )}
                    </span>
                    {isGroupCollapsed ? (
                      <ChevronRight className="h-3.5 w-3.5 opacity-60 transition-transform group-hover:opacity-100" />
                    ) : (
                      <ChevronDown className="h-3.5 w-3.5 opacity-60 transition-transform group-hover:opacity-100" />
                    )}
                  </button>
                )}

                {/* Garis pemisah halus saat sidebar mode mini (collapsed) */}
                {isCollapsed && (
                  <div className="hidden my-1.5 border-t border-border/60 lg:block first:hidden" />
                )}

                {/* Daftar item menu di dalam grup */}
                <div
                  className={cn(
                    'flex gap-1 lg:flex-col',
                    isGroupCollapsed && 'lg:hidden',
                  )}
                >
                  {group.items.map((item) => (
                    <NavLink
                      key={item.to}
                      to={item.to}
                      end={item.end}
                      title={isCollapsed ? item.label : undefined}
                      className={({ isActive }) =>
                        cn(
                          'flex items-center gap-2 whitespace-nowrap rounded-md px-3 py-1.5 text-[13px] transition-colors',
                          isCollapsed && 'lg:justify-center lg:px-2.5 lg:py-2',
                          isActive
                            ? 'bg-muted font-medium text-foreground'
                            : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground',
                        )
                      }
                    >
                      <item.icon className="h-4 w-4 shrink-0" aria-hidden />
                      <span className={cn(isCollapsed && 'lg:hidden')}>{item.label}</span>
                    </NavLink>
                  ))}
                </div>
              </div>
            )
          })}
        </nav>

        {/* Footer info user & logout */}
        <div className="hidden shrink-0 border-t border-border p-2.5 lg:block">
          {isCollapsed ? (
            <div className="flex flex-col items-center gap-2">
              <button
                type="button"
                onClick={signOut}
                className="flex h-9 w-9 items-center justify-center rounded-md border border-border text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                title={`Keluar (${email || admin?.name || 'Administrator'})`}
                aria-label="Keluar dari panel admin"
              >
                <LogOut className="h-4 w-4" aria-hidden="true" />
              </button>
            </div>
          ) : (
            <div className="flex items-center gap-2 px-1">
              <div className="min-w-0 flex-1">
                <p className="text-[10px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">
                  Masuk sebagai
                </p>
                <p className="truncate text-[11px] font-medium text-foreground">
                  {email || admin?.name || 'Administrator'}
                </p>
              </div>
              <button
                type="button"
                onClick={signOut}
                className="inline-flex shrink-0 items-center gap-1.5 rounded-md border border-border px-2 py-1.5 text-[11px] font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                aria-label="Keluar dari panel admin"
              >
                <LogOut className="h-3.5 w-3.5" aria-hidden="true" />
                Keluar
              </button>
            </div>
          )}
        </div>
      </aside>

      <main className="min-w-0 flex-1">
        <div className="mx-auto flex max-w-6xl flex-col gap-5 p-4 sm:p-6">{children}</div>
      </main>
    </div>
  )
}
