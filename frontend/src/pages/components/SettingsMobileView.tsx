import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import {
  Bell,
  Bot,
  CheckCircle2,
  ChevronRight,
  CreditCard,
  FileText,
  HelpCircle,
  LogOut,
  Mail,
  Monitor,
  ShieldCheck,
  Sparkles,
  User as UserIcon,
  Wallet,
} from 'lucide-react'
import { PUSH_STATUS_QUERY_KEY } from '@/features/notifications/PushNotificationPanel'
import { pushApi } from '@/lib/push-notifications'
import { useSignOut, useAuthContext } from '@/features/auth/hooks/use-auth'
import { accountFlowApi } from '@/lib/auth-api'
import { useQuery } from '@tanstack/react-query'
import { useInviteOnly } from '@/lib/use-invite-only'
import { cn } from '@/lib/utils'
import type { User } from '@/types/api'

const TIER_LABEL: Record<string, string> = { free: 'Free', basic: 'Basic', plus: 'Plus' }

type MenuRowProps = {
  icon: typeof UserIcon
  label: string
  description: string
  to?: string
  /** Tautan lintas bundel (mis. ke entri publik /bantuan) memakai <a href>. */
  href?: string
  onClick?: () => void
  trailing?: string
  disabled?: boolean
}

function MenuRow({ icon: Icon, label, description, to, href, onClick, trailing, disabled }: MenuRowProps) {
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
            trailing === 'Terverifikasi' || trailing === 'Terhubung' || trailing === 'Aktif'
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

  if (href) {
    return (
      <a href={href} className={className}>
        {inner}
      </a>
    )
  }

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

function Group({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="rounded-2xl border border-border bg-card shadow-xs">
      <h2 className="border-b border-border px-3.5 py-2.5 text-[10px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">
        {title}
      </h2>
      <div className="divide-y divide-border/70">{children}</div>
    </section>
  )
}

export function SettingsMobileView() {
  const { user } = useAuthContext()
  const signOut = useSignOut()

  const { data: onboarding } = useQuery({
    queryKey: ['me', 'onboarding'],
    queryFn: accountFlowApi.onboarding,
  })
  const { data: pushStatus } = useQuery({
    queryKey: PUSH_STATUS_QUERY_KEY,
    queryFn: pushApi.status,
  })

  const isTelegramConnected = Boolean(onboarding?.telegram_linked)
  const pushDeviceCount = pushStatus?.subscriptions?.length ?? 0
  const notifAktif = isTelegramConnected || pushDeviceCount > 0

  const profile: User | null = user ?? null
  const initial = (profile?.name || profile?.email || 'I').trim().charAt(0).toUpperCase()
  const isVerified = Boolean(profile?.email_verified_at)
  const inviteOnly = useInviteOnly()

  return (
    <div className="flex flex-col gap-3.5 pb-2 lg:hidden">
      {/* Header */}
      <div>
        <h1 className="text-lg font-bold tracking-tight text-foreground">Pengaturan</h1>
        <p className="mt-0.5 text-[11.5px] leading-relaxed text-muted-foreground">
          Kelola akun, notifikasi, dan keamanan Anda.
        </p>
      </div>

      {/* Kartu profil */}
      <Link
        to="/pengaturan/profil"
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

      {/* Grup Umum */}
      <Group title="Umum">
        <MenuRow
          icon={UserIcon}
          label="Profil & Akun"
          description="Data diri dan informasi kontak"
          to="/pengaturan/profil"
        />
        <MenuRow
          icon={Bell}
          label="Notifikasi"
          description="Telegram dan Web Push"
          trailing={notifAktif ? 'Aktif' : 'Belum'}
          to="/pengaturan/notifikasi"
        />
        {!inviteOnly && (
          <MenuRow
            icon={CreditCard}
            label="Langganan & Tagihan"
            description="Paket aktif dan riwayat invoice"
            to="/langganan"
          />
        )}
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

      {/* Grup Keamanan */}
      <Group title="Keamanan">
        <MenuRow
          icon={ShieldCheck}
          label="Kata Sandi & 2FA"
          description="Kata sandi dan autentikasi dua faktor"
          to="/pengaturan/keamanan"
        />
        <MenuRow
          icon={Mail}
          label="Verifikasi Email"
          description="Status verifikasi email akun"
          to="/pengaturan/keamanan?sub=email"
          trailing={isVerified ? 'Terverifikasi' : 'Belum'}
        />
        <MenuRow
          icon={Monitor}
          label="Perangkat & Sesi"
          description="Perangkat yang masuk dan keluar otomatis"
          to="/pengaturan/perangkat"
        />
      </Group>

      {/* Grup Tentang */}
      <Group title="Tentang">
        <MenuRow
          icon={HelpCircle}
          label="Pusat Bantuan"
          description="Panduan setup broker dan bot Telegram"
          href="/bantuan"
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
    </div>
  )
}
