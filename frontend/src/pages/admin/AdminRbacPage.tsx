import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { ShieldCheck, Users } from 'lucide-react'
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
  Modal,
  PageHeader,
  Select,
  Skeleton,
} from '@/components/ui'
import { adminRoleApi, adminUserApi } from '@/lib/admin-api'
import { usePermissions } from '@/features/admin/hooks/use-permissions'
import { cn } from '@/lib/utils'
import type { AdminPermissionGroup, AdminRole, AdminUserListItem } from '@/types/admin'

type Tab = 'role' | 'user'

const TABS: { key: Tab; label: string; icon: typeof ShieldCheck }[] = [
  { key: 'role', label: 'Role', icon: ShieldCheck },
  { key: 'user', label: 'User', icon: Users },
]

const RISK_LABEL: Record<string, string> = {
  rendah: 'Risiko rendah',
  sedang: 'Risiko sedang',
  tinggi: 'Risiko tinggi',
}

const PAGE_SIZE_USERS = 25

/**
 * AdminRbacPage mengelola role, permission, dan penugasan role ke akun admin.
 *
 * Dua tab dipisah karena pertanyaannya berbeda: "role ini boleh apa" dan
 * "akun ini memegang role apa". Tab aktif disimpan di query string agar tautan
 * ke tab tertentu bisa dibagikan dan tombol kembali bekerja.
 *
 * Server tetap penentu akhir. Layar ini hanya menyembunyikan aksi yang pasti
 * ditolak, dan menandai permission yang tidak boleh diberikan pelaku.
 */
export default function AdminRbacPage() {
  const [params, setParams] = useSearchParams()
  const tab: Tab = params.get('tab') === 'user' ? 'user' : 'role'

  const switchTab = (next: Tab) => {
    const updated = new URLSearchParams(params)
    updated.set('tab', next)
    setParams(updated, { replace: true })
  }

  return (
    <>
      <PageHeader
        title="Role & Permission"
        description="Atur kewenangan tiap role dan tugaskan role ke akun pengelola. Perubahan berlaku seketika untuk sesi berikutnya."
      />

      <div role="tablist" aria-label="Kelola RBAC" className="flex gap-1 border-b border-border">
        {TABS.map((item) => {
          const active = tab === item.key
          return (
            <button
              key={item.key}
              type="button"
              role="tab"
              aria-selected={active}
              aria-controls={`rbac-panel-${item.key}`}
              id={`rbac-tab-${item.key}`}
              onClick={() => switchTab(item.key)}
              className={cn(
                '-mb-px inline-flex items-center gap-2 border-b-2 px-3 py-2 text-[13px] transition-colors',
                active
                  ? 'border-foreground font-semibold text-foreground'
                  : 'border-transparent text-muted-foreground hover:text-foreground',
              )}
            >
              <item.icon className="h-4 w-4" aria-hidden="true" />
              {item.label}
            </button>
          )
        })}
      </div>

      <div role="tabpanel" id={`rbac-panel-${tab}`} aria-labelledby={`rbac-tab-${tab}`}>
        {tab === 'role' ? <RoleTab /> : <UserRoleTab />}
      </div>
    </>
  )
}

// --- Tab Role ---------------------------------------------------------------

function RoleTab() {
  const queryClient = useQueryClient()
  const { can, permissions: mine } = usePermissions()
  const [notice, setNotice] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<AdminRole | null>(null)
  const [removing, setRemoving] = useState<AdminRole | null>(null)

  const roles = useQuery({ queryKey: ['admin', 'rbac', 'roles'], queryFn: adminRoleApi.list })
  const catalog = useQuery({
    queryKey: ['admin', 'rbac', 'permissions'],
    queryFn: adminRoleApi.permissions,
  })

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['admin', 'rbac'] })
  }

  const createRole = useMutation({
    mutationFn: adminRoleApi.create,
    onSuccess: (role) => {
      setNotice(`Role "${role.name}" dibuat.`)
      setCreating(false)
      invalidate()
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal membuat role.'),
  })

  const updateRole = useMutation({
    mutationFn: (input: { id: string; permissions: string[] }) =>
      adminRoleApi.updatePermissions(input.id, input.permissions),
    onSuccess: (role) => {
      setNotice(`Permission role "${role.name}" diperbarui.`)
      setEditing(null)
      invalidate()
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal menyimpan role.'),
  })

  const deleteRole = useMutation({
    mutationFn: (id: string) => adminRoleApi.remove(id),
    onSuccess: () => {
      setNotice('Role dihapus.')
      setRemoving(null)
      invalidate()
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal menghapus role.'),
  })

  const rows = roles.data ?? []

  return (
    <div className="flex flex-col gap-4">
      {notice && <Alert>{notice}</Alert>}

      <Card className="p-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <p className="text-[12px] leading-relaxed text-muted-foreground">
            Role bawaan sistem bersifat tetap dan tidak dapat diubah. Role buatan Anda bisa diatur
            permission-nya kapan saja.
          </p>
          {can('role.create') && (
            <Button size="sm" onClick={() => setCreating(true)} disabled={catalog.isLoading}>
              Buat role
            </Button>
          )}
        </div>
      </Card>

      {roles.isLoading && (
        <div className="flex flex-col gap-2" aria-busy="true">
          {Array.from({ length: 3 }).map((_, index) => (
            <Skeleton key={index} className="h-16 w-full" />
          ))}
        </div>
      )}

      {roles.isError && (
        <EmptyState
          title="Daftar role gagal dimuat"
          description={roles.error instanceof Error ? roles.error.message : 'Silakan coba lagi.'}
        />
      )}

      {!roles.isLoading && !roles.isError && rows.length === 0 && (
        <EmptyState title="Belum ada role" description="Buat role pertama untuk memulai." />
      )}

      {rows.map((role) => (
        <Card key={role.id}>
          <CardHeader>
            <div className="flex flex-wrap items-start justify-between gap-2">
              <div className="min-w-0">
                <CardTitle className="flex items-center gap-2 text-sm">
                  {role.name}
                  <Badge variant="muted">{role.code}</Badge>
                  {role.is_system && <Badge variant="outline">Bawaan sistem</Badge>}
                </CardTitle>
                <CardDescription className="text-xs">
                  {role.description || 'Tanpa deskripsi.'}
                </CardDescription>
              </div>
              <div className="flex items-center gap-2">
                <Badge variant="default">
                  {role.user_count} akun · {role.permissions.length} izin
                </Badge>
                {role.editable && can('role.edit') && (
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => setEditing(role)}
                    disabled={catalog.isLoading || catalog.isError}
                  >
                    Ubah izin
                  </Button>
                )}
                {role.editable && role.user_count === 0 && can('role.delete') && (
                  <Button size="sm" variant="destructive" onClick={() => setRemoving(role)}>
                    Hapus
                  </Button>
                )}
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <PermissionSummary permissions={role.permissions} catalog={catalog.data ?? []} />
          </CardContent>
        </Card>
      ))}

      {creating && (
        <CreateRoleModal
          catalog={catalog.data ?? []}
          owned={mine}
          pending={createRole.isPending}
          onClose={() => setCreating(false)}
          onSubmit={(payload) => createRole.mutate(payload)}
        />
      )}

      {editing && (
        <EditRoleModal
          role={editing}
          catalog={catalog.data ?? []}
          pending={updateRole.isPending}
          onClose={() => setEditing(null)}
          onSubmit={(permissions) => updateRole.mutate({ id: editing.id, permissions })}
        />
      )}

      <Modal
        open={Boolean(removing)}
        onClose={() => setRemoving(null)}
        title={`Hapus role "${removing?.name ?? ''}"?`}
        description="Role hanya bisa dihapus bila tidak dipakai akun mana pun. Tindakan ini tercatat di jejak audit."
        footer={
          <>
            <Button variant="outline" size="sm" onClick={() => setRemoving(null)}>
              Batal
            </Button>
            <Button
              variant="destructive"
              size="sm"
              isLoading={deleteRole.isPending}
              onClick={() => removing && deleteRole.mutate(removing.id)}
            >
              Hapus role
            </Button>
          </>
        }
      >
        <p className="text-[12px] leading-relaxed text-muted-foreground">
          Penghapusan tidak dapat dibatalkan. Bila role ini masih dipakai, server akan menolak dan
          Anda perlu memindahkan akunnya lebih dulu.
        </p>
      </Modal>
    </div>
  )
}

/** PermissionSummary merangkum izin yang dipegang role, dikelompokkan. */
function PermissionSummary({
  permissions,
  catalog,
}: {
  permissions: string[]
  catalog: AdminPermissionGroup[]
}) {
  const grouped = useMemo(() => groupOwned(permissions, catalog), [permissions, catalog])

  if (permissions.length === 0) {
    return <p className="text-[12px] text-muted-foreground">Belum memegang permission apa pun.</p>
  }

  return (
    <div className="flex flex-col gap-2">
      {grouped.map((group) => (
        <div key={group.group} className="flex flex-wrap items-baseline gap-1.5">
          <span className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
            {group.group}
          </span>
          {group.items.map((item) => (
            <Badge key={item.key} variant="default" title={item.key}>
              {item.label}
            </Badge>
          ))}
        </div>
      ))}
    </div>
  )
}

/** groupOwned mengelompokkan izin role menurut grup katalog. */
function groupOwned(permissions: string[], catalog: AdminPermissionGroup[]) {
  const owned = new Set(permissions)
  return catalog
    .map((group) => ({
      group: group.group,
      items: group.permissions.filter((item) => owned.has(item.key)),
    }))
    .filter((group) => group.items.length > 0)
}

// --- Modal role -------------------------------------------------------------

function CreateRoleModal({
  catalog,
  owned,
  pending,
  onClose,
  onSubmit,
}: {
  catalog: AdminPermissionGroup[]
  owned: string[]
  pending: boolean
  onClose: () => void
  onSubmit: (payload: { code: string; name: string; description?: string; permissions: string[] }) => void
}) {
  const [code, setCode] = useState('')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [selected, setSelected] = useState<string[]>([])

  const ownedSet = useMemo(() => new Set(owned), [owned])
  const valid = code.trim().length >= 2 && name.trim().length >= 2 && selected.length > 0

  return (
    <Modal
      open
      onClose={onClose}
      size="2xl"
      title="Buat role baru"
      description="Pilih permission yang dipegang role ini. Anda hanya bisa memberikan izin yang Anda miliki sendiri."
      footer={
        <>
          <Button variant="outline" size="sm" onClick={onClose}>
            Batal
          </Button>
          <Button
            size="sm"
            isLoading={pending}
            disabled={!valid}
            onClick={() =>
              onSubmit({
                code: code.trim(),
                name: name.trim(),
                description: description.trim(),
                permissions: selected,
              })
            }
          >
            Simpan role
          </Button>
        </>
      }
    >
      <div className="grid gap-3 sm:grid-cols-2">
        <div>
          <Label htmlFor="role-code">Kode role</Label>
          <Input
            id="role-code"
            className="mt-1"
            value={code}
            placeholder="mis. marketing"
            onChange={(event) => setCode(event.target.value)}
          />
          <p className="mt-1 text-[11px] text-muted-foreground">
            Huruf kecil dan tanda hubung; dipakai untuk mengenali role di audit.
          </p>
        </div>
        <div>
          <Label htmlFor="role-name">Nama tampilan</Label>
          <Input
            id="role-name"
            className="mt-1"
            value={name}
            placeholder="mis. Marketing"
            onChange={(event) => setName(event.target.value)}
          />
        </div>
      </div>
      <div>
        <Label htmlFor="role-desc">Deskripsi</Label>
        <Input
          id="role-desc"
          className="mt-1"
          value={description}
          placeholder="Ringkas tujuan role ini"
          onChange={(event) => setDescription(event.target.value)}
        />
      </div>

      <PermissionMatrix
        catalog={catalog}
        selected={selected}
        isGrantable={(key) => ownedSet.has(key)}
        onChange={setSelected}
      />
    </Modal>
  )
}

function EditRoleModal({
  role,
  catalog,
  pending,
  onClose,
  onSubmit,
}: {
  role: AdminRole
  catalog: AdminPermissionGroup[]
  pending: boolean
  onClose: () => void
  onSubmit: (permissions: string[]) => void
}) {
  const { permissions: mine } = usePermissions()
  const [selected, setSelected] = useState<string[]>(role.permissions)

  const ownedSet = useMemo(() => new Set(mine), [mine])
  const unchanged =
    selected.length === role.permissions.length &&
    selected.every((key) => role.permissions.includes(key))

  return (
    <Modal
      open
      onClose={onClose}
      size="2xl"
      title={`Izin untuk ${role.name}`}
      description="PATCH mengirim seluruh daftar izin; yang tidak dicentang akan dicabut."
      footer={
        <>
          <Button variant="outline" size="sm" onClick={onClose}>
            Batal
          </Button>
          <Button
            size="sm"
            isLoading={pending}
            disabled={selected.length === 0 || unchanged}
            onClick={() => onSubmit(selected)}
          >
            Simpan izin
          </Button>
        </>
      }
    >
      <PermissionMatrix
        catalog={catalog}
        selected={selected}
        isGrantable={(key) => ownedSet.has(key) || role.permissions.includes(key)}
        onChange={setSelected}
      />
    </Modal>
  )
}

/**
 * PermissionMatrix menampilkan centang bergrup dengan risiko tiap izin.
 *
 * Izin yang tidak boleh diberikan pelaku (`grantable=false`) dinonaktifkan
 * dengan alasan tertulis, bukan disembunyikan — agar jelas mengapa maksud
 * pemberian izin tertentu tidak bisa dijalankan.
 */
function PermissionMatrix({
  catalog,
  selected,
  isGrantable,
  onChange,
}: {
  catalog: AdminPermissionGroup[]
  selected: string[]
  isGrantable: (key: string) => boolean
  onChange: (next: string[]) => void
}) {
  const selectedSet = new Set(selected)

  const toggle = (key: string) => {
    const next = new Set(selectedSet)
    if (next.has(key)) next.delete(key)
    else next.add(key)
    onChange([...next])
  }

  return (
    <div className="flex max-h-[55vh] flex-col gap-4 overflow-y-auto pr-1">
      {catalog.map((group) => {
        const keys = group.permissions.map((item) => item.key)
        const chosen = keys.filter((key) => selectedSet.has(key)).length

        return (
          <fieldset key={group.group} className="rounded-lg border border-border p-3">
            <legend className="flex items-baseline gap-2 px-1">
              <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                {group.group}
              </span>
              <span className="text-[11px] text-muted-foreground">
                {chosen}/{keys.length}
              </span>
            </legend>
            <div className="grid gap-2 sm:grid-cols-2">
              {group.permissions.map((item) => {
                const grantable = isGrantable(item.key)
                const checked = selectedSet.has(item.key)
                const id = `perm-${item.key}`

                return (
                  <label
                    key={item.key}
                    htmlFor={id}
                    className={cn(
                      'flex cursor-pointer items-start gap-2 rounded-md border p-2 transition-colors',
                      checked ? 'border-foreground/30 bg-secondary/50' : 'border-border',
                      !grantable && 'cursor-not-allowed opacity-60',
                    )}
                  >
                    <input
                      id={id}
                      type="checkbox"
                      className="mt-0.5 h-3.5 w-3.5 shrink-0"
                      checked={checked}
                      disabled={!grantable}
                      onChange={() => toggle(item.key)}
                    />
                    <span className="min-w-0">
                      <span className="block text-[12px] font-medium leading-tight">
                        {item.label}
                      </span>
                      <span className="block text-[11px] text-muted-foreground">
                        {item.key} · {RISK_LABEL[item.risk] ?? item.risk}
                        {!grantable && ' · tidak dapat Anda berikan'}
                      </span>
                    </span>
                  </label>
                )
              })}
            </div>
          </fieldset>
        )
      })}
    </div>
  )
}

// --- Tab User ---------------------------------------------------------------

/**
 * UserRoleTab menugaskan role ke akun admin.
 *
 * Daftar memakai filter `role_id` di server supaya role dengan pemakai banyak
 * tidak perlu disaring di browser setelah seluruh halaman ditarik.
 */
function UserRoleTab() {
  const queryClient = useQueryClient()
  const { can, permissions: mine } = usePermissions()
  const [roleFilter, setRoleFilter] = useState('')
  const [page, setPage] = useState(1)
  const [notice, setNotice] = useState<string | null>(null)
  const [assigning, setAssigning] = useState<AdminUserListItem | null>(null)

  const roles = useQuery({ queryKey: ['admin', 'rbac', 'roles'], queryFn: adminRoleApi.list })

  const usersQuery = useMemo(
    () => ({
      page,
      per_page: PAGE_SIZE_USERS,
      role: 'admin',
      role_id: roleFilter || undefined,
    }),
    [page, roleFilter],
  )

  const users = useQuery({
    queryKey: ['admin', 'rbac', 'users', usersQuery],
    queryFn: () => adminUserApi.list(usersQuery),
  })

  const assign = useMutation({
    mutationFn: (input: { userId: string; roleId: string | null }) =>
      adminRoleApi.assign(input.userId, input.roleId),
    onSuccess: () => {
      setNotice('Role akun diperbarui. Akun tersebut perlu masuk ulang pada sesi berikutnya.')
      setAssigning(null)
      void queryClient.invalidateQueries({ queryKey: ['admin', 'rbac', 'users'] })
      void queryClient.invalidateQueries({ queryKey: ['admin', 'rbac', 'roles'] })
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal mengubah role.'),
  })

  const roleRows = roles.data ?? []
  const rows = users.data?.data ?? []
  const meta = users.data?.meta

  const roleName = (roleId?: string) =>
    roleRows.find((role) => role.id === roleId)?.name ?? 'Belum ditugaskan'

  return (
    <div className="flex flex-col gap-4">
      {notice && <Alert>{notice}</Alert>}

      <Card className="p-3">
        <div className="flex flex-wrap items-end gap-2">
          <div className="min-w-[200px] flex-1">
            <Label htmlFor="rbac-role-filter">Saring menurut role</Label>
            <Select
              id="rbac-role-filter"
              className="mt-1"
              value={roleFilter}
              onChange={(event) => {
                setRoleFilter(event.target.value)
                setPage(1)
              }}
            >
              <option value="">Semua akun admin</option>
              {roleRows.map((role) => (
                <option key={role.id} value={role.id}>
                  {role.name} ({role.user_count})
                </option>
              ))}
            </Select>
          </div>
          <p className="text-[11px] leading-relaxed text-muted-foreground">
            Yang tampil hanya akun ber-role kasar <code>admin</code>. Pelanggan biasa tidak dapat
            memegang role RBAC.
          </p>
        </div>
      </Card>

      {users.isLoading && (
        <div className="flex flex-col gap-2" aria-busy="true">
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className="h-12 w-full" />
          ))}
        </div>
      )}

      {users.isError && (
        <EmptyState
          title="Daftar akun gagal dimuat"
          description={users.error instanceof Error ? users.error.message : 'Silakan coba lagi.'}
        />
      )}

      {!users.isLoading && !users.isError && rows.length === 0 && (
        <EmptyState
          title="Tidak ada akun admin"
          description="Ubah filter role atau tugaskan role melalui detail pengguna."
        />
      )}

      {rows.length > 0 && (
        <Card className="overflow-x-auto">
          <table className="w-full text-[13px]">
            <thead className="bg-secondary/40">
              <tr className="border-b border-border">
                <th className="h-9 px-3 text-left text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                  Akun
                </th>
                <th className="h-9 px-3 text-left text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                  Role saat ini
                </th>
                <th className="h-9 px-3 text-right text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                  Aksi
                </th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={row.id} className="border-b border-border last:border-0">
                  <td className="px-3 py-2.5">
                    <div className="flex flex-col">
                      <span className="text-[13px] font-medium">{row.name}</span>
                      <span className="text-[11px] text-muted-foreground">{row.email}</span>
                    </div>
                  </td>
                  <td className="px-3 py-2.5 text-[12px]">
                    {row.role_code ? (
                      <Badge variant="muted">{roleName(roleRows.find((r) => r.code === row.role_code)?.id)}</Badge>
                    ) : (
                      <span className="text-muted-foreground">Belum ditugaskan</span>
                    )}
                  </td>
                  <td className="px-3 py-2.5 text-right">
                    {can('role.edit') && (
                      <Button size="sm" variant="outline" onClick={() => setAssigning(row)}>
                        Ubah role
                      </Button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>

          <div className="flex flex-wrap items-center justify-between gap-2 border-t border-border px-3 py-2">
            <p className="text-[11px] text-muted-foreground">
              {meta ? `${meta.total} akun · halaman ${meta.page}/${meta.total_pages || 1}` : ''}
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

      {assigning && (
        <AssignRoleModal
          user={assigning}
          roles={roleRows}
          owned={mine}
          pending={assign.isPending}
          onClose={() => setAssigning(null)}
          onSubmit={(roleId) => assign.mutate({ userId: assigning.id, roleId })}
        />
      )}
    </div>
  )
}

function AssignRoleModal({
  user,
  roles,
  owned,
  pending,
  onClose,
  onSubmit,
}: {
  user: AdminUserListItem
  roles: AdminRole[]
  owned: string[]
  pending: boolean
  onClose: () => void
  onSubmit: (roleId: string | null) => void
}) {
  const current = useMemo(
    () => roles.find((role) => role.code === user.role_code)?.id ?? '',
    [roles, user.role_code],
  )
  const [roleId, setRoleId] = useState(current)

  const ownedSet = useMemo(() => new Set(owned), [owned])

  // Role hanya bisa diberikan bila seluruh izinnya dimiliki pelaku; server
  // menolak pemberian di luar kewenangan (larangan eskalasi).
  const notGrantable = roles.filter(
    (role) => role.id !== current && !role.permissions.every((key) => ownedSet.has(key)),
  )

  return (
    <Modal
      open
      onClose={onClose}
      title={`Role untuk ${user.name}`}
      description="Memilih role baru mencabut sesi admin tersebut agar kewenangan baru segera berlaku."
      footer={
        <>
          <Button variant="outline" size="sm" onClick={onClose}>
            Batal
          </Button>
          <Button
            size="sm"
            isLoading={pending}
            disabled={roleId === current}
            onClick={() => onSubmit(roleId || null)}
          >
            Simpan role
          </Button>
        </>
      }
    >
      <div>
        <Label htmlFor="assign-role">Role</Label>
        <Select
          id="assign-role"
          className="mt-1"
          value={roleId}
          onChange={(event) => setRoleId(event.target.value)}
        >
          <option value="">Tanpa role (cabut akses)</option>
          {roles.map((role) => {
            const blocked = role.id !== current && !role.permissions.every((key) => ownedSet.has(key))
            return (
              <option key={role.id} value={role.id} disabled={blocked}>
                {role.name}
                {blocked ? ' — melebihi kewenangan Anda' : ''}
              </option>
            )
          })}
        </Select>
        {notGrantable.length > 0 && (
          <p className="mt-2 text-[11px] leading-relaxed text-muted-foreground">
            Role bertanda "melebihi kewenangan" memuat izin yang tidak Anda pegang, sehingga tidak
            bisa ditugaskan dari akun Anda.
          </p>
        )}
      </div>
    </Modal>
  )
}
