import type * as React from 'react'
import { cn } from '@/lib/utils'

/**
 * Label status monokrom.
 *
 * Status dibedakan oleh latar dan gaya garis, bukan warna, supaya tetap
 * bermakna bagi pengguna dengan buta warna dan saat dicetak.
 */
type Tone = 'neutral' | 'solid' | 'outline' | 'muted'

const TONES: Record<Tone, string> = {
  neutral: 'border-border bg-transparent text-foreground',
  solid: 'border-foreground bg-foreground text-background',
  outline: 'border-foreground/40 bg-transparent text-foreground',
  muted: 'border-border bg-secondary text-muted-foreground',
}

export function StatusPill({
  children,
  tone = 'neutral',
  className,
}: {
  children: React.ReactNode
  tone?: Tone
  className?: string
}) {
  return (
    <span
      className={cn(
        'inline-flex items-center whitespace-nowrap rounded-md border px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wide',
        TONES[tone],
        className,
      )}
    >
      {children}
    </span>
  )
}
