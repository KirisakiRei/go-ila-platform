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
    <div className={cn('flex flex-col gap-1 border border-border p-4', className)}>
      <dt className="text-micro font-medium uppercase tracking-wide text-muted-foreground">
        {label}
      </dt>
      <dd className="text-xl font-semibold tabular leading-none">{value}</dd>
      {showHint && <p className="text-micro text-muted-foreground">{hint}</p>}
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
  return <dl className={cn('grid grid-cols-2 gap-3 lg:grid-cols-4', className)}>{children}</dl>
}
