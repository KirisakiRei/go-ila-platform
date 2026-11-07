import type * as React from 'react'
import { cn } from '@/lib/utils'

/** Label status bersudut bulat dengan warna sesuai kelas semantik. */
export function StatusPill({
  children,
  tone = 'muted',
  className,
}: {
  children: React.ReactNode
  tone?: 'muted' | 'success' | 'warning' | 'danger' | 'info'
  className?: string
}) {
  const tones = {
    muted: 'border-border text-muted-foreground',
    success: 'border-success/35 bg-success/10 text-success',
    warning: 'border-warning/35 bg-warning/10 text-warning',
    danger: 'border-destructive/35 bg-destructive/10 text-destructive',
    info: 'border-primary/35 bg-primary/10 text-primary',
  } as const

  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium',
        tones[tone],
        className,
      )}
    >
      {children}
    </span>
  )
}
