import { NavLink } from 'react-router-dom'
import { BarChart3, CreditCard, LayoutDashboard, Radio, Settings } from 'lucide-react'
import { cn } from '@/lib/utils'

const BOTTOM_NAV_ITEMS = [
  { to: '/dashboard', label: 'Dashboard', icon: LayoutDashboard, end: true },
  { to: '/sinyal', label: 'Radar', icon: Radio },
  { to: '/posisi', label: 'Posisi', icon: BarChart3 },
  { to: '/langganan', label: 'Billing', icon: CreditCard },
  { to: '/pengaturan', label: 'Pengaturan', icon: Settings },
]

export function MobileBottomNav() {
  return (
    <nav
      aria-label="Navigasi bawah mobile"
      className="fixed bottom-0 left-0 right-0 z-40 border-t border-border bg-card/95 pb-safe backdrop-blur-xl lg:hidden shadow-[0_-2px_12px_rgba(0,0,0,0.04)]"
    >
      <div className="flex h-15 items-center justify-around px-1 py-1">
        {BOTTOM_NAV_ITEMS.map(({ to, label, icon: Icon, end }) => (
          <NavLink
            key={to}
            to={to}
            end={end}
            className={({ isActive }) =>
              cn(
                'flex flex-col items-center justify-center w-14 py-1 transition-colors active:scale-95',
                isActive
                  ? 'font-semibold text-foreground'
                  : 'text-muted-foreground hover:text-foreground',
              )
            }
          >
            {({ isActive }) => (
              <>
                <div
                  className={cn(
                    'flex h-7 w-7 items-center justify-center rounded-lg transition-colors',
                    isActive ? 'text-foreground' : 'text-muted-foreground',
                  )}
                >
                  <Icon className="h-5 w-5" />
                </div>
                <span
                  className={cn(
                    'text-[10px] tracking-tight',
                    isActive ? 'font-semibold text-foreground' : 'text-muted-foreground',
                  )}
                >
                  {label}
                </span>
              </>
            )}
          </NavLink>
        ))}
      </div>
    </nav>
  )
}
