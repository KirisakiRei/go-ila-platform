import { useEffect, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { AnimatePresence, motion, useReducedMotion } from 'framer-motion'
import { Minimize2 } from 'lucide-react'
import { cn } from '@/lib/utils'

const EASE_OUT = [0.23, 1, 0.32, 1] as const

type ChartFullscreenModalProps = {
  open: boolean
  onClose: () => void
  title?: ReactNode
  toolbar?: ReactNode
  footer?: ReactNode
  ariaLabel?: string
  contentClassName?: string
  children: ReactNode
}

/**
 * ChartFullscreenModal membuka chart apa pun ke tampilan penuh viewport.
 *
 * Memakai portal ke document.body agar tidak pernah terpotong oleh ancestor
 * dengan overflow-hidden atau transform (mis. kartu Framer Motion), sekaligus
 * tetap mewarisi kelas `.dark` dari <html> dan berfungsi mulus di iOS Safari
 * yang tidak mendukung Element.requestFullscreen() pada <div>.
 */
export function ChartFullscreenModal({
  open,
  onClose,
  title,
  toolbar,
  footer,
  ariaLabel,
  contentClassName,
  children,
}: ChartFullscreenModalProps) {
  const reduceMotion = useReducedMotion() ?? false
  const resolvedAriaLabel =
    ariaLabel ?? (typeof title === 'string' && title.trim() !== '' ? title : 'Tampilan layar penuh chart')

  useEffect(() => {
    if (!open) return
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        onClose()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => {
      document.body.style.overflow = previousOverflow
      window.removeEventListener('keydown', handleKeyDown)
    }
  }, [open, onClose])

  if (typeof document === 'undefined') return null

  return createPortal(
    <AnimatePresence>
      {open && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center sm:p-3 md:p-4"
          role="dialog"
          aria-modal="true"
          aria-label={resolvedAriaLabel}
        >
          {/* Backdrop */}
          <motion.div
            aria-hidden="true"
            onClick={onClose}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: reduceMotion ? 0.1 : 0.16, ease: EASE_OUT }}
            className="fixed inset-0 bg-background/85 backdrop-blur-sm"
          />

          {/* Panel layar penuh */}
          <motion.div
            initial={reduceMotion ? { opacity: 0 } : { opacity: 0, scale: 0.97 }}
            animate={reduceMotion ? { opacity: 1 } : { opacity: 1, scale: 1 }}
            exit={
              reduceMotion
                ? { opacity: 0, transition: { duration: 0.1 } }
                : { opacity: 0, scale: 0.98, transition: { duration: 0.15, ease: EASE_OUT } }
            }
            transition={{ duration: reduceMotion ? 0.12 : 0.2, ease: EASE_OUT }}
            className="relative z-10 flex h-[100dvh] w-full flex-col overflow-hidden bg-card text-foreground pt-[env(safe-area-inset-top)] pb-[env(safe-area-inset-bottom)] shadow-2xl sm:h-[calc(100dvh-1.5rem)] sm:rounded-2xl sm:border sm:border-border sm:pt-0 sm:pb-0 md:h-[calc(100dvh-2rem)]"
          >
            {/* Bar atas ramping */}
            <header className="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-border bg-card/95 px-3 py-2 sm:px-4 sm:py-2.5">
              <div className="flex min-w-0 flex-1 items-center gap-2">
                {title ? (
                  <div className="min-w-0 truncate text-xs font-semibold tracking-tight text-foreground sm:text-sm">
                    {title}
                  </div>
                ) : (
                  <span className="text-xs font-semibold tracking-tight text-foreground sm:text-sm">
                    Tampilan Layar Penuh
                  </span>
                )}
              </div>

              <div className="flex flex-wrap items-center justify-end gap-1.5 sm:gap-2">
                {toolbar}
                <button
                  type="button"
                  onClick={onClose}
                  aria-label="Keluar layar penuh"
                  title="Keluar layar penuh (Esc)"
                  className="press inline-flex h-8 items-center gap-1.5 rounded-lg border border-border bg-secondary/50 px-2.5 text-xs font-medium text-foreground hover:border-foreground/30 hover:bg-secondary"
                >
                  <Minimize2 className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
                  <span className="hidden sm:inline">Tutup</span>
                  <kbd className="hidden rounded border border-border/80 bg-card px-1 py-0.5 font-mono text-[9.5px] font-medium text-muted-foreground md:inline">
                    Esc
                  </kbd>
                </button>
              </div>
            </header>

            {/* Kanvas chart mengisi penuh ruang tersisa */}
            <div className={cn('relative flex min-h-0 flex-1 flex-col', contentClassName)}>
              {children}
            </div>

            {footer && (
              <div className="shrink-0 border-t border-border bg-card px-3 py-2 text-[11px] text-muted-foreground sm:px-4">
                {footer}
              </div>
            )}
          </motion.div>
        </div>
      )}
    </AnimatePresence>,
    document.body,
  )
}
