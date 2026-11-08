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
  Input,
  Label,
  PageHeader,
  Skeleton,
} from '@/components/ui'
import { adminSystemApi } from '@/lib/system-api'
import { formatDateTime } from '@/lib/format'

/** MODES menjelaskan akibat tiap mode tepat seperti yang ditegakkan sistem. */
const MODES = [
  {
    value: 'normal',
    label: 'Normal',
    effect: 'Seluruh alur berjalan: login, entry baru, dan pemantauan posisi.',
    confirm: false,
  },
  {
    value: 'read-only',
    label: 'Hanya baca',
    effect: 'Entry baru dilewati. Posisi berjalan tetap dipantau sampai ditutup broker.',
    confirm: true,
  },
  {
    value: 'maintenance',
    label: 'Pemeliharaan',
    effect: 'Pengguna melihat banner pemeliharaan. Entry baru ditahan, posisi tidak disentuh.',
    confirm: true,
  },
] as const

/**
 * AdminMaintenancePage mengubah mode platform.
 *
 * Posisi berjalan tidak pernah ditutup oleh mode apa pun: yang diatur hanya
 * penerimaan entry baru. Sinyal saat read-only dilewati, bukan ditunda,
 * karena harga sudah berubah (plans/18 §4).
 */
export default function AdminMaintenancePage() {
  const queryClient = useQueryClient()
  const [notice, setNotice] = useState<string | null>(null)
  const [mode, setMode] = useState<'normal' | 'read-only' | 'maintenance'>('normal')
  const [reason, setReason] = useState('')
  const [confirm, setConfirm] = useState('')
  const [expectedEnd, setExpectedEnd] = useState('')

  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'system', 'overview'],
    queryFn: adminSystemApi.overview,
  })

  const change = useMutation({
    mutationFn: () =>
      adminSystemApi.changeMode({
        mode,
        reason,
        confirm,
        expected_end: expectedEnd ? new Date(expectedEnd).toISOString() : undefined,
      }),
    onSuccess: () => {
      setNotice('Mode platform diperbarui. Posisi berjalan tidak tersentuh.')
      setConfirm('')
      queryClient.setQueryData(['admin', 'system', 'overview'], undefined)
      void queryClient.invalidateQueries({ queryKey: ['admin', 'system'] })
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal mengubah mode.'),
  })

  const selected = MODES.find((item) => item.value === mode) ?? MODES[0]
  const valid = reason.trim().length >= 10 && (!selected.confirm || confirm.trim() === 'MAINTENANCE')

  return (
    <>
      <PageHeader
        title="Pemeliharaan"
        description="Mengatur apakah entry baru diterima. Posisi yang sudah berjalan selalu dibiarkan berjalan."
      />

      {notice && <Alert>{notice}</Alert>}

      {isLoading && <Skeleton className="h-32 w-full" />}

      {data && (
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Mode saat ini</CardTitle>
            <CardDescription className="text-xs">
              Diubah {formatDateTime(data.mode.changed_at)}
              {data.mode.reason ? ` · ${data.mode.reason}` : ''}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Badge variant="solid">{data.mode.label}</Badge>
            {data.mode.expected_end && (
              <p className="mt-2 text-[12px] text-muted-foreground">
                Perkiraan selesai {formatDateTime(data.mode.expected_end)}
              </p>
            )}
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Ubah mode</CardTitle>
          <CardDescription className="text-xs">
            Setiap perubahan tercatat atas nama Anda beserta alasannya.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-2">
            {MODES.map((item) => (
              <label
                key={item.value}
                className="flex cursor-pointer items-start gap-2 rounded-md border border-border p-3 text-[13px] hover:bg-muted/40"
              >
                <input
                  type="radio"
                  name="mode"
                  value={item.value}
                  checked={mode === item.value}
                  onChange={() => setMode(item.value)}
                  className="mt-1"
                />
                <span>
                  <span className="font-medium">{item.label}</span>
                  <span className="block text-[12px] text-muted-foreground">{item.effect}</span>
                </span>
              </label>
            ))}
          </div>

          <div className="grid gap-3 sm:grid-cols-2">
            <div>
              <Label htmlFor="reason">Alasan (minimal 10 karakter)</Label>
              <Input
                id="reason"
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                placeholder="Contoh: gangguan sisi broker, menahan entry sementara"
                className="mt-1"
              />
            </div>
            <div>
              <Label htmlFor="expected">Perkiraan selesai (opsional)</Label>
              <Input
                id="expected"
                type="datetime-local"
                value={expectedEnd}
                onChange={(event) => setExpectedEnd(event.target.value)}
                className="mt-1"
              />
            </div>
          </div>

          {selected.confirm && (
            <div>
              <Label htmlFor="confirm">Ketik MAINTENANCE untuk konfirmasi</Label>
              <Input
                id="confirm"
                value={confirm}
                onChange={(event) => setConfirm(event.target.value)}
                placeholder="MAINTENANCE"
                className="mt-1 font-mono text-xs"
              />
            </div>
          )}

          <Button disabled={!valid || change.isPending} onClick={() => change.mutate()}>
            Terapkan mode
          </Button>
        </CardContent>
      </Card>

      <Alert variant="info">
        Saat mode hanya-baca atau pemeliharaan, sinyal baru <strong>dilewati</strong> dan tidak
        ditunda. Menjalankan sinyal lama setelah harga bergerak lebih berbahaya daripada melewatkannya.
      </Alert>
    </>
  )
}
