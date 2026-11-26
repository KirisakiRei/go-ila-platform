import { useId, useState } from 'react'
import { FileSpreadsheet } from 'lucide-react'
import { Alert, Button } from '@/components/ui'
import { journalApi } from './api'
import { useJournalMutation } from './hooks'
import { wibTime } from './model'
import type { CSVPreview, JournalAccount } from './types'

export function CSVImport({
  accounts = [],
  onDone,
}: {
  accounts?: JournalAccount[]
  onDone: () => void
}) {
  const [csv, setCSV] = useState('')
  const [fileName, setFileName] = useState('')
  const [preview, setPreview] = useState<CSVPreview | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const fileId = useId()
  const commit = useJournalMutation(journalApi.import)

  function accountName(id: string) {
    return accounts.find((a) => a.id === id)?.name ?? id
  }

  return (
    <div className="space-y-4">
      <p className="text-xs text-muted-foreground">
        Gunakan format CSV ekspor Jurnal (maks 500 baris / 1 MiB).
      </p>

      {error && <Alert variant="destructive">{error}</Alert>}

      <label
        htmlFor={fileId}
        className="flex cursor-pointer flex-col items-center justify-center gap-1.5 rounded-xl border border-dashed border-border bg-secondary/20 px-4 py-6 text-center transition-colors hover:bg-secondary/40 focus-within:ring-2 focus-within:ring-ring focus-within:ring-offset-1"
      >
        <FileSpreadsheet className="h-5 w-5 text-muted-foreground" aria-hidden="true" />
        <span className="text-sm font-medium text-foreground">
          {fileName || 'Pilih berkas CSV'}
        </span>
        <span className="text-xs text-muted-foreground">Format .csv · maks 1 MiB</span>
        <input
          id={fileId}
          type="file"
          accept=".csv,text/csv"
          disabled={busy || commit.isPending}
          className="sr-only"
          aria-label="Berkas CSV"
          onChange={async (e) => {
            setPreview(null)
            setCSV('')
            setError('')
            const file = e.target.files?.[0]
            if (!file) return
            setFileName(file.name)
            if (file.size > 1024 * 1024) {
              setError('CSV melebihi 1 MiB.')
              return
            }
            setBusy(true)
            try {
              const text = await file.text()
              setCSV(text)
              setPreview(await journalApi.preview(text))
            } catch (err) {
              setError(err instanceof Error ? err.message : 'Gagal membaca CSV.')
            } finally {
              setBusy(false)
              e.target.value = ''
            }
          }}
        />
      </label>

      {busy && (
        <p role="status" className="text-xs text-muted-foreground">
          Memvalidasi CSV…
        </p>
      )}

      {preview && (
        <div className="space-y-3">
          <div className="flex flex-wrap items-baseline justify-between gap-2">
            <p className="text-sm font-semibold">{preview.count} baris valid siap diimpor</p>
            <span className="text-[11px] text-muted-foreground">
              Berlaku s/d {wibTime(preview.expires_at)} WIB
            </span>
          </div>

          <div className="max-h-56 overflow-auto rounded-lg border border-border">
            <table className="w-full text-left text-xs">
              <caption className="sr-only">Preview impor, maksimal 20 baris pertama</caption>
              <thead className="bg-secondary/50 text-muted-foreground">
                <tr>
                  {['Jurnal', 'Pair', 'Status', 'Net P&L'].map((h) => (
                    <th key={h} className="px-3 py-2 font-medium" scope="col">
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {preview.rows.slice(0, 20).map((row, i) => (
                  <tr key={i} className="border-t border-border">
                    <td className="px-3 py-2">{accountName(row.account_id)}</td>
                    <td className="px-3 py-2 font-medium">{row.asset || '—'}</td>
                    <td className="px-3 py-2">{row.status}</td>
                    <td className="px-3 py-2 tabular-nums">{row.net_pnl ?? '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="flex flex-wrap items-center justify-between gap-2">
            <span className="text-[11px] text-muted-foreground">
              Menampilkan hingga 20 baris pertama.
            </span>
            <Button
              size="sm"
              disabled={!preview.count}
              isLoading={commit.isPending}
              onClick={async () => {
                setError('')
                try {
                  if (!preview.token || Date.parse(preview.expires_at) <= Date.now()) {
                    setError('Preview kedaluwarsa. Pilih berkas kembali untuk preview baru.')
                    return
                  }
                  await commit.mutateAsync({ csv, token: preview.token })
                  onDone()
                } catch (err) {
                  setError(
                    err instanceof Error
                      ? err.message
                      : 'Impor gagal. Tidak ada baris yang disimpan.',
                  )
                }
              }}
            >
              Konfirmasi impor {preview.count} trade
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
