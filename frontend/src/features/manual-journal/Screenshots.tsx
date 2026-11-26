import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Alert, Button, Input } from '@/components/ui'
import { journalApi } from './api'
import { journalKey } from './hooks'

export function Screenshots({
  entryId,
  readOnly = false,
}: {
  entryId: string
  readOnly?: boolean
}) {
  const client = useQueryClient()
  const key = [...journalKey, 'screenshots', entryId]
  const query = useQuery({ queryKey: key, queryFn: () => journalApi.screenshots(entryId) })
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  return (
    <section className="space-y-3 border-t border-border pt-4">
      <h3 className="text-sm font-semibold">Screenshot sebelum / sesudah</h3>
      <p className="text-xs text-muted-foreground">
        Gunakan nama berkas “sebelum” atau “sesudah” untuk membedakan bukti. Hanya pemilik yang
        dapat membuka berkas. Maksimal 5 MiB per gambar.
      </p>
      {(error || query.error) && (
        <Alert variant="destructive">{error || query.error?.message}</Alert>
      )}
      {query.isLoading && (
        <p role="status" className="text-xs">
          Memuat screenshot…
        </p>
      )}
      {!query.isLoading && !query.error && !query.data?.length && (
        <p className="text-xs text-muted-foreground">Belum ada screenshot.</p>
      )}
      <ul className="space-y-2">
        {query.data?.map((shot, i) => (
          <li key={shot.id} className="flex items-center justify-between gap-2">
            <a
              href={journalApi.screenshotUrl(entryId, shot.id)}
              target="_blank"
              rel="noopener noreferrer"
              className="min-w-0 truncate rounded text-sm underline underline-offset-4 focus-visible:ring-2 focus-visible:ring-ring"
            >
              {shot.file_name || `Screenshot ${i + 1}`} (tab baru)
            </a>
            <Button
              variant="ghost"
              size="sm"
              disabled={busy || readOnly}
              onClick={async () => {
                if (!window.confirm('Hapus screenshot ini?')) return
                setBusy(true)
                setError('')
                try {
                  await journalApi.deleteScreenshot(entryId, shot.id)
                  await client.invalidateQueries({ queryKey: journalKey })
                } catch (err) {
                  setError(err instanceof Error ? err.message : 'Gagal menghapus screenshot.')
                } finally {
                  setBusy(false)
                }
              }}
            >
              Hapus
            </Button>
          </li>
        ))}
      </ul>
      <label className="block space-y-1 text-sm">
        Unggah screenshot privat
        <Input
          type="file"
          accept="image/png,image/jpeg,image/webp,image/gif"
          disabled={busy || readOnly}
          onChange={async (e) => {
            const file = e.target.files?.[0]
            if (!file) return
            setBusy(true)
            setError('')
            try {
              await journalApi.uploadScreenshot(entryId, file)
              await client.invalidateQueries({ queryKey: journalKey })
              e.target.value = ''
            } catch (err) {
              setError(err instanceof Error ? err.message : 'Gagal mengunggah screenshot.')
            } finally {
              setBusy(false)
            }
          }}
        />
      </label>
      {busy && (
        <p role="status" className="text-xs">
          Memproses screenshot…
        </p>
      )}
    </section>
  )
}
