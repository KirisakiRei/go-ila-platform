import { Link } from 'react-router-dom'
import { ArrowRight, Bot, ShieldCheck, Sparkles, Zap } from 'lucide-react'
import { Badge, Button, Card, CardContent } from '@/components/ui'
import { useAuthContext } from '@/features/auth/hooks/use-auth'
import { useTrial, useSystemStatus } from '@/lib/queries'
import { useInviteOnly } from '@/lib/use-invite-only'
import { formatDate } from '@/lib/format'
import type { Tier } from '@/types/api'

const TIER_META: Record<Tier, { label: string; badgeVariant: 'muted' | 'outline' | 'solid'; description: string }> = {
  free: {
    label: 'Free Tier',
    badgeVariant: 'muted',
    description: 'Akses pemantauan sinyal dasar dalam mode simulasi.',
  },
  basic: {
    label: 'Paket Basic',
    badgeVariant: 'outline',
    description: 'Sinyal real-time tanpa jeda, 1 akun broker Match-Trader, strategi ILA 2.0.',
  },
  plus: {
    label: 'Paket Plus',
    badgeVariant: 'solid',
    description: 'Fitur terlengkap, 2 akun broker, eksekusi otomatis, akses penuh ILA 2.0 & ILA 3.0.',
  },
}

export function AccountStatusCard() {
  const { user } = useAuthContext()
  const trial = useTrial()
  const { data: status } = useSystemStatus()
  const inviteOnly = useInviteOnly()

  if (!user) return null

  const tier = user.tier ?? 'free'
  const meta = TIER_META[tier] ?? TIER_META.free
  const isTrialActive = Boolean(trial.data?.active)
  const isTrialUsed = Boolean(trial.data?.used)

  const isSimulated = tier === 'free' || isTrialActive

  return (
    <Card className="border-border bg-gradient-to-br from-card via-card to-secondary/20">
      <CardContent className="flex flex-col gap-4 p-4 sm:flex-row sm:items-center sm:justify-between sm:p-5">
        <div className="flex items-start gap-3.5">
          <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-foreground text-background shadow-sm">
            {tier === 'plus' ? (
              <Zap className="h-5 w-5" />
            ) : isTrialActive ? (
              <Sparkles className="h-5 w-5" />
            ) : (
              <Bot className="h-5 w-5" />
            )}
          </span>

          <div className="min-w-0 space-y-1">
            <div className="flex flex-wrap items-center gap-2">
              <h2 className="text-sm font-semibold tracking-tight">{user.name}</h2>
              <Badge variant={meta.badgeVariant}>{meta.label}</Badge>
              {isTrialActive && (
                <span className="inline-flex items-center gap-1 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-2 py-0.5 text-[11px] font-semibold text-emerald-700 dark:text-emerald-400">
                  <Sparkles className="h-3 w-3" />
                  Trial Aktif ({trial.data?.days_remaining} hari lagi)
                </span>
              )}
            </div>

            <p className="text-[12.5px] text-muted-foreground">{meta.description}</p>

            <div className="flex flex-wrap items-center gap-3 pt-1 text-[11.5px] text-muted-foreground">
              <span className="flex items-center gap-1.5">
                <ShieldCheck className="h-3.5 w-3.5 text-foreground" />
                Mode: <strong className="font-medium text-foreground">{isSimulated ? 'Simulasi' : 'Live Broker'}</strong>
              </span>

              {isTrialActive && trial.data?.trial_expires_at && (
                <span>
                  Berakhir: <strong className="font-medium text-foreground">{formatDate(trial.data.trial_expires_at)}</strong>
                </span>
              )}

              {status && (
                <span>
                  Broker tersambung:{' '}
                  <strong className="font-medium text-foreground">
                    {status.brokers.connected} / {status.brokers.total}
                  </strong>
                </span>
              )}
            </div>
          </div>
        </div>

        {/* Ajakan trial dan billing disembunyikan selama invite only: tidak
            ada yang bisa didaftarkan maupun dibeli dari sini. */}
        {!inviteOnly && (
          <div className="flex flex-wrap items-center gap-2 self-start sm:self-center">
            {tier === 'free' && !isTrialActive && !isTrialUsed && (
              <Button size="sm" variant="default" asChild className="gap-1.5 text-xs font-semibold">
                <Link to="/pengaturan">
                  <Sparkles className="h-3.5 w-3.5" />
                  Mulai Uji Coba 7 Hari
                </Link>
              </Button>
            )}

            {tier === 'free' && (isTrialActive || isTrialUsed) && (
              <Button size="sm" variant="default" asChild className="gap-1.5 text-xs font-semibold">
                <Link to="/langganan">
                  Tingkatkan Paket
                  <ArrowRight className="h-3.5 w-3.5" />
                </Link>
              </Button>
            )}

            {tier !== 'free' && (
              <Button size="sm" variant="outline" asChild className="gap-1.5 text-xs">
                <Link to="/langganan">Kelola Langganan</Link>
              </Button>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
