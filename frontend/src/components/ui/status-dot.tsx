import { cn } from '@/lib/utils'

export type StatusState = 'online' | 'offline' | 'pending' | 'idle'

const STATE_CLASS: Record<StatusState, string> = {
  online: 'bg-success shadow-[0_0_0_3px_hsl(var(--success)/0.15)]',
  offline: 'bg-destructive shadow-[0_0_0_3px_hsl(var(--destructive)/0.15)]',
  pending: 'bg-warning shadow-[0_0_0_3px_hsl(var(--warning)/0.15)]',
  idle: 'bg-border',
}

type StatusDotProps = {
  state: StatusState
  label: string
  className?: string
}

/** Titik status kecil; teks lengkap dibacakan pembaca layar lewat sr-only. */
export function StatusDot({ state, label, className }: StatusDotProps) {
  return (
    <span className={cn('inline-flex items-center', className)}>
      <span className={cn('block h-2 w-2 rounded-full', STATE_CLASS[state])} aria-hidden="true" />
      <span className="sr-only">{label}</span>
    </span>
  )
}
