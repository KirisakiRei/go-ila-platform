import type * as React from 'react'
import { cn } from '@/lib/utils'

/** Stat menampilkan satu angka penting beserta keterangan singkatnya. */
export function Stat({
  label,
  value,
  hint,
  className,
}: {
  label: string
  value: React.ReactNode
  hint?: string
  className?: string
}) {
  // Keterangan yang mengulang nilai utama hanya menambah kebisingan visual.
  const showHint = hint && hint !== value

  return (
    <div
      className={cn(
        'flex min-w-0 flex-col gap-0.5 rounded-xl border border-border bg-card p-3 sm:gap-1 sm:p-4',
        className,
      )}
    >
      <dt className="truncate text-[10.5px] font-medium uppercase tracking-wide text-muted-foreground sm:text-[11px]">
        {label}
      </dt>
      <dd className="truncate font-mono text-base font-semibold tabular leading-tight sm:text-xl">
        {value}
      </dd>
      {showHint && (
        <p className="line-clamp-1 text-[10.5px] text-muted-foreground sm:text-[11px]">{hint}</p>
      )}
    </div>
  )
}

/** StatGrid menyusun Stat dalam kisi rapi yang menyesuaikan lebar layar. */
export function StatGrid({
  children,
  className,
}: {
  children: React.ReactNode
  className?: string
}) {
  return <dl className={cn('grid grid-cols-2 gap-2.5 sm:gap-3 lg:grid-cols-4', className)}>{children}</dl>
}
