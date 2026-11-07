import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { useDebouncedValue } from '@/features/admin/hooks/use-debounce'
import { formatInvoice } from '@/features/admin/lib/format'
import { adminUserApi } from '@/lib/admin-api'
import {
  Badge,
  Button,
  Card,
  EmptyState,
  Input,
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
import type { AdminUserListItem } from '@/types/admin'

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
  const [page, setPage] = useState(1)

  const debounced = useDebouncedValue(search, 300)

  const query = useMemo(
    () => ({ page, per_page: 25, q: debounced, status, tier, accounts }),
    [page, debounced, status, tier, accounts],
  )

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['admin', 'users', query],
    queryFn: () => adminUserApi.list(query),
  })

  const rows: AdminUserListItem[] = data?.data ?? []
  const meta = data?.meta

  return (
    <>
      <PageHeader
        title="Pengguna"
        description="Daftar pelanggan beserta tier dan status. Membuka detail tercatat di jejak audit."
        actions={
          <Button variant="outline" size="sm" onClick={() => window.open('/api/v1/admin/users-export')}>
            Ekspor CSV
          </Button>
        }
      />

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
    </>
  )
}
