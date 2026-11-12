import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Copy, Check, Link2 } from 'lucide-react'
import { inviteApi, type CreatedInvite } from '@/lib/invite-api'
import { Can } from '@/features/admin/components/Can'
import { formatDate } from '@/lib/format'
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

/**
 * AdminInvitesPage membuat dan mengelola tautan undangan komunitas.
 *
 * Kode mentah hanya tampil SEKALI saat dibuat: backend hanya menyimpan hash-nya,
 * sehingga daftar tidak bisa menampilkan ulang tautan yang sudah ada. Bila
 * tautan hilang, admin membuat yang baru.
 */
export default function AdminInvitesPage() {
  const queryClient = useQueryClient()
  const [status, setStatus] = useState('')
  const [creating, setCreating] = useState(false)
  const [created, setCreated] = useState<CreatedInvite | null>(null)
  const [galat, setGalat] = useState<string | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'invites', status],
    queryFn: () => inviteApi.list(status || undefined),
  })

  const rows = data ?? []
  // Daftar dipecah per halaman agar tabel panjang tidak memaksa menggulir jauh.
  const paginasi = usePaginasiLokal(rows)
  const invalidate = () => void queryClient.invalidateQueries({ queryKey: ['admin', 'invites'] })

  const revoke = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) => inviteApi.revoke(id, reason),
    onSuccess: invalidate,
    onError: () => setGalat('Gagal mencabut undangan.'),
  })

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title="Undangan"
        description="Tautan untuk membuat akun selama platform masih invite only."
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
                      <span className="ml-1 text-[11px] text-muted-foreground">({row.email})</span>
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

      <BuatUndanganModal
        open={creating}
        onClose={() => setCreating(false)}
        onCreated={(result) => {
          setCreating(false)
          setCreated(result)
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
                  value={`${window.location.origin}${created.url}`}
                  onFocus={(e) => e.currentTarget.select()}
                />
                <TombolSalin teks={`${window.location.origin}${created.url}`} />
              </div>
            </div>
            <p className="text-[12px] text-muted-foreground">
              Kuota {created.max_uses} akun · tier {created.tier} · berlaku sampai{' '}
              {formatDate(created.expires_at)}
            </p>
          </div>
        )}
      </Modal>
    </div>
  )
}

/** TombolSalin menyalin teks dan memberi tanda sesaat bahwa itu berhasil. */
function TombolSalin({ teks }: { teks: string }) {
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
      {tersalin ? 'Tersalin' : 'Salin'}
    </Button>
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
