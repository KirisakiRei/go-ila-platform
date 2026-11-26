import { useId, useState } from 'react'
import { ExternalLink } from 'lucide-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Alert } from '@/components/ui'
import { journalApi } from './api'
import { journalKey } from './hooks'
import { ConfirmButton, ImageDropzone } from './ui'

export function Screenshots({
  entryId,
  readOnly = false,
}: {
  entryId: string
  readOnly?: boolean
}) {
  const client = useQueryClient()
  const inputId = useId()
  const key = [...journalKey, 'screenshots', entryId]
  const query = useQuery({ queryKey: key, queryFn: () => journalApi.screenshots(entryId) })
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  async function handleFiles(files: File[]) {
    if (!files.length || busy || readOnly) return
    setBusy(true)
    setError('')
    const failed: string[] = []
    try {
      for (const file of files) {
        try {
          await journalApi.uploadScreenshot(entryId, file)
        } catch (err) {
          failed.push(err instanceof Error ? `${file.name}: ${err.message}` : file.name)
        }
      }
      await client.invalidateQueries({ queryKey: journalKey })
      if (failed.length) {
        setError(failed[0]!)
      }
    } finally {
      setBusy(false)
    }
  }

  return (
    <section className="space-y-3 border-t border-border pt-4">
      <div>
        <h3 className="text-sm font-semibold">Screenshot chart</h3>
      </div>

      {!readOnly && (
        <ImageDropzone
          id={inputId}
          multiple
          busy={busy}
          disabled={readOnly}
          ariaLabel="Unggah gambar chart"
          hint="Privat · otomatis dikompres · PNG, JPG, WebP, GIF"
          onFiles={(files) => void handleFiles(files)}
        />
      )}

      {(error || query.error) && (
        <Alert variant="destructive">{error || query.error?.message}</Alert>
      )}

      {query.isLoading && (
        <p role="status" className="text-xs text-muted-foreground">
          Memuat screenshot…
        </p>
      )}

      {!query.isLoading && !query.error && !query.data?.length && (
        <p className="rounded-lg border border-dashed border-border py-5 text-center text-xs text-muted-foreground">
          Belum ada screenshot terlampir
        </p>
      )}

      {!!query.data?.length && (
        <ul className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          {query.data.map((shot, i) => {
            const url = journalApi.screenshotUrl(entryId, shot.id)
            const name = shot.file_name || `Screenshot ${i + 1}`
            return (
              <li
                key={shot.id}
                className="overflow-hidden rounded-lg border border-border bg-secondary/20"
              >
                <a
                  href={url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="group relative block aspect-video overflow-hidden bg-secondary/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  <img
                    src={url}
                    alt={name}
                    loading="lazy"
                    className="h-full w-full object-cover transition-transform duration-200 ease-[var(--ease-out)] group-hover:scale-[1.02]"
                  />
                  <span className="absolute right-2 top-2 inline-flex items-center gap-1 rounded-md bg-background/85 px-1.5 py-0.5 text-[10px] font-medium text-foreground backdrop-blur-sm">
                    <ExternalLink className="h-2.5 w-2.5" aria-hidden="true" />
                    Buka
                  </span>
                </a>
                <div className="flex items-center justify-between gap-2 px-2.5 py-2">
                  <span className="min-w-0 truncate text-xs text-muted-foreground">{name}</span>
                  {!readOnly && (
                    <ConfirmButton
                      label="Hapus"
                      confirmLabel="Yakin?"
                      variant="ghost"
                      size="sm"
                      disabled={busy}
                      className="h-7 px-2 text-xs"
                      onConfirm={async () => {
                        setBusy(true)
                        setError('')
                        try {
                          await journalApi.deleteScreenshot(entryId, shot.id)
                          await client.invalidateQueries({ queryKey: journalKey })
                        } catch (err) {
                          setError(
                            err instanceof Error ? err.message : 'Gagal menghapus screenshot.',
                          )
                        } finally {
                          setBusy(false)
                        }
                      }}
                    />
                  )}
                </div>
              </li>
            )
          })}
        </ul>
      )}
    </section>
  )
}
