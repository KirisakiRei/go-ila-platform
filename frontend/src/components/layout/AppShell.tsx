import type { ReactNode } from 'react'
import { NavLink } from 'react-router-dom'
import { cn } from '@/lib/utils'
import { Header } from './Header'
import type { User } from '@/types/api'

type AppShellProps = {
  user: User
  children: ReactNode
}

const NAV_ITEMS = [
  { to: '/', label: 'Ringkasan', end: true },
  { to: '/sinyal', label: 'Sinyal', end: false },
  { to: '/posisi', label: 'Posisi', end: false },
  { to: '/akun', label: 'Akun Broker', end: false },
  { to: '/strategi', label: 'Strategi', end: false },
  { to: '/pengaturan', label: 'Pengaturan', end: false },
]

/** Kerangka halaman: header tetap, navigasi utama, konten terbatas lebarnya. */
export function AppShell({ user, children }: AppShellProps) {
  return (
    <div className="flex min-h-full flex-col">
      <Header user={user} />

      <nav
        aria-label="Navigasi utama"
        className="border-b border-border bg-background/60 backdrop-blur"
      >
        <div className="mx-auto flex w-full max-w-5xl gap-1 overflow-x-auto px-4">
          {NAV_ITEMS.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={({ isActive }) =>
                cn(
                  'whitespace-nowrap border-b-2 px-3 py-2.5 text-[13px] transition-colors',
                  isActive
                    ? 'border-primary font-semibold text-foreground'
                    : 'border-transparent text-muted-foreground hover:text-foreground',
                )
              }
            >
              {item.label}
            </NavLink>
          ))}
        </div>
      </nav>

      <main className="mx-auto w-full max-w-5xl flex-1 px-4 py-6">{children}</main>
    </div>
  )
}
