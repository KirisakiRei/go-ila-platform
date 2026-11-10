import { useQuery } from '@tanstack/react-query'
import { ArrowRight, CheckCircle2 } from 'lucide-react'
import { accountFlowApi } from '@/lib/auth-api'

type DashboardOnboardingWidgetProps = {
  onOpenOnboarding: () => void
}

export function DashboardOnboardingWidget({ onOpenOnboarding }: DashboardOnboardingWidgetProps) {
  const { data, isLoading } = useQuery({
    queryKey: ['me', 'onboarding'],
    queryFn: accountFlowApi.onboarding,
    staleTime: 60 * 1000,
  })

  if (isLoading || !data || data.done) return null

  const stepDoneMap = data.step_done ?? {
    verifikasi_email: data.email_verified,
    hubungkan_broker: data.has_broker,
    pilih_strategi: Boolean(data.strategy_selected),
    hubungkan_telegram: data.telegram_linked,
    setel_risiko: Boolean(data.risk_configured),
  }

  const completed = Object.values(stepDoneMap).filter(Boolean).length
  const total = data.total_steps || 5
  const percentage = Math.round((completed / total) * 100)

  // Kalau sudah selesai semua, tidak perlu tampil lagi
  if (completed >= total) return null

  return (
    <div className="flex flex-col gap-1.5 rounded-xl border border-border bg-card px-3.5 py-2.5 shadow-xs">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5 min-w-0">
          <CheckCircle2 className="h-4 w-4 text-emerald-600 dark:text-emerald-400 shrink-0" />
          <span className="truncate text-xs font-semibold text-foreground">
            Penyiapan Akun · {completed} dari {total} selesai ({percentage}%)
          </span>
        </div>

        <button
          type="button"
          onClick={onOpenOnboarding}
          className="inline-flex items-center gap-0.5 text-xs font-bold text-foreground hover:underline shrink-0 ml-2"
        >
          <span>Lanjutkan</span>
          <ArrowRight className="h-3.5 w-3.5" />
        </button>
      </div>

      <div className="h-1 w-full overflow-hidden rounded-full bg-secondary">
        <div
          className="h-full rounded-full bg-emerald-500 transition-all duration-500"
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  )
}
