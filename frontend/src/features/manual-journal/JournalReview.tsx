import { useState } from 'react'
import { Lightbulb, Settings2, Sparkles } from 'lucide-react'
import { Alert, Button, Skeleton } from '@/components/ui'
import { cn } from '@/lib/utils'
import { journalApi } from './api'
import { useJournalMutation, useJournalReview, useJournalReviewSettings } from './hooks'
import { wibTime } from './model'
import { InfoTooltip } from './ui'

export function JournalReview({
  accountId,
  onOpenSettings,
}: {
  accountId: string
  onOpenSettings?: () => void
}) {
  const review = useJournalReview(accountId)
  const pending = review.data?.status === 'pending' || review.data?.status === 'running'
  const isAI = review.data?.source === 'ai'

  if (review.isLoading) {
    return (
      <div role="status" aria-label="Memuat insight jurnal">
        <Skeleton className="h-10 w-full rounded-lg" />
      </div>
    )
  }

  if (review.error) {
    return (
      <div className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-border bg-secondary/40 px-3.5 py-2 text-xs text-muted-foreground">
        <span>Insight harian belum termuat: {review.error.message}</span>
        <Button variant="ghost" size="sm" className="h-7 px-2 text-xs" onClick={() => void review.refetch()}>
          Coba lagi
        </Button>
      </div>
    )
  }

  const text =
    review.data?.text ||
    (pending
      ? 'Menyiapkan insight harian…'
      : 'Catat trade closed untuk melihat insight harian')

  const sourceLabel = isAI ? 'AI' : review.data?.source === 'rules' ? 'Aturan' : 'Menunggu'
  const metaTip = review.data
    ? `Update ${wibTime(review.data.updated_at)} WIB · Sumber: ${sourceLabel}`
    : 'Insight harian otomatis (WIB).'

  return (
    <div
      role="status"
      aria-label="Insight harian jurnal"
      className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1.5 rounded-lg border border-border/80 bg-secondary/35 px-3.5 py-2.5 text-xs sm:flex-nowrap"
    >
      <div className="flex min-w-0 flex-1 items-center gap-2.5">
        <span
          className={cn(
            'inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-md border border-border/70 bg-background text-muted-foreground',
            isAI && 'text-foreground',
          )}
        >
          {isAI ? (
            <Sparkles className="h-3.5 w-3.5" aria-hidden="true" />
          ) : (
            <Lightbulb className="h-3.5 w-3.5" aria-hidden="true" />
          )}
        </span>
        <p className="min-w-0 flex-1 text-[13px] leading-snug text-foreground">
          {text}
        </p>
      </div>

      <div className="flex shrink-0 items-center gap-2 pl-8 sm:pl-0">
        <span className="inline-flex items-center gap-1 rounded-md border border-border/70 bg-background px-2 py-0.5 text-[11px] font-medium text-muted-foreground">
          {sourceLabel}
          {pending ? ' · proses' : ''}
        </span>
        <InfoTooltip label="Info insight harian" content={metaTip} align="end" />
        {onOpenSettings && (
          <button
            type="button"
            onClick={onOpenSettings}
            className="inline-flex items-center gap-1 rounded-md px-1.5 py-1 text-[11px] font-medium text-muted-foreground transition-colors duration-150 hover:bg-secondary hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <Settings2 className="h-3 w-3" aria-hidden="true" />
            <span>Atur</span>
          </button>
        )}
      </div>
    </div>
  )
}

export function JournalAIPreferences({ accountId }: { accountId?: string }) {
  const settings = useJournalReviewSettings(accountId)
  const globalSettings = useJournalReviewSettings()
  const save = useJournalMutation(({ enabled, account }: { enabled: boolean; account?: string }) =>
    journalApi.saveReviewSettings(enabled, account),
  )
  const [error, setError] = useState('')

  async function toggle(enabled: boolean, account?: string) {
    setError('')
    try {
      await save.mutateAsync({ enabled, account })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Gagal menyimpan preferensi AI.')
    }
  }

  return (
    <div className="space-y-4">
      <p className="rounded-lg border border-border bg-secondary/35 p-3.5 text-xs leading-relaxed text-muted-foreground">
        Saat AI aktif, sistem hanya membaca ringkasan angka (jumlah trade, win rate, net P&amp;L,
        drawdown). Catatan, tag, emosi, checklist, link, dan screenshot tetap privat.
      </p>

      {(error || settings.error || globalSettings.error) && (
        <Alert variant="destructive">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <span>{error || settings.error?.message || globalSettings.error?.message}</span>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                void globalSettings.refetch()
                if (accountId) void settings.refetch()
              }}
            >
              Muat ulang
            </Button>
          </div>
        </Alert>
      )}

      {(globalSettings.isLoading || (accountId && settings.isLoading)) && (
        <p className="text-xs text-muted-foreground" role="status">
          Memuat pengaturan privasi…
        </p>
      )}

      <div className="divide-y divide-border rounded-lg border border-border bg-card">
        <label className="flex cursor-pointer items-center justify-between gap-4 p-3.5">
          <span className="min-w-0">
            <span className="block text-sm font-medium text-foreground">
              Aktifkan insight AI
            </span>
            <span className="mt-0.5 block text-xs text-muted-foreground">
              Berlaku untuk semua jurnal. Jika mati, hanya evaluasi aturan yang jalan.
            </span>
          </span>
          <input
            type="checkbox"
            role="switch"
            className="h-4 w-4 shrink-0 accent-primary"
            checked={globalSettings.data?.ai_enabled ?? false}
            disabled={save.isPending || !globalSettings.data || globalSettings.isError}
            onChange={(e) => void toggle(e.target.checked)}
          />
        </label>

        {accountId && (
          <label className="flex cursor-pointer items-center justify-between gap-4 p-3.5">
            <span className="min-w-0">
              <span className="block text-sm font-medium text-foreground">
                Aktifkan AI untuk jurnal ini
              </span>
              <span className="mt-0.5 block text-xs text-muted-foreground">
                Matikan jika jurnal ini cukup pakai evaluasi aturan.
              </span>
            </span>
            <input
              type="checkbox"
              role="switch"
              className="h-4 w-4 shrink-0 accent-primary"
              checked={settings.data?.ai_enabled ?? false}
              disabled={save.isPending || !settings.data || settings.isError}
              onChange={(e) => void toggle(e.target.checked, accountId)}
            />
          </label>
        )}
      </div>

      {settings.data && (
        <p role="status" className="text-xs text-muted-foreground">
          Status:{' '}
          <span className="font-medium text-foreground">
            {settings.data.effective_ai_enabled ? 'AI aktif' : 'Hanya aturan'}
          </span>
        </p>
      )}
    </div>
  )
}
