import { cn } from '@/lib/utils'

export type StatusState = 'online' | 'offline' | 'pending' | 'idle'

/**
 * Indikator keadaan monokrom.
 * Bentuk berbeda per keadaan agar tidak bergantung pada warna:
 * terisi penuh = aktif, kosong = mati, separuh = menunggu.
 */
const STATE_CLASS: Record<StatusState, string> = {
  online: 'bg-foreground',
  offline: 'border border-foreground bg-transparent',
  pending:
    'border border-foreground bg-[linear-gradient(90deg,hsl(var(--foreground))_50%,transparent_50%)]',
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
      <span
        className={cn('block h-2.5 w-2.5 rounded-full', STATE_CLASS[state])}
        aria-hidden="true"
      />
      <span className="sr-only">{label}</span>
    </span>
  )
}
