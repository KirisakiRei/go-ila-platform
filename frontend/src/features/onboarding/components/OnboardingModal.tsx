import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Sparkles } from 'lucide-react'
import { accountFlowApi } from '@/lib/auth-api'
import { Button, Modal } from '@/components/ui'
import { OnboardingChecklist } from './OnboardingChecklist'

const DISMISS_KEY = 'ila_dismiss_onboarding_modal'

type OnboardingModalProps = {
  forceOpen?: boolean
  onCloseForce?: () => void
}

export function OnboardingModal({ forceOpen, onCloseForce }: OnboardingModalProps = {}) {
  const queryClient = useQueryClient()
  const [userDismissed, setUserDismissed] = useState(() => {
    return typeof window !== 'undefined' && sessionStorage.getItem(DISMISS_KEY) === 'true'
  })

  const { data } = useQuery({
    queryKey: ['me', 'onboarding'],
    queryFn: accountFlowApi.onboarding,
    staleTime: 60 * 1000,
  })

  const skipAll = useMutation({
    mutationFn: () => accountFlowApi.saveOnboarding(5, true),
    onSuccess: () => {
      handleDismiss()
      void queryClient.invalidateQueries({ queryKey: ['me', 'onboarding'] })
    },
  })

  const shouldAutoOpen = Boolean(data && !data.done && !userDismissed)
  const isOpen = forceOpen !== undefined ? forceOpen : shouldAutoOpen

  const handleDismiss = () => {
    sessionStorage.setItem(DISMISS_KEY, 'true')
    setUserDismissed(true)
    if (onCloseForce) onCloseForce()
  }

  if (!data) return null

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

  return (
    <Modal
      open={isOpen}
      onClose={handleDismiss}
      size="xl"
      title={
        <div className="flex items-center gap-2.5">
          <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-zinc-900 text-white dark:bg-white dark:text-zinc-900">
            <Sparkles className="h-4 w-4" />
          </span>
          <div>
            <h2 className="text-base font-semibold tracking-tight">Penyiapan Akun</h2>
            <p className="text-xs font-normal text-muted-foreground">
              {completed === total
                ? 'Semua langkah telah selesai dikonfigurasi!'
                : `${completed} dari ${total} langkah penyiapan selesai (${percentage}%)`}
            </p>
          </div>
        </div>
      }
      description={
        <div className="mt-2 space-y-2">
          <p className="text-[12.5px] text-muted-foreground">
            Lengkapi langkah di bawah ini agar akun Anda siap menerima sinyal trading dan mengeksekusi
            order secara aman.
          </p>
          <div className="h-1.5 w-full overflow-hidden rounded-full bg-secondary">
            <div
              className="h-full bg-emerald-500 transition-all duration-500 ease-out"
              style={{ width: `${percentage}%` }}
            />
          </div>
        </div>
      }
      footer={
        <div className="flex w-full flex-wrap items-center justify-between gap-2">
          <Button
            variant="ghost"
            size="sm"
            className="text-xs text-muted-foreground hover:text-foreground"
            onClick={() => skipAll.mutate()}
            disabled={skipAll.isPending}
          >
            {skipAll.isPending ? 'Menyimpan...' : 'Tandai Selesai & Lewati'}
          </Button>

          <Button variant="default" size="sm" onClick={handleDismiss}>
            {completed === total ? 'Tutup & Mulai Trading' : 'Nanti Saja'}
          </Button>
        </div>
      }
    >
      <div className="max-h-[60vh] overflow-y-auto px-1 py-1">
        <OnboardingChecklist data={data} onNavigate={handleDismiss} />
      </div>
    </Modal>
  )
}
