import type { ReactNode } from 'react'
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
import { cn } from '@/lib/utils'

type NavItem = {
  to: string
  label: string
  icon: typeof LayoutDashboard
  end?: boolean
}

type NavGroup = {
  category: string
  items: NavItem[]
}

/** Menu panel admin yang dikelompokkan berdasarkan kategori kerja. */
const NAV_GROUPS: NavGroup[] = [
  {
    category: 'Utama',
    items: [
      { to: '/admin', label: 'Ringkasan', icon: LayoutDashboard, end: true },
      { to: '/admin/pengguna', label: 'Pengguna', icon: Users },
      { to: '/admin/dukungan', label: 'Dukungan', icon: LifeBuoy },
    ],
  },
  {
    category: 'Katalog & Layanan',
    items: [{ to: '/admin/pairs', label: 'Katalog Pair', icon: Coins }],
  },
  {
    category: 'Billing & Monetisasi',
    items: [
      { to: '/admin/paket', label: 'Paket & Harga', icon: Wallet },
      { to: '/admin/tagihan', label: 'Tagihan', icon: Receipt },
      { to: '/admin/kupon', label: 'Kupon', icon: Percent },
      { to: '/admin/kode-kredit', label: 'Kode Kredit', icon: Ticket },
      { to: '/admin/trial', label: 'Uji Coba', icon: Timer },
    ],
  },
  {
    category: 'Konten & CMS',
    items: [
      { to: '/admin/konten/halaman', label: 'Halaman Publik', icon: FileText },
      { to: '/admin/konten/legal', label: 'Dokumen Legal', icon: ShieldCheck },
      { to: '/admin/konten', label: 'Template Konten', icon: FileText, end: true },
    ],
  },
  {
    category: 'Sistem & Infrastruktur',
    items: [
      { to: '/admin/engine', label: 'Engine', icon: Radio },
      { to: '/admin/runner', label: 'Runner', icon: Cpu },
      { to: '/admin/kesehatan', label: 'Ringkasan Kesehatan', icon: Gauge },
      { to: '/admin/kesehatan-broker', label: 'Kesehatan Broker', icon: Activity },
      { to: '/admin/pengaturan-sistem', label: 'Flag Sistem', icon: Settings2 },
      { to: '/admin/pemeliharaan', label: 'Pemeliharaan', icon: ShieldAlert },
    ],
  },
  {
    category: 'Kepatuhan & Laporan',
    items: [
      { to: '/admin/laporan', label: 'Laporan', icon: BarChart3 },
      { to: '/admin/audit', label: 'Jejak Audit', icon: FileClock },
    ],
  },
]

/**
 * AdminShell adalah kerangka panel admin. Terpisah dari AppShell pengguna
 * karena sesi admin memakai pintu sendiri dan tidak saling bergantian.
 */
export function AdminShell({ children, actorEmail }: { children: ReactNode; actorEmail?: string }) {
  const { admin, signOut } = useAdminAuth()
  const email = admin?.email ?? actorEmail

  return (
    <div className="flex min-h-full flex-col lg:flex-row">
      <aside className="shrink-0 border-b border-border lg:sticky lg:top-0 lg:flex lg:h-screen lg:w-64 lg:flex-col lg:border-b-0 lg:border-r">
        <div className="flex h-14 shrink-0 items-center gap-2.5 border-b border-border px-4">
          {/* logo-light.png = garis hitam (latar terang), logo-dark.png = garis putih (mode gelap). */}
          <img src="/logo-light.png" alt="ILA" className="h-7 w-auto object-contain dark:hidden" />
          <img src="/logo-dark.png" alt="ILA" className="h-7 w-auto object-contain hidden dark:block" />
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
          {NAV_GROUPS.map((group) => (
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

        {email && (
          <div className="hidden shrink-0 border-t border-border p-3 lg:block">
            <div className="flex items-center gap-2">
              <div className="min-w-0 flex-1">
                <p className="text-[10px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">
                  Masuk sebagai
                </p>
                <p className="truncate text-[11px] text-foreground">{email}</p>
              </div>
              <button
                type="button"
                onClick={signOut}
                className="inline-flex shrink-0 items-center gap-1.5 rounded-md border border-border px-2 py-1.5 text-[11px] text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                aria-label="Keluar dari panel admin"
              >
                <LogOut className="h-3.5 w-3.5" aria-hidden="true" />
                Keluar
              </button>
            </div>
          </div>
        )}
      </aside>

      <main className="min-w-0 flex-1">
        <div className="mx-auto flex max-w-6xl flex-col gap-5 p-4 sm:p-6">{children}</div>
      </main>
    </div>
  )
}
