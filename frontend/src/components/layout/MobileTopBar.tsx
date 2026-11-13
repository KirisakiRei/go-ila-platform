import { Link } from 'react-router-dom'
import { Bell, Menu, User as UserIcon } from 'lucide-react'
import type { Tier, User } from '@/types/api'

const TIER_LABEL: Record<Tier, string> = { free: 'Free', basic: 'Basic', plus: 'Plus' }

type MobileTopBarProps = {
  user: User
  onOpenMenu: () => void
}

export function MobileTopBar({ user, onOpenMenu }: MobileTopBarProps) {
  const tierName = TIER_LABEL[user.tier] ?? 'Free'

  return (
    <header className="sticky top-0 z-40 flex h-14 w-full items-center justify-between border-b border-border bg-card/90 px-4 backdrop-blur-md lg:hidden">
      {/* Kiri: Hamburger Menu + Logo ILA + Tier Badge */}
      <div className="flex items-center gap-2.5">
        <button
          type="button"
          aria-label="Buka Menu"
          onClick={onOpenMenu}
          className="flex h-9 w-9 items-center justify-center rounded-lg text-foreground hover:bg-secondary/60 transition-colors active:scale-95"
        >
          <Menu className="h-5 w-5" />
        </button>

        <Link to="/dashboard" className="flex items-center gap-1.5">
          <span className="text-base font-bold tracking-tight text-foreground">ILA</span>
          <span className="rounded bg-foreground px-1.5 py-0.5 text-[9.5px] font-bold uppercase tracking-wider text-background">
            {tierName}
          </span>
        </Link>
      </div>

      {/* Kanan: Notifikasi (menuju Sinyal) + Avatar Profil (menuju Pengaturan) */}
      <div className="flex items-center gap-2">
        <Link
          to="/sinyal"
          aria-label="Sinyal & Radar"
          className="relative flex h-9 w-9 items-center justify-center rounded-lg text-muted-foreground hover:bg-secondary/60 hover:text-foreground transition-colors"
        >
          <Bell className="h-5 w-5" />
          <span className="absolute top-2 right-2 h-2 w-2 rounded-full bg-emerald-500 ring-2 ring-card" />
        </Link>

        <Link
          to="/pengaturan"
          aria-label="Pengaturan Profil"
          className="flex h-8 w-8 items-center justify-center rounded-full bg-foreground text-background font-semibold text-xs transition-transform active:scale-95"
        >
          <UserIcon className="h-4 w-4" />
        </Link>
      </div>
    </header>
  )
}
