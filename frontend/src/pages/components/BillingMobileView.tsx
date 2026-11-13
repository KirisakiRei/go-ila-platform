import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  ArrowLeft,
  ArrowRight,
  Check,
  ChevronRight,
  Crown,
  FileText,
  Receipt,
  Tag,
} from 'lucide-react'
import { Button, EmptyState, Skeleton } from '@/components/ui'
import { formatIDR, formatInvoice } from '@/features/admin/lib/format'
import { formatPlanBenefits, getPlanAudience } from '@/pages/public/lib/plan-benefits'
import { cn } from '@/lib/utils'
import type { PublicPlan } from '@/types/admin'
import type { TimePair } from '@/types/api'

export type MobileInvoice = {
  id: string
  number: string
  plan_code: string
  total_idr: number
  status: string
  created_at: TimePair
  paid_at?: TimePair
}

type InvoiceFilter = 'all' | 'paid' | 'pending' | 'expired'

type BillingMobileViewProps = {
  userName: string
  userEmail: string
  currentTier: string
  plans: PublicPlan[]
  plansLoading: boolean
  subscription?: { plan_code: string; period_end: TimePair; status: string } | null
  invoices: MobileInvoice[]
  invoicesLoading: boolean
  onOpenRedeem: () => void
}

const TIER_LABEL: Record<string, string> = { free: 'Free', basic: 'Basic', plus: 'Plus' }

const STATUS_LABEL: Record<string, string> = {
  pending: 'Menunggu',
  paid: 'Lunas',
  expired: 'Kedaluwarsa',
  cancelled: 'Dibatalkan',
  refunded: 'Dikembalikan',
}

const STATUS_STYLE: Record<string, string> = {
  paid: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400',
  pending: 'border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-400',
  expired: 'border-border bg-secondary text-muted-foreground',
}

const FILTERS: { key: InvoiceFilter; label: string }[] = [
  { key: 'all', label: 'Semua' },
  { key: 'paid', label: 'Lunas' },
  { key: 'pending', label: 'Menunggu' },
  { key: 'expired', label: 'Kedaluwarsa' },
]

export function BillingMobileView({
  userName,
  userEmail,
  currentTier,
  plans,
  plansLoading,
  subscription,
  invoices,
  invoicesLoading,
  onOpenRedeem,
}: BillingMobileViewProps) {
  const [view, setView] = useState<'main' | 'invoices'>('main')
  const [filter, setFilter] = useState<InvoiceFilter>('all')

  const tierName = TIER_LABEL[currentTier] ?? currentTier
  const initial = (userName || userEmail || 'I').trim().charAt(0).toUpperCase()

  const counts = useMemo(() => {
    const base: Record<InvoiceFilter, number> = { all: invoices.length, paid: 0, pending: 0, expired: 0 }
    for (const row of invoices) {
      if (row.status === 'paid') base.paid++
      else if (row.status === 'pending') base.pending++
      else if (row.status === 'expired') base.expired++
    }
    return base
  }, [invoices])

  const filteredInvoices = useMemo(() => {
    if (filter === 'all') return invoices
    return invoices.filter((row) => row.status === filter)
  }, [invoices, filter])

  const recentInvoices = invoices.slice(0, 3)

  if (view === 'invoices') {
    return (
      <div className="flex flex-col gap-4 pb-2 lg:hidden">
        {/* Header riwayat tagihan */}
        <div className="flex items-start gap-3">
          <button
            type="button"
            onClick={() => setView('main')}
            aria-label="Kembali ke Langganan"
            className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-border bg-card text-foreground active:scale-95"
          >
            <ArrowLeft className="h-4 w-4" />
          </button>
          <div className="min-w-0">
            <h1 className="text-lg font-bold tracking-tight text-foreground">Riwayat Tagihan</h1>
            <p className="mt-0.5 text-caption leading-relaxed text-muted-foreground">
              Daftar seluruh riwayat pembayaran paket dan langganan resmi Anda.
            </p>
          </div>
        </div>

        {/* Filter status */}
        <div className="flex items-center gap-1.5 overflow-x-auto pb-1 scrollbar-none">
          {FILTERS.map((item) => (
            <button
              key={item.key}
              type="button"
              onClick={() => setFilter(item.key)}
              className={cn(
                'flex shrink-0 items-center gap-1.5 rounded-xl px-3 py-1.5 text-xs font-semibold transition-all active:scale-95',
                filter === item.key
                  ? 'bg-foreground text-background shadow-xs'
                  : 'border border-border bg-card text-muted-foreground hover:text-foreground',
              )}
            >
              <span>{item.label}</span>
              <span
                className={cn(
                  'rounded-full px-1.5 text-micro',
                  filter === item.key
                    ? 'bg-background/20 text-background'
                    : 'bg-secondary text-muted-foreground',
                )}
              >
                {counts[item.key]}
              </span>
            </button>
          ))}
        </div>

        {/* Daftar tagihan */}
        {invoicesLoading ? (
          <div className="space-y-3">
            {[1, 2, 3].map((i) => (
              <Skeleton key={i} className="h-28 w-full rounded-2xl" />
            ))}
          </div>
        ) : filteredInvoices.length === 0 ? (
          <div className="rounded-2xl border border-border bg-card p-4">
            <EmptyState
              title="Belum ada tagihan"
              description="Tagihan akan tampil di sini setelah Anda menyelesaikan checkout paket."
            />
          </div>
        ) : (
          <div className="space-y-3">
            {filteredInvoices.map((row) => (
              <div
                key={row.id}
                className="rounded-2xl border border-border bg-card p-3.5 shadow-xs"
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <p className="truncate font-mono text-[13px] font-bold tracking-tight text-foreground">
                      {row.number}
                    </p>
                    <p className="mt-0.5 text-micro text-muted-foreground">
                      {formatInvoice(row.created_at?.wib)} WIB
                    </p>
                  </div>
                  <span
                    className={cn(
                      'shrink-0 rounded-lg border px-2 py-0.5 text-micro font-semibold uppercase tracking-wide',
                      STATUS_STYLE[row.status] ?? 'border-border bg-secondary text-muted-foreground',
                    )}
                  >
                    {STATUS_LABEL[row.status] ?? row.status}
                  </span>
                </div>

                <div className="mt-3 flex items-end justify-between gap-2 border-t border-border/70 pt-3">
                  <div className="min-w-0">
                    <p className="truncate text-caption font-medium text-muted-foreground">
                      Paket {TIER_LABEL[row.plan_code] ?? row.plan_code.toUpperCase()}
                    </p>
                    <p className="text-[15px] font-bold tabular-nums text-foreground">
                      {formatIDR(row.total_idr)}
                    </p>
                  </div>
                  <span className="shrink-0 text-micro font-semibold text-muted-foreground">
                    {row.status === 'pending' ? 'Menunggu pembayaran' : STATUS_LABEL[row.status] ?? row.status}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}

        <Button variant="outline" size="sm" className="mt-1 w-full gap-1.5 text-xs" onClick={() => setView('main')}>
          <ArrowLeft className="h-3.5 w-3.5" />
          Kembali ke Langganan
        </Button>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-4 pb-2 lg:hidden">
      {/* Header halaman */}
      <div>
        <h1 className="text-lg font-bold tracking-tight text-foreground">Langganan &amp; Tagihan</h1>
        <p className="mt-0.5 text-caption leading-relaxed text-muted-foreground">
          Kelola paket aktif, bandingkan benefit, dan akses riwayat tagihan resmi.
        </p>
      </div>

      {/* Kartu profil ringkas */}
      <div className="flex items-center gap-3 rounded-2xl border border-border bg-card p-3.5 shadow-xs">
        <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-foreground text-base font-bold text-background">
          {initial}
        </span>
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-bold tracking-tight text-foreground">{userName}</p>
          <p className="truncate text-micro text-muted-foreground">{userEmail}</p>
        </div>
        <span className="shrink-0 rounded-lg border border-border bg-secondary px-2.5 py-1 text-micro font-semibold uppercase tracking-wider text-foreground">
          {tierName}
        </span>
      </div>

      {/* Paket aktif saat ini */}
      <div className="rounded-2xl border border-border bg-card p-4 shadow-xs">
        <p className="text-micro font-semibold uppercase tracking-[0.14em] text-muted-foreground">
          Paket Aktif Saat Ini
        </p>

        <div className="mt-2 flex items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <Crown className="h-4 w-4 text-foreground" />
            <span className="text-[17px] font-bold tracking-tight text-foreground">
              {subscription ? `Paket ${TIER_LABEL[subscription.plan_code] ?? subscription.plan_code}` : `Paket ${tierName}`}
            </span>
          </div>
          <span
            className={cn(
              'rounded-lg border px-2 py-0.5 text-micro font-semibold uppercase tracking-wide',
              subscription
                ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400'
                : 'border-border bg-secondary text-muted-foreground',
            )}
          >
            {subscription ? 'Aktif' : 'Tidak aktif'}
          </span>
        </div>

        <p className="mt-1 text-caption text-muted-foreground">
          {subscription
            ? `Berlaku hingga ${formatInvoice(subscription.period_end?.wib)} WIB`
            : 'Belum ada langganan berbayar yang aktif saat ini.'}
        </p>

        <div className="mt-3 flex items-center gap-2 border-t border-border/70 pt-3">
          <Button size="sm" className="flex-1 gap-1 text-xs font-semibold" asChild>
            <Link to={`/langganan/bayar?paket=${currentTier === 'plus' ? 'plus' : 'basic'}`}>
              {currentTier === 'free' ? 'Aktifkan Paket' : 'Perpanjang Paket'}
              <ArrowRight className="h-3.5 w-3.5" />
            </Link>
          </Button>
          <Button size="sm" variant="outline" className="gap-1 text-xs font-semibold" onClick={onOpenRedeem}>
            <Tag className="h-3.5 w-3.5" />
            Tukar Kode
          </Button>
        </div>
      </div>

      {/* Toggle siklus penagihan */}
      <div className="space-y-1.5">
        <div className="grid grid-cols-2 gap-1 rounded-xl border border-border bg-secondary/60 p-1">
          <button
            type="button"
            aria-pressed="true"
            className="rounded-lg bg-card py-2 text-xs font-semibold text-foreground shadow-xs"
          >
            Bulanan
          </button>
          <button
            type="button"
            disabled
            aria-pressed="false"
            className="flex items-center justify-center gap-1 rounded-lg py-2 text-micro font-semibold text-muted-foreground"
          >
            Tahunan
            <span className="rounded bg-secondary px-1 py-0.5 text-micro font-bold uppercase tracking-wide text-muted-foreground">
              -20%
            </span>
          </button>
        </div>
        <p className="text-micro text-muted-foreground">
          Penagihan tahunan segera hadir — semua paket saat ini ditagihkan per 30 hari.
        </p>
      </div>

      {/* Daftar paket */}
      <div className="space-y-3">
        <div>
          <h2 className="text-sm font-bold tracking-tight text-foreground">Pilih Paket Langganan</h2>
          <p className="mt-0.5 text-micro text-muted-foreground">
            Harga sudah termasuk pajak sesuai ketentuan resmi.
          </p>
        </div>

        {plansLoading ? (
          <div className="space-y-3">
            {[1, 2, 3].map((i) => (
              <Skeleton key={i} className="h-60 w-full rounded-2xl" />
            ))}
          </div>
        ) : (
          plans.map((plan) => {
            const isCurrent = currentTier === plan.code
            const isPopular = plan.code === 'basic'
            const benefits = formatPlanBenefits(plan)
            const isFree = plan.code === 'free'

            return (
              <div
                key={plan.code}
                className={cn(
                  'relative rounded-2xl border bg-card p-4 shadow-xs',
                  isCurrent || isPopular ? 'border-foreground/70' : 'border-border',
                )}
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <div className="flex items-center gap-1.5">
                      <h3 className="text-sm font-bold tracking-tight text-foreground">{plan.name}</h3>
                      {isPopular && (
                        <span className="rounded bg-secondary px-1.5 py-0.5 text-micro font-bold uppercase tracking-wider text-foreground">
                          Paling Populer
                        </span>
                      )}
                      {isCurrent && (
                        <span className="rounded bg-foreground px-1.5 py-0.5 text-micro font-bold uppercase tracking-wider text-background">
                          Paket Anda
                        </span>
                      )}
                    </div>
                    <p className="mt-1 text-micro leading-relaxed text-muted-foreground">
                      {getPlanAudience(plan)}
                    </p>
                  </div>
                </div>

                <div className="mt-3 flex items-baseline gap-1">
                  <span className="text-xl font-bold tracking-tight tabular-nums text-foreground">
                    {isFree ? 'Gratis' : formatIDR(plan.price_idr)}
                  </span>
                  {!isFree && <span className="text-micro text-muted-foreground">/ 30 hari</span>}
                </div>

                <ul className="mt-3 space-y-1.5 border-t border-border/70 pt-3">
                  {benefits.map((benefit) => (
                    <li key={benefit.label} className="flex items-start gap-2 text-caption text-foreground">
                      <Check className="mt-0.5 h-3.5 w-3.5 shrink-0 text-emerald-600 dark:text-emerald-400" />
                      <span className="leading-snug">{benefit.label}</span>
                    </li>
                  ))}
                </ul>

                <div className="mt-3">
                  {isCurrent || isFree ? (
                    <Button variant="outline" size="sm" className="w-full text-xs" disabled>
                      {isCurrent ? 'Paket Aktif Anda' : 'Paket Dasar'}
                    </Button>
                  ) : (
                    <Button
                      size="sm"
                      variant={isPopular ? 'default' : 'outline'}
                      className="w-full gap-1 text-xs font-semibold"
                      asChild
                    >
                      <Link to={`/langganan/bayar?paket=${plan.code}`}>
                        Pilih Paket
                        <ArrowRight className="h-3.5 w-3.5" />
                      </Link>
                    </Button>
                  )}
                </div>
              </div>
            )
          })
        )}
      </div>

      {/* Riwayat tagihan ringkas */}
      <div className="rounded-2xl border border-border bg-card shadow-xs">
        <div className="flex items-center justify-between gap-2 border-b border-border px-3.5 py-3">
          <div className="flex items-center gap-2">
            <Receipt className="h-4 w-4 text-muted-foreground" />
            <h2 className="text-[13px] font-bold tracking-tight text-foreground">Tagihan Terakhir</h2>
          </div>
          <button
            type="button"
            onClick={() => setView('invoices')}
            className="flex items-center gap-0.5 text-micro font-semibold text-foreground"
          >
            Lihat Semua
            <ChevronRight className="h-3.5 w-3.5" />
          </button>
        </div>

        {invoicesLoading ? (
          <div className="space-y-2 p-3.5">
            {[1, 2].map((i) => (
              <Skeleton key={i} className="h-12 w-full rounded-xl" />
            ))}
          </div>
        ) : recentInvoices.length === 0 ? (
          <p className="px-3.5 py-5 text-center text-caption text-muted-foreground">
            Belum ada tagihan. Riwayat akan muncul setelah checkout pertama Anda.
          </p>
        ) : (
          <div className="divide-y divide-border/70">
            {recentInvoices.map((row) => (
              <div key={row.id} className="flex items-center justify-between gap-2 px-3.5 py-2.5">
                <div className="min-w-0">
                  <p className="truncate font-mono text-caption font-semibold text-foreground">
                    {row.number}
                  </p>
                  <p className="text-micro text-muted-foreground">
                    {formatInvoice(row.created_at?.wib)} WIB
                  </p>
                </div>
                <div className="flex shrink-0 items-center gap-2">
                  <span className="text-caption font-semibold tabular-nums text-foreground">
                    {formatIDR(row.total_idr)}
                  </span>
                  <span
                    className={cn(
                      'rounded-md border px-1.5 py-0.5 text-micro font-semibold uppercase tracking-wide',
                      STATUS_STYLE[row.status] ?? 'border-border bg-secondary text-muted-foreground',
                    )}
                  >
                    {STATUS_LABEL[row.status] ?? row.status}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Voucher / kredit */}
      <div className="rounded-2xl border border-border bg-card p-3.5 shadow-xs">
        <div className="flex items-start gap-2.5">
          <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-secondary text-foreground">
            <Tag className="h-4 w-4" />
          </span>
          <div className="min-w-0 flex-1">
            <p className="text-[12.5px] font-semibold text-foreground">Punya kode voucher?</p>
            <p className="mt-0.5 text-micro leading-relaxed text-muted-foreground">
              Tukarkan kode kredit promosi untuk memotong total tagihan saat checkout.
            </p>
            <Button size="sm" variant="outline" className="mt-2 gap-1 text-micro font-semibold" onClick={onOpenRedeem}>
              <FileText className="h-3.5 w-3.5" />
              Tukar Kode Kredit
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
