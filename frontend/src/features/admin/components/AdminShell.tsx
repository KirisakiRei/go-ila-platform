import type { ReactNode } from 'react'
import { useMemo } from 'react'
import { NavLink } from 'react-router-dom'
import {
  Activity,
  Coins,
  Cpu,
  Gauge,
  Radio,
  BarChart3,
  FileClock,
  FileText,
  LayoutDashboard,
  LogOut,
  Percent,
  Receipt,
  Ticket,
  Timer,
  Settings2,
  ShieldAlert,
  LifeBuoy,
  ShieldCheck,
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
  const email = admin?.email ?? actorEmail

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
      <aside className="shrink-0 border-b border-border lg:sticky lg:top-0 lg:flex lg:h-screen lg:w-64 lg:flex-col lg:border-b-0 lg:border-r">
        <div className="flex h-14 shrink-0 items-center gap-2.5 border-b border-border px-4">
          {/* Logo bersama memakai turunan 64/128px, bukan berkas 660 KB asli. */}
          <Logo className="h-7 w-auto" />
          <div className="flex items-baseline gap-1">
            <span className="text-sm font-bold tracking-tight">ILA</span>
            <span className="text-[10px] uppercase tracking-[0.2em] text-muted-foreground">
              Admin
            </span>
          </div>

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
          {navGroups.map((group) => (
            <div key={group.category} className="mb-2 last:mb-0">
              <p className="hidden px-3 pt-2 pb-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground lg:block">
                {group.category}
              </p>
              <div className="flex gap-1 lg:flex-col">
                {group.items.map((item) => (
                  <NavLink
                    key={item.to}
                    to={item.to}
                    end={item.end}
                    className={({ isActive }) =>
                      cn(
                        'flex items-center gap-2 whitespace-nowrap rounded-md px-3 py-1.5 text-[13px] transition-colors',
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
              </div>
            </div>
          ))}
        </nav>

        <div className="hidden shrink-0 border-t border-border p-3 lg:block">
          <div className="flex items-center gap-2">
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
        </div>
      </aside>

      <main className="min-w-0 flex-1">
        <div className="mx-auto flex max-w-6xl flex-col gap-5 p-4 sm:p-6">{children}</div>
      </main>
    </div>
  )
}
