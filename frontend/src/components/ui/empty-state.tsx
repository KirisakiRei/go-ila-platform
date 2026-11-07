import type * as React from 'react'
import { cn } from '@/lib/utils'

/**
 * EmptyState menampilkan keadaan kosong secara jelas: apa yang belum ada dan
 * langkah berikutnya, bukan sekadar ruang kosong.
 */
export function EmptyState({
  icon,
  title,
  description,
  action,
  className,
}: {
  icon?: React.ReactNode
  title: string
  description?: string
  action?: React.ReactNode
  className?: string
}) {
  return (
    <div
      className={cn(
        'flex flex-col items-center gap-2 border border-dashed border-border px-6 py-10 text-center',
        className,
      )}
    >
      {icon && <div className="text-muted-foreground">{icon}</div>}
      <p className="text-[13px] font-semibold">{title}</p>
      {description && (
        <p className="max-w-sm text-[12px] leading-relaxed text-muted-foreground">{description}</p>
      )}
      {action && <div className="mt-2">{action}</div>}
    </div>
  )
}
