import { useState } from 'react'
import { Alert, Button, Skeleton } from '@/components/ui'
import { journalApi } from './api'
import { useJournalMutation, useJournalReview, useJournalReviewSettings } from './hooks'
import { wibDay, wibTime } from './model'
import type { JournalReview as Review } from './types'

export function JournalReview({ accountId }: { accountId: string }) {
  const review = useJournalReview(accountId)
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
      setError(err instanceof Error ? err.message : 'Gagal menyimpan preferensi AI. Coba lagi.')
    }
  }
  const pending = review.data?.status === 'pending' || review.data?.status === 'running'
  return (
    <section
      className="space-y-3 rounded-lg border border-border bg-card p-4"
      aria-label="Review harian jurnal"
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="font-semibold">Review harian jurnal</h2>
        <span className="text-xs text-muted-foreground">
          {review.data?.source === 'ai'
            ? 'Sumber: AI'
            : review.data?.source === 'rules'
              ? 'Sumber: aturan'
              : 'Sumber: menunggu'}
          {pending ? ' · diproses' : ''}
        </span>
      </div>
      <p className="text-xs text-muted-foreground">
        Review akun sampai hari ini dalam WIB, tidak mengikuti filter tabel. Evaluasi berbasis
        aturan tetap tersedia walau AI dimatikan.
      </p>
      {review.isLoading && (
        <div role="status" aria-label="Memuat review harian">
          <Skeleton className="h-12 w-full" />
        </div>
      )}
      {review.error && (
        <Alert variant="warning">
          Review belum dapat dimuat: {review.error.message}
          <Button variant="ghost" size="sm" onClick={() => void review.refetch()}>
            Coba lagi
          </Button>
        </Alert>
      )}
      {review.data && (
        <>
          <p className="whitespace-pre-wrap text-sm" role="status">
            {review.data.text ||
              (pending
                ? 'Review sedang disiapkan. Sambil menunggu, gunakan evaluasi berbasis aturan di bawah.'
                : 'Belum ada teks review. Evaluasi berbasis aturan tetap bisa digunakan.')}
          </p>
          <p className="text-xs text-muted-foreground">
            Tanggal review {review.data.date} WIB · Diperbarui {wibTime(review.data.updated_at)} WIB
            {pending && review.data.text ? ' · Teks sebelumnya ditampilkan selama pembaruan.' : ''}
          </p>
          <DailyReviewBanner key={`${accountId}:${review.data.date}`} review={review.data} />
        </>
      )}
      <details className="border-t border-border pt-3">
        <summary className="w-fit cursor-pointer rounded text-sm font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring">
          Privasi & pengaturan AI
        </summary>
        <p className="mt-3 max-w-prose text-xs leading-relaxed text-muted-foreground">
          Jika AI efektif aktif, backend mengirim agregat numerik: jumlah trade, win/loss/breakeven,
          net P&L, win rate, dan drawdown ke penyedia AI yang dikonfigurasi server. Catatan,
          nama/tag setup, emosi, kesalahan, checklist, tautan, screenshot, dan isi jurnal mentah
          tidak dikirim ke AI. Mematikan salah satu pilihan di bawah menghentikan penggunaan AI
          untuk review berikutnya; evaluasi aturan tetap berjalan. Ini bukan rekomendasi transaksi.
        </p>
        {(error || settings.error || globalSettings.error) && (
          <Alert variant="destructive" className="mt-3">
            {error || settings.error?.message || globalSettings.error?.message}
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                void settings.refetch()
                void globalSettings.refetch()
              }}
            >
              Muat ulang pengaturan
            </Button>
          </Alert>
        )}
        {(settings.isLoading || globalSettings.isLoading) && (
          <p className="mt-3 text-xs" role="status">
            Memuat pengaturan privasi…
          </p>
        )}
        <div className="mt-3 space-y-3">
          <label className="flex items-center gap-3 text-sm">
            <input
              type="checkbox"
              className="h-4 w-4 accent-primary"
              checked={globalSettings.data?.ai_enabled ?? false}
              disabled={save.isPending || !globalSettings.data || globalSettings.isError}
              onChange={(e) => void toggle(e.target.checked)}
            />
            Izinkan review AI untuk seluruh jurnal saya
          </label>
          <label className="flex items-center gap-3 text-sm">
            <input
              type="checkbox"
              className="h-4 w-4 accent-primary"
              checked={settings.data?.ai_enabled ?? false}
              disabled={save.isPending || !settings.data || settings.isError}
              onChange={(e) => void toggle(e.target.checked, accountId)}
            />
            Izinkan review AI untuk akun ini
          </label>
        </div>
        {settings.data && (
          <p role="status" className="mt-3 text-xs text-muted-foreground">
            {settings.data.effective_ai_enabled
              ? 'AI efektif aktif untuk akun ini. Jika layanan AI gagal, backend menggunakan fallback aturan.'
              : 'AI efektif tidak aktif: preferensi global/akun dimatikan atau layanan AI server tidak tersedia. Review tetap memakai aturan.'}
          </p>
        )}
      </details>
    </section>
  )
}

function DailyReviewBanner({ review }: { review: Review }) {
  const key = `ila:manual-journal:daily-review:${review.account_id}:${review.date}`
  const [dismissed, setDismissed] = useState(() => {
    try {
      return localStorage.getItem(key) === 'dismissed'
    } catch {
      return false
    }
  })
  if (
    dismissed ||
    review.status !== 'ready' ||
    !review.text ||
    review.date !== wibDay(new Date().toISOString())
  )
    return null
  return (
    <aside className="rounded-md bg-secondary p-3" aria-label="Pengingat review hari ini">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 className="text-sm font-medium">Catatan untuk hari ini</h3>
          <p className="mt-1 text-sm">{review.text}</p>
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => {
            setDismissed(true)
            try {
              localStorage.setItem(key, 'dismissed')
            } catch {
              /* Storage may be blocked; dismissal still applies for this mounted session. */
            }
          }}
          aria-label="Tutup pengingat review hari ini"
        >
          Tutup
        </Button>
      </div>
    </aside>
  )
}
