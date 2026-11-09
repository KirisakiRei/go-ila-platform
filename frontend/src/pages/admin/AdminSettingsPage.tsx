import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Alert,
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  EmptyState,
  Input,
  Label,
  PageHeader,
  Skeleton,
} from '@/components/ui'
import { adminSystemApi } from '@/lib/system-api'
import type { SettingRow } from '@/types/system'

/**
 * AdminSettingsPage mengatur flag operasional.
 *
 * Tabel ini murni override: baris yang dihapus membuat nilai environment
 * berlaku kembali. Setiap perubahan wajib menyebut alasan dan mengetik ulang
 * nama flag, karena flag di sini menyentuh jalur eksekusi (plans/11 I3–I4).
 */
export default function AdminSettingsPage() {
  const queryClient = useQueryClient()
  const [notice, setNotice] = useState<string | null>(null)
  const [editing, setEditing] = useState<string | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'system', 'settings'],
    queryFn: adminSystemApi.settings,
  })

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['admin', 'system'] })
  }

  const save = useMutation({
    mutationFn: (payload: { key: string; value: string; reason: string; confirm: string }) =>
      adminSystemApi.updateSetting(payload.key, payload),
    onSuccess: () => {
      setNotice('Flag diperbarui. Nilai berlaku paling lama 10 detik.')
      setEditing(null)
      invalidate()
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal menyimpan flag.'),
  })

  const reset = useMutation({
    mutationFn: (payload: { key: string; reason: string; confirm: string }) =>
      adminSystemApi.resetSetting(payload.key, payload),
    onSuccess: () => {
      setNotice('Flag dikembalikan ke nilai bawaan (environment).')
      setEditing(null)
      invalidate()
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal mengembalikan flag.'),
  })

  const rows: SettingRow[] = data ?? []

  return (
    <>
      <PageHeader
        title="Pengaturan Sistem"
        description="Perubahan berlaku untuk seluruh pengguna platform. Nilai dari database menimpa konfigurasi bawaan."
      />

      {notice && <Alert>{notice}</Alert>}

      <Alert variant="warning">
        Mengubah flag saat pasar berjalan memengaruhi sinyal berikutnya. Posisi yang sudah terbuka tidak
        tersentuh: batas harian dan berhenti-setelah-menang hanya menahan entry baru.
      </Alert>

      {isLoading && (
        <div className="space-y-2">
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-12 w-full" />
          <Skeleton className="h-12 w-full" />
        </div>
      )}

      {!isLoading && rows.length === 0 && (
        <EmptyState title="Belum ada flag" description="Katalog flag kosong pada build ini." />
      )}

      <div className="space-y-3">
        {rows.map((row) => (
          <Card key={row.key}>
            <CardHeader>
              <div className="flex flex-wrap items-start justify-between gap-2">
                <div>
                  <CardTitle className="text-sm">{row.label}</CardTitle>
                  <CardDescription className="mt-1 text-xs">{row.help}</CardDescription>
                </div>
                <div className="flex items-center gap-2">
                  {row.danger && <Badge variant="solid">Berisiko</Badge>}
                  <Badge variant={row.source === 'database' ? 'outline' : 'default'}>
                    {row.source === 'database' ? 'Dari database' : 'Bawaan env'}
                  </Badge>
                </div>
              </div>
            </CardHeader>
            <CardContent>
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="flex items-baseline gap-2">
                  <code className="text-[11px] uppercase tracking-wider text-muted-foreground">{row.key}</code>
                  <span className="text-lg font-semibold tabular-nums">{row.value}</span>
                </div>
                <Button size="sm" variant="outline" onClick={() => setEditing(editing === row.key ? null : row.key)}>
                  {editing === row.key ? 'Tutup' : 'Ubah'}
                </Button>
              </div>

              {editing === row.key && (
                <FlagForm
                  row={row}
                  busy={save.isPending || reset.isPending}
                  onSubmit={(value, reason, confirm) => save.mutate({ key: row.key, value, reason, confirm })}
                  onReset={(reason, confirm) => reset.mutate({ key: row.key, reason, confirm })}
                />
              )}
            </CardContent>
          </Card>
        ))}
      </div>
    </>
  )
}

type FlagFormProps = {
  row: SettingRow
  busy: boolean
  onSubmit: (value: string, reason: string, confirm: string) => void
  onReset: (reason: string, confirm: string) => void
}

/** FlagForm memaksa alasan dan pengetikan ulang nama flag sebelum menyimpan. */
function FlagForm({ row, busy, onSubmit, onReset }: FlagFormProps) {
  const [value, setValue] = useState(row.value)
  const [reason, setReason] = useState('')
  const [confirm, setConfirm] = useState('')

  const valid = reason.trim().length >= 10 && confirm.trim() === row.key

  return (
    <div className="mt-4 space-y-3 border-t border-border pt-4">
      <div className="grid gap-3 sm:grid-cols-2">
        <div>
          <Label htmlFor={`value-${row.key}`}>Nilai baru</Label>
          {row.kind === 'bool' ? (
            <select
              id={`value-${row.key}`}
              value={value}
              onChange={(event) => setValue(event.target.value)}
              className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            >
              <option value="true">true</option>
              <option value="false">false</option>
            </select>
          ) : (
            <Input
              id={`value-${row.key}`}
              type="number"
              min={row.min ?? 1}
              max={row.max ?? 20}
              value={value}
              onChange={(event) => setValue(event.target.value)}
              className="mt-1"
            />
          )}
          {row.kind === 'int' && (
            <p className="mt-1 text-[11px] text-muted-foreground">
              Rentang yang diizinkan {row.min}–{row.max}.
            </p>
          )}
        </div>

        <div>
          <Label htmlFor={`confirm-${row.key}`}>Ketik ulang nama flag</Label>
          <Input
            id={`confirm-${row.key}`}
            value={confirm}
            onChange={(event) => setConfirm(event.target.value)}
            placeholder={row.key}
            className="mt-1 font-mono text-xs"
          />
        </div>
      </div>

      <div>
        <Label htmlFor={`reason-${row.key}`}>Alasan perubahan (minimal 10 karakter)</Label>
        <Input
          id={`reason-${row.key}`}
          value={reason}
          onChange={(event) => setReason(event.target.value)}
          placeholder="Contoh: menahan entry sementara karena broker gangguan"
          className="mt-1"
        />
      </div>

      <div className="flex flex-wrap gap-2">
        <Button size="sm" disabled={!valid || busy} onClick={() => onSubmit(value, reason, confirm)}>
          Simpan
        </Button>
        <Button
          size="sm"
          variant="outline"
          disabled={!valid || busy || row.source !== 'database'}
          onClick={() => onReset(reason, confirm)}
        >
          Kembalikan ke bawaan
        </Button>
      </div>
    </div>
  )
}
