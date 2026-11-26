import { Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface SwitchProps {
  id?: string
  checked: boolean
  onCheckedChange: (next: boolean) => void
  disabled?: boolean
  loading?: boolean
  'aria-label'?: string
  title?: string
  className?: string
}

export function Switch({
  id,
  checked,
  onCheckedChange,
  disabled = false,
  loading = false,
  'aria-label': ariaLabel,
  title,
  className,
}: SwitchProps) {
  const isDisabled = disabled || loading

  return (
    <button
      id={id}
      type="button"
      role="switch"
      aria-label={ariaLabel}
      aria-checked={checked}
      aria-disabled={disabled || undefined}
      disabled={isDisabled}
      title={title}
      onClick={() => onCheckedChange(!checked)}
      className={cn(
        'inline-flex h-6 w-11 shrink-0 cursor-pointer items-center rounded-full border p-0.5 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background',
        checked
          ? 'border-primary bg-primary'
          : 'border-border bg-input/70 hover:bg-input dark:bg-secondary dark:hover:bg-secondary/80',
        disabled && 'cursor-not-allowed opacity-50',
        loading && 'cursor-wait opacity-70',
        className,
      )}
    >
      <span
        className={cn(
          'pointer-events-none flex h-[18px] w-[18px] items-center justify-center rounded-full shadow-xs transition-transform duration-200',
          checked
            ? 'translate-x-5 bg-primary-foreground'
            : 'translate-x-0 bg-background dark:bg-muted-foreground',
        )}
      >
        {loading && <Loader2 className="h-3 w-3 animate-spin text-muted-foreground" />}
      </span>
    </button>
  )
}
