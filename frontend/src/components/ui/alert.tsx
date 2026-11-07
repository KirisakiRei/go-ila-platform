import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { AlertCircle, AlertTriangle, CheckCircle2, Info } from 'lucide-react'
import { cn } from '@/lib/utils'

const alertVariants = cva('flex items-start gap-2.5 rounded-md border px-3 py-2.5 text-[13px]', {
  variants: {
    variant: {
      default: 'border-foreground/20 bg-secondary text-foreground',
      destructive: 'border-foreground bg-foreground text-background',
      warning: 'border-foreground/40 bg-transparent text-foreground',
      success: 'border-border bg-secondary text-foreground',
      info: 'border-border bg-transparent text-muted-foreground',
    },
  },
  defaultVariants: {
    variant: 'default',
  },
})

const ICONS = {
  default: Info,
  destructive: AlertCircle,
  warning: AlertTriangle,
  success: CheckCircle2,
  info: Info,
} as const

export interface AlertProps
  extends React.HTMLAttributes<HTMLDivElement>, VariantProps<typeof alertVariants> {}

function Alert({ className, variant = 'default', children, ...props }: AlertProps) {
  const Icon = ICONS[variant ?? 'default']

  return (
    <div
      role="alert"
      aria-live="polite"
      className={cn(alertVariants({ variant }), className)}
      {...props}
    >
      <Icon className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
      <div className="min-w-0 flex-1">{children}</div>
    </div>
  )
}

export { Alert, alertVariants }
