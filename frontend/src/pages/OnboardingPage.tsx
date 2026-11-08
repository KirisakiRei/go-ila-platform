import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Check } from 'lucide-react'
import { Alert, Button, Card, CardContent, CardDescription, CardHeader, CardTitle, PageHeader, Skeleton } from '@/components/ui'
import { accountFlowApi, authApi } from '@/lib/auth-api'
import { cn } from '@/lib/utils'

/** LABELS menerjemahkan nama langkah teknis menjadi kalimat pendek. */
const LABELS: Record<string, { title: string; hint: string }> = {
  verifikasi_email: { title: 'Verifikasi email', hint: 'Wajib sebelum trial, broker, dan pembayaran.' },
  hubungkan_broker: { title: 'Hubungkan akun broker', hint: 'Gunakan token akun MatchTrader Anda.' },
  pilih_strategi: { title: 'Pilih strategi', hint: 'ILA 2.0 untuk semua paket; ILA 3.0 hanya Plus.' },
  hubungkan_telegram: { title: 'Hubungkan Telegram', hint: 'Notifikasi dan konfirmasi entry lewat tautan pendek.' },
  setel_risiko: { title: 'Setel risiko', hint: 'Persentase risiko per trade pada akun broker Anda.' },
}

/**
 * OnboardingPage memandu lima langkah pertama.
 *
 * Langkah yang sudah selesai tidak bisa mundur; pengguna yang keluar di
 * tengah jalan kembali ke langkah terakhirnya (plans/17 §5).
 */
export default function OnboardingPage() {
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ['me', 'onboarding'],
    queryFn: accountFlowApi.onboarding,
  })

  const invalidate = () => void queryClient.invalidateQueries({ queryKey: ['me', 'onboarding'] })

  const resend = useMutation({
    mutationFn: () => authApi.resendVerification(),
    onSuccess: invalidate,
  })

  const save = useMutation({
    mutationFn: (step: number) => accountFlowApi.saveOnboarding(step),
    onSuccess: invalidate,
  })

  return (
    <>
      <PageHeader
        title="Mulai"
        description="Lima langkah singkat agar sistem mengenal akun dan preferensi Anda."
      />

      {isLoading && <Skeleton className="h-64 w-full" />}

      {data && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">
              Langkah {Math.min(data.step + 1, data.total_steps)} dari {data.total_steps}
            </CardTitle>
            <CardDescription className="text-xs">
              Anda bisa menutup halaman ini kapan saja; posisi akan tersimpan.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            {data.steps.map((step, index) => {
              const done = index < data.step || (index === 0 && data.email_verified)
              const active = index === data.step
              const meta = LABELS[step] ?? { title: step, hint: '' }

              return (
                <div
                  key={step}
                  className={cn(
                    'flex items-start gap-3 rounded-md border p-3',
                    active ? 'border-foreground' : 'border-border',
                  )}
                >
                  <span
                    className={cn(
                      'mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full border text-[10px]',
                      done ? 'border-foreground bg-foreground text-background' : 'border-border',
                    )}
                    aria-hidden="true"
                  >
                    {done ? <Check className="h-3 w-3" /> : index + 1}
                  </span>
                  <div className="min-w-0 flex-1">
                    <p className="text-[13px] font-medium">{meta.title}</p>
                    <p className="text-[12px] text-muted-foreground">{meta.hint}</p>
                    {active && (
                      <div className="mt-2 flex flex-wrap gap-2">
                        {step === 'verifikasi_email' && (
                          <>
                            <Button size="sm" disabled={resend.isPending} onClick={() => resend.mutate()}>
                              Kirim tautan verifikasi
                            </Button>
                            {resend.isSuccess && <span className="text-[12px] text-muted-foreground">Tautan terkirim.</span>}
                          </>
                        )}
                        {step === 'hubungkan_broker' && (
                          <Button size="sm" onClick={() => (window.location.href = '/akun')}>
                            Buka halaman akun
                          </Button>
                        )}
                        {step === 'pilih_strategi' && (
                          <Button size="sm" onClick={() => (window.location.href = '/strategi')}>
                            Pilih strategi
                          </Button>
                        )}
                        {step === 'hubungkan_telegram' && (
                          <Button size="sm" onClick={() => (window.location.href = '/pengaturan')}>
                            Buka pengaturan
                          </Button>
                        )}
                        {step === 'setel_risiko' && (
                          <Button size="sm" onClick={() => (window.location.href = '/akun')}>
                            Atur risiko
                          </Button>
                        )}
                        <Button size="sm" variant="outline" onClick={() => save.mutate(index + 1)}>
                          Tandai selesai
                        </Button>
                      </div>
                    )}
                  </div>
                </div>
              )
            })}

            {!data.email_verified && (
              <Alert variant="warning">
                Beberapa fitur terkunci sampai email terverifikasi: trial, menghubungkan broker, dan pembayaran.
              </Alert>
            )}
          </CardContent>
        </Card>
      )}
    </>
  )
}
