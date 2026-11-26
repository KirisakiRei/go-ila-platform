import { useState } from 'react'
import { Alert, Button, Input } from '@/components/ui'
import { journalApi } from './api'
import { useJournalMutation } from './hooks'
import { wibTime } from './model'
import type { CSVPreview } from './types'

export function CSVImport({ onDone }: { onDone: () => void }) {
  const [csv, setCSV] = useState('')
  const [preview, setPreview] = useState<CSVPreview | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const commit = useJournalMutation(journalApi.import)
  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        Gunakan format CSV hasil ekspor jurnal. Maksimal 1 MiB / 500 baris. Preview divalidasi
        server dan belum menyimpan apa pun.
      </p>
      {error && <Alert variant="destructive">{error}</Alert>}
      <label className="block space-y-1 text-sm">
        Berkas CSV
        <Input
          type="file"
          accept=".csv,text/csv"
          disabled={busy || commit.isPending}
          onChange={async (e) => {
            setPreview(null)
            setCSV('')
            setError('')
            const file = e.target.files?.[0]
            if (!file) return
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
            }
          }}
        />
      </label>
      {busy && (
        <p role="status" className="text-sm">
          Memvalidasi CSV…
        </p>
      )}
      {preview && (
        <>
          <p className="text-sm font-semibold">{preview.count} baris valid siap diimpor</p>
          <p className="text-xs text-muted-foreground">
            Preview berlaku sampai {wibTime(preview.expires_at)} WIB. Checklist divalidasi terhadap
            template akun saat impor.
          </p>
          <div className="max-h-64 overflow-auto rounded border border-border">
            <table className="w-full text-left text-xs">
              <caption className="sr-only">Preview impor, maksimal 20 baris pertama</caption>
              <thead>
                <tr>
                  {['Akun', 'Aset', 'Status', 'Net P&L'].map((h) => (
                    <th key={h} className="p-2" scope="col">
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {preview.rows.slice(0, 20).map((row, i) => (
                  <tr key={i} className="border-t border-border">
                    <td className="p-2">{row.account_id}</td>
                    <td className="p-2">{row.asset}</td>
                    <td className="p-2">{row.status}</td>
                    <td className="p-2">{row.net_pnl ?? '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <p className="text-xs text-muted-foreground">
            Menampilkan hingga 20 baris pertama. Konfirmasi memasukkan seluruh {preview.count}{' '}
            baris. Retry memakai token preview yang sama agar tidak menggandakan impor. Preview baru
            menghasilkan sesi impor baru.
          </p>
          <Button
            disabled={!preview.count}
            isLoading={commit.isPending}
            onClick={async () => {
              setError('')
              try {
                if (!preview.token || Date.parse(preview.expires_at) <= Date.now()) {
                  setError('Preview kedaluwarsa. Pilih berkas kembali untuk membuat preview baru.')
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
        </>
      )}
    </div>
  )
}
