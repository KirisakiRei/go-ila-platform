import { Check, Loader2, AlertCircle } from 'lucide-react'
import type { UsedToolMeta } from './types'

type ThinkingIndicatorProps = {
  label: string
  tools: UsedToolMeta[]
  activeToolLabel?: string | null
}

/**
 * Menampilkan status proses AI (thinking -> pemanggilan tool) dengan transisi
 * masuk halus tanpa memicu layout shift.
 */
export function ThinkingIndicator({ label, tools, activeToolLabel }: ThinkingIndicatorProps) {
  const displayLabel = activeToolLabel || label || 'Sedang berpikir...'

  return (
    <div className="space-y-2 py-1" role="status" aria-live="polite">
      {tools.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {tools.map((tool, idx) => (
            <span
              key={`${tool.name}-${idx}`}
              className="inline-flex items-center gap-1.5 rounded-md border border-border bg-secondary/50 px-2 py-0.5 text-[11.5px] text-muted-foreground"
            >
              {tool.ok ? (
                <Check className="h-3 w-3 shrink-0 text-foreground" aria-hidden="true" />
              ) : (
                <AlertCircle className="h-3 w-3 shrink-0 text-muted-foreground" aria-hidden="true" />
              )}
              <span>{tool.summary || tool.label}</span>
            </span>
          ))}
        </div>
      )}
      <div className="flex items-center gap-2 text-[13px] text-muted-foreground">
        <Loader2 className="h-3.5 w-3.5 shrink-0 animate-spin text-foreground" aria-hidden="true" />
        <span key={displayLabel} className="animate-in fade-in duration-150">
          {displayLabel}
        </span>
      </div>
    </div>
  )
}
