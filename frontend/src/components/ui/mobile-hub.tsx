import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { ArrowLeft, ChevronRight, type LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'

/**
 * MobileMenuGroup mengelompokkan deretan menu drill-down bergaya `SettingsMobileView`.
 */
export function MobileMenuGroup({
  title,
  children,
  className,
}: {
  title: string
  children: ReactNode
  className?: string
}) {
  return (
    <div className={cn('space-y-1.5', className)}>
      <h2 className="px-1 text-[10px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">
        {title}
      </h2>
      <div className="divide-y divide-border/70 overflow-hidden rounded-2xl border border-border bg-card shadow-xs">
        {children}
      </div>
    </div>
  )
}

export type MobileMenuRowProps = {
  icon: LucideIcon
  label: string
  description?: string
  badge?: ReactNode
  to?: string
  onClick?: () => void
  tone?: 'default' | 'danger'
}

/**
 * MobileMenuRow menampilkan satu baris menu ringkas bebas horizontal scroll
 * untuk berpindah ke sub-halaman atau membuka dialog fokus di layar ponsel.
 */
export function MobileMenuRow({
  icon: Icon,
  label,
  description,
  badge,
  to,
  onClick,
  tone = 'default',
}: MobileMenuRowProps) {
  const isDanger = tone === 'danger'

  const content = (
    <div className="flex min-w-0 items-center justify-between gap-3 px-3.5 py-3 transition-colors hover:bg-secondary/40 active:bg-secondary/60">
      <div className="flex min-w-0 flex-1 items-center gap-3">
        <div
          className={cn(
            'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border',
            isDanger
              ? 'border-destructive/30 bg-destructive/10 text-destructive'
              : 'border-border bg-secondary text-foreground',
          )}
        >
          <Icon className="h-4 w-4" aria-hidden="true" />
        </div>
        <div className="min-w-0 flex-1">
          <p
            className={cn(
              'truncate text-[12.5px] font-semibold',
              isDanger ? 'text-destructive' : 'text-foreground',
            )}
          >
            {label}
          </p>
          {description && (
            <p className="truncate text-[10.5px] text-muted-foreground">{description}</p>
          )}
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-1.5">
        {badge}
        <ChevronRight className="h-4 w-4 text-muted-foreground/70" aria-hidden="true" />
      </div>
    </div>
  )

  if (to) {
    return (
      <Link to={to} className="block focus-visible:outline-none focus-visible:bg-secondary/40">
        {content}
      </Link>
    )
  }

  return (
    <button
      type="button"
      onClick={onClick}
      className="block w-full text-left focus-visible:outline-none focus-visible:bg-secondary/40"
    >
      {content}
    </button>
  )
}

/**
 * MobileSubpageHeader menyajikan tombol kembali dan judul ringkas pada sub-tampilan mobile.
 */
export function MobileSubpageHeader({
  backLabel,
  onBack,
  backTo,
  title,
  description,
  actions,
}: {
  backLabel: string
  onBack?: () => void
  backTo?: string
  title: string
  description?: string
  actions?: ReactNode
}) {
  return (
    <div className="space-y-2">
      {backTo ? (
        <Link
          to={backTo}
          className="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5" aria-hidden="true" />
          {backLabel}
        </Link>
      ) : (
        <button
          type="button"
          onClick={onBack}
          className="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5" aria-hidden="true" />
          {backLabel}
        </button>
      )}
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <h2 className="text-base font-bold tracking-tight text-foreground">{title}</h2>
          {description && (
            <p className="mt-0.5 text-[11.5px] leading-relaxed text-muted-foreground">
              {description}
            </p>
          )}
        </div>
        {actions && <div className="shrink-0">{actions}</div>}
      </div>
    </div>
  )
}

export type MobileSegmentOption<T extends string> = {
  id: T
  label: string
  count?: number
}

/**
 * MobileSegmentedTabs menyajikan tab kontrol satu baris (grid merata) yang aman
 * di layar 360px tanpa terlipat ke baris kedua.
 */
export function MobileSegmentedTabs<T extends string>({
  options,
  value,
  onChange,
  ariaLabel,
  className,
}: {
  options: MobileSegmentOption<T>[]
  value: T
  onChange: (next: T) => void
  ariaLabel?: string
  className?: string
}) {
  return (
    <div
      role="tablist"
      aria-label={ariaLabel}
      style={{ gridTemplateColumns: `repeat(${Math.max(options.length, 1)}, minmax(0, 1fr))` }}
      className={cn(
        'grid gap-1 rounded-xl border border-border bg-secondary/50 p-1',
        className,
      )}
    >
      {options.map((opt) => {
        const active = opt.id === value
        return (
          <button
            key={opt.id}
            type="button"
            role="tab"
            aria-selected={active}
            onClick={() => onChange(opt.id)}
            className={cn(
              'flex min-w-0 items-center justify-center gap-1 rounded-lg px-2 py-1.5 text-[11.5px] font-medium transition-all',
              active
                ? 'bg-card font-semibold text-foreground shadow-xs'
                : 'text-muted-foreground hover:text-foreground',
            )}
          >
            <span className="truncate">{opt.label}</span>
            {opt.count !== undefined && (
              <span
                className={cn(
                  'shrink-0 rounded-full px-1.5 py-0.5 text-[9.5px] font-mono tabular',
                  active
                    ? 'bg-foreground text-background'
                    : 'bg-background/80 text-muted-foreground',
                )}
              >
                {opt.count}
              </span>
            )}
          </button>
        )
      })}
    </div>
  )
}
