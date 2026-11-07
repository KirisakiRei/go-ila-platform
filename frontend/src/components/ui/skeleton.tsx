import type * as React from 'react'
import { cn } from '@/lib/utils'

/** Skeleton menandai area yang sedang dimuat tanpa menggeser tata letak. */
export function Skeleton({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn('animate-pulse rounded-md bg-secondary/70', className)}
      aria-hidden="true"
      {...props}
    />
  )
}
