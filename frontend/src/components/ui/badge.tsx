import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'

const badgeVariants = cva(
  'inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors',
  {
    variants: {
      variant: {
        default: 'border-border text-muted-foreground',
        primary: 'border-primary/35 bg-primary/10 text-primary',
        success: 'border-success/35 bg-success/10 text-success',
        warning: 'border-warning/35 bg-warning/10 text-warning',
        destructive: 'border-destructive/35 bg-destructive/10 text-destructive',
        accent: 'border-[hsl(270_80%_78%)/35] bg-[hsl(270_80%_78%)/10] text-[hsl(270_80%_78%)]',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  },
)

export interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>, VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ variant }), className)} {...props} />
}

export { Badge, badgeVariants }
