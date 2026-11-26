import {
  useCallback,
  useEffect,
  useId,
  useLayoutEffect,
  useRef,
  useState,
  type DragEvent,
  type KeyboardEvent,
  type ReactNode,
} from 'react'
import { createPortal } from 'react-dom'
import { ImagePlus, Info, Plus, X } from 'lucide-react'
import { Button } from '@/components/ui'
import { cn } from '@/lib/utils'
import { mergeChipDraft } from './model'

// Shared timestamp so hovering adjacent tooltips opens instantly with no delay.
let lastTooltipClosedAt = 0

export function InfoTooltip({
  label,
  content,
  align = 'center',
  className,
}: {
  label: string
  content: ReactNode
  align?: 'start' | 'center' | 'end'
  className?: string
}) {
  const [open, setOpen] = useState(false)
  const [instant, setInstant] = useState(false)
  const [coords, setCoords] = useState<{ top: number; left: number; placeBelow: boolean } | null>(
    null,
  )
  const timerRef = useRef<number | null>(null)
  const pointerDownRef = useRef(false)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const tooltipRef = useRef<HTMLSpanElement>(null)
  const tooltipId = useId()

  const clearTimer = useCallback(() => {
    if (timerRef.current !== null) {
      window.clearTimeout(timerRef.current)
      timerRef.current = null
    }
  }, [])

  const openWithDelay = useCallback(
    (immediate = false) => {
      clearTimer()
      const warm = immediate || Date.now() - lastTooltipClosedAt < 350
      if (warm) {
        setInstant(true)
        setOpen(true)
        return
      }
      setInstant(false)
      timerRef.current = window.setTimeout(() => {
        setOpen(true)
        timerRef.current = null
      }, 180)
    },
    [clearTimer],
  )

  const closeNow = useCallback(() => {
    clearTimer()
    setOpen((prev) => {
      if (prev) lastTooltipClosedAt = Date.now()
      return false
    })
  }, [clearTimer])

  const updatePosition = useCallback(() => {
    const btn = buttonRef.current
    if (!btn) return
    const rect = btn.getBoundingClientRect()
    const tipWidth = Math.min(256, window.innerWidth - 24)
    const tipHeight = tooltipRef.current?.offsetHeight ?? 72
    const margin = 12
    const gap = 8

    let left: number
    if (align === 'start') {
      left = rect.left
    } else if (align === 'end') {
      left = rect.right - tipWidth
    } else {
      left = rect.left + rect.width / 2 - tipWidth / 2
    }
    left = Math.max(margin, Math.min(left, window.innerWidth - tipWidth - margin))

    const placeBelow = rect.top - tipHeight - gap < margin
    const top = placeBelow ? rect.bottom + gap : Math.max(margin, rect.top - tipHeight - gap)
    setCoords({ top, left, placeBelow })
  }, [align])

  useLayoutEffect(() => {
    if (!open) return
    updatePosition()
    window.addEventListener('resize', updatePosition)
    window.addEventListener('scroll', updatePosition, true)
    return () => {
      window.removeEventListener('resize', updatePosition)
      window.removeEventListener('scroll', updatePosition, true)
    }
  }, [open, updatePosition])

  useEffect(() => {
    if (!open) return
    const onKey = (e: globalThis.KeyboardEvent) => {
      if (e.key === 'Escape') closeNow()
    }
    const onPointerDown = (e: PointerEvent) => {
      const target = e.target as Node
      if (
        !buttonRef.current?.contains(target) &&
        !tooltipRef.current?.contains(target)
      ) {
        closeNow()
      }
    }
    window.addEventListener('keydown', onKey)
    window.addEventListener('pointerdown', onPointerDown)
    return () => {
      window.removeEventListener('keydown', onKey)
      window.removeEventListener('pointerdown', onPointerDown)
    }
  }, [open, closeNow])

  useEffect(() => () => clearTimer(), [clearTimer])

  return (
    <span
      className={cn('relative inline-flex items-center', className)}
      onMouseEnter={() => {
        if (!pointerDownRef.current) openWithDelay(false)
      }}
      onMouseLeave={closeNow}
    >
      <button
        ref={buttonRef}
        type="button"
        aria-label={label}
        aria-expanded={open}
        aria-describedby={open ? tooltipId : undefined}
        onPointerDown={() => {
          pointerDownRef.current = true
        }}
        onFocus={() => {
          if (pointerDownRef.current) return
          openWithDelay(true)
        }}
        onBlur={(e) => {
          pointerDownRef.current = false
          const next = e.relatedTarget as Node | null
          if (!buttonRef.current?.contains(next) && !tooltipRef.current?.contains(next)) {
            closeNow()
          }
        }}
        onClick={(e) => {
          e.stopPropagation()
          pointerDownRef.current = false
          if (open) closeNow()
          else openWithDelay(true)
        }}
        className="inline-flex h-5 w-5 items-center justify-center rounded-full text-muted-foreground/80 transition-colors duration-150 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        <Info className="h-3.5 w-3.5" aria-hidden="true" />
      </button>
      {open &&
        typeof document !== 'undefined' &&
        createPortal(
          <span
            ref={tooltipRef}
            id={tooltipId}
            role="tooltip"
            data-instant={instant ? 'true' : 'false'}
            style={
              coords
                ? { top: coords.top, left: coords.left, width: 'min(16rem, calc(100vw - 1.5rem))' }
                : { visibility: 'hidden', top: 0, left: 0 }
            }
            className={cn(
              'ui-tooltip-enter pointer-events-auto fixed z-[60] rounded-lg border border-border bg-popover px-3 py-2 text-left text-xs font-normal leading-relaxed text-popover-foreground shadow-md',
              coords?.placeBelow ? 'origin-top' : 'origin-bottom',
            )}
          >
            {content}
          </span>,
          document.body,
        )}
    </span>
  )
}

export interface SegmentedOption<T extends string> {
  value: T
  label: string
  tone?: 'default' | 'buy' | 'sell'
  badge?: string | number
  disabled?: boolean
}

export function SegmentedControl<T extends string>({
  value,
  onChange,
  options,
  ariaLabel,
  size = 'md',
  fullWidth = false,
  disabled = false,
  invalid = false,
  className,
}: {
  value: T | ''
  onChange: (value: T) => void
  options: readonly SegmentedOption<T>[]
  ariaLabel: string
  size?: 'sm' | 'md'
  fullWidth?: boolean
  disabled?: boolean
  invalid?: boolean
  className?: string
}) {
  return (
    <div
      role="group"
      aria-label={ariaLabel}
      aria-invalid={invalid || undefined}
      className={cn(
        'inline-flex items-center gap-1 rounded-lg border bg-secondary/50 p-1',
        invalid ? 'border-destructive ring-1 ring-destructive/40' : 'border-border',
        fullWidth && 'grid w-full',
        className,
      )}
      style={
        fullWidth
          ? { gridTemplateColumns: `repeat(${options.length}, minmax(0, 1fr))` }
          : undefined
      }
    >
      {options.map((opt) => {
        const active = value === opt.value
        const activeClass =
          opt.tone === 'buy'
            ? 'bg-emerald-500/15 text-emerald-700 border-emerald-500/35 dark:text-emerald-300'
            : opt.tone === 'sell'
              ? 'bg-rose-500/15 text-rose-700 border-rose-500/35 dark:text-rose-300'
              : 'bg-background text-foreground border-border/80 shadow-sm'
        return (
          <button
            key={opt.value}
            type="button"
            disabled={disabled || opt.disabled}
            aria-pressed={active}
            onClick={() => onChange(opt.value)}
            className={cn(
              'inline-flex items-center justify-center gap-1.5 rounded-md border border-transparent font-medium transition-[color,background-color,border-color,transform] duration-150 ease-[var(--ease-out)] active:scale-[0.98] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50',
              size === 'sm' ? 'h-7 px-2.5 text-xs' : 'h-8 px-3 text-xs sm:text-[13px]',
              active ? activeClass : 'text-muted-foreground hover:text-foreground',
            )}
          >
            <span className="truncate">{opt.label}</span>
            {opt.badge !== undefined && opt.badge !== '' && opt.badge !== 0 && (
              <span
                className={cn(
                  'rounded px-1.5 py-0.5 text-[10px] font-medium tabular-nums',
                  typeof opt.badge === 'string' && opt.badge.includes('error')
                    ? 'bg-destructive/15 text-destructive'
                    : 'bg-secondary text-muted-foreground',
                )}
              >
                {opt.badge}
              </span>
            )}
          </button>
        )
      })}
    </div>
  )
}

export function ChipInput({
  id,
  label,
  values,
  onChange,
  draftValue,
  onDraftChange,
  placeholder,
  suggestions,
  disabled = false,
  maxItems = 30,
  error,
}: {
  id?: string
  label: string
  values: string[]
  onChange: (next: string[]) => void
  draftValue?: string
  onDraftChange?: (next: string) => void
  placeholder?: string
  suggestions?: readonly string[]
  disabled?: boolean
  maxItems?: number
  error?: string
}) {
  const [internalDraft, setInternalDraft] = useState('')
  const draft = draftValue ?? internalDraft
  const setDraft = onDraftChange ?? setInternalDraft
  const inputId = useId()
  const resolvedId = id ?? inputId

  function addValue(raw: string) {
    const next = mergeChipDraft(values, raw, maxItems)
    if (next !== values) onChange(next)
    setDraft('')
  }

  function removeValue(target: string) {
    onChange(values.filter((v) => v !== target))
  }

  function toggleSuggestion(item: string) {
    if (disabled) return
    if (values.includes(item)) {
      removeValue(item)
    } else if (values.length < maxItems) {
      onChange([...values, item])
    }
  }

  function onKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter') {
      e.preventDefault()
      addValue(draft)
    } else if (e.key === 'Backspace' && !draft && values.length > 0) {
      e.preventDefault()
      onChange(values.slice(0, -1))
    }
  }

  return (
    <div className="space-y-2">
      <label htmlFor={resolvedId} className="block text-xs font-medium text-muted-foreground">
        {label}
      </label>
      <div
        className={cn(
          'flex min-h-10 flex-wrap items-center gap-1.5 rounded-lg border bg-background px-2.5 py-1.5 transition-colors focus-within:ring-2 focus-within:ring-offset-1',
          error
            ? 'border-destructive focus-within:ring-destructive'
            : 'border-input focus-within:ring-ring',
          disabled && 'opacity-60',
        )}
      >
        {values.map((item) => (
          <span
            key={item}
            className="inline-flex max-w-full items-center gap-1 rounded-md border border-border bg-secondary/70 px-2 py-0.5 text-xs text-foreground"
          >
            <span className="truncate">{item}</span>
            {!disabled && (
              <button
                type="button"
                aria-label={`Hapus ${item}`}
                onClick={() => removeValue(item)}
                className="rounded text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              >
                <X className="h-3 w-3" aria-hidden="true" />
              </button>
            )}
          </span>
        ))}
        <div className="flex min-w-[120px] flex-1 items-center gap-1">
          <input
            id={resolvedId}
            type="text"
            value={draft}
            disabled={disabled || values.length >= maxItems}
            placeholder={values.length ? 'Tambah lagi…' : placeholder}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={onKeyDown}
            onBlur={() => {
              if (draft.trim()) addValue(draft)
            }}
            className="h-7 w-full bg-transparent text-sm text-foreground placeholder:text-muted-foreground focus:outline-none disabled:cursor-not-allowed"
          />
          {draft.trim() && !disabled && (
            <button
              type="button"
              aria-label={`Tambahkan ${label}`}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => addValue(draft)}
              className="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded bg-secondary text-foreground transition-colors hover:bg-accent"
            >
              <Plus className="h-3.5 w-3.5" aria-hidden="true" />
            </button>
          )}
        </div>
      </div>
      {error && (
        <p role="alert" className="text-[11px] font-medium text-destructive">
          {error}
        </p>
      )}
      {suggestions && suggestions.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {suggestions.map((item) => {
            const picked = values.includes(item)
            return (
              <button
                key={item}
                type="button"
                disabled={disabled}
                aria-pressed={picked}
                onClick={() => toggleSuggestion(item)}
                className={cn(
                  'rounded-md border px-2 py-0.5 text-[11px] font-medium transition-[color,background-color,border-color,transform] duration-150 active:scale-[0.97] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50',
                  picked
                    ? 'border-foreground/30 bg-foreground text-background'
                    : 'border-border bg-transparent text-muted-foreground hover:bg-secondary hover:text-foreground',
                )}
              >
                {picked ? `✓ ${item}` : `+ ${item}`}
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}

export function ConfirmButton({
  label,
  confirmLabel = 'Yakin hapus?',
  variant = 'ghost',
  size = 'sm',
  disabled = false,
  isLoading = false,
  onConfirm,
  className,
}: {
  label: string
  confirmLabel?: string
  variant?: 'ghost' | 'outline' | 'destructive'
  size?: 'sm' | 'default'
  disabled?: boolean
  isLoading?: boolean
  onConfirm: () => void | Promise<void>
  className?: string
}) {
  const [armed, setArmed] = useState(false)

  if (!armed) {
    return (
      <Button
        type="button"
        variant={variant}
        size={size}
        disabled={disabled}
        isLoading={isLoading}
        className={className}
        onClick={() => setArmed(true)}
      >
        {label}
      </Button>
    )
  }

  return (
    <span className={cn('inline-flex items-center gap-1.5', className)}>
      <Button
        type="button"
        variant="destructive"
        size={size}
        disabled={disabled}
        isLoading={isLoading}
        onClick={async () => {
          setArmed(false)
          await onConfirm()
        }}
      >
        {confirmLabel}
      </Button>
      <Button
        type="button"
        variant="ghost"
        size={size}
        disabled={disabled || isLoading}
        onClick={() => setArmed(false)}
      >
        Batal
      </Button>
    </span>
  )
}

export function ImageDropzone({
  id,
  multiple = true,
  disabled = false,
  busy = false,
  hint = 'PNG, JPG, WebP, GIF · maks 5 MiB',
  ariaLabel = 'Unggah gambar chart',
  onFiles,
}: {
  id: string
  multiple?: boolean
  disabled?: boolean
  busy?: boolean
  hint?: string
  ariaLabel?: string
  onFiles: (files: File[]) => void
}) {
  const [dragging, setDragging] = useState(false)
  const dragDepthRef = useRef(0)

  function handleDragEnter(e: DragEvent<HTMLLabelElement>) {
    e.preventDefault()
    if (disabled || busy) return
    dragDepthRef.current += 1
    setDragging(true)
  }

  function handleDragOver(e: DragEvent<HTMLLabelElement>) {
    e.preventDefault()
    if (disabled || busy) return
    e.dataTransfer.dropEffect = 'copy'
  }

  function handleDragLeave(e: DragEvent<HTMLLabelElement>) {
    e.preventDefault()
    dragDepthRef.current = Math.max(0, dragDepthRef.current - 1)
    if (dragDepthRef.current === 0) setDragging(false)
  }

  function handleDrop(e: DragEvent<HTMLLabelElement>) {
    e.preventDefault()
    dragDepthRef.current = 0
    setDragging(false)
    if (disabled || busy) return
    const dropped = Array.from(e.dataTransfer.files ?? [])
    if (dropped.length) onFiles(dropped)
  }

  return (
    <label
      htmlFor={id}
      onDragEnter={handleDragEnter}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
      className={cn(
        'flex min-h-[76px] cursor-pointer flex-col items-center justify-center gap-1 rounded-xl border border-dashed px-4 py-3.5 text-center transition-[color,background-color,border-color] duration-150 ease-[var(--ease-out)] focus-within:ring-2 focus-within:ring-ring focus-within:ring-offset-1',
        dragging
          ? 'border-foreground/60 bg-secondary/60 text-foreground'
          : 'border-border bg-secondary/20 text-muted-foreground hover:bg-secondary/35',
        (disabled || busy) && 'pointer-events-none opacity-60',
      )}
    >
      <ImagePlus className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
      <span className="text-xs font-medium text-foreground">
        {busy
          ? 'Mengunggah gambar…'
          : dragging
            ? 'Lepas gambar di sini'
            : 'Tarik & lepas chart di sini, atau klik pilih gambar'}
      </span>
      <span className="text-[11px] text-muted-foreground">{hint}</span>
      <input
        id={id}
        type="file"
        multiple={multiple}
        accept="image/png,image/jpeg,image/webp,image/gif"
        disabled={disabled || busy}
        aria-label={ariaLabel}
        className="sr-only"
        onChange={(e) => {
          const picked = Array.from(e.target.files ?? [])
          if (picked.length) onFiles(picked)
          e.target.value = ''
        }}
      />
    </label>
  )
}
