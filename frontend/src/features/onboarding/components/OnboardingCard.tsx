import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ArrowRight, CheckCircle2, Sparkles } from 'lucide-react'
import { accountFlowApi } from '@/lib/auth-api'
import { Button, Card, CardContent } from '@/components/ui'
import { OnboardingModal } from './OnboardingModal'

export function OnboardingCard() {
  const [modalOpen, setModalOpen] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ['me', 'onboarding'],
    queryFn: accountFlowApi.onboarding,
    staleTime: 60 * 1000,
  })

  if (isLoading || !data) return null
  if (data.done) return null

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

  // Kalau sudah beres 5 dari 5, tidak perlu tampilkan banner lagi
  if (completed >= total) return null

  return (
    <>
      <Card className="border-border bg-gradient-to-r from-card to-secondary/30">
        <CardContent className="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between sm:p-5">
          <div className="flex items-start gap-3.5">
            <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-foreground text-background">
              <Sparkles className="h-4 w-4" />
            </span>
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <h3 className="text-sm font-semibold tracking-tight">Lengkapi Penyiapan Akun</h3>
                <span className="inline-flex items-center gap-1 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5 text-[11px] font-medium text-emerald-700 dark:text-emerald-400">
                  <CheckCircle2 className="h-3 w-3" />
                  {completed} dari {total} langkah selesai ({percentage}%)
                </span>
              </div>
              <p className="mt-1 text-[12.5px] text-muted-foreground">
                Selesaikan langkah penyiapan agar akun siap menerima sinyal dan mengeksekusi order.
              </p>
              <div className="mt-2.5 h-1.5 w-48 overflow-hidden rounded-full bg-secondary sm:w-64">
                <div
                  className="h-full bg-emerald-500 transition-all duration-500"
                  style={{ width: `${percentage}%` }}
                />
              </div>
            </div>
          </div>

          <div className="flex items-center gap-2 self-end sm:self-center">
            <Button
              size="sm"
              variant="default"
              className="gap-1.5 text-xs font-semibold"
              onClick={() => setModalOpen(true)}
            >
              Lihat Checklist
              <ArrowRight className="h-3.5 w-3.5" />
            </Button>
          </div>
        </CardContent>
      </Card>

      <OnboardingModal forceOpen={modalOpen} onCloseForce={() => setModalOpen(false)} />
    </>
  )
}
