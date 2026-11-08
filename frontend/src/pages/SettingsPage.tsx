import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { ArrowRight, CheckCircle2 } from 'lucide-react'
import { Alert, Badge, Button, PageHeader, Section, Skeleton } from '@/components/ui'
import { useStartTrial, useTelegramLink, useTelegramUnlink, useTrial } from '@/lib/queries'
import { accountFlowApi } from '@/lib/auth-api'
import { ApiError } from '@/lib/api-error'
import { formatDate, formatDateTime } from '@/lib/format'

function TrialSection() {
  const { data, isLoading, isError, error } = useTrial()
  const mulai = useStartTrial()
  const [galat, setGalat] = useState<string | null>(null)

  const { data: onboarding, isLoading: isLoadingOnboarding } = useQuery({
    queryKey: ['me', 'onboarding'],
    queryFn: accountFlowApi.onboarding,
  })

  const isEmailVerified = Boolean(onboarding?.email_verified)
  const stepDoneMap = onboarding?.step_done ?? {
    verifikasi_email: Boolean(onboarding?.email_verified),
    hubungkan_broker: Boolean(onboarding?.has_broker),
    pilih_strategi: Boolean(onboarding?.strategy_selected),
    hubungkan_telegram: Boolean(onboarding?.telegram_linked),
    setel_risiko: Boolean(onboarding?.risk_configured),
  }
  const completedSteps = Object.values(stepDoneMap).filter(Boolean).length
  const totalSteps = onboarding?.total_steps || 5
  const isOnboardingDone = Boolean(onboarding?.done || completedSteps >= totalSteps)

  const canStartTrial = isEmailVerified && isOnboardingDone

  const onMulai = async () => {
    setGalat(null)
    try {
      await mulai.mutateAsync()
    } catch (err) {
      setGalat(err instanceof ApiError ? err.message : 'Gagal memulai uji coba.')
    }
  }

  return (
    <Section
      title="Uji coba 7 hari"
      description="Setara tier Plus, namun seluruh eksekusi berjalan sebagai simulasi."
    >
      <div className="flex flex-col gap-3 border border-border p-4">
        {(isLoading || isLoadingOnboarding) && <Skeleton className="h-16 w-full" />}
        {isError && <Alert variant="destructive">{(error as Error).message}</Alert>}

        {data && !isLoadingOnboarding && (
          <>
            {data.active ? (
              <div className="flex flex-col gap-2 text-[13px]">
                <div className="flex flex-wrap items-center gap-2">
                  <Badge variant="solid">Aktif</Badge>
                  <span>
                    sisa <strong>{data.days_remaining} hari</strong>
                  </span>
                  <Badge variant="outline">Simulasi wajib</Badge>
                </div>
                <p className="text-[12px] text-muted-foreground">
                  Mulai {formatDateTime(data.trial_started_at)} · berakhir{' '}
                  {formatDate(data.trial_expires_at)}
                </p>
              </div>
            ) : data.used ? (
              <p className="text-[13px] text-muted-foreground">
                Uji coba sudah pernah dipakai. Masa berlaku tidak dapat diulang.
              </p>
            ) : (
              <div className="flex flex-col gap-3">
                <p className="text-[13px] text-muted-foreground">
                  Anda belum memakai uji coba. Uji coba hanya bisa dipakai sekali per akun.
                </p>

                {!isEmailVerified && (
                  <Alert variant="warning" className="text-xs">
                    <div className="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:justify-between">
                      <span>
                        Verifikasi email diperlukan sebelum memulai uji coba 7 hari.
                      </span>
                      <Link
                        to="/pengaturan/keamanan"
                        className="inline-flex items-center gap-1 font-semibold text-amber-800 underline dark:text-amber-300"
                      >
                        Verifikasi Email <ArrowRight className="h-3 w-3" />
                      </Link>
                    </div>
                  </Alert>
                )}

                {isEmailVerified && !isOnboardingDone && (
                  <Alert variant="warning" className="text-xs">
                    <div className="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:justify-between">
                      <span>
                        Selesaikan 5 langkah penyiapan akun terlebih dahulu ({completedSteps}/{totalSteps} selesai) agar preferensi broker dan risiko Anda siap.
                      </span>
                      <Link
                        to="/mulai"
                        className="inline-flex items-center gap-1 font-semibold text-amber-800 underline dark:text-amber-300"
                      >
                        Lanjutkan Penyiapan <ArrowRight className="h-3 w-3" />
                      </Link>
                    </div>
                  </Alert>
                )}

                {canStartTrial && (
                  <div className="flex items-center gap-2 text-xs text-emerald-600 dark:text-emerald-400">
                    <CheckCircle2 className="h-3.5 w-3.5" />
                    <span>Persyaratan terpenuhi. Anda dapat memulai uji coba simulasi sekarang.</span>
                  </div>
                )}

                <Button
                  size="sm"
                  className="self-start"
                  disabled={!canStartTrial || mulai.isPending}
                  isLoading={mulai.isPending}
                  onClick={() => void onMulai()}
                >
                  Mulai uji coba
                </Button>
              </div>
            )}
          </>
        )}

        {galat && <Alert variant="destructive">{galat}</Alert>}
      </div>
    </Section>
  )
}

function TelegramSection() {
  const link = useTelegramLink()
  const unlink = useTelegramUnlink()
  const [galat, setGalat] = useState<string | null>(null)

  const onLink = async () => {
    setGalat(null)
    try {
      const hasil = await link.mutateAsync()
      // Dibuka di tab baru agar sesi dashboard tidak hilang.
      window.open(hasil.url, '_blank', 'noopener,noreferrer')
    } catch (err) {
      setGalat(err instanceof ApiError ? err.message : 'Gagal membuat tautan Telegram.')
    }
  }

  return (
    <Section
      title="Notifikasi Telegram"
      description="Bot hanya mengirim pemberitahuan. Bot tidak pernah mengakses kredensial akun Anda."
    >
      <div className="flex flex-col gap-3 border border-border p-4">
        <div className="flex flex-wrap gap-2">
          <Button size="sm" isLoading={link.isPending} onClick={() => void onLink()}>
            Hubungkan Telegram
          </Button>
          <Button
            size="sm"
            variant="outline"
            isLoading={unlink.isPending}
            onClick={() => void unlink.mutateAsync()}
          >
            Putuskan
          </Button>
        </div>

        <p className="text-[11px] text-muted-foreground">
          Tautan berlaku 5 menit dan hanya bisa dipakai sekali.
        </p>

        {galat && <Alert variant="destructive">{galat}</Alert>}
        {unlink.isSuccess && <Alert variant="success">Chat Telegram sudah diputus.</Alert>}
      </div>
    </Section>
  )
}

export default function SettingsPage() {
  return (
    <>
      <PageHeader
        title="Pengaturan"
        description="Uji coba, saluran notifikasi, dan keamanan akun."
      />
      <TrialSection />
      <TelegramSection />

      <Section
        title="Keamanan"
        description="Kata sandi, verifikasi email, 2FA, dan perangkat yang sedang masuk."
      >
        <div className="flex flex-col gap-3 border border-border p-4">
          <p className="text-[13px] text-muted-foreground">
            Pengaturan keamanan lengkap dipisah ke halaman sendiri agar tidak tercampur dengan uji coba
            dan notifikasi.
          </p>
          <Link
            to="/pengaturan/keamanan"
            className="inline-flex w-fit rounded-md border border-border px-3 py-1.5 text-[13px] font-medium hover:bg-secondary"
          >
            Buka pengaturan keamanan
          </Link>
        </div>
      </Section>
    </>
  )
}
