import { useState } from 'react'
import { Link } from 'react-router-dom'
import {
  ArrowRight,
  Bell,
  Bot,
  CheckCircle2,
  ChevronRight,
  Clock,
  CreditCard,
  FileText,
  HelpCircle,
  LogOut,
  Mail,
  ShieldCheck,
  Sparkles,
  User as UserIcon,
  Wallet,
} from 'lucide-react'
import { Alert, Badge, Button, Modal, Skeleton } from '@/components/ui'
import { useSignOut, useAuthContext } from '@/features/auth/hooks/use-auth'
import { useStartTrial, useTelegramLink, useTelegramUnlink, useTrial } from '@/lib/queries'
import { accountFlowApi } from '@/lib/auth-api'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { ApiError } from '@/lib/api-error'
import { formatDate } from '@/lib/format'
import { cn } from '@/lib/utils'
import type { User } from '@/types/api'

const TIER_LABEL: Record<string, string> = { free: 'Free', basic: 'Basic', plus: 'Plus' }

type MenuRowProps = {
  icon: typeof UserIcon
  label: string
  description: string
  to?: string
  onClick?: () => void
  trailing?: string
  disabled?: boolean
}

function MenuRow({ icon: Icon, label, description, to, onClick, trailing, disabled }: MenuRowProps) {
  const inner = (
    <>
      <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-border bg-secondary text-foreground">
        <Icon className="h-4 w-4" />
      </span>
      <div className="min-w-0 flex-1">
        <p className="truncate text-[12.5px] font-semibold text-foreground">{label}</p>
        <p className="truncate text-[10.5px] text-muted-foreground">{description}</p>
      </div>
      {trailing && (
        <span
          className={cn(
            'shrink-0 rounded-md border px-1.5 py-0.5 text-[9.5px] font-semibold uppercase tracking-wide',
            trailing === 'Terverifikasi' || trailing === 'Terhubung'
              ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400'
              : 'border-border bg-secondary text-muted-foreground',
          )}
        >
          {trailing}
        </span>
      )}
      <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />
    </>
  )

  const className = cn(
    'flex w-full items-center gap-2.5 px-3.5 py-2.5 text-left transition-colors active:bg-secondary/50',
    disabled && 'pointer-events-none opacity-50',
  )

  if (to) {
    return (
      <Link to={to} className={className}>
        {inner}
      </Link>
    )
  }

  return (
    <button type="button" onClick={onClick} className={className} disabled={disabled}>
      {inner}
    </button>
  )
}

function Group({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="rounded-2xl border border-border bg-card shadow-xs">
      <h2 className="border-b border-border px-3.5 py-2.5 text-[10px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">
        {title}
      </h2>
      <div className="divide-y divide-border/70">{children}</div>
    </section>
  )
}

function TrialCard() {
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

  if (isLoading || isLoadingOnboarding) {
    return <Skeleton className="h-24 w-full rounded-2xl" />
  }

  if (isError) {
    return (
      <div className="rounded-2xl border border-destructive/40 bg-destructive/5 p-3.5 text-[11.5px] text-destructive">
        {(error as Error).message}
      </div>
    )
  }

  if (!data) return null

  if (data.active) {
    return (
      <div className="rounded-2xl border border-emerald-500/30 bg-emerald-500/[0.06] p-3.5">
        <div className="flex items-start gap-2.5">
          <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-emerald-500/30 bg-background text-emerald-600 dark:text-emerald-400">
            <Sparkles className="h-4 w-4" />
          </span>
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-1.5">
              <p className="text-[12.5px] font-bold text-foreground">Uji Coba 7 Hari Aktif</p>
              <Badge variant="solid" className="text-[9px]">
                {data.days_remaining} hari lagi
              </Badge>
            </div>
            <p className="mt-1 text-[10.5px] leading-relaxed text-muted-foreground">
              Setara tier Plus, seluruh eksekusi berjalan sebagai simulasi hingga{' '}
              {formatDate(data.trial_expires_at)}.
            </p>
          </div>
        </div>
      </div>
    )
  }

  if (data.used) {
    return (
      <div className="rounded-2xl border border-border bg-card p-3.5">
        <div className="flex items-start gap-2.5">
          <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-border bg-secondary text-muted-foreground">
            <Clock className="h-4 w-4" />
          </span>
          <div className="min-w-0">
            <p className="text-[12.5px] font-semibold text-foreground">Uji Coba Sudah Dipakai</p>
            <p className="mt-0.5 text-[10.5px] leading-relaxed text-muted-foreground">
              Masa uji coba hanya bisa dipakai sekali per akun dan tidak dapat diulang.
            </p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="rounded-2xl border border-border bg-card p-3.5">
      <div className="flex items-start gap-2.5">
        <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-border bg-secondary text-foreground">
          <Sparkles className="h-4 w-4" />
        </span>
        <div className="min-w-0 flex-1">
          <p className="text-[12.5px] font-semibold text-foreground">Coba Tier Plus Gratis</p>
          <p className="mt-0.5 text-[10.5px] leading-relaxed text-muted-foreground">
            Nikmati seluruh fitur tier Plus selama 7 hari dengan eksekusi simulasi.
          </p>
        </div>
      </div>

      {!canStartTrial && (
        <div className="mt-2.5 flex items-start gap-1.5 rounded-lg border border-amber-500/30 bg-amber-500/10 px-2.5 py-2">
          <Mail className="mt-0.5 h-3.5 w-3.5 shrink-0 text-amber-700 dark:text-amber-400" />
          <p className="text-[10.5px] leading-relaxed text-amber-800 dark:text-amber-300">
            {!isEmailVerified
              ? 'Verifikasi email diperlukan sebelum memulai uji coba.'
              : `Selesaikan penyiapan akun dulu (${completedSteps}/${totalSteps} langkah).`}
          </p>
        </div>
      )}

      {galat && <p className="mt-2 text-[10.5px] font-medium text-destructive">{galat}</p>}

      <div className="mt-2.5 flex items-center gap-2">
        <Button
          size="sm"
          className="flex-1 text-[11px] font-semibold"
          disabled={!canStartTrial || mulai.isPending}
          isLoading={mulai.isPending}
          onClick={() => void onMulai()}
        >
          Mulai Uji Coba
        </Button>
        {!isEmailVerified ? (
          <Button size="sm" variant="outline" className="gap-1 text-[11px] font-semibold" asChild>
            <Link to="/pengaturan/keamanan">
              Verifikasi
              <ArrowRight className="h-3 w-3" />
            </Link>
          </Button>
        ) : (
          !isOnboardingDone && (
            <Button size="sm" variant="outline" className="gap-1 text-[11px] font-semibold" asChild>
              <Link to="/mulai">
                Lanjutkan
                <ArrowRight className="h-3 w-3" />
              </Link>
            </Button>
          )
        )}
      </div>
    </div>
  )
}

export function SettingsMobileView() {
  const { user } = useAuthContext()
  const signOut = useSignOut()
  const queryClient = useQueryClient()
  const telegramLink = useTelegramLink()
  const telegramUnlink = useTelegramUnlink()
  const [telegramNotice, setTelegramNotice] = useState<string | null>(null)
  const [telegramSuccess, setTelegramSuccess] = useState<string | null>(null)
  const [showTelegramModal, setShowTelegramModal] = useState(false)

  const { data: onboarding } = useQuery({
    queryKey: ['me', 'onboarding'],
    queryFn: accountFlowApi.onboarding,
  })
  const isTelegramConnected = Boolean(onboarding?.telegram_linked)

  const profile: User | null = user ?? null
  const initial = (profile?.name || profile?.email || 'I').trim().charAt(0).toUpperCase()
  const isVerified = Boolean(profile?.email_verified_at)

  const onConnectTelegram = async () => {
    setTelegramNotice(null)
    setTelegramSuccess(null)
    try {
      const hasil = await telegramLink.mutateAsync()
      window.open(hasil.url, '_blank', 'noopener,noreferrer')
    } catch (err) {
      setTelegramNotice(err instanceof ApiError ? err.message : 'Gagal membuat tautan Telegram.')
    }
  }

  const onDisconnectTelegram = async () => {
    setTelegramNotice(null)
    setTelegramSuccess(null)
    try {
      await telegramUnlink.mutateAsync()
      await queryClient.invalidateQueries({ queryKey: ['me', 'onboarding'] })
      await queryClient.invalidateQueries({ queryKey: ['trial'] })
      setTelegramSuccess('Koneksi Telegram berhasil diputus.')
    } catch (err) {
      setTelegramNotice(err instanceof ApiError ? err.message : 'Gagal memutuskan Telegram.')
    }
  }

  return (
    <div className="flex flex-col gap-3.5 pb-2 lg:hidden">
      {/* Header */}
      <div>
        <h1 className="text-lg font-bold tracking-tight text-foreground">Pengaturan</h1>
        <p className="mt-0.5 text-[11.5px] leading-relaxed text-muted-foreground">
          Kelola akun, preferensi trading, dan keamanan Anda.
        </p>
      </div>

      {/* Kartu profil */}
      <Link
        to="/pengaturan/keamanan"
        className="flex items-center gap-3 rounded-2xl border border-border bg-card p-3.5 shadow-xs active:scale-[0.99]"
      >
        <span className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-foreground text-lg font-bold text-background">
          {initial}
        </span>
        <div className="min-w-0 flex-1">
          <p className="truncate text-[13.5px] font-bold tracking-tight text-foreground">
            {profile?.name ?? 'Pengguna ILA'}
          </p>
          <p className="truncate text-[11px] text-muted-foreground">{profile?.email ?? '—'}</p>
          <div className="mt-1 flex items-center gap-1.5">
            <span className="rounded-md border border-border bg-secondary px-1.5 py-0.5 text-[9px] font-semibold uppercase tracking-wider text-foreground">
              Tier {TIER_LABEL[profile?.tier ?? 'free'] ?? profile?.tier}
            </span>
            {isVerified && (
              <span className="inline-flex items-center gap-0.5 rounded-md border border-emerald-500/30 bg-emerald-500/10 px-1.5 py-0.5 text-[9px] font-semibold uppercase tracking-wider text-emerald-700 dark:text-emerald-400">
                <CheckCircle2 className="h-2.5 w-2.5" />
                Terverifikasi
              </span>
            )}
          </div>
        </div>
        <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />
      </Link>

      {/* Banner uji coba */}
      <TrialCard />

      {/* Grup Umum */}
      <Group title="Umum">
        <MenuRow
          icon={UserIcon}
          label="Profil & Akun"
          description="Data diri dan informasi kontak"
          to="/pengaturan/keamanan"
        />
        <MenuRow
          icon={CreditCard}
          label="Langganan & Tagihan"
          description="Paket aktif dan riwayat invoice"
          to="/langganan"
        />
        <MenuRow
          icon={Sparkles}
          label="Penyiapan Akun"
          description="Lima langkah awal penggunaan sistem"
          to="/mulai"
        />
      </Group>

      {/* Grup Trading */}
      <Group title="Trading">
        <MenuRow
          icon={Wallet}
          label="Akun Broker"
          description="Kelola koneksi cTrader & Match-Trader"
          to="/akun"
        />
        <MenuRow
          icon={Bot}
          label="Watchlist Pair"
          description="Instrumen dan metode analisis ILA"
          to="/strategi"
        />
      </Group>

      {/* Grup Notifikasi */}
      <Group title="Notifikasi">
        <MenuRow
          icon={Bell}
          label="Bot Telegram"
          description={
            isTelegramConnected
              ? 'Terhubung — notifikasi sinyal & alert aktif'
              : 'Peringatan sinyal dan konfirmasi entry'
          }
          trailing={isTelegramConnected ? 'Terhubung' : 'Belum'}
          onClick={() => {
            setTelegramNotice(null)
            setTelegramSuccess(null)
            setShowTelegramModal(true)
          }}
        />
      </Group>

      {/* Grup Keamanan */}
      <Group title="Keamanan">
        <MenuRow
          icon={ShieldCheck}
          label="Kata Sandi & 2FA"
          description="Autentikasi dan sesi perangkat"
          to="/pengaturan/keamanan"
        />
        <MenuRow
          icon={Mail}
          label="Verifikasi Email"
          description="Wajib untuk simulasi dan pembayaran"
          to="/pengaturan/keamanan"
          trailing={isVerified ? 'Terverifikasi' : 'Belum'}
        />
      </Group>

      {/* Grup Tentang */}
      <Group title="Tentang">
        <MenuRow
          icon={HelpCircle}
          label="Pusat Bantuan"
          description="Panduan setup broker dan bot Telegram"
          to="/bantuan"
        />
        <MenuRow
          icon={FileText}
          label="Legal & Risiko"
          description="Ketentuan layanan dan disclaimer trading"
          to="/legal"
        />
      </Group>

      {/* Keluar */}
      <button
        type="button"
        onClick={signOut}
        className="mt-1 flex w-full items-center justify-center gap-1.5 rounded-xl border border-destructive/40 bg-destructive/[0.04] py-2.5 text-[12px] font-semibold text-destructive active:scale-[0.99]"
      >
        <LogOut className="h-4 w-4" />
        Keluar dari Akun
      </button>

      <p className="pb-1 text-center text-[10px] text-muted-foreground">
        ILA Trading Platform · Sinyal, radar, dan eksekusi terpandu
      </p>

      {/* Modal Pengaturan Telegram Mobile */}
      <Modal
        open={showTelegramModal}
        onClose={() => setShowTelegramModal(false)}
        title="Notifikasi Telegram"
        description="Peringatan sinyal, peluang entry, dan status trading."
      >
        <div className="flex flex-col gap-3.5 py-1">
          <div className="flex items-center gap-2">
            <span className="text-[12px] font-medium text-muted-foreground">Status:</span>
            {isTelegramConnected ? (
              <span className="inline-flex items-center gap-1 rounded-md border border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5 text-[11px] font-semibold text-emerald-700 dark:text-emerald-400">
                <CheckCircle2 className="h-3 w-3" />
                Terhubung
              </span>
            ) : (
              <Badge variant="outline" className="text-muted-foreground">
                Belum Terhubung
              </Badge>
            )}
          </div>

          <p className="text-[12px] leading-relaxed text-muted-foreground">
            {isTelegramConnected
              ? 'Akun Anda sudah terhubung ke bot resmi (@imaginaryliquiditybot). Notifikasi setup valid, peluang entry, dan status trading akan otomatis dikirimkan ke chat Telegram Anda.'
              : 'Hubungkan akun Anda dengan bot Telegram (@imaginaryliquiditybot) untuk menerima sinyal setup dan konfirmasi entry secara langsung.'}
          </p>

          {telegramNotice && <Alert variant="destructive">{telegramNotice}</Alert>}
          {telegramSuccess && <Alert variant="success">{telegramSuccess}</Alert>}

          <div className="flex flex-col gap-2 pt-2">
            {isTelegramConnected ? (
              <>
                <Button
                  variant="outline"
                  className="w-full text-destructive hover:text-destructive"
                  isLoading={telegramUnlink.isPending}
                  onClick={() => void onDisconnectTelegram()}
                >
                  Putuskan Koneksi
                </Button>
                <Button
                  variant="secondary"
                  className="w-full"
                  isLoading={telegramLink.isPending}
                  onClick={() => void onConnectTelegram()}
                >
                  Hubungkan Ulang / Akun Lain
                </Button>
              </>
            ) : (
              <Button
                className="w-full"
                isLoading={telegramLink.isPending}
                onClick={() => void onConnectTelegram()}
              >
                Hubungkan ke Bot Telegram
              </Button>
            )}
          </div>
        </div>
      </Modal>
    </div>
  )
}
