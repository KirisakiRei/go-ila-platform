import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { adminAuditApi } from '@/lib/admin-api'
import { formatInvoice } from '@/features/admin/lib/format'
import {
  Card,
  EmptyState,
  Input,
  Label,
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
} from '@/components/ui'
import type { AuditEntry } from '@/types/admin'

/**
 * AdminAuditPage menampilkan jejak audit.
 *
 * Rentang tanggal WAJIB: tanpa batas, halaman ini akan memindai seluruh tabel
 * yang tumbuh paling cepat (plans/24 §4.3). Bawaannya 7 hari terakhir.
 */
export default function AdminAuditPage() {
  const today = new Date()
  const weekAgo = new Date(today.getTime() - 6 * 24 * 60 * 60 * 1000)

  const [from, setFrom] = useState(isoDate(weekAgo))
  const [to, setTo] = useState(isoDate(today))
  const [action, setAction] = useState('')
  const [targetType, setTargetType] = useState('')
  const [actorID, setActorID] = useState('')
  const [cursor, setCursor] = useState<string | undefined>(undefined)
  const [trail, setTrail] = useState<string[]>([])
  /** Jumlah baris per halaman; dikirim ke server sebagai per_page. */
  const [ukuranHalaman, setUkuranHalaman] = useState(50)

  const query = useMemo(
    () => ({ from, to, action, target_type: targetType, actor_id: actorID, cursor, per_page: ukuranHalaman }),
    [from, to, action, targetType, actorID, cursor, ukuranHalaman],
  )

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ['admin', 'audit', query],
    queryFn: () => adminAuditApi.list(query),
  })

  const catalog = useQuery({ queryKey: ['admin', 'audit-catalog'], queryFn: adminAuditApi.catalog })

  const rows: AuditEntry[] = data?.data ?? []
  const meta = data?.meta

  const resetCursor = () => {
    setCursor(undefined)
    setTrail([])
  }

  return (
    <>
      <PageHeader
        title="Jejak Audit"
        description="Setiap tindakan admin tercatat. Catatan ini tidak bisa diubah atau dihapus siapa pun."
      />

      <Card className="p-3">
        <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-5">
          <div className="flex flex-col gap-1">
            <Label htmlFor="from">Dari (WIB)</Label>
            <Input
              id="from"
              type="date"
              value={from}
              onChange={(event) => {
                setFrom(event.target.value)
                resetCursor()
              }}
            />
          </div>
          <div className="flex flex-col gap-1">
            <Label htmlFor="to">Sampai (WIB)</Label>
            <Input
              id="to"
              type="date"
              value={to}
              onChange={(event) => {
                setTo(event.target.value)
                resetCursor()
              }}
            />
          </div>
          <div className="flex flex-col gap-1">
            <Label htmlFor="action">Aksi</Label>
            <Select
              id="action"
              value={action}
              onChange={(event) => {
                setAction(event.target.value)
                resetCursor()
              }}
            >
              <option value="">Semua aksi</option>
              {(catalog.data ?? []).map((item) => (
                <option key={item.action} value={item.action}>
                  {item.label}
                </option>
              ))}
            </Select>
          </div>
          <div className="flex flex-col gap-1">
            <Label htmlFor="target">Jenis sasaran</Label>
            <Select
              id="target"
              value={targetType}
              onChange={(event) => {
                setTargetType(event.target.value)
                resetCursor()
              }}
            >
              <option value="">Semua</option>
              <option value="user">Pengguna</option>
              <option value="account">Akun broker</option>
              <option value="invoice">Invoice</option>
              <option value="plan">Paket</option>
              <option value="coupon">Kupon</option>
            </Select>
          </div>
          <div className="flex flex-col gap-1">
            <Label htmlFor="actor">ID pelaku</Label>
            <Input
              id="actor"
              value={actorID}
              placeholder="ID admin"
              onChange={(event) => {
                setActorID(event.target.value)
                resetCursor()
              }}
            />
          </div>
        </div>
      </Card>

      {isLoading && <Skeleton className="h-40 w-full" />}

      {isError && (
        <EmptyState
          title="Jejak audit gagal dimuat"
          description={error instanceof Error ? error.message : 'Persempit rentang tanggal lalu coba lagi.'}
        />
      )}

      {!isLoading && !isError && rows.length === 0 && (
        <EmptyState title="Tidak ada aktivitas" description="Tidak ada tindakan pada rentang ini." />
      )}

      {rows.length > 0 && (
        <Card className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Waktu (WIB)</TableHead>
                <TableHead>Aksi</TableHead>
                <TableHead>Pelaku</TableHead>
                <TableHead>Sasaran</TableHead>
                <TableHead>Alasan</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((row) => (
                <TableRow key={row.id}>
                  <TableCell className="whitespace-nowrap text-[12px] text-muted-foreground">
                    {formatInvoice(row.created_at.wib)}
                  </TableCell>
                  <TableCell className="text-[12px] font-medium">{row.action}</TableCell>
                  <TableCell className="text-[12px]">
                    {row.actor_email || row.actor_id || 'sistem'}
                  </TableCell>
                  <TableCell className="text-[12px] text-muted-foreground">
                    {row.target_type ? `${row.target_type}:${row.target_id.slice(0, 8)}` : '—'}
                  </TableCell>
                  <TableCell className="max-w-[22rem] truncate text-[12px] text-muted-foreground">
                    {row.reason || '—'}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>

          <div className="border-t border-border px-3 py-2">
            {/* Audit memakai kursor, bukan nomor halaman: hanya mundur dan maju
                yang bermakna. Nomor halaman diturunkan dari panjang jejak agar
                pengguna tetap tahu sedang di halaman berapa. */}
            <Pagination
              page={trail.length + 1}
              totalPages={meta?.next_cursor ? trail.length + 2 : trail.length + 1}
              shown={rows.length}
              total={meta?.total ?? rows.length}
              unit={meta?.approx_total ? 'perkiraan baris' : 'baris'}
              pageSize={ukuranHalaman}
              onPageChange={(berikut) => {
                if (berikut > trail.length + 1) {
                  if (!meta?.next_cursor) return
                  setTrail([...trail, meta.next_cursor])
                  setCursor(meta.next_cursor)
                  return
                }
                const next = trail.slice(0, Math.max(0, berikut - 1))
                setTrail(next)
                setCursor(next[next.length - 1])
              }}
              onPageSizeChange={(ukuran) => {
                setUkuranHalaman(ukuran)
                // Ukuran berubah berarti kursor lama tidak lagi sebanding.
                resetCursor()
              }}
            />
          </div>
        </Card>
      )}
    </>
  )
}

/** isoDate mengubah tanggal menjadi YYYY-MM-DD untuk input tanggal. */
function isoDate(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}
