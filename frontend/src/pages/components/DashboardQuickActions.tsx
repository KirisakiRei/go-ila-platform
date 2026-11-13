import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  BarChart3,
  CreditCard,
  Grid,
  Landmark,
  ListChecks,
  Radio,
  ShieldCheck,
  Star,
} from 'lucide-react'
import { useAuthContext } from '@/features/auth/hooks/use-auth'
import { accountFlowApi } from '@/lib/auth-api'
import { useInviteOnly } from '@/lib/use-invite-only'
import type { Tier } from '@/types/api'

const TIER_SHORT: Record<Tier, string> = { free: 'Free', basic: 'Basic', plus: 'Plus' }

type DashboardQuickActionsProps = {
  onOpenMore: () => void
  onOpenOnboarding: () => void
}

export function DashboardQuickActions({ onOpenMore, onOpenOnboarding }: DashboardQuickActionsProps) {
  const { user } = useAuthContext()
  const inviteOnly = useInviteOnly()

  // Ambil progres checklist onboarding
  const { data: onboarding } = useQuery({
    queryKey: ['me', 'onboarding'],
    queryFn: accountFlowApi.onboarding,
    staleTime: 60 * 1000,
  })

  const stepDoneMap = onboarding?.step_done ?? {
    verifikasi_email: onboarding?.email_verified,
    hubungkan_broker: onboarding?.has_broker,
    pilih_strategi: Boolean(onboarding?.strategy_selected),
    hubungkan_telegram: onboarding?.telegram_linked,
    setel_risiko: Boolean(onboarding?.risk_configured),
  }
  const completed = Object.values(stepDoneMap).filter(Boolean).length
  const total = onboarding?.total_steps || 5
  const percentage = Math.round((completed / total) * 100) || 40

  const tier = (user?.tier ?? 'free') as Tier
  const tierLabel = TIER_SHORT[tier] ?? 'Free'

  return (
    <section className="flex flex-col gap-2">
      <h2 className="text-base font-bold tracking-tight text-foreground">Akses Cepat</h2>

      <div className="rounded-xl border border-border bg-card p-3 shadow-xs">
        <div className="grid grid-cols-4 gap-x-2 gap-y-3.5">
          {/* 1. Sinyal */}
          <Link
            to="/sinyal"
            className="flex flex-col items-center gap-1 text-center transition-transform active:scale-95 group"
          >
            <div className="relative flex h-11 w-11 items-center justify-center rounded-2xl bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 transition-colors group-hover:bg-emerald-500/25">
              <Radio className="h-5 w-5" />
              <span className="absolute top-1 right-1 h-2 w-2 rounded-full bg-emerald-500 ring-2 ring-card" />
            </div>
            <span className="text-micro font-medium text-foreground leading-tight">Sinyal</span>
          </Link>

          {/* 2. Posisi */}
          <Link
            to="/posisi"
            className="flex flex-col items-center gap-1 text-center transition-transform active:scale-95 group"
          >
            <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-secondary text-foreground transition-colors group-hover:bg-secondary/80">
              <BarChart3 className="h-5 w-5" />
            </div>
            <span className="text-micro font-medium text-foreground leading-tight">Posisi</span>
          </Link>

          {/* 3. Broker */}
          <Link
            to="/akun"
            className="flex flex-col items-center gap-1 text-center transition-transform active:scale-95 group"
          >
            <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-secondary text-foreground transition-colors group-hover:bg-secondary/80">
              <Landmark className="h-5 w-5" />
            </div>
            <span className="text-micro font-medium text-foreground leading-tight">Broker</span>
          </Link>

          {/* 4. Watchlist */}
          <Link
            to="/strategi"
            className="flex flex-col items-center gap-1 text-center transition-transform active:scale-95 group"
          >
            <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-secondary text-foreground transition-colors group-hover:bg-secondary/80">
              <Star className="h-5 w-5" />
            </div>
            <span className="text-micro font-medium text-foreground leading-tight">Watchlist</span>
          </Link>

          {/* 5. Checklist (dengan badge persentase) */}
          <button
            type="button"
            onClick={onOpenOnboarding}
            className="flex flex-col items-center gap-1 text-center transition-transform active:scale-95 group"
          >
            <div className="relative flex h-11 w-11 items-center justify-center rounded-2xl bg-secondary text-foreground transition-colors group-hover:bg-secondary/80">
              <ListChecks className="h-5 w-5" />
              <span className="absolute -top-1 -right-1 rounded-full bg-foreground px-1.5 text-[10px] font-bold text-background ring-1 ring-card leading-tight">
                {percentage}%
              </span>
            </div>
            <span className="text-micro font-medium text-foreground leading-tight">Checklist</span>
          </button>

          {/* 6. Billing (dengan badge tier) — disembunyikan saat invite only
              karena tidak ada paket yang bisa dibeli. */}
          {!inviteOnly && (
            <Link
              to="/langganan"
              className="flex flex-col items-center gap-1 text-center transition-transform active:scale-95 group"
            >
              <div className="relative flex h-11 w-11 items-center justify-center rounded-2xl bg-secondary text-foreground transition-colors group-hover:bg-secondary/80">
                <CreditCard className="h-5 w-5" />
                <span className="absolute -top-1 -right-1 rounded-full bg-emerald-600 px-1.5 text-[10px] font-bold text-white ring-1 ring-card leading-tight">
                  {tierLabel}
                </span>
              </div>
              <span className="text-micro font-medium text-foreground leading-tight">Billing</span>
            </Link>
          )}

          {/* 7. Keamanan */}
          <Link
            to="/pengaturan/keamanan"
            className="flex flex-col items-center gap-1 text-center transition-transform active:scale-95 group"
          >
            <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-secondary text-foreground transition-colors group-hover:bg-secondary/80">
              <ShieldCheck className="h-5 w-5" />
            </div>
            <span className="text-micro font-medium text-foreground leading-tight">Keamanan</span>
          </Link>

          {/* 8. Lainnya */}
          <button
            type="button"
            onClick={onOpenMore}
            className="flex flex-col items-center gap-1 text-center transition-transform active:scale-95 group"
          >
            <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-secondary text-foreground transition-colors group-hover:bg-secondary/80">
              <Grid className="h-5 w-5" />
            </div>
            <span className="text-micro font-medium text-foreground leading-tight">Lainnya</span>
          </button>
        </div>
      </div>
    </section>
  )
}
