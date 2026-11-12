import { useState } from 'react'
import type { ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { adminUserApi } from '@/lib/admin-api'
import { formatIDR, formatInvoice } from '@/features/admin/lib/format'
import { Can } from '@/features/admin/components/Can'
import { usePermissions } from '@/features/admin/hooks/use-permissions'
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
  Pagination,
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
  usePaginasiLokal,
} from '@/components/ui'
import type {
  AdminAccountItem,
  AdminPositionItem,
  AdminUserDetail,
  UserActionPayload,
} from '@/types/admin'
import type { Tier } from '@/types/api'

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
  const { can, canAny } = usePermissions()

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
    // Dokumen persetujuan dijaga permission sendiri; jangan menembak endpoint
    // yang pasti ditolak 403.
    enabled: can('user.legal'),
  })
  // Persetujuan dokumen bisa bertambah tiap versi; dipecah per halaman.
  const paginasiLegal = usePaginasiLokal(legal.data)
  const credit = useQuery({
    queryKey: ['admin', 'user', id, 'credit'],
    queryFn: () => adminUserApi.credit(id),
    retry: false,
    enabled: canAny(['billing.view', 'billing.voucher']),
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
          <div className="flex items-center gap-2">
            {user.role === 'admin' && can('role.view') && (
              <Button variant="outline" size="sm" asChild>
                <Link to="/admin/rbac?tab=user">Kelola role admin</Link>
              </Button>
            )}
            <Button variant="outline" size="sm" asChild>
              <Link to="/admin/pengguna">Kembali</Link>
            </Button>
          </div>
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
          <EmptyState title="Belum ada akun broker" description="Pengguna belum menghubungkan broker." />
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

      <Section title="Catatan internal" description="Hanya terlihat staf pengelola platform.">
        <NotesBox userID={id} notes={notes.data ?? []} onDone={refresh} />
      </Section>

      <Section title="Persetujuan dokumen">
        {!can('user.legal') ? (
          <p className="text-[12px] text-muted-foreground">
            Role Anda tidak berwenang melihat bukti persetujuan legal.
          </p>
        ) : legal.data && legal.data.length > 0 ? (
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
                {paginasiLegal.baris.map((item) => (
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
            <div className="border-t border-border px-3 py-2">
              <Pagination
                page={paginasiLegal.page}
                totalPages={paginasiLegal.totalPages}
                shown={paginasiLegal.baris.length}
                total={paginasiLegal.total}
                unit="dokumen"
                pageSize={paginasiLegal.pageSize}
                onPageChange={paginasiLegal.setPage}
                onPageSizeChange={paginasiLegal.setPageSize}
              />
            </div>
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
function UserActions({ user, onDone }: { user: AdminUserDetail; onDone: () => void }) {
  const { canAny } = usePermissions()
  const [pending, setPending] = useState<string | null>(null)
  const [reason, setReason] = useState('')
  const [category, setCategory] = useState('tos')
  const [confirmEmail, setConfirmEmail] = useState('')
  const [acknowledge, setAcknowledge] = useState(false)
  const [tier, setTier] = useState<Tier>(user.tier)
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

  // Akun terhapus tetap dirender sebagai pemberitahuan; early-return diletakkan
  // setelah seluruh hook agar urutan hook tidak berubah antar render.
  if (user.status === 'deleted') {
    return (
      <Section title="Tindakan" description="Akun ini telah dihapus.">
        <div className="rounded-md border border-border bg-muted/40 p-3 text-[13px] text-muted-foreground">
          Akun ini sudah dihapus dan dianonimkan. Tidak ada tindakan lebih lanjut yang dapat dijalankan.
        </div>
      </Section>
    )
  }

  const needsReason = pending !== null
  const needsEmail = pending === 'ban' || pending === 'delete'
  const emailMatched = confirmEmail.trim().toLowerCase() === user.email.trim().toLowerCase()
  const canSubmit =
    !mutation.isPending &&
    (!needsReason || reason.trim().length >= 10) &&
    (!needsEmail || emailMatched) &&
    (pending !== 'delete' || acknowledge)

  return (
    <Section title="Tindakan" description="Setiap tindakan wajib beralasan dan tercatat.">
      {notice && <Alert>{notice}</Alert>}

      <div className="flex flex-wrap gap-2">
        <Can permission="user.suspend">
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
        </Can>
        <Can permission="user.ban">
          {/* Admin tidak bisa diblokir dari sini: hak admin dicabut lewat role. */}
          {user.status !== 'banned' && user.role !== 'admin' && (
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
        </Can>
        <Can permission="user.tier">
          <Button variant="outline" size="sm" onClick={() => setPending('tier')}>
            Ubah tier
          </Button>
        </Can>
        <Can permission="user.delete">
          <Button variant="destructive" size="sm" onClick={() => setPending('delete')}>
            Hapus akun
          </Button>
        </Can>
      </div>

      {!canAny(['user.suspend', 'user.ban', 'user.tier', 'user.delete']) && (
        <p className="text-[12px] text-muted-foreground">
          Role Anda hanya dapat melihat data pelanggan ini; tidak ada tindakan yang dapat dijalankan.
        </p>
      )}

      {pending && (
        <Card className="p-4">
          <div className="flex flex-col gap-3">
            {pending === 'tier' && (
              <div className="flex flex-col gap-1">
                <Label htmlFor="tier-select">Tier baru</Label>
                <Select id="tier-select" value={tier} onChange={(event) => setTier(event.target.value as Tier)}>
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
                <div className="flex items-center justify-between">
                  <Label htmlFor="confirm-email">Ketik ulang email pelanggan</Label>
                  <button
                    type="button"
                    className="text-[11px] text-primary hover:underline"
                    onClick={() => setConfirmEmail(user.email)}
                  >
                    Salin email
                  </button>
                </div>
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
  const { can } = usePermissions()
  // Daftar akun dipecah per halaman agar seragam dengan tabel lain.
  const paginasi = usePaginasiLokal(accounts)
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
          {paginasi.baris.map((account) => (
            <TableRow key={account.id}>
              <TableCell className="text-[13px]">{account.label || '—'}</TableCell>
              <TableCell className="text-[12px]">{account.broker_type}</TableCell>
              <TableCell className="text-[12px]">{account.mode}</TableCell>
              <TableCell className="text-[12px]">
                {account.is_frozen ? 'Dibekukan' : account.is_active ? 'Aktif' : 'Nonaktif'}
              </TableCell>
              <TableCell className="text-right">
                {can('account.freeze') && (
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
                )}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <div className="border-t border-border px-3 py-2">
        <Pagination
          page={paginasi.page}
          totalPages={paginasi.totalPages}
          shown={paginasi.baris.length}
          total={paginasi.total}
          unit="akun"
          pageSize={paginasi.pageSize}
          onPageChange={paginasi.setPage}
          onPageSizeChange={paginasi.setPageSize}
        />
      </div>
    </Card>
  )
}

/** PositionsSection dimuat terpisah: membukanya tercatat sebagai baca sensitif. */
function PositionsSection({ userID }: { userID: string }) {
  const { can } = usePermissions()
  const [open, setOpen] = useState(false)
  const positions = useQuery({
    queryKey: ['admin', 'user', userID, 'positions'],
    queryFn: () => adminUserApi.positions(userID),
    enabled: open && can('trade.view'),
  })

  const rows: AdminPositionItem[] = positions.data ?? []
  // Riwayat posisi bisa panjang; dipecah per halaman agar tabel tetap ringkas.
  const paginasi = usePaginasiLokal(rows)

  if (!can('trade.view')) {
    return (
      <Section title="Riwayat posisi">
        <p className="text-[12px] text-muted-foreground">
          Role Anda tidak berwenang melihat riwayat posisi pelanggan.
        </p>
      </Section>
    )
  }

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
              {paginasi.baris.map((row) => (
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
          <div className="border-t border-border px-3 py-2">
            <Pagination
              page={paginasi.page}
              totalPages={paginasi.totalPages}
              shown={paginasi.baris.length}
              total={paginasi.total}
              unit="posisi"
              pageSize={paginasi.pageSize}
              onPageChange={paginasi.setPage}
              onPageSizeChange={paginasi.setPageSize}
            />
          </div>
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
