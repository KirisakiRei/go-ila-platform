import { Link } from 'react-router-dom'
import {
  ArrowRight,
  BookOpen,
  CheckCircle2,
  HelpCircle,
  ShieldAlert,
  ShieldCheck,
  Sparkles,
} from 'lucide-react'
import { Alert, Button, Skeleton } from '@/components/ui'
import { OnboardingChecklist } from '@/features/onboarding'
import type { OnboardingState } from '@/types/api'

type OnboardingMobileViewProps = {
  data: OnboardingState | undefined
  isLoading: boolean
  completed: number
  total: number
  percentage: number
  onSkip: () => void
  isSkipping: boolean
  canSkip: boolean
  onNavigate: () => void
}

type HelpRow = {
  icon: typeof BookOpen
  title: string
  description: string
  href: string
}

const HELP_ROWS: HelpRow[] = [
  {
    icon: BookOpen,
    title: 'Pusat Bantuan',
    description: 'Panduan token Match-Trader & bot Telegram',
    href: '/bantuan',
  },
  {
    icon: HelpCircle,
    title: 'Tanya Jawab (FAQ)',
    description: 'Sesi pasar, batas risiko, dan eksekusi order',
    href: '/bantuan#faq-bantuan',
  },
  {
    icon: ShieldAlert,
    title: 'Legal & Risiko',
    description: 'Ketentuan layanan dan disclaimer trading',
    href: '/legal',
  },
]

export function OnboardingMobileView({
  data,
  isLoading,
  completed,
  total,
  percentage,
  onSkip,
  isSkipping,
  canSkip,
  onNavigate,
}: OnboardingMobileViewProps) {
  const isDone = completed >= total

  return (
    <div className="flex flex-col gap-3.5 pb-2 lg:hidden">
      {/* Header */}
      <div>
        <h1 className="text-lg font-bold tracking-tight text-foreground">Mulai</h1>
        <p className="mt-0.5 text-caption leading-relaxed text-muted-foreground">
          Lima langkah singkat agar sistem mengenal akun, broker, dan preferensi risiko Anda.
        </p>
      </div>

      {/* Kartu progres */}
      <div className="rounded-2xl border border-border bg-card p-3.5 shadow-xs">
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-foreground text-background">
              <Sparkles className="h-4 w-4" />
            </span>
            <div>
              <p className="text-[12.5px] font-bold tracking-tight text-foreground">Progres Penyiapan</p>
              <p className="text-micro text-muted-foreground">
                {completed} dari {total} langkah selesai
              </p>
            </div>
          </div>
          <span className="flex flex-col items-end">
            <span className="text-xl font-bold tabular-nums text-foreground">{percentage}%</span>
            <span className="text-micro uppercase tracking-wider text-muted-foreground">selesai</span>
          </span>
        </div>

        <div className="mt-3 h-2 w-full overflow-hidden rounded-full bg-secondary">
          <div
            className="h-full rounded-full bg-emerald-500 transition-all duration-500 ease-out"
            style={{ width: `${Math.min(100, Math.max(0, percentage))}%` }}
          />
        </div>

        <p className="mt-2 text-micro leading-relaxed text-muted-foreground">
          {isDone
            ? 'Semua langkah selesai. Akun Anda siap menerima sinyal.'
            : 'Posisi tersimpan otomatis — Anda bisa melanjutkan kapan saja.'}
        </p>

        {isDone && (
          <div className="mt-2.5 flex items-center gap-1.5 rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-2.5 py-2">
            <CheckCircle2 className="h-3.5 w-3.5 shrink-0 text-emerald-700 dark:text-emerald-400" />
            <p className="text-micro font-medium text-emerald-800 dark:text-emerald-300">
              Penyiapan akun selesai sepenuhnya.
            </p>
          </div>
        )}
      </div>

      {isLoading && <Skeleton className="h-72 w-full rounded-2xl" />}

      {data && (
        <>
          {/* Daftar langkah */}
          <div className="rounded-2xl border border-border bg-card p-3 shadow-xs">
            <OnboardingChecklist data={data} onNavigate={onNavigate} />
          </div>

          {!data.email_verified && (
            <Alert variant="warning" className="text-micro leading-relaxed">
              Beberapa fitur terkunci sampai email terverifikasi: uji coba simulasi, menghubungkan akun
              broker, dan pembayaran langganan.
            </Alert>
          )}

          {canSkip && (
            <Button
              variant="outline"
              size="sm"
              className="w-full text-caption font-semibold"
              disabled={isSkipping}
              onClick={onSkip}
            >
              {isSkipping ? 'Menyimpan…' : 'Tandai Selesai / Lewati'}
            </Button>
          )}
        </>
      )}

      {/* Banner keamanan */}
      <div className="flex items-start gap-2.5 rounded-2xl border border-border bg-secondary/40 p-3.5">
        <ShieldCheck className="mt-0.5 h-4 w-4 shrink-0 text-emerald-600 dark:text-emerald-400" />
        <div>
          <p className="text-caption font-semibold text-foreground">Keamanan data Anda</p>
          <p className="mt-0.5 text-micro leading-relaxed text-muted-foreground">
            Kredensial broker disimpan terenkripsi AES-256 dan tidak pernah dibagikan ke pihak ketiga.
          </p>
        </div>
      </div>

      {/* Bantuan cepat */}
      <div className="rounded-2xl border border-border bg-card shadow-xs">
        <h2 className="border-b border-border px-3.5 py-2.5 text-micro font-semibold uppercase tracking-[0.14em] text-muted-foreground">
          Butuh Panduan?
        </h2>
        <div className="divide-y divide-border/70">
          {HELP_ROWS.map(({ icon: Icon, title, description, href }) => (
            <a
              key={title}
              href={href}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-2.5 px-3.5 py-2.5 active:bg-secondary/50"
            >
              <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-border bg-secondary text-foreground">
                <Icon className="h-4 w-4" />
              </span>
              <div className="min-w-0 flex-1">
                <p className="truncate text-[12px] font-semibold text-foreground">{title}</p>
                <p className="truncate text-micro text-muted-foreground">{description}</p>
              </div>
              <ArrowRight className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
            </a>
          ))}
        </div>
      </div>

      <Link
        to="/pengaturan/keamanan"
        className="pb-1 text-center text-micro font-medium text-muted-foreground underline"
      >
        Pelajari cara kami menjaga keamanan akun
      </Link>
    </div>
  )
}
