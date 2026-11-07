import { useState } from 'react'
import type { ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { adminUserApi } from '@/lib/admin-api'
import { formatIDR, formatInvoice } from '@/features/admin/lib/format'
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
  Select,
  Section,
  Skeleton,
  StatusPill,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui'
import type { AdminAccountItem, AdminPositionItem, UserActionPayload } from '@/types/admin'

const TIER_LABEL: Record<string, string> = { free: 'Free', basic: 'Basic', plus: 'Plus' }

const STATUS_TEXT: Record<string, string> = {
  active: 'Aktif',
  suspended: 'Ditangguhkan',
  banned: 'Diblokir',
  deleted: 'Dihapus',
}

/**
 * AdminUserDetailPage adalah pusat tindakan terhadap satu pelanggan.
 *
 * Batasnya ketat: kredensial broker tidak punya endpoint, dan admin tidak
 * bisa mengubah kata sandi atau menutup posisi (plans/07 §6.3, §8.1).
 */
export default function AdminUserDetailPage() {
  const { id = '' } = useParams()
  const queryClient = useQueryClient()

  const detail = useQuery({ queryKey: ['admin', 'user', id], queryFn: () => adminUserApi.detail(id) })
  const accounts = useQuery({
    queryKey: ['admin', 'user', id, 'accounts'],
    queryFn: () => adminUserApi.accounts(id),
  })
  const notes = useQuery({
    queryKey: ['admin', 'user', id, 'notes'],
    queryFn: () => adminUserApi.notes(id),
  })
  const legal = useQuery({
    queryKey: ['admin', 'user', id, 'legal'],
    queryFn: () => adminUserApi.legal(id),
  })
  const credit = useQuery({
    queryKey: ['admin', 'user', id, 'credit'],
    queryFn: () => adminUserApi.credit(id),
    retry: false,
  })

  const refresh = () => {
    void queryClient.invalidateQueries({ queryKey: ['admin', 'user', id] })
    void queryClient.invalidateQueries({ queryKey: ['admin', 'users'] })
  }

  if (detail.isLoading) {
    return <Skeleton className="h-64 w-full" />
  }
  if (detail.isError || !detail.data) {
    return (
      <EmptyState
        title="Pengguna tidak ditemukan"
        description="Data mungkin sudah berubah. Kembali ke daftar pengguna."
      />
    )
  }

  const user = detail.data

  return (
    <>
      <PageHeader
        title={user.name}
        description={`${user.email} · terdaftar ${formatInvoice(user.created_at.wib)} WIB`}
        actions={
          <Button variant="outline" size="sm" asChild>
            <Link to="/admin/pengguna">Kembali</Link>
          </Button>
        }
      />

      <Card>
        <CardHeader>
          <CardTitle>Ringkasan akun</CardTitle>
          <CardDescription>Status ini menentukan apa yang boleh dilakukan Runner.</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <Field label="Tier">
            <Badge variant={user.tier === 'free' ? 'default' : 'solid'}>
              {TIER_LABEL[user.tier] ?? user.tier}
            </Badge>
          </Field>
          <Field label="Status">
            <StatusPill
              tone={
                user.status === 'active'
                  ? 'outline'
                  : user.status === 'suspended'
                    ? 'muted'
                    : 'solid'
              }
            >
              {STATUS_TEXT[user.status] ?? user.status}
            </StatusPill>
          </Field>
          <Field label="Telegram">
            <span className="text-[13px]">{user.telegram_linked ? 'Terhubung' : 'Belum'}</span>
          </Field>
          <Field label="2FA">
            <span className="text-[13px]">{user.two_factor_enabled ? 'Aktif' : 'Tidak aktif'}</span>
          </Field>
        </CardContent>
        {user.ban_reason && <Alert className="mx-5 mb-5">{user.ban_reason}</Alert>}
      </Card>

      <UserActions user={user} onDone={refresh} />

      <Section title="Akun broker" description="Kredensial tidak pernah ditampilkan, termasuk untuk admin.">
        {accounts.data && accounts.data.length > 0 ? (
          <AccountsTable accounts={accounts.data} onDone={refresh} />
        ) : (
          <EmptyState title="Belum ada akun broker" description="Pelanggan belum menghubungkan broker." />
        )}
      </Section>

      <PositionsSection userID={id} />

      <Section title="Kredit" description="Setiap perubahan saldo tercatat di buku besar.">
        {credit.data ? (
          <Card className="p-4">
            <p className="text-[13px]">
              Saldo:{' '}
              <span className="font-semibold tabular-nums">{formatIDR(credit.data.balance_idr)}</span>
            </p>
          </Card>
        ) : (
          <p className="text-[12px] text-muted-foreground">Saldo kredit belum tersedia.</p>
        )}
      </Section>

      <Section title="Catatan internal" description="Hanya terlihat staf. Tidak pernah dikirim ke pelanggan.">
        <NotesBox userID={id} notes={notes.data ?? []} onDone={refresh} />
      </Section>

      <Section title="Persetujuan dokumen">
        {legal.data && legal.data.length > 0 ? (
          <Card className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Dokumen</TableHead>
                  <TableHead>Versi</TableHead>
                  <TableHead>Disetujui (WIB)</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {legal.data.map((item) => (
                  <TableRow key={`${item.slug}-${item.version}`}>
                    <TableCell className="text-[13px]">{item.title}</TableCell>
                    <TableCell className="text-[12px]">{item.version}</TableCell>
                    <TableCell className="text-[12px] text-muted-foreground">
                      {formatInvoice(item.accepted_at.wib)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Card>
        ) : (
          <p className="text-[12px] text-muted-foreground">Belum ada persetujuan tercatat.</p>
        )}
      </Section>
    </>
  )
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-[10px] uppercase tracking-wide text-muted-foreground">{label}</span>
      {children}
    </div>
  )
}

/** UserActions menaungi seluruh aksi berlapis terhadap pelanggan. */
function UserActions({ user, onDone }: { user: { id: string; status: string; email: string; tier: string }; onDone: () => void }) {
  const [pending, setPending] = useState<string | null>(null)
  const [reason, setReason] = useState('')
  const [category, setCategory] = useState('tos')
  const [confirmEmail, setConfirmEmail] = useState('')
  const [acknowledge, setAcknowledge] = useState(false)
  const [tier, setTier] = useState(user.tier)
  const [notice, setNotice] = useState<string | null>(null)

  const mutation = useMutation({
    mutationFn: async (action: string) => {
      const payload: UserActionPayload = { reason, category }

      switch (action) {
        case 'suspend':
          return adminUserApi.suspend(user.id, payload)
        case 'unsuspend':
          return adminUserApi.unsuspend(user.id)
        case 'ban':
          return adminUserApi.ban(user.id, { ...payload, confirm_email: confirmEmail })
        case 'unban':
          return adminUserApi.unban(user.id)
        case 'delete':
          return adminUserApi.remove(user.id, {
            ...payload,
            confirm_email: confirmEmail,
            acknowledge,
          })
        case 'tier':
          return adminUserApi.changeTier(user.id, { tier, reason })
        default:
          throw new Error('Aksi tidak dikenal')
      }
    },
    onSuccess: () => {
      setNotice('Tindakan tersimpan dan tercatat di audit.')
      setPending(null)
      setReason('')
      setConfirmEmail('')
      setAcknowledge(false)
      onDone()
    },
    onError: (error) => {
      setNotice(error instanceof Error ? error.message : 'Tindakan gagal.')
    },
  })

  const needsReason = pending !== null
  const needsEmail = pending === 'ban' || pending === 'delete'
  const canSubmit =
    !mutation.isPending &&
    (!needsReason || reason.trim().length >= 10) &&
    (!needsEmail || confirmEmail.trim() === user.email) &&
    (pending !== 'delete' || acknowledge)

  return (
    <Section title="Tindakan" description="Setiap tindakan wajib beralasan dan tercatat.">
      {notice && <Alert>{notice}</Alert>}

      <div className="flex flex-wrap gap-2">
        {user.status !== 'suspended' && (
          <Button variant="outline" size="sm" onClick={() => setPending('suspend')}>
            Tangguhkan
          </Button>
        )}
        {user.status === 'suspended' && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => mutation.mutate('unsuspend')}
            disabled={mutation.isPending}
          >
            Buka penangguhan
          </Button>
        )}
        {user.status !== 'banned' && (
          <Button variant="destructive" size="sm" onClick={() => setPending('ban')}>
            Blokir
          </Button>
        )}
        {user.status === 'banned' && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => mutation.mutate('unban')}
            disabled={mutation.isPending}
          >
            Buka blokir
          </Button>
        )}
        <Button variant="outline" size="sm" onClick={() => setPending('tier')}>
          Ubah tier
        </Button>
        <Button variant="destructive" size="sm" onClick={() => setPending('delete')}>
          Hapus akun
        </Button>
      </div>

      {pending && (
        <Card className="p-4">
          <div className="flex flex-col gap-3">
            {pending === 'tier' && (
              <div className="flex flex-col gap-1">
                <Label htmlFor="tier-select">Tier baru</Label>
                <Select id="tier-select" value={tier} onChange={(event) => setTier(event.target.value)}>
                  <option value="free">Free</option>
                  <option value="basic">Basic</option>
                  <option value="plus">Plus</option>
                </Select>
              </div>
            )}

            <div className="flex flex-col gap-1">
              <Label htmlFor="reason">Alasan</Label>
              <textarea
                id="reason"
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                rows={3}
                maxLength={500}
                className="w-full rounded-md border border-border bg-background px-3 py-2 text-[13px] outline-none focus-visible:ring-2 focus-visible:ring-ring"
                placeholder="Ditampilkan ke pelanggan, jadi tulis dengan bahasa yang pantas dibaca."
              />
              <p className="text-[11px] text-muted-foreground">
                Minimal 10 karakter. {reason.trim().length}/10
              </p>
            </div>

            {pending !== 'tier' && (
              <div className="flex flex-col gap-1">
                <Label htmlFor="category">Kategori</Label>
                <Select id="category" value={category} onChange={(event) => setCategory(event.target.value)}>
                  <option value="tos">Pelanggaran ToS</option>
                  <option value="trial_abuse">Penyalahgunaan trial</option>
                  <option value="unpaid">Gagal bayar</option>
                  <option value="user_request">Permintaan pelanggan</option>
                  <option value="fraud">Penipuan</option>
                  <option value="other">Lainnya</option>
                </Select>
              </div>
            )}

            {needsEmail && (
              <div className="flex flex-col gap-1">
                <Label htmlFor="confirm-email">Ketik ulang email pelanggan</Label>
                <Input
                  id="confirm-email"
                  value={confirmEmail}
                  onChange={(event) => setConfirmEmail(event.target.value)}
                  placeholder={user.email}
                  autoComplete="off"
                />
              </div>
            )}

            {pending === 'delete' && (
              <label className="flex items-start gap-2 text-[12px]">
                <input
                  type="checkbox"
                  checked={acknowledge}
                  onChange={(event) => setAcknowledge(event.target.checked)}
                  className="mt-0.5"
                />
                <span>Saya paham penghapusan tidak bisa dibatalkan.</span>
              </label>
            )}

            <div className="flex gap-2">
              <Button size="sm" disabled={!canSubmit} onClick={() => mutation.mutate(pending)}>
                Konfirmasi
              </Button>
              <Button variant="ghost" size="sm" onClick={() => setPending(null)}>
                Batal
              </Button>
            </div>
          </div>
        </Card>
      )}
    </Section>
  )
}

function AccountsTable({ accounts, onDone }: { accounts: AdminAccountItem[]; onDone: () => void }) {
  const freeze = useMutation({
    mutationFn: async ({ id, frozen, reason }: { id: string; frozen: boolean; reason: string }) =>
      frozen ? adminUserApi.freezeAccount(id, reason) : adminUserApi.unfreezeAccount(id),
    onSuccess: onDone,
  })

  return (
    <Card className="overflow-x-auto">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Label</TableHead>
            <TableHead>Broker</TableHead>
            <TableHead>Mode</TableHead>
            <TableHead>Status</TableHead>
            <TableHead className="text-right">Tindakan</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {accounts.map((account) => (
            <TableRow key={account.id}>
              <TableCell className="text-[13px]">{account.label || '—'}</TableCell>
              <TableCell className="text-[12px]">{account.broker_type}</TableCell>
              <TableCell className="text-[12px]">{account.mode}</TableCell>
              <TableCell className="text-[12px]">
                {account.is_frozen ? 'Dibekukan' : account.is_active ? 'Aktif' : 'Nonaktif'}
              </TableCell>
              <TableCell className="text-right">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={freeze.isPending}
                  onClick={() => {
                    const reason = account.is_frozen
                      ? ''
                      : window.prompt('Alasan pembekuan (minimal 10 karakter):') ?? ''
                    if (!account.is_frozen && reason.trim().length < 10) return
                    freeze.mutate({ id: account.id, frozen: !account.is_frozen, reason })
                  }}
                >
                  {account.is_frozen ? 'Buka beku' : 'Bekukan'}
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Card>
  )
}

/** PositionsSection dimuat terpisah: membukanya tercatat sebagai baca sensitif. */
function PositionsSection({ userID }: { userID: string }) {
  const [open, setOpen] = useState(false)
  const positions = useQuery({
    queryKey: ['admin', 'user', userID, 'positions'],
    queryFn: () => adminUserApi.positions(userID),
    enabled: open,
  })

  const rows: AdminPositionItem[] = positions.data ?? []

  return (
    <Section
      title="Riwayat posisi"
      description="Membuka data ini tercatat di audit sebagai pembacaan sensitif."
      actions={
        <Button variant="outline" size="sm" onClick={() => setOpen((value) => !value)}>
          {open ? 'Sembunyikan' : 'Tampilkan riwayat'}
        </Button>
      }
    >
      {open && positions.isLoading && <Skeleton className="h-32 w-full" />}
      {open && !positions.isLoading && rows.length === 0 && (
        <p className="text-[12px] text-muted-foreground">Belum ada posisi.</p>
      )}
      {open && rows.length > 0 && (
        <Card className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Simbol</TableHead>
                <TableHead>Arah</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">P/L</TableHead>
                <TableHead>Dibuka (WIB)</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => (
                <TableRow key={row.id}>
                  <TableCell className="text-[13px]">{row.symbol}</TableCell>
                  <TableCell className="text-[12px]">{row.direction}</TableCell>
                  <TableCell className="text-[12px]">{row.status}</TableCell>
                  <TableCell className="text-right text-[12px] tabular-nums">
                    {row.profit_loss.toFixed(2)}
                  </TableCell>
                  <TableCell className="text-[12px] text-muted-foreground">
                    {formatInvoice(row.opened_at.wib)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Card>
      )}
    </Section>
  )
}

function NotesBox({
  userID,
  notes,
  onDone,
}: {
  userID: string
  notes: { id: string; body: string; author_name: string; created_at: { wib: string } }[]
  onDone: () => void
}) {
  const [body, setBody] = useState('')
  const add = useMutation({
    mutationFn: () => adminUserApi.addNote(userID, body),
    onSuccess: () => {
      setBody('')
      onDone()
    },
  })

  return (
    <Card className="p-4">
      <div className="flex flex-col gap-2">
        <textarea
          value={body}
          onChange={(event) => setBody(event.target.value)}
          rows={2}
          maxLength={2000}
          placeholder="Tambahkan catatan untuk tim"
          className="w-full rounded-md border border-border bg-background px-3 py-2 text-[13px] outline-none focus-visible:ring-2 focus-visible:ring-ring"
        />
        <div>
          <Button size="sm" disabled={body.trim().length < 3 || add.isPending} onClick={() => add.mutate()}>
            Simpan catatan
          </Button>
        </div>
      </div>

      <ul className="mt-4 flex flex-col gap-2">
        {notes.map((note) => (
          <li key={note.id} className="rounded-md border border-border p-2">
            <p className="text-[12px]">{note.body}</p>
            <p className="mt-1 text-[10px] text-muted-foreground">
              {note.author_name || 'Staf'} · {note.created_at.wib}
            </p>
          </li>
        ))}
        {notes.length === 0 && <li className="text-[12px] text-muted-foreground">Belum ada catatan.</li>}
      </ul>
    </Card>
  )
}
