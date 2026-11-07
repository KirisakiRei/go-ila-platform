import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { AlertCircle, AlertTriangle, Info } from 'lucide-react'
import { cn } from '@/lib/utils'

const alertVariants = cva('flex items-start gap-2.5 rounded-md border px-3 py-2.5 text-[13px]', {
  variants: {
    variant: {
      destructive: 'border-destructive/35 bg-destructive/10 text-[hsl(4_90%_80%)]',
      warning: 'border-warning/35 bg-warning/10 text-[hsl(40_80%_70%)]',
      info: 'border-primary/35 bg-primary/10 text-[hsl(217_90%_80%)]',
    },
  },
  defaultVariants: {
    variant: 'destructive',
  },
})

const ICONS = {
  destructive: AlertCircle,
  warning: AlertTriangle,
  info: Info,
} as const

export interface AlertProps
  extends React.HTMLAttributes<HTMLDivElement>, VariantProps<typeof alertVariants> {}

function Alert({ className, variant = 'destructive', children, ...props }: AlertProps) {
  const Icon = ICONS[variant ?? 'destructive']

  return (
    <div
      role="alert"
      aria-live="polite"
      className={cn(alertVariants({ variant }), className)}
      {...props}
    >
      <Icon className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
      <div>{children}</div>
    </div>
  )
}

export { Alert, alertVariants }
