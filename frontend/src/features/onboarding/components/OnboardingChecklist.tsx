import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { ArrowRight, Check, CheckCircle2, Send } from 'lucide-react'
import { authApi } from '@/lib/auth-api'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui'
import type { OnboardingState } from '@/types/api'

type StepMeta = {
  title: string
  hint: string
  actionLabel: string
  href: string
}

const STEP_META: Record<string, StepMeta> = {
  verifikasi_email: {
    title: 'Verifikasi Email',
    hint: 'Wajib aktif sebelum simulasi, menghubungkan broker, dan memilih paket.',
    actionLabel: 'Kirim tautan verifikasi',
    href: '',
  },
  hubungkan_broker: {
    title: 'Hubungkan Akun Broker',
    hint: 'Gunakan cTrader OAuth atau access token broker Anda untuk eksekusi sinyal.',
    actionLabel: 'Buka halaman akun',
    href: '/akun',
  },
  pilih_strategi: {
    title: 'Pilih Watchlist Pair',
    hint: 'Tentukan instrumen pair dan metode analisis ILA yang ingin Anda pantau.',
    actionLabel: 'Pilih pair',
    href: '/strategi',
  },
  hubungkan_telegram: {
    title: 'Hubungkan Telegram',
    hint: 'Terima peringatan real-time dan konfirmasi entry sinyal lewat Telegram bot.',
    actionLabel: 'Buka pengaturan Telegram',
    href: '/pengaturan',
  },
  setel_risiko: {
    title: 'Setel Aturan Risiko',
    hint: 'Tetapkan persentase risiko per trade atau ukuran lot untuk akun trading Anda.',
    actionLabel: 'Atur risiko broker',
    href: '/akun',
  },
}

type OnboardingChecklistProps = {
  data: OnboardingState
  onNavigate?: () => void
}

export function OnboardingChecklist({ data, onNavigate }: OnboardingChecklistProps) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [resendNotice, setResendNotice] = useState<string | null>(null)

  const resend = useMutation({
    mutationFn: () => authApi.resendVerification(),
    onSuccess: () => {
      setResendNotice('Tautan verifikasi baru berhasil dikirim ke email Anda.')
      void queryClient.invalidateQueries({ queryKey: ['me', 'onboarding'] })
    },
    onError: (err) => {
      setResendNotice(err instanceof Error ? err.message : 'Gagal mengirim tautan verifikasi.')
    },
  })

  const stepDoneMap = data.step_done ?? {
    verifikasi_email: data.email_verified,
    hubungkan_broker: data.has_broker,
    pilih_strategi: Boolean(data.strategy_selected),
    hubungkan_telegram: data.telegram_linked,
    setel_risiko: Boolean(data.risk_configured),
  }

  const handleGo = (href: string) => {
    if (onNavigate) onNavigate()
    navigate(href)
  }

  return (
    <div className="space-y-3">
      {resendNotice && (
        <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 p-3 text-[12px] font-medium text-emerald-800 dark:text-emerald-300">
          {resendNotice}
        </div>
      )}

      <div className="space-y-2.5">
        {data.steps.map((stepKey, index) => {
          const isDone = Boolean(stepDoneMap[stepKey])
          const meta = STEP_META[stepKey] ?? {
            title: stepKey,
            hint: '',
            actionLabel: 'Buka',
            href: '/',
          }

          return (
            <div
              key={stepKey}
              className={cn(
                'flex flex-col gap-3 rounded-xl border p-4 transition-colors sm:flex-row sm:items-center sm:justify-between',
                isDone
                  ? 'border-emerald-500/30 bg-emerald-500/[0.03]'
                  : 'border-border bg-card hover:border-zinc-300',
              )}
            >
              <div className="flex items-start gap-3.5">
                <span
                  className={cn(
                    'mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full border text-xs font-semibold transition-transform',
                    isDone
                      ? 'border-emerald-500 bg-emerald-500 text-white shadow-sm shadow-emerald-500/20'
                      : 'border-zinc-300 bg-secondary/80 text-foreground',
                  )}
                  aria-label={isDone ? 'Selesai' : `Langkah ${index + 1}`}
                >
                  {isDone ? <Check className="h-4 w-4 stroke-[2.5]" /> : index + 1}
                </span>

                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <p
                      className={cn(
                        'text-[13.5px] font-semibold tracking-tight',
                        isDone ? 'text-foreground' : 'text-foreground',
                      )}
                    >
                      {meta.title}
                    </p>
                    {isDone ? (
                      <span className="inline-flex items-center gap-1 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5 text-micro font-medium text-emerald-700 dark:text-emerald-400">
                        <CheckCircle2 className="h-3 w-3" />
                        Selesai
                      </span>
                    ) : (
                      <span className="rounded-full border border-zinc-200 bg-zinc-100 px-2 py-0.5 text-micro font-medium text-muted-foreground">
                        Belum selesai
                      </span>
                    )}
                  </div>
                  <p className="mt-1 text-[12px] leading-relaxed text-muted-foreground">
                    {meta.hint}
                  </p>
                </div>
              </div>

              {!isDone && (
                <div className="flex shrink-0 items-center gap-2 pl-10 sm:pl-0">
                  {stepKey === 'verifikasi_email' ? (
                    <Button
                      size="sm"
                      variant="outline"
                      className="gap-1.5 text-xs font-medium"
                      disabled={resend.isPending}
                      onClick={() => resend.mutate()}
                    >
                      <Send className="h-3.5 w-3.5" />
                      {resend.isPending ? 'Mengirim...' : 'Kirim Ulang Email'}
                    </Button>
                  ) : (
                    <Button
                      size="sm"
                      variant="default"
                      className="gap-1.5 text-xs font-semibold"
                      onClick={() => handleGo(meta.href)}
                    >
                      {meta.actionLabel}
                      <ArrowRight className="h-3.5 w-3.5" />
                    </Button>
                  )}
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
