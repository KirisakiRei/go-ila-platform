import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Copy, Check, Link2, MailCheck, Inbox } from 'lucide-react'
import {
  inviteApi,
  inviteRequestApi,
  type ApprovedInviteRequestResult,
  type CreatedInvite,
  type InviteRequestItem,
} from '@/lib/invite-api'
import { Can } from '@/features/admin/components/Can'
import { formatDate, formatDateTime } from '@/lib/format'
import {
  Alert,
  Badge,
  Button,
  Card,
  EmptyState,
  Input,
  Label,
  Modal,
  PageHeader,
  Pagination,
  Select,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  usePaginasiLokal,
} from '@/components/ui'
import { cn } from '@/lib/utils'

const STATUS_VARIANT: Record<string, 'muted' | 'outline' | 'solid'> = {
  aktif: 'solid',
  habis: 'outline',
  kedaluwarsa: 'muted',
  dicabut: 'muted',
}

const STATUS_TEXT: Record<string, string> = {
  aktif: 'Aktif',
  habis: 'Habis',
  kedaluwarsa: 'Kedaluwarsa',
  dicabut: 'Dicabut',
}

const REQ_STATUS_VARIANT: Record<string, 'muted' | 'outline' | 'solid'> = {
  pending: 'solid',
  approved: 'outline',
  rejected: 'muted',
}

const REQ_STATUS_TEXT: Record<string, string> = {
  pending: 'Menunggu',
  approved: 'Disetujui',
  rejected: 'Ditolak',
}

type CreatedModalState = {
  invite: CreatedInvite
  recipientName?: string
  emailQueued?: boolean
}

/**
 * AdminInvitesPage mengelola permohonan akses masuk dan tautan undangan komunitas.
 *
 * Kode mentah hanya tampil SEKALI saat dibuat: backend hanya menyimpan hash-nya,
 * sehingga daftar tidak bisa menampilkan ulang tautan yang sudah ada. Bila
 * tautan hilang, admin membuat yang baru.
 */
export default function AdminInvitesPage() {
  const queryClient = useQueryClient()
  const [tab, setTab] = useState<'requests' | 'invites'>('requests')
  const [status, setStatus] = useState('')
  const [reqStatus, setReqStatus] = useState<'pending' | 'approved' | 'rejected' | ''>('pending')
  const [creating, setCreating] = useState(false)
  const [approving, setApproving] = useState<InviteRequestItem | null>(null)
  const [created, setCreated] = useState<CreatedModalState | null>(null)
  const [galat, setGalat] = useState<string | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'invites', status],
    queryFn: () => inviteApi.list(status || undefined),
  })

  const requestsQuery = useQuery({
    queryKey: ['admin', 'invite-requests', reqStatus],
    queryFn: () => inviteRequestApi.list({ status: reqStatus || undefined, per_page: 50 }),
  })

  const rows = data ?? []
  const paginasi = usePaginasiLokal(rows)
  const requestRows = requestsQuery.data?.items ?? []
  const pendingCount = requestsQuery.data?.pending_count ?? 0

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['admin', 'invites'] })
    void queryClient.invalidateQueries({ queryKey: ['admin', 'invite-requests'] })
  }

  const revoke = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) => inviteApi.revoke(id, reason),
    onSuccess: invalidate,
    onError: () => setGalat('Gagal mencabut undangan.'),
  })

  const rejectRequest = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      inviteRequestApi.reject(id, reason),
    onSuccess: invalidate,
    onError: (err) => setGalat(err instanceof Error ? err.message : 'Gagal menolak permohonan.'),
  })

  const fullInviteUrl = created ? `${window.location.origin}${created.invite.url}` : ''
  const chatTemplate = created
    ? `Halo${created.recipientName ? ` ${created.recipientName}` : ''},\n\nPermohonan akses Anda ke Imaginary Liquidity telah disetujui. Silakan daftar memakai tautan undangan pribadi berikut:\n${fullInviteUrl}\n\nTautan berlaku 1x pakai sampai ${formatDate(created.invite.expires_at)}.`
    : ''

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title="Undangan"
        description="Tinjau permohonan akses masuk dan kelola tautan undangan selama platform invite only."
        actions={
          <Can permission="user.edit">
            <Button size="sm" onClick={() => setCreating(true)} className="gap-1.5">
              <Link2 className="h-4 w-4" />
              Buat Undangan
            </Button>
          </Can>
        }
      />
      {galat && <Alert>{galat}</Alert>}

      {/* Tab pemilih Permintaan Masuk vs Daftar Tautan */}
      <div className="inline-flex w-fit items-center gap-1 rounded-lg border border-border bg-secondary/40 p-1">
        <button
          type="button"
          onClick={() => setTab('requests')}
          className={cn(
            'inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors',
            tab === 'requests'
              ? 'bg-background text-foreground shadow-2xs'
              : 'text-muted-foreground hover:text-foreground',
          )}
        >
          <Inbox className="h-3.5 w-3.5" />
          Permintaan Masuk
          {pendingCount > 0 && (
            <Badge variant="solid" className="ml-0.5 px-1.5 py-0 text-[10px] tabular-nums">
              {pendingCount}
            </Badge>
          )}
        </button>
        <button
          type="button"
          onClick={() => setTab('invites')}
          className={cn(
            'inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-colors',
            tab === 'invites'
              ? 'bg-background text-foreground shadow-2xs'
              : 'text-muted-foreground hover:text-foreground',
          )}
        >
          <Link2 className="h-3.5 w-3.5" />
          Tautan Undangan
        </button>
      </div>

      {tab === 'requests' ? (
        <Card className="p-4">
          <div className="mb-3 flex flex-wrap items-end gap-3">
            <div className="flex flex-col gap-1">
              <Label htmlFor="status-permohonan">Status permohonan</Label>
              <Select
                id="status-permohonan"
                value={reqStatus}
                onChange={(e) =>
                  setReqStatus(e.target.value as 'pending' | 'approved' | 'rejected' | '')
                }
                className="w-44"
              >
                <option value="pending">Menunggu</option>
                <option value="approved">Disetujui</option>
                <option value="rejected">Ditolak</option>
                <option value="">Semua</option>
              </Select>
            </div>
          </div>

          {requestsQuery.isLoading ? (
            <Skeleton className="h-40 w-full rounded-xl" />
          ) : requestRows.length === 0 ? (
            <EmptyState
              title="Belum ada permohonan"
              description="Permohonan akses yang dikirim dari beranda publik akan masuk ke sini."
            />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Pemohon</TableHead>
                  <TableHead>Kontak</TableHead>
                  <TableHead>Catatan</TableHead>
                  <TableHead>Masuk</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Aksi</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {requestRows.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell>
                      <p className="text-xs font-medium text-foreground">{row.name}</p>
                      <p className="font-mono text-[11px] text-muted-foreground">{row.email}</p>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {row.contact || '—'}
                    </TableCell>
                    <TableCell className="max-w-xs text-xs text-muted-foreground">
                      <span className="line-clamp-2">{row.note || '—'}</span>
                    </TableCell>
                    <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                      {formatDateTime(row.created_at)}
                    </TableCell>
                    <TableCell>
                      <Badge variant={REQ_STATUS_VARIANT[row.status] ?? 'muted'}>
                        {REQ_STATUS_TEXT[row.status] ?? row.status}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {row.status === 'pending' ? (
                        <Can permission="user.edit">
                          <div className="flex flex-wrap items-center gap-1.5">
                            <Button
                              size="sm"
                              onClick={() => {
                                setGalat(null)
                                setApproving(row)
                              }}
                            >
                              Setujui
                            </Button>
                            <Button
                              size="sm"
                              variant="outline"
                              disabled={rejectRequest.isPending}
                              onClick={() => {
                                const reason = window.prompt(
                                  'Alasan penolakan (opsional, tidak dikirim ke pemohon):',
                                  '',
                                )
                                if (reason === null) return
                                setGalat(null)
                                rejectRequest.mutate({ id: row.id, reason })
                              }}
                            >
                              Tolak
                            </Button>
                          </div>
                        </Can>
                      ) : (
                        <span className="text-[11px] text-muted-foreground">
                          {row.review_note || 'Selesai'}
                        </span>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </Card>
      ) : (
        <Card className="p-4">
          <div className="mb-3 flex flex-wrap items-end gap-3">
            <div className="flex flex-col gap-1">
              <Label htmlFor="status-undangan">Status</Label>
              <Select
                id="status-undangan"
                value={status}
                onChange={(e) => setStatus(e.target.value)}
                className="w-44"
              >
                <option value="">Semua</option>
                <option value="aktif">Aktif</option>
                <option value="habis">Habis</option>
                <option value="kedaluwarsa">Kedaluwarsa</option>
                <option value="dicabut">Dicabut</option>
              </Select>
            </div>
          </div>

          {isLoading ? (
            <Skeleton className="h-40 w-full rounded-xl" />
          ) : rows.length === 0 ? (
            <EmptyState
              title="Belum ada undangan"
              description="Buat tautan pertama untuk mengundang anggota komunitas."
            />
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Kode</TableHead>
                    <TableHead>Tier</TableHead>
                    <TableHead>Kuota</TableHead>
                    <TableHead>Berlaku s.d.</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Aksi</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {paginasi.baris.map((row) => (
                    <TableRow key={row.id}>
                      <TableCell className="font-mono text-xs">
                        {row.prefix}…
                        {row.email && (
                          <span className="ml-1 text-[11px] text-muted-foreground">
                            ({row.email})
                          </span>
                        )}
                      </TableCell>
                      <TableCell className="text-xs uppercase">{row.tier}</TableCell>
                      <TableCell className="text-xs">
                        {row.used_count}/{row.max_uses}
                      </TableCell>
                      <TableCell className="text-xs">{formatDate(row.expires_at)}</TableCell>
                      <TableCell>
                        <Badge variant={STATUS_VARIANT[row.status] ?? 'muted'}>
                          {STATUS_TEXT[row.status] ?? row.status}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {row.status !== 'dicabut' && (
                          <Can permission="user.edit">
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() => {
                                const reason = window.prompt('Alasan mencabut undangan ini:')
                                if (!reason || reason.trim().length < 3) return
                                revoke.mutate({ id: row.id, reason })
                              }}
                            >
                              Cabut
                            </Button>
                          </Can>
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
                  unit="undangan"
                  pageSize={paginasi.pageSize}
                  onPageChange={paginasi.setPage}
                  onPageSizeChange={paginasi.setPageSize}
                />
              </div>
            </>
          )}
        </Card>
      )}

      <BuatUndanganModal
        open={creating}
        onClose={() => setCreating(false)}
        onCreated={(result) => {
          setCreating(false)
          setCreated({ invite: result })
          invalidate()
        }}
        onError={setGalat}
      />

      <SetujuiPermohonanModal
        item={approving}
        onClose={() => setApproving(null)}
        onApproved={(result) => {
          setApproving(null)
          setCreated({
            invite: result.invite,
            recipientName: result.request.name,
            emailQueued: result.email_queued,
          })
          invalidate()
        }}
        onError={setGalat}
      />

      {/* Kode hanya bisa dilihat sekarang. Peringatan ini bagian dari desain. */}
      <Modal
        open={Boolean(created)}
        onClose={() => setCreated(null)}
        title="Salin tautan sekarang"
      >
        {created && (
          <div className="flex flex-col gap-3">
            {created.emailQueued === true && (
              <Alert>
                <span className="inline-flex items-center gap-1.5 font-medium">
                  <MailCheck className="h-4 w-4 shrink-0" />
                  Email undangan otomatis masuk antrean SMTP ke {created.invite.email}.
                </span>
              </Alert>
            )}
            {created.emailQueued === false && (
              <Alert variant="warning">
                SMTP belum aktif di server — kirim tautan atau pesan di bawah secara manual ke
                pemohon ({created.invite.email}).
              </Alert>
            )}
            <Alert>
              Tautan ini <strong>tidak bisa dilihat lagi</strong>. Bila hilang, buat undangan
              baru — kode lama tidak bisa ditampilkan ulang.
            </Alert>
            <div className="flex flex-col gap-1">
              <Label htmlFor="tautan-undangan">Tautan undangan</Label>
              <div className="flex gap-2">
                <Input
                  id="tautan-undangan"
                  readOnly
                  value={fullInviteUrl}
                  onFocus={(e) => e.currentTarget.select()}
                />
                <TombolSalin teks={fullInviteUrl} label="Salin" />
              </div>
            </div>
            <div className="flex flex-col gap-1">
              <Label htmlFor="template-pesan">Template pesan (Telegram / Discord / Email)</Label>
              <textarea
                id="template-pesan"
                readOnly
                rows={4}
                value={chatTemplate}
                className="w-full rounded-md border border-input bg-secondary/30 px-3 py-2 text-xs text-foreground"
              />
              <div className="flex justify-end">
                <TombolSalin teks={chatTemplate} label="Salin Pesan" />
              </div>
            </div>
            <p className="text-[12px] text-muted-foreground">
              Kuota {created.invite.max_uses} akun · tier {created.invite.tier} · berlaku sampai{' '}
              {formatDate(created.invite.expires_at)}
            </p>
          </div>
        )}
      </Modal>
    </div>
  )
}

/** TombolSalin menyalin teks dan memberi tanda sesaat bahwa itu berhasil. */
function TombolSalin({ teks, label = 'Salin' }: { teks: string; label?: string }) {
  const [tersalin, setTersalin] = useState(false)
  async function salin() {
    try {
      await navigator.clipboard.writeText(teks)
      setTersalin(true)
      window.setTimeout(() => setTersalin(false), 2000)
    } catch {
      // Klipboard bisa ditolak browser; isian tetap bisa dipilih manual.
    }
  }
  return (
    <Button type="button" size="sm" variant="outline" onClick={() => void salin()} className="gap-1.5">
      {tersalin ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
      {tersalin ? 'Tersalin' : label}
    </Button>
  )
}

/** SetujuiPermohonanModal menentukan tier & masa berlaku sebelum membuat undangan terkunci email. */
function SetujuiPermohonanModal({
  item,
  onClose,
  onApproved,
  onError,
}: {
  item: InviteRequestItem | null
  onClose: () => void
  onApproved: (result: ApprovedInviteRequestResult) => void
  onError: (msg: string) => void
}) {
  const [tier, setTier] = useState('plus')
  const [expiresDays, setExpiresDays] = useState('7')
  const [note, setNote] = useState('')

  const approve = useMutation({
    mutationFn: () => {
      if (!item) return Promise.reject(new Error('Permohonan tidak ditemukan'))
      return inviteRequestApi.approve(item.id, {
        tier,
        expires_days: Math.max(1, Number(expiresDays) || 7),
        note: note.trim() || undefined,
      })
    },
    onSuccess: (res) => {
      setNote('')
      onApproved(res)
    },
    onError: (err) => onError(err instanceof Error ? err.message : 'Gagal menyetujui permohonan.'),
  })

  return (
    <Modal open={Boolean(item)} onClose={onClose} title="Setujui & buat tautan undangan">
      {item && (
        <div className="flex flex-col gap-3">
          <div className="rounded-lg border border-border bg-secondary/30 p-3 text-xs">
            <p className="font-semibold text-foreground">{item.name}</p>
            <p className="font-mono text-muted-foreground">{item.email}</p>
            {item.contact && <p className="mt-1 text-muted-foreground">Kontak: {item.contact}</p>}
            {item.note && <p className="mt-1 text-muted-foreground">Catatan: {item.note}</p>}
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="flex flex-col gap-1">
              <Label htmlFor="approve-tier">Tier akun</Label>
              <Select
                id="approve-tier"
                value={tier}
                onChange={(e) => setTier(e.target.value)}
              >
                <option value="plus">Plus</option>
                <option value="basic">Basic</option>
                <option value="free">Free</option>
              </Select>
            </div>
            <div className="flex flex-col gap-1">
              <Label htmlFor="approve-days">Berlaku (hari)</Label>
              <Input
                id="approve-days"
                type="number"
                min={1}
                max={365}
                value={expiresDays}
                onChange={(e) => setExpiresDays(e.target.value)}
              />
            </div>
          </div>
          <div className="flex flex-col gap-1">
            <Label htmlFor="approve-note">Catatan internal (opsional)</Label>
            <Input
              id="approve-note"
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="Kosong = otomatis pakai nama pemohon"
            />
          </div>
          <p className="text-[11px] text-muted-foreground">
            Tautan akan dikunci 1× pakai khusus untuk <span className="font-mono">{item.email}</span>{' '}
            dan otomatis masuk antrean email bila SMTP server aktif.
          </p>
          <div className="flex justify-end gap-2">
            <Button variant="outline" type="button" onClick={onClose}>
              Batal
            </Button>
            <Button type="button" isLoading={approve.isPending} onClick={() => approve.mutate()}>
              Setujui &amp; Buat Tautan
            </Button>
          </div>
        </div>
      )}
    </Modal>
  )
}

/** BuatUndanganModal mengumpulkan kuota, masa berlaku, tier, dan alamat tujuan. */
function BuatUndanganModal({
  open,
  onClose,
  onCreated,
  onError,
}: {
  open: boolean
  onClose: () => void
  onCreated: (result: CreatedInvite) => void
  onError: (pesan: string) => void
}) {
  const [tier, setTier] = useState('plus')
  const [maxUses, setMaxUses] = useState('1')
  const [hari, setHari] = useState('7')
  const [email, setEmail] = useState('')
  const [note, setNote] = useState('')

  const buat = useMutation({
    mutationFn: () => {
      const kuota = Number.parseInt(maxUses, 10)
      const masaHari = Number.parseInt(hari, 10)
      const expires = new Date(Date.now() + masaHari * 24 * 60 * 60 * 1000).toISOString()
      return inviteApi.create({
        tier,
        max_uses: Number.isFinite(kuota) ? kuota : 1,
        expires_at: expires,
        email: email.trim() || undefined,
        note: note.trim() || undefined,
      })
    },
    onSuccess: onCreated,
    onError: (err) => {
      const pesan = err instanceof Error ? err.message : 'Gagal membuat undangan.'
      onError(pesan)
    },
  })

  return (
    <Modal open={open} onClose={onClose} title="Buat undangan">
      <div className="flex flex-col gap-3">
        <div className="grid grid-cols-2 gap-3">
          <div className="flex flex-col gap-1">
            <Label htmlFor="tier-undangan">Tier akun</Label>
            <Select id="tier-undangan" value={tier} onChange={(e) => setTier(e.target.value)}>
              <option value="plus">Plus</option>
              <option value="basic">Basic</option>
              <option value="free">Free</option>
            </Select>
          </div>
          <div className="flex flex-col gap-1">
            <Label htmlFor="kuota-undangan">Kuota akun</Label>
            <Input
              id="kuota-undangan"
              type="number"
              min={1}
              value={maxUses}
              onChange={(e) => setMaxUses(e.target.value)}
            />
          </div>
          <div className="flex flex-col gap-1">
            <Label htmlFor="hari-undangan">Berlaku (hari)</Label>
            <Input
              id="hari-undangan"
              type="number"
              min={1}
              max={365}
              value={hari}
              onChange={(e) => setHari(e.target.value)}
            />
          </div>
          <div className="flex flex-col gap-1">
            <Label htmlFor="email-undangan">Khusus surel (opsional)</Label>
            <Input
              id="email-undangan"
              type="email"
              placeholder="kosong = siapa pun"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>
        </div>
        <div className="flex flex-col gap-1">
          <Label htmlFor="catatan-undangan">Catatan (opsional)</Label>
          <Input
            id="catatan-undangan"
            placeholder="misal: kelas batch Januari"
            value={note}
            onChange={(e) => setNote(e.target.value)}
          />
        </div>
        <div className="flex justify-end gap-2">
          <Button variant="outline" onClick={onClose} type="button">
            Batal
          </Button>
          <Button onClick={() => buat.mutate()} isLoading={buat.isPending} type="button">
            Buat tautan
          </Button>
        </div>
      </div>
    </Modal>
  )
}
