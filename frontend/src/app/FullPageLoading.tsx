import { Loader2 } from 'lucide-react'

export function FullPageLoading({ label }: { label: string }) {
  return (
    <div
      className="flex min-h-full flex-col items-center justify-center gap-3 text-muted-foreground"
      role="status"
      aria-live="polite"
    >
      <Loader2 className="h-5 w-5 animate-spin" aria-hidden="true" />
      <span className="text-[13px]">{label}</span>
    </div>
  )
}
