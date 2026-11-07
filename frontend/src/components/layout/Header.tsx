import { Link } from 'react-router-dom'
import { useSignOut } from '@/features/auth/hooks/use-auth'
import { Badge, Button } from '@/components/ui'
import type { Tier, User } from '@/types/api'

const TIER_LABEL: Record<Tier, string> = {
  free: 'Free',
  basic: 'Basic',
  plus: 'Plus',
}

const TIER_VARIANT = {
  free: 'default',
  basic: 'primary',
  plus: 'accent',
} as const

type HeaderProps = {
  user: User
}

export function Header({ user }: HeaderProps) {
  const signOut = useSignOut()

  return (
    <header className="sticky top-0 z-10 border-b border-border bg-background/80 backdrop-blur">
      <div className="mx-auto flex h-14 w-full max-w-5xl items-center justify-between px-4">
        <Link to="/" className="flex items-baseline gap-1.5" aria-label="ILA Bot, halaman utama">
          <span className="text-base font-bold tracking-tight">ILA</span>
          <span className="text-[11px] uppercase tracking-[0.18em] text-muted-foreground">Bot</span>
        </Link>

        <div className="flex items-center gap-3">
          <Badge variant={TIER_VARIANT[user.tier]}>{TIER_LABEL[user.tier]}</Badge>
          <span className="hidden text-[13px] text-muted-foreground sm:inline">{user.name}</span>
          <Button variant="ghost" size="sm" onClick={signOut}>
            Keluar
          </Button>
        </div>
      </div>
    </header>
  )
}
