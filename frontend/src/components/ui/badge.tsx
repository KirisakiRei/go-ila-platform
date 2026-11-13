import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'

/**
 * Badge monokrom. Perbedaan makna disampaikan lewat latar dan bobot huruf,
 * bukan warna, sehingga tetap terbaca pada layar apa pun.
 */
const badgeVariants = cva(
  'inline-flex items-center gap-1 rounded-md border px-1.5 py-0.5 text-micro font-medium leading-4',
  {
    variants: {
      variant: {
        default: 'border-border bg-transparent text-muted-foreground',
        solid: 'border-foreground bg-foreground text-background',
        outline: 'border-foreground/25 bg-transparent text-foreground',
        muted: 'border-border bg-secondary text-foreground',
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
