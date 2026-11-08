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
      className="fixed inset-0 z-50 flex items-center justify-center overflow-y-auto bg-black/60 p-4 backdrop-blur-sm animate-in fade-in duration-150"
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
          'relative w-full rounded-xl border border-border bg-card p-6 shadow-xl animate-in zoom-in-95 duration-150',
          SIZE_CLASSES[size],
          className,
        )}
      >
        <button
          type="button"
          onClick={onClose}
          className="absolute top-4 right-4 rounded-md p-1 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          aria-label="Tutup dialog"
        >
          <X className="h-4 w-4" aria-hidden="true" />
        </button>

        {(title || description) && (
          <header className="mb-5 pr-6">
            {title && <h2 className="text-base font-semibold tracking-tight">{title}</h2>}
            {description && (
              <p className="mt-1 text-xs leading-relaxed text-muted-foreground">{description}</p>
            )}
          </header>
        )}

        <div className="space-y-4">{children}</div>

        {footer && <footer className="mt-6 flex justify-end gap-2 border-t border-border pt-4">{footer}</footer>}
      </div>
    </div>
  )
}
