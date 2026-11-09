import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useDebouncedValue } from '@/features/admin/hooks/use-debounce'
import { formatInvoice } from '@/features/admin/lib/format'
import { adminUserApi } from '@/lib/admin-api'
import { Can } from '@/features/admin/components/Can'
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
  Select,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui'
import type { AdminCreateUserPayload, AdminUserListItem } from '@/types/admin'
import type { Tier } from '@/types/api'

const STATUS_LABEL: Record<string, string> = {
  active: 'Aktif',
  suspended: 'Ditangguhkan',
  banned: 'Diblokir',
  deleted: 'Dihapus',
}

const TIER_LABEL: Record<string, string> = { free: 'Free', basic: 'Basic', plus: 'Plus' }

/**
 * AdminUsersPage menampilkan pelanggan dengan pencarian dan filter.
 * Saldo dan kredensial tidak pernah muncul di daftar (plans/07 §2).
 */
export default function AdminUsersPage() {
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState('')
  const [tier, setTier] = useState('')
  const [accounts, setAccounts] = useState('')
  const [trial, setTrial] = useState('')
  const [role, setRole] = useState('')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [page, setPage] = useState(1)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)

  const queryClient = useQueryClient()
  const debounced = useDebouncedValue(search, 300)

  const query = useMemo(
    () => ({
      page,
      per_page: 25,
      q: debounced,
      status,
      tier,
      accounts,
      trial,
      role,
      from,
      to,
    }),
    [page, debounced, status, tier, accounts, trial, role, from, to],
  )

  const exportUrl = useMemo(() => {
    // Ekspor mengikuti filter yang sedang aktif, bukan seluruh basis pengguna.
    const params = new URLSearchParams({ q: debounced, status, tier, accounts, trial, role, from, to })
    for (const [key, value] of [...params]) {
      if (!value) params.delete(key)
    }
    const qs = params.toString()
    return `/api/v1/admin/users-export${qs ? `?${qs}` : ''}`
  }, [debounced, status, tier, accounts, trial, role, from, to])

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['admin', 'users', query],
    queryFn: () => adminUserApi.list(query),
  })

  const createUser = useMutation({
    mutationFn: (payload: AdminCreateUserPayload) => adminUserApi.create(payload),
    onSuccess: () => {
      setNotice('Pengguna pelanggan berhasil dibuat.')
      setShowCreateModal(false)
      void queryClient.invalidateQueries({ queryKey: ['admin', 'users'] })
    },
    onError: (err) => {
      setNotice(err instanceof Error ? err.message : 'Gagal membuat pengguna.')
    },
  })

  const rows: AdminUserListItem[] = data?.data ?? []
  const meta = data?.meta

  return (
    <>
      <PageHeader
        title="Pengguna"
        description="Daftar pelanggan beserta tier dan status. Membuka detail tercatat di jejak audit."
        actions={
          <div className="flex flex-wrap gap-2">
            <Can permission="user.edit">
              <Button size="sm" onClick={() => setShowCreateModal(true)}>
                + Tambah Pengguna
              </Button>
            </Can>
            <Can permission="user.export">
              <Button variant="outline" size="sm" onClick={() => window.open(exportUrl)}>
                Ekspor CSV
              </Button>
            </Can>
          </div>
        }
      />

      {notice && (
        <Alert variant={notice.includes('Gagal') ? 'destructive' : 'default'} className="mb-4">
          <div className="flex items-center justify-between">
            <span>{notice}</span>
            <Button variant="ghost" size="sm" onClick={() => setNotice(null)}>
              Tutup
            </Button>
          </div>
        </Alert>
      )}

      <Card className="p-3">
        <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
          <Input
            placeholder="Cari nama, email, atau ID"
            value={search}
            aria-label="Cari pengguna"
            onChange={(event) => {
              setSearch(event.target.value)
              setPage(1)
            }}
          />
          <Select
            aria-label="Filter status"
            value={status}
            onChange={(event) => {
              setStatus(event.target.value)
              setPage(1)
            }}
          >
            <option value="">Semua status</option>
            <option value="active">Aktif</option>
            <option value="suspended">Ditangguhkan</option>
            <option value="banned">Diblokir</option>
            <option value="deleted">Dihapus</option>
          </Select>
          <Select
            aria-label="Filter tier"
            value={tier}
            onChange={(event) => {
              setTier(event.target.value)
              setPage(1)
            }}
          >
            <option value="">Semua tier</option>
            <option value="free">Free</option>
            <option value="basic">Basic</option>
            <option value="plus">Plus</option>
          </Select>
          <Select
            aria-label="Filter akun broker"
            value={accounts}
            onChange={(event) => {
              setAccounts(event.target.value)
              setPage(1)
            }}
          >
            <option value="">Semua akun broker</option>
            <option value="yes">Punya akun broker</option>
            <option value="no">Belum punya akun</option>
          </Select>
          <Select
            aria-label="Filter trial"
            value={trial}
            onChange={(event) => {
              setTrial(event.target.value)
              setPage(1)
            }}
          >
            <option value="">Semua pengguna</option>
            <option value="yes">Pernah memakai trial</option>
          </Select>
          <Select
            aria-label="Filter peran"
            value={role}
            onChange={(event) => {
              setRole(event.target.value)
              setPage(1)
            }}
          >
            <option value="">Semua peran</option>
            <option value="admin">Admin</option>
          </Select>
          <Input
            type="date"
            aria-label="Terdaftar mulai"
            value={from}
            onChange={(event) => {
              setFrom(event.target.value)
              setPage(1)
            }}
          />
          <Input
            type="date"
            aria-label="Terdaftar sampai"
            value={to}
            onChange={(event) => {
              setTo(event.target.value)
              setPage(1)
            }}
          />
        </div>
      </Card>

      {isLoading && (
        <div className="flex flex-col gap-2" aria-busy="true">
          {Array.from({ length: 5 }).map((_, index) => (
            <Skeleton key={index} className="h-12 w-full" />
          ))}
        </div>
      )}

      {isError && (
        <EmptyState
          title="Daftar pengguna gagal dimuat"
          description={error instanceof Error ? error.message : 'Silakan coba lagi.'}
        />
      )}

      {!isLoading && !isError && rows.length === 0 && (
        <EmptyState
          title="Tidak ada pengguna cocok"
          description="Ubah kata kunci atau filter untuk melihat hasil lain."
        />
      )}

      {rows.length > 0 && (
        <Card className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Pengguna</TableHead>
                <TableHead>Tier</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Akun</TableHead>
                <TableHead>Terdaftar (WIB)</TableHead>
                <TableHead>Login terakhir</TableHead>
                <TableHead className="text-right">Aksi</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => (
                <TableRow key={row.id}>
                  <TableCell>
                    <Link
                      to={`/admin/pengguna/${row.id}`}
                      className="flex flex-col gap-0.5 hover:underline"
                    >
                      <span className="text-[13px] font-medium">{row.name}</span>
                      <span className="text-[11px] text-muted-foreground">{row.email}</span>
                    </Link>
                  </TableCell>
                  <TableCell>
                    <Badge variant={row.tier === 'free' ? 'outline' : 'default'}>
                      {TIER_LABEL[row.tier] ?? row.tier}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-[12px]">
                    {STATUS_LABEL[row.status] ?? row.status}
                  </TableCell>
                  <TableCell className="text-right text-[12px] tabular-nums">
                    {row.account_count}
                  </TableCell>
                  <TableCell className="text-[12px] text-muted-foreground">
                    {formatInvoice(row.created_at.wib)}
                  </TableCell>
                  <TableCell className="text-[12px] text-muted-foreground">
                    {row.last_login_at ? formatInvoice(row.last_login_at.wib) : '—'}
                  </TableCell>
                  <TableCell className="text-right">
                    <Link to={`/admin/pengguna/${row.id}`}>
                      <Button variant="outline" size="sm" className="h-7 px-2 text-[12px]">
                        Kelola
                      </Button>
                    </Link>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>

          <div className="flex flex-wrap items-center justify-between gap-2 border-t border-border px-3 py-2">
            <p className="text-[11px] text-muted-foreground">
              {meta ? `${meta.total} pengguna · halaman ${meta.page}/${meta.total_pages || 1}` : ''}
            </p>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage((value) => Math.max(1, value - 1))}
              >
                Sebelumnya
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={!meta || page >= (meta.total_pages || 1)}
                onClick={() => setPage((value) => value + 1)}
              >
                Berikutnya
              </Button>
            </div>
          </div>
        </Card>
      )}

      {showCreateModal && (
        <CreateCustomerModal
          pending={createUser.isPending}
          onClose={() => setShowCreateModal(false)}
          onSubmit={(payload) => createUser.mutate(payload)}
        />
      )}
    </>
  )
}

function CreateCustomerModal({
  pending,
  onClose,
  onSubmit,
}: {
  pending: boolean
  onClose: () => void
  onSubmit: (payload: AdminCreateUserPayload) => void
}) {
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [tier, setTier] = useState<Tier>('free')

  const emailSah = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())
  const nameSah = name.trim().length >= 2
  const usernameSah = username.trim() === '' || /^[a-z0-9._-]{3,32}$/.test(username.trim().toLowerCase())
  const passwordSah = password.length >= 8 && password.length <= 72
  const cocok = password === confirmPassword && password.length > 0
  const siap = nameSah && emailSah && usernameSah && passwordSah && cocok

  return (
    <Modal
      open
      onClose={onClose}
      size="lg"
      title="Tambah pengguna pelanggan"
      description="Akun ini adalah pelanggan (bukan staf/admin). Tidak memerlukan verifikasi email dan langsung aktif."
      footer={
        <>
          <Button variant="outline" size="sm" onClick={onClose}>
            Batal
          </Button>
          <Button
            size="sm"
            isLoading={pending}
            disabled={!siap}
            onClick={() =>
              onSubmit({
                name: name.trim(),
                email: email.trim().toLowerCase(),
                username: username.trim().toLowerCase() || undefined,
                password,
                tier,
              })
            }
          >
            Buat pelanggan
          </Button>
        </>
      }
    >
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="sm:col-span-2">
          <Label htmlFor="customer-name">Nama lengkap</Label>
          <Input
            id="customer-name"
            className="mt-1"
            value={name}
            placeholder="mis. Budi Santoso"
            onChange={(event) => setName(event.target.value)}
          />
        </div>

        <div>
          <Label htmlFor="customer-email">Alamat email</Label>
          <Input
            id="customer-email"
            type="email"
            className="mt-1"
            value={email}
            placeholder="mis. budi@gmail.com"
            onChange={(event) => setEmail(event.target.value)}
          />
          {email.trim() !== '' && !emailSah && (
            <p className="mt-1 text-[11px] text-destructive">Format email belum valid.</p>
          )}
        </div>

        <div>
          <Label htmlFor="customer-username">Username (opsional)</Label>
          <Input
            id="customer-username"
            className="mt-1"
            value={username}
            placeholder="Kosongkan untuk otomatis dari email"
            onChange={(event) => setUsername(event.target.value)}
          />
          {username.trim() !== '' && !usernameSah && (
            <p className="mt-1 text-[11px] text-destructive">3-32 huruf kecil, angka, titik, atau strip.</p>
          )}
        </div>

        <div>
          <Label htmlFor="customer-tier">Tier layanan awal</Label>
          <Select
            id="customer-tier"
            className="mt-1"
            value={tier}
            onChange={(event) => setTier(event.target.value as Tier)}
          >
            <option value="free">Free</option>
            <option value="basic">Basic</option>
            <option value="plus">Plus</option>
          </Select>
        </div>

        <div>
          <Label htmlFor="customer-password">Kata sandi</Label>
          <Input
            id="customer-password"
            type="password"
            className="mt-1"
            value={password}
            placeholder="Minimal 8 karakter"
            onChange={(event) => setPassword(event.target.value)}
          />
          {password !== '' && !passwordSah && (
            <p className="mt-1 text-[11px] text-destructive">Minimal 8 karakter.</p>
          )}
        </div>

        <div className="sm:col-span-2">
          <Label htmlFor="customer-confirm-password">Konfirmasi kata sandi</Label>
          <Input
            id="customer-confirm-password"
            type="password"
            className="mt-1"
            value={confirmPassword}
            placeholder="Ketik ulang kata sandi di atas"
            onChange={(event) => setConfirmPassword(event.target.value)}
          />
          {confirmPassword !== '' && !cocok && (
            <p className="mt-1 text-[11px] text-destructive">Kata sandi tidak sama.</p>
          )}
        </div>
      </div>
    </Modal>
  )
}
