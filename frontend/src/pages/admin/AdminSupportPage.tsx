import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Alert, Badge, Button, Card, CardContent, CardDescription, CardHeader, CardTitle, Input, Label, PageHeader, Skeleton } from '@/components/ui'
import { adminSystemApi } from '@/lib/system-api'
import { adminUserApi } from '@/lib/admin-api'
import { useDebouncedValue } from '@/features/admin/hooks/use-debounce'
import { formatDateTime } from '@/lib/format'
import type { AdminUserListItem } from '@/types/admin'

/**
 * AdminSupportPage adalah jurnal dukungan per pelanggan.
 *
 * Ini bukan sistem tiket: catatan ditulis staf setelah percakapan, dan
 * channel menandai dari mana percakapan datang. Alasan pemisahannya dari
 * catatan internal: jurnal dibaca saat menjawab, catatan internal saat
 * memutuskan (plans/18 §3).
 */
export default function AdminSupportPage() {
  const [search, setSearch] = useState('')
  const debounced = useDebouncedValue(search, 350)
  const [selected, setSelected] = useState<AdminUserListItem | null>(null)

  const users = useQuery({
    queryKey: ['admin', 'users', 'support-search', debounced],
    queryFn: () => adminUserApi.list({ q: debounced, per_page: 20 }),
    enabled: debounced.trim().length > 1,
  })

  return (
    <>
      <PageHeader
        title="Dukungan"
        description="Catat hasil percakapan agar staf lain bisa melanjutkan tanpa menanyakan hal yang sama."
      />

      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Pilih pelanggan</CardTitle>
          <CardDescription className="text-xs">
            Cari berdasarkan email atau nama, minimal dua karakter.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-2">
          <Input
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Cari email atau nama"
            aria-label="Cari pelanggan"
          />

          {users.isFetching && <Skeleton className="h-10 w-full" />}

          {(users.data?.data ?? []).map((user) => (
            <button
              key={user.id}
              type="button"
              onClick={() => setSelected(user)}
              className="flex w-full items-center justify-between gap-2 rounded-md border border-border px-3 py-2 text-left text-[13px] hover:bg-secondary/40"
            >
              <span className="min-w-0">
                <span className="block truncate font-medium">{user.name}</span>
                <span className="block truncate text-[11px] text-muted-foreground">{user.email}</span>
              </span>
              <Badge variant="outline">{user.tier}</Badge>
            </button>
          ))}
        </CardContent>
      </Card>

      {selected && <SupportThread userID={selected.id} label={selected.email} />}
    </>
  )
}

function SupportThread({ userID, label }: { userID: string; label: string }) {
  const queryClient = useQueryClient()
  const [body, setBody] = useState('')
  const [channel, setChannel] = useState<'internal' | 'telegram' | 'email' | 'phone'>('telegram')
  const [pinned, setPinned] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)

  const notes = useQuery({
    queryKey: ['admin', 'support', userID],
    queryFn: () => adminSystemApi.supportNotes(userID),
  })

  const add = useMutation({
    mutationFn: () =>
      adminSystemApi.addSupportNote(userID, { body, channel, is_pinned: pinned }),
    onSuccess: () => {
      setBody('')
      setPinned(false)
      setNotice('Catatan tersimpan.')
      void queryClient.invalidateQueries({ queryKey: ['admin', 'support', userID] })
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal menyimpan catatan.'),
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Jurnal — {label}</CardTitle>
        <CardDescription className="text-xs">
          Catatan bersifat tambahan; tidak bisa diubah setelah disimpan.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        {notice && <Alert>{notice}</Alert>}
        {notes.isLoading && <Skeleton className="h-20 w-full" />}

        {(notes.data ?? []).length === 0 && !notes.isLoading && (
          <p className="text-[13px] text-muted-foreground">Belum ada catatan untuk pelanggan ini.</p>
        )}

        {(notes.data ?? []).map((note) => (
          <div key={note.id} className="border-b border-border pb-2 last:border-0">
            <div className="flex items-center gap-2 text-[11px] text-muted-foreground">
              <Badge variant={note.is_pinned ? 'solid' : 'default'}>{note.channel}</Badge>
              <span>{formatDateTime(note.created_at)}</span>
              {note.is_pinned && <span>· disematkan</span>}
            </div>
            <p className="mt-1 whitespace-pre-line text-[13px]">{note.body}</p>
          </div>
        ))}

        <div className="space-y-2 border-t border-border pt-3">
          <div>
            <Label htmlFor="note-body">Catatan baru</Label>
            <textarea
              id="note-body"
              value={body}
              onChange={(event) => setBody(event.target.value)}
              rows={3}
              className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              placeholder="Contoh: pengguna menanyakan kenapa sinyal tidak dieksekusi; dijawab kuota harian sudah habis."
            />
          </div>
          <div className="flex flex-wrap items-end gap-3">
            <div>
              <Label htmlFor="note-channel">Saluran</Label>
              <select
                id="note-channel"
                value={channel}
                onChange={(event) => setChannel(event.target.value as typeof channel)}
                className="mt-1 rounded-md border border-input bg-background px-3 py-2 text-sm"
              >
                <option value="telegram">Telegram</option>
                <option value="email">Email</option>
                <option value="phone">Telepon</option>
                <option value="internal">Internal</option>
              </select>
            </div>
            <label className="flex items-center gap-2 text-[13px]">
              <input type="checkbox" checked={pinned} onChange={(event) => setPinned(event.target.checked)} />
              Sematkan
            </label>
            <Button size="sm" disabled={body.trim().length < 2 || add.isPending} onClick={() => add.mutate()}>
              Simpan catatan
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
