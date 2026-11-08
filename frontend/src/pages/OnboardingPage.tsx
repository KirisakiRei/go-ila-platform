import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ArrowRight, BookOpen, CheckCircle2, HelpCircle, ShieldAlert, Sparkles } from 'lucide-react'
import {
  Alert,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  PageHeader,
  Skeleton,
} from '@/components/ui'
import { accountFlowApi } from '@/lib/auth-api'
import { OnboardingChecklist } from '@/features/onboarding'

/**
 * OnboardingPage memandu lima langkah pertama penyiapan akun.
 * Langkah yang selesai otomatis ditandai centang hijau dari basis data.
 */
export default function OnboardingPage() {
  const queryClient = useQueryClient()

  const { data, isLoading } = useQuery({
    queryKey: ['me', 'onboarding'],
    queryFn: accountFlowApi.onboarding,
  })

  const invalidate = () => void queryClient.invalidateQueries({ queryKey: ['me', 'onboarding'] })

  const skipAll = useMutation({
    mutationFn: () => accountFlowApi.saveOnboarding(5, true),
    onSuccess: invalidate,
  })

  const stepDoneMap = data?.step_done ?? {
    verifikasi_email: Boolean(data?.email_verified),
    hubungkan_broker: Boolean(data?.has_broker),
    pilih_strategi: Boolean(data?.strategy_selected),
    hubungkan_telegram: Boolean(data?.telegram_linked),
    setel_risiko: Boolean(data?.risk_configured),
  }

  const completed = Object.values(stepDoneMap).filter(Boolean).length
  const total = data?.total_steps || 5
  const percentage = Math.round((completed / total) * 100)

  return (
    <>
      <PageHeader
        title="Mulai"
        description="Lima langkah singkat agar sistem mengenal akun, broker, dan preferensi risiko Anda."
        actions={
          data && !data.done && completed < total ? (
            <Button
              variant="outline"
              size="sm"
              disabled={skipAll.isPending}
              onClick={() => skipAll.mutate()}
            >
              {skipAll.isPending ? 'Menyimpan...' : 'Tandai Selesai / Lewati'}
            </Button>
          ) : null
        }
      />

      {isLoading && <Skeleton className="h-64 w-full" />}

      {data && (
        <div className="space-y-4">
          <Card>
            <CardHeader className="pb-4">
              <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <div className="flex items-center gap-2">
                    <span className="flex h-6 w-6 items-center justify-center rounded-md bg-foreground text-background">
                      <Sparkles className="h-3.5 w-3.5" />
                    </span>
                    <CardTitle className="text-sm font-semibold">Progres Penyiapan Akun</CardTitle>
                  </div>
                  <CardDescription className="mt-1 text-xs">
                    {completed === total
                      ? 'Semua langkah penyiapan sudah selesai!'
                      : `${completed} dari ${total} langkah selesai (${percentage}%). Posisi tersimpan otomatis.`}
                  </CardDescription>
                </div>
                <span className="inline-flex items-center gap-1.5 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-3 py-1 text-xs font-semibold text-emerald-700 dark:text-emerald-400">
                  <CheckCircle2 className="h-3.5 w-3.5" />
                  {percentage}% Selesai
                </span>
              </div>
              <div className="mt-4 h-2 w-full overflow-hidden rounded-full bg-secondary">
                <div
                  className="h-full bg-emerald-500 transition-all duration-500 ease-out"
                  style={{ width: `${percentage}%` }}
                />
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              <OnboardingChecklist data={data} onNavigate={invalidate} />

              {!data.email_verified && (
                <Alert variant="warning" className="text-xs">
                  Beberapa fitur terkunci sampai email terverifikasi: uji coba simulasi,
                  menghubungkan akun broker, dan pembayaran langganan.
                </Alert>
              )}
            </CardContent>
          </Card>

          {/* Navigasi Cepat Pusat Bantuan, FAQ, dan Legal */}
          <div className="grid grid-cols-1 gap-4 pt-2 md:grid-cols-3">
            <Card className="flex flex-col justify-between">
              <CardHeader className="pb-3">
                <div className="flex items-center gap-2">
                  <span className="flex h-8 w-8 items-center justify-center rounded-lg border border-border bg-secondary text-foreground">
                    <BookOpen className="h-4 w-4" />
                  </span>
                  <CardTitle className="text-sm font-semibold">Pusat Bantuan</CardTitle>
                </div>
                <CardDescription className="mt-2 text-xs leading-relaxed">
                  Panduan langkah awal, cara mendapatkan token Match-Trader, dan pengaturan bot Telegram.
                </CardDescription>
              </CardHeader>
              <CardContent className="pt-0">
                <a
                  href="/bantuan"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1.5 text-xs font-medium text-foreground hover:underline"
                >
                  Buka Panduan & Tutorial <ArrowRight className="h-3 w-3" />
                </a>
              </CardContent>
            </Card>

            <Card className="flex flex-col justify-between">
              <CardHeader className="pb-3">
                <div className="flex items-center gap-2">
                  <span className="flex h-8 w-8 items-center justify-center rounded-lg border border-border bg-secondary text-foreground">
                    <HelpCircle className="h-4 w-4" />
                  </span>
                  <CardTitle className="text-sm font-semibold">Tanya Jawab (FAQ)</CardTitle>
                </div>
                <CardDescription className="mt-2 text-xs leading-relaxed">
                  Jawaban cepat seputar sesi pasar New York vs WIB, batasan risiko, dan eksekusi order.
                </CardDescription>
              </CardHeader>
              <CardContent className="pt-0">
                <a
                  href="/bantuan#faq-bantuan"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1.5 text-xs font-medium text-foreground hover:underline"
                >
                  Lihat Pertanyaan Umum <ArrowRight className="h-3 w-3" />
                </a>
              </CardContent>
            </Card>

            <Card className="flex flex-col justify-between">
              <CardHeader className="pb-3">
                <div className="flex items-center gap-2">
                  <span className="flex h-8 w-8 items-center justify-center rounded-lg border border-border bg-secondary text-foreground">
                    <ShieldAlert className="h-4 w-4" />
                  </span>
                  <CardTitle className="text-sm font-semibold">Legal & Risiko</CardTitle>
                </div>
                <CardDescription className="mt-2 text-xs leading-relaxed">
                  Ketentuan Layanan, Kebijakan Privasi, dan Pernyataan Disclaimer Risiko Trading.
                </CardDescription>
              </CardHeader>
              <CardContent className="pt-0">
                <a
                  href="/legal"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1.5 text-xs font-medium text-foreground hover:underline"
                >
                  Baca Dokumen Legal <ArrowRight className="h-3 w-3" />
                </a>
              </CardContent>
            </Card>
          </div>
        </div>
      )}
    </>
  )
}
