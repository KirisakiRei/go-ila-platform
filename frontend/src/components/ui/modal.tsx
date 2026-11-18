import { useEffect, useRef, type ReactNode } from 'react'
import { X } from 'lucide-react'
import { cn } from '@/lib/utils'

type ModalProps = {
  open: boolean
  onClose: () => void
  title?: ReactNode
  description?: ReactNode
  children: ReactNode
  footer?: ReactNode
  size?: 'sm' | 'md' | 'lg' | 'xl' | '2xl'
  className?: string
}

const SIZE_CLASSES = {
  sm: 'max-w-sm',
  md: 'max-w-md',
  lg: 'max-w-lg',
  xl: 'max-w-xl',
  '2xl': 'max-w-2xl',
}

/**
 * Modal seragam untuk seluruh dialog aksi (tambah, edit, konfirmasi hapus).
 * Mendukung penutupan dengan tombol ESC, klik backdrop, dan tombol X.
 */
export function Modal({
  open,
  onClose,
  title,
  description,
  children,
  footer,
  size = 'md',
  className,
}: ModalProps) {
  const contentRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return

    const originalOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose()
      }
    }

    window.addEventListener('keydown', onKeyDown)
    return () => {
      document.body.style.overflow = originalOverflow
      window.removeEventListener('keydown', onKeyDown)
    }
  }, [open, onClose])

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center overflow-y-auto bg-black/60 p-0 backdrop-blur-sm sm:items-center sm:p-4 animate-in fade-in duration-150"
      role="dialog"
      aria-modal="true"
      onClick={(event) => {
        if (event.target === event.currentTarget) {
          onClose()
        }
      }}
    >
      <div
        ref={contentRef}
        className={cn(
          'relative max-h-[90dvh] w-full overflow-y-auto rounded-t-2xl border border-border bg-card p-4 pb-[calc(1rem+env(safe-area-inset-bottom))] shadow-xl sm:max-h-[90vh] sm:rounded-xl sm:p-6 animate-in zoom-in-95 duration-150',
          SIZE_CLASSES[size],
          className,
        )}
      >
        <button
          type="button"
          onClick={onClose}
          className="absolute top-3.5 right-3.5 rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground sm:top-4 sm:right-4"
          aria-label="Tutup dialog"
        >
          <X className="h-4 w-4" aria-hidden="true" />
        </button>

        {(title || description) && (
          <header className="mb-4 pr-7 sm:mb-5">
            {title &&
              (typeof title === 'string' ? (
                <h2 className="text-base font-semibold tracking-tight">{title}</h2>
              ) : (
                <div className="text-base font-semibold tracking-tight">{title}</div>
              ))}
            {description &&
              (typeof description === 'string' ? (
                <p className="mt-1 text-xs leading-relaxed text-muted-foreground">{description}</p>
              ) : (
                <div className="mt-1 text-xs leading-relaxed text-muted-foreground">{description}</div>
              ))}
          </header>
        )}

        <div className="min-w-0 space-y-4">{children}</div>

        {footer && (
          <footer className="mt-5 flex flex-wrap justify-end gap-2 border-t border-border pt-4 sm:mt-6">
            {footer}
          </footer>
        )}
      </div>
    </div>
  )
}
