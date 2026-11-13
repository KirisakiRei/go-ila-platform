import { useAuthContext } from '@/features/auth/hooks/use-auth'
import { useSystemStatus } from '@/lib/queries'
import type { Tier } from '@/types/api'

const TIER_LABEL: Record<Tier, string> = { free: 'FREE', basic: 'BASIC', plus: 'PLUS' }

export function DashboardGreetingBar() {
  const { user } = useAuthContext()
  const { data: status } = useSystemStatus()

  if (!user) return null

  const tier = (user.tier ?? 'free') as Tier
  const tierName = TIER_LABEL[tier] ?? 'FREE'

  const connectedBrokers = status?.brokers?.connected ?? 0
  const totalBrokers = status?.brokers?.total ?? 0

  return (
    <div className="flex items-center justify-between border-b border-border/60 pb-3 pt-1">
      {/* Kiri: Halo, Nama User + Tier Badge */}
      <div className="flex items-center gap-2">
        <h1 className="text-base font-bold tracking-tight text-foreground sm:text-lg">
          Halo, {user.name}
        </h1>
        <span className="rounded bg-foreground px-1.5 py-0.5 text-micro font-bold uppercase tracking-wider text-background">
          {tierName}
        </span>
      </div>

      {/* Kanan: Live Broker Indicator */}
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
        <span className="h-2 w-2 rounded-full bg-emerald-500 shrink-0" />
        <span className="text-caption">
          Live Broker · <strong className="font-semibold text-foreground">{connectedBrokers}/{totalBrokers} Aktif</strong>
        </span>
      </div>
    </div>
  )
}
