import { useMemo, useState, type FormEvent, type ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Pencil, Plus, Trash2 } from 'lucide-react'
import {
  Alert,
  Badge,
  Button,
  Card,
  CardDescription,
  CardHeader,
  EmptyState,
  Input,
  Label,
  Modal,
  PageHeader,
  Pagination,
  Section,
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
import { adminPairApi } from '@/lib/admin-api'
import { queryKeys } from '@/lib/queries'
import type { AdminPairCategory, AdminPairGroup, AdminPairItem } from '@/types/admin'

type Tab = 'pair' | 'kategori' | 'grup'

const TABS: { key: Tab; label: string; hint: string }[] = [
  { key: 'pair', label: 'Pair', hint: 'Instrumen yang dapat dipilih user.' },
  { key: 'kategori', label: 'Kategori', hint: 'Kelas aset: Forex, Indeks, Komoditas, Kripto.' },
  { key: 'grup', label: 'Sub-kategori', hint: 'Pengelompokan halus di dalam satu kategori.' },
]

const mesinGalat = (error: unknown, fallback: string) =>
  error instanceof Error && error.message ? error.message : fallback

/** Pilihan feed yang biasa dipakai; admin tetap boleh menulis feed lain. */
const FEED_OPTIONS = ['FOREXCOM', 'ICMARKETS', 'OANDA', 'BINANCE']

/** adminPairKey adalah kunci tunggal katalog admin; satu kali muat, dipakai tiga tab. */
const adminPairKey = ['admin', 'pairs'] as const

/**
 * AdminPairsPage mengelola kamus pair yang menjadi satu-satunya sumber pilihan
 * bagi user. User tidak pernah mengetik simbol sendiri: halaman Pair Pantauan
 * hanya menampilkan isi katalog ini.
 *
 * Tiga entitas dikelola di sini karena saling mengunci: pair wajib punya
 * kategori, dan grup wajib berada di dalam kategori. Tanpa tab kategori dan
 * grup, admin tidak akan bisa menambah pair pertama.
 */
export default function AdminPairsPage() {
  const queryClient = useQueryClient()

  const [tab, setTab] = useState<Tab>('pair')
  const [notice, setNotice] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const [pairForm, setPairForm] = useState<'baru' | AdminPairItem | null>(null)
  const [pairToDelete, setPairToDelete] = useState<AdminPairItem | null>(null)
  const [categoryForm, setCategoryForm] = useState<'baru' | AdminPairCategory | null>(null)
  const [categoryToDelete, setCategoryToDelete] = useState<AdminPairCategory | null>(null)
  const [groupForm, setGroupForm] =
    useState<{ categoryId: string } | { categoryId: string; group: AdminPairGroup } | null>(null)
  const [groupToDelete, setGroupToDelete] =
    useState<{ group: AdminPairGroup; category: AdminPairCategory } | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: adminPairKey,
    queryFn: adminPairApi.catalog,
    staleTime: 60_000,
  })

  const categories = useMemo(() => data?.categories ?? [], [data])

  const daftarPair = useMemo(
    () =>
      categories.flatMap((category) =>
        category.pairs.map((pair) => ({ pair, category })),
      ),
    [categories],
  )

  const daftarGroup = useMemo(
    () =>
      categories.flatMap((category) =>
        category.groups.map((group) => ({ group, category })),
      ),
    [categories],
  )

  // Daftar pair dan grup bisa panjang; dipecah per halaman agar tabel
  // tetap ringkas tanpa kehilangan baris.
  const paginasiPair = usePaginasiLokal(daftarPair)
  const paginasiGroup = usePaginasiLokal(daftarGroup)
  const paginasiKategori = usePaginasiLokal(categories)

  const segarkan = () => {
    void queryClient.invalidateQueries({ queryKey: adminPairKey })
    // Katalog publik ikut berubah agar halaman Pair Pantauan user segera sinkron.
    void queryClient.invalidateQueries({ queryKey: queryKeys.pairCatalog })
  }

  const sukses = (pesan: string) => {
    setError(null)
    setNotice(pesan)
    segarkan()
  }

  const gagal = (error_: unknown, fallback: string) => {
    setNotice(null)
    setError(mesinGalat(error_, fallback))
  }

  const togglePair = useMutation({
    mutationFn: (pair: AdminPairItem) =>
      adminPairApi.updatePair(pair.id, { is_enabled: !pair.is_enabled }),
    onSuccess: (_, pair) =>
      sukses(pair.is_enabled ? `${pair.symbol} dinonaktifkan.` : `${pair.symbol} diaktifkan.`),
    onError: (error_) => gagal(error_, 'Gagal mengubah status pair.'),
  })

  const hapusPair = useMutation({
    mutationFn: (pair: AdminPairItem) => adminPairApi.deletePair(pair.id),
    onSuccess: () => sukses('Pair dihapus dari katalog.'),
    onError: (error_) => gagal(error_, 'Gagal menghapus pair.'),
  })

  const toggleCategory = useMutation({
    mutationFn: (category: AdminPairCategory) =>
      adminPairApi.updateCategory(category.id, { is_enabled: !category.is_enabled }),
    onSuccess: () => sukses('Status kategori diperbarui.'),
    onError: (error_) => gagal(error_, 'Gagal mengubah status kategori.'),
  })

  const hapusCategory = useMutation({
    mutationFn: (category: AdminPairCategory) => adminPairApi.deleteCategory(category.id),
    onSuccess: () => sukses('Kategori dihapus.'),
    onError: (error_) => gagal(error_, 'Gagal menghapus kategori.'),
  })

  const toggleGroup = useMutation({
    mutationFn: (group: AdminPairGroup) =>
      adminPairApi.updateGroup(group.id, { is_enabled: !group.is_enabled }),
    onSuccess: () => sukses('Status sub-kategori diperbarui.'),
    onError: (error_) => gagal(error_, 'Gagal mengubah status sub-kategori.'),
  })

  const hapusGroup = useMutation({
    mutationFn: (group: AdminPairGroup) => adminPairApi.deleteGroup(group.id),
    onSuccess: () => sukses('Sub-kategori dihapus.'),
    onError: (error_) => gagal(error_, 'Gagal menghapus sub-kategori.'),
  })

  return (
    <>
      <PageHeader
        title="Katalog Pair"
        description="Kamus simbol yang boleh dipilih user. Watchlist Pair hanya menampilkan isi katalog ini, sehingga penulisan simbol bebas tidak dimungkinkan."
        actions={
          tab === 'pair' ? (
            <Button size="sm" onClick={() => setPairForm('baru')} disabled={categories.length === 0}>
              <Plus className="h-3.5 w-3.5" aria-hidden="true" />
              Tambah Pair
            </Button>
          ) : tab === 'kategori' ? (
            <Button size="sm" onClick={() => setCategoryForm('baru')}>
              <Plus className="h-3.5 w-3.5" aria-hidden="true" />
              Tambah Kategori
            </Button>
          ) : (
            <Button
              size="sm"
              onClick={() => setGroupForm({ categoryId: categories[0]?.id ?? '' })}
              disabled={categories.length === 0}
            >
              <Plus className="h-3.5 w-3.5" aria-hidden="true" />
              Tambah Sub-kategori
            </Button>
          )
        }
      />

      {error && <Alert variant="destructive">{error}</Alert>}
      {!error && notice && <Alert>{notice}</Alert>}

      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex flex-wrap gap-2">
              {TABS.map((entry) => (
                <Button
                  key={entry.key}
                  size="sm"
                  variant={tab === entry.key ? 'default' : 'outline'}
                  onClick={() => setTab(entry.key)}
                >
                  {entry.label}
                </Button>
              ))}
            </div>
            <CardDescription className="text-xs">
              {TABS.find((entry) => entry.key === tab)?.hint}
            </CardDescription>
          </div>
        </CardHeader>
      </Card>

      {isLoading && <Skeleton className="h-40 w-full" />}

      {!isLoading && tab === 'pair' && (
        <Section
          title="Daftar Pair"
          description="Nonaktifkan pair untuk menghentikan sinyal baru tanpa menghapus riwayat."
        >
          {daftarPair.length === 0 ? (
            <EmptyState
              title="Katalog pair masih kosong"
              description={
                categories.length === 0
                  ? 'Buat kategori terlebih dahulu, lalu tambahkan pair ke dalamnya.'
                  : 'Tambahkan pair pertama agar user dapat memilihnya di halaman Watchlist Pair.'
              }
              action={
                categories.length === 0 ? (
                  <Button size="sm" onClick={() => setCategoryForm('baru')}>
                    Tambah Kategori
                  </Button>
                ) : (
                  <Button size="sm" onClick={() => setPairForm('baru')}>
                    Tambah Pair
                  </Button>
                )
              }
            />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Simbol</TableHead>
                  <TableHead>Nama Tampilan</TableHead>
                  <TableHead>Kategori / Sub</TableHead>
                  <TableHead>Alias</TableHead>
                  <TableHead>Feed</TableHead>
                  <TableHead className="text-right">Pelanggan</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Aksi</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {paginasiPair.baris.map(({ pair, category }) => (
                  <TableRow key={pair.id}>
                    <TableCell className="font-semibold text-foreground">{pair.symbol}</TableCell>
                    <TableCell>{pair.display_name || <span className="text-muted-foreground">—</span>}</TableCell>
                    <TableCell className="text-muted-foreground">
                      {category.label}
                      {pair.group_key ? ` · ${labelGroup(categories, pair)}` : ''}
                    </TableCell>
                    <TableCell className="text-muted-foreground">{pair.aliases || '—'}</TableCell>
                    <TableCell>
                      <Badge variant="outline" className="text-[10px] uppercase">
                        {pair.feed_source || 'Bawaan'}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right tabular">{pair.subscribers}</TableCell>
                    <TableCell>
                      <Badge variant={pair.is_enabled ? 'outline' : 'muted'}>
                        {pair.is_enabled ? 'Aktif' : 'Nonaktif'}
                      </Badge>
                      {!category.is_enabled && <Badge variant="muted">Kategori off</Badge>}
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end gap-1">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => setPairForm(pair)}
                          aria-label={`Sunting ${pair.symbol}`}
                        >
                          <Pencil className="h-3.5 w-3.5" aria-hidden="true" />
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          disabled={togglePair.isPending}
                          onClick={() => togglePair.mutate(pair)}
                        >
                          {pair.is_enabled ? 'Nonaktifkan' : 'Aktifkan'}
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          className="text-destructive hover:bg-destructive/10"
                          onClick={() => setPairToDelete(pair)}
                          aria-label={`Hapus ${pair.symbol}`}
                        >
                          <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
          <div className="border-t border-border px-3 py-2">
            <Pagination
              page={paginasiPair.page}
              totalPages={paginasiPair.totalPages}
              shown={paginasiPair.baris.length}
              total={paginasiPair.total}
              unit="pair"
              pageSize={paginasiPair.pageSize}
              onPageChange={paginasiPair.setPage}
              onPageSizeChange={paginasiPair.setPageSize}
            />
          </div>
        </Section>
      )}

      {!isLoading && tab === 'kategori' && (
        <Section
          title="Kategori"
          description="Menonaktifkan kategori ikut menyembunyikan seluruh pair di dalamnya dari user."
        >
          {categories.length === 0 ? (
            <EmptyState
              title="Belum ada kategori"
              description="Kategori adalah wadah tertinggi katalog. Buat satu dulu sebelum menambah pair."
              action={
                <Button size="sm" onClick={() => setCategoryForm('baru')}>
                  Tambah Kategori
                </Button>
              }
            />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Kunci</TableHead>
                  <TableHead>Label</TableHead>
                  <TableHead className="text-right">Urutan</TableHead>
                  <TableHead className="text-right">Pair</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Aksi</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {paginasiKategori.baris.map((category) => (
                  <TableRow key={category.id}>
                    <TableCell className="font-mono text-[12px]">{category.key}</TableCell>
                    <TableCell className="font-medium">{category.label}</TableCell>
                    <TableCell className="text-right tabular">{category.sort_order}</TableCell>
                    <TableCell className="text-right tabular">{category.pairs.length}</TableCell>
                    <TableCell>
                      <Badge variant={category.is_enabled ? 'outline' : 'muted'}>
                        {category.is_enabled ? 'Aktif' : 'Nonaktif'}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end gap-1">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => setCategoryForm(category)}
                          aria-label={`Sunting kategori ${category.label}`}
                        >
                          <Pencil className="h-3.5 w-3.5" aria-hidden="true" />
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          disabled={toggleCategory.isPending}
                          onClick={() => toggleCategory.mutate(category)}
                        >
                          {category.is_enabled ? 'Nonaktifkan' : 'Aktifkan'}
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          className="text-destructive hover:bg-destructive/10"
                          onClick={() => setCategoryToDelete(category)}
                          aria-label={`Hapus kategori ${category.label}`}
                        >
                          <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
          <div className="border-t border-border px-3 py-2">
            <Pagination
              page={paginasiKategori.page}
              totalPages={paginasiKategori.totalPages}
              shown={paginasiKategori.baris.length}
              total={paginasiKategori.total}
              unit="kategori"
              pageSize={paginasiKategori.pageSize}
              onPageChange={paginasiKategori.setPage}
              onPageSizeChange={paginasiKategori.setPageSize}
            />
          </div>
        </Section>
      )}

      {!isLoading && tab === 'grup' && (
        <Section
          title="Sub-kategori"
          description="Grup memecah kategori menjadi bagian kecil, misalnya Forex · Major."
        >
          {daftarGroup.length === 0 ? (
            <EmptyState
              title="Belum ada sub-kategori"
              description="Sub-kategori bersifat opsional; pair tanpa grup otomatis masuk grup “Umum”."
              action={
                categories.length === 0 ? undefined : (
                  <Button size="sm" onClick={() => setGroupForm({ categoryId: categories[0]?.id ?? '' })}>
                    Tambah Sub-kategori
                  </Button>
                )
              }
            />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Kategori</TableHead>
                  <TableHead>Kunci</TableHead>
                  <TableHead>Label</TableHead>
                  <TableHead className="text-right">Urutan</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Aksi</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {paginasiGroup.baris.map(({ group, category }) => (
                  <TableRow key={group.id}>
                    <TableCell className="text-muted-foreground">{category.label}</TableCell>
                    <TableCell className="font-mono text-[12px]">{group.key}</TableCell>
                    <TableCell className="font-medium">{group.label}</TableCell>
                    <TableCell className="text-right tabular">{group.sort_order}</TableCell>
                    <TableCell>
                      <Badge variant={group.is_enabled ? 'outline' : 'muted'}>
                        {group.is_enabled ? 'Aktif' : 'Nonaktif'}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end gap-1">
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => setGroupForm({ categoryId: category.id, group })}
                          aria-label={`Sunting sub-kategori ${group.label}`}
                        >
                          <Pencil className="h-3.5 w-3.5" aria-hidden="true" />
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          disabled={toggleGroup.isPending}
                          onClick={() => toggleGroup.mutate(group)}
                        >
                          {group.is_enabled ? 'Nonaktifkan' : 'Aktifkan'}
                        </Button>
                        <Button
                          size="sm"
                          variant="ghost"
                          className="text-destructive hover:bg-destructive/10"
                          onClick={() => setGroupToDelete({ group, category })}
                          aria-label={`Hapus sub-kategori ${group.label}`}
                        >
                          <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
          <div className="border-t border-border px-3 py-2">
            <Pagination
              page={paginasiGroup.page}
              totalPages={paginasiGroup.totalPages}
              shown={paginasiGroup.baris.length}
              total={paginasiGroup.total}
              unit="grup"
              pageSize={paginasiGroup.pageSize}
              onPageChange={paginasiGroup.setPage}
              onPageSizeChange={paginasiGroup.setPageSize}
            />
          </div>
        </Section>
      )}

      {/* --- Form pair --------------------------------------------------- */}
      {pairForm && (
        <PairFormModal
          key={pairForm === 'baru' ? 'pair-baru' : pairForm.id}
          categories={categories}
          initial={pairForm === 'baru' ? null : pairForm}
          onClose={() => setPairForm(null)}
          onSaved={(pesan) => {
            setPairForm(null)
            sukses(pesan)
          }}
          onError={(error_) => gagal(error_, 'Gagal menyimpan pair.')}
        />
      )}

      {/* --- Form kategori ---------------------------------------------- */}
      {categoryForm && (
        <CategoryFormModal
          key={categoryForm === 'baru' ? 'kategori-baru' : categoryForm.id}
          initial={categoryForm === 'baru' ? null : categoryForm}
          onClose={() => setCategoryForm(null)}
          onSaved={(pesan) => {
            setCategoryForm(null)
            sukses(pesan)
          }}
          onError={(error_) => gagal(error_, 'Gagal menyimpan kategori.')}
        />
      )}

      {/* --- Form sub-kategori ------------------------------------------ */}
      {groupForm && (
        <GroupFormModal
          key={
            'group' in groupForm
              ? `grup-${groupForm.group.id}`
              : `grup-baru-${groupForm.categoryId}`
          }
          categories={categories}
          categoryId={groupForm.categoryId}
          initial={'group' in groupForm ? groupForm.group : null}
          onClose={() => setGroupForm(null)}
          onSaved={(pesan) => {
            setGroupForm(null)
            sukses(pesan)
          }}
          onError={(error_) => gagal(error_, 'Gagal menyimpan sub-kategori.')}
        />
      )}

      {/* --- Konfirmasi hapus ------------------------------------------- */}
      <ConfirmModal
        open={Boolean(pairToDelete)}
        title={`Hapus pair ${pairToDelete?.symbol ?? ''}?`}
        description="Pair yang masih dipakai langganan aktif tidak dapat dihapus; nonaktifkan saja. Tindakan ini tidak dapat dibatalkan."
        confirmLabel="Hapus Pair"
        pending={hapusPair.isPending}
        onClose={() => setPairToDelete(null)}
        onConfirm={() => {
          if (!pairToDelete) return
          hapusPair.mutate(pairToDelete, { onSuccess: () => setPairToDelete(null) })
        }}
      />

      <ConfirmModal
        open={Boolean(categoryToDelete)}
        title={`Hapus kategori ${categoryToDelete?.label ?? ''}?`}
        description="Kategori hanya dapat dihapus bila tidak lagi menyimpan pair maupun sub-kategori."
        confirmLabel="Hapus Kategori"
        pending={hapusCategory.isPending}
        onClose={() => setCategoryToDelete(null)}
        onConfirm={() => {
          if (!categoryToDelete) return
          hapusCategory.mutate(categoryToDelete, { onSuccess: () => setCategoryToDelete(null) })
        }}
      />

      <ConfirmModal
        open={Boolean(groupToDelete)}
        title={`Hapus sub-kategori ${groupToDelete?.group.label ?? ''}?`}
        description="Pair di dalam grup ini tidak terhapus; pair tersebut kembali ke grup “Umum”."
        confirmLabel="Hapus Sub-kategori"
        pending={hapusGroup.isPending}
        onClose={() => setGroupToDelete(null)}
        onConfirm={() => {
          if (!groupToDelete) return
          hapusGroup.mutate(groupToDelete.group, { onSuccess: () => setGroupToDelete(null) })
        }}
      />
    </>
  )
}

/** labelGroup mencari label grup dari pair; dipakai pada kolom kategori. */
function labelGroup(categories: AdminPairCategory[], pair: AdminPairItem): string {
  for (const category of categories) {
    const found = category.groups.find((group) => group.id === pair.group_id)
    if (found) return found.label
  }
  return 'Umum'
}

type FormShellProps = {
  title: string
  description: string
  onClose: () => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
  pending: boolean
  submitLabel: string
  children: ReactNode
  error: string | null
}

/** FormShell merangkum tata letak dan tombol aksi yang sama untuk tiga formulir. */
function FormShell({
  title,
  description,
  onClose,
  onSubmit,
  pending,
  submitLabel,
  children,
  error,
}: FormShellProps) {
  return (
    <Modal open onClose={onClose} title={title} description={description} size="xl">
      <form onSubmit={onSubmit} className="space-y-4" noValidate>
        {error && <Alert variant="destructive">{error}</Alert>}
        {children}
        <div className="flex items-center justify-between border-t border-border pt-4">
          <Button variant="ghost" size="sm" type="button" onClick={onClose}>
            Batal
          </Button>
          <Button size="sm" type="submit" isLoading={pending}>
            {submitLabel}
          </Button>
        </div>
      </form>
    </Modal>
  )
}

function PairFormModal({
  categories,
  initial,
  onClose,
  onSaved,
  onError,
}: {
  categories: AdminPairCategory[]
  initial: AdminPairItem | null
  onClose: () => void
  onSaved: (pesan: string) => void
  onError: (error: unknown) => void
}) {
  const [symbol, setSymbol] = useState(initial?.symbol ?? '')
  const [displayName, setDisplayName] = useState(initial?.display_name ?? '')
  const [description, setDescription] = useState(initial?.description ?? '')
  const [categoryId, setCategoryId] = useState(initial?.category_id ?? categories[0]?.id ?? '')
  const [groupId, setGroupId] = useState(initial?.group_id ?? '')
  const [aliases, setAliases] = useState(initial?.aliases ?? '')
  const [correlated, setCorrelated] = useState(initial?.correlated_symbols ?? '')
  const [feedSource, setFeedSource] = useState(initial?.feed_source ?? '')
  const [sortOrder, setSortOrder] = useState(String(initial?.sort_order ?? 0))
  const [enabled, setEnabled] = useState(initial?.is_enabled ?? true)
  const [error, setError] = useState<string | null>(null)

  const category = categories.find((entry) => entry.id === categoryId)
  const groups = category?.groups ?? []

  const simpan = useMutation({
    mutationFn: async () => {
      if (!categoryId) throw new Error('Pilih kategori terlebih dahulu.')
      if (!initial && symbol.trim().length < 3) throw new Error('Simbol minimal 3 karakter.')

      const payload: Record<string, unknown> = {
        display_name: displayName.trim(),
        description: description.trim(),
        category_id: categoryId,
        group_id: groupId,
        aliases: aliases.trim(),
        correlated_symbols: correlated.trim(),
        feed_source: feedSource.trim().toUpperCase(),
        sort_order: Number(sortOrder) || 0,
        is_enabled: enabled,
      }

      if (initial) {
        await adminPairApi.updatePair(initial.id, payload)
        return `Pair ${initial.symbol} diperbarui.`
      }

      const simbolBersih = symbol.trim().toUpperCase().replace(/\s+/g, '')
      await adminPairApi.createPair({ ...payload, symbol: simbolBersih })
      return `Pair ${simbolBersih} ditambahkan ke katalog.`
    },
    onSuccess: (pesan) => onSaved(pesan),
    onError: (error_) => {
      setError(mesinGalat(error_, 'Gagal menyimpan pair.'))
      onError(error_)
    },
  })

  return (
    <FormShell
      title={initial ? `Sunting Pair ${initial.symbol}` : 'Tambah Pair Baru'}
      description={
        initial
          ? 'Simbol tidak dapat diubah setelah dibuat agar riwayat langganan tetap utuh.'
          : 'Simbol harus unik, termasuk terhadap alias pair lain.'
      }
      onClose={onClose}
      onSubmit={(event) => {
        event.preventDefault()
        setError(null)
        simpan.mutate()
      }}
      pending={simpan.isPending}
      submitLabel={initial ? 'Simpan Perubahan' : 'Tambah Pair'}
      error={error}
    >
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="pair-symbol">Simbol</Label>
          <Input
            id="pair-symbol"
            value={symbol}
            disabled={Boolean(initial)}
            placeholder="XAUUSD"
            onChange={(event) => setSymbol(event.target.value.toUpperCase())}
          />
          <p className="text-[11px] text-muted-foreground">
            Tanpa pemisah, contoh XAUUSD atau EURUSD.
          </p>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="pair-name">Nama Tampilan</Label>
          <Input
            id="pair-name"
            value={displayName}
            placeholder="Gold / US Dollar"
            onChange={(event) => setDisplayName(event.target.value)}
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="pair-category">Kategori</Label>
          <Select
            id="pair-category"
            value={categoryId}
            onChange={(event) => {
              setCategoryId(event.target.value)
              setGroupId('')
            }}
          >
            <option value="">Pilih kategori</option>
            {categories.map((entry) => (
              <option key={entry.id} value={entry.id}>
                {entry.label}
              </option>
            ))}
          </Select>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="pair-group">Sub-kategori</Label>
          <Select
            id="pair-group"
            value={groupId}
            onChange={(event) => setGroupId(event.target.value)}
            disabled={groups.length === 0}
          >
            <option value="">{groups.length === 0 ? 'Kategori ini tanpa grup' : 'Tanpa grup (Umum)'}</option>
            {groups.map((group) => (
              <option key={group.id} value={group.id}>
                {group.label}
              </option>
            ))}
          </Select>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="pair-aliases">Alias</Label>
          <Input
            id="pair-aliases"
            value={aliases}
            placeholder="XAU/USD, GOLD"
            onChange={(event) => setAliases(event.target.value)}
          />
          <p className="text-[11px] text-muted-foreground">Dipisah koma; ikut divalidasi server.</p>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="pair-correlated">Simbol Pembanding (SMT)</Label>
          <Input
            id="pair-correlated"
            value={correlated}
            placeholder="DXY, US10Y"
            onChange={(event) => setCorrelated(event.target.value)}
          />
          <p className="text-[11px] text-muted-foreground">Opsional, dipisah koma.</p>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="pair-sort">Urutan Tampil</Label>
          <Input
            id="pair-sort"
            type="number"
            value={sortOrder}
            onChange={(event) => setSortOrder(event.target.value)}
          />
          <p className="text-[11px] text-muted-foreground">Angka kecil tampil lebih dulu.</p>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="pair-feed">Data Feed Broker</Label>
          <Input
            id="pair-feed"
            list="pair-feed-options"
            value={feedSource}
            placeholder="FOREXCOM"
            onChange={(event) => setFeedSource(event.target.value.toUpperCase())}
          />
          <datalist id="pair-feed-options">
            {FEED_OPTIONS.map((feed) => (
              <option key={feed} value={feed} />
            ))}
          </datalist>
          <p className="text-[11px] text-muted-foreground">
            Sumber candle yang dibaca engine dan chart. Kosongkan untuk memakai bawaan
            (FOREXCOM untuk forex/logam/indeks, BINANCE untuk kripto).
          </p>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="pair-enabled">Status</Label>
          <Select
            id="pair-enabled"
            value={enabled ? 'aktif' : 'nonaktif'}
            onChange={(event) => setEnabled(event.target.value === 'aktif')}
          >
            <option value="aktif">Aktif</option>
            <option value="nonaktif">Nonaktif</option>
          </Select>
        </div>
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="pair-description">Deskripsi Singkat</Label>
        <Input
          id="pair-description"
          value={description}
          placeholder="Emas terhadap dolar AS"
          onChange={(event) => setDescription(event.target.value)}
        />
      </div>
    </FormShell>
  )
}

function CategoryFormModal({
  initial,
  onClose,
  onSaved,
  onError,
}: {
  initial: AdminPairCategory | null
  onClose: () => void
  onSaved: (pesan: string) => void
  onError: (error: unknown) => void
}) {
  const [key, setKey] = useState(initial?.key ?? '')
  const [label, setLabel] = useState(initial?.label ?? '')
  const [sortOrder, setSortOrder] = useState(String(initial?.sort_order ?? 0))
  const [enabled, setEnabled] = useState(initial?.is_enabled ?? true)
  const [error, setError] = useState<string | null>(null)

  const simpan = useMutation({
    mutationFn: async () => {
      if (label.trim().length < 2) throw new Error('Label minimal 2 karakter.')
      if (!initial && key.trim().length < 2) throw new Error('Kunci minimal 2 karakter.')

      if (initial) {
        await adminPairApi.updateCategory(initial.id, {
          label: label.trim(),
          sort_order: Number(sortOrder) || 0,
          is_enabled: enabled,
        })
        return `Kategori ${label.trim()} diperbarui.`
      }

      await adminPairApi.createCategory({
        key: key.trim(),
        label: label.trim(),
        sort_order: Number(sortOrder) || 0,
        is_enabled: enabled,
      })
      return `Kategori ${label.trim()} dibuat.`
    },
    onSuccess: (pesan) => onSaved(pesan),
    onError: (error_) => {
      setError(mesinGalat(error_, 'Gagal menyimpan kategori.'))
      onError(error_)
    },
  })

  return (
    <FormShell
      title={initial ? `Sunting Kategori ${initial.label}` : 'Tambah Kategori'}
      description="Kunci kategori dipakai API dan frontend, jadi tidak dapat diubah setelah dibuat."
      onClose={onClose}
      onSubmit={(event) => {
        event.preventDefault()
        setError(null)
        simpan.mutate()
      }}
      pending={simpan.isPending}
      submitLabel={initial ? 'Simpan Perubahan' : 'Tambah Kategori'}
      error={error}
    >
      <div className="grid gap-3 sm:grid-cols-3">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="category-key">Kunci</Label>
          <Input
            id="category-key"
            value={key}
            disabled={Boolean(initial)}
            placeholder="forex"
            onChange={(event) => setKey(event.target.value.toLowerCase())}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="category-label">Label</Label>
          <Input
            id="category-label"
            value={label}
            placeholder="Forex"
            onChange={(event) => setLabel(event.target.value)}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="category-sort">Urutan</Label>
          <Input
            id="category-sort"
            type="number"
            value={sortOrder}
            onChange={(event) => setSortOrder(event.target.value)}
          />
        </div>
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="category-enabled">Status</Label>
        <Select
          id="category-enabled"
          value={enabled ? 'aktif' : 'nonaktif'}
          onChange={(event) => setEnabled(event.target.value === 'aktif')}
        >
          <option value="aktif">Aktif</option>
          <option value="nonaktif">Nonaktif</option>
        </Select>
        <p className="text-[11px] text-muted-foreground">
          Kategori nonaktif menyembunyikan seluruh pair di dalamnya dari user.
        </p>
      </div>
    </FormShell>
  )
}

function GroupFormModal({
  categories,
  categoryId,
  initial,
  onClose,
  onSaved,
  onError,
}: {
  categories: AdminPairCategory[]
  categoryId: string
  initial: AdminPairGroup | null
  onClose: () => void
  onSaved: (pesan: string) => void
  onError: (error: unknown) => void
}) {
  const [kategoriId, setKategoriId] = useState(categoryId)
  const [key, setKey] = useState(initial?.key ?? '')
  const [label, setLabel] = useState(initial?.label ?? '')
  const [sortOrder, setSortOrder] = useState(String(initial?.sort_order ?? 0))
  const [enabled, setEnabled] = useState(initial?.is_enabled ?? true)
  const [error, setError] = useState<string | null>(null)

  const simpan = useMutation({
    mutationFn: async () => {
      if (label.trim().length < 2) throw new Error('Label minimal 2 karakter.')
      if (!initial && key.trim().length < 2) throw new Error('Kunci minimal 2 karakter.')
      if (!initial && !kategoriId) throw new Error('Pilih kategori induk.')

      if (initial) {
        await adminPairApi.updateGroup(initial.id, {
          label: label.trim(),
          sort_order: Number(sortOrder) || 0,
          is_enabled: enabled,
        })
        return `Sub-kategori ${label.trim()} diperbarui.`
      }

      await adminPairApi.createGroup({
        category_id: kategoriId,
        key: key.trim(),
        label: label.trim(),
        sort_order: Number(sortOrder) || 0,
        is_enabled: enabled,
      })
      return `Sub-kategori ${label.trim()} dibuat.`
    },
    onSuccess: (pesan) => onSaved(pesan),
    onError: (error_) => {
      setError(mesinGalat(error_, 'Gagal menyimpan sub-kategori.'))
      onError(error_)
    },
  })

  return (
    <FormShell
      title={initial ? `Sunting Sub-kategori ${initial.label}` : 'Tambah Sub-kategori'}
      description="Sub-kategori selalu berada di dalam satu kategori dan tidak dapat dipindah."
      onClose={onClose}
      onSubmit={(event) => {
        event.preventDefault()
        setError(null)
        simpan.mutate()
      }}
      pending={simpan.isPending}
      submitLabel={initial ? 'Simpan Perubahan' : 'Tambah Sub-kategori'}
      error={error}
    >
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="group-category">Kategori Induk</Label>
          <Select
            id="group-category"
            value={kategoriId}
            disabled={Boolean(initial)}
            onChange={(event) => setKategoriId(event.target.value)}
          >
            <option value="">Pilih kategori</option>
            {categories.map((entry) => (
              <option key={entry.id} value={entry.id}>
                {entry.label}
              </option>
            ))}
          </Select>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="group-key">Kunci</Label>
          <Input
            id="group-key"
            value={key}
            disabled={Boolean(initial)}
            placeholder="major"
            onChange={(event) => setKey(event.target.value.toLowerCase())}
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="group-label">Label</Label>
          <Input
            id="group-label"
            value={label}
            placeholder="Major Pairs"
            onChange={(event) => setLabel(event.target.value)}
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="group-sort">Urutan</Label>
          <Input
            id="group-sort"
            type="number"
            value={sortOrder}
            onChange={(event) => setSortOrder(event.target.value)}
          />
        </div>
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="group-enabled">Status</Label>
        <Select
          id="group-enabled"
          value={enabled ? 'aktif' : 'nonaktif'}
          onChange={(event) => setEnabled(event.target.value === 'aktif')}
        >
          <option value="aktif">Aktif</option>
          <option value="nonaktif">Nonaktif</option>
        </Select>
      </div>
    </FormShell>
  )
}

function ConfirmModal({
  open,
  title,
  description,
  confirmLabel,
  pending,
  onClose,
  onConfirm,
}: {
  open: boolean
  title: string
  description: string
  confirmLabel: string
  pending: boolean
  onClose: () => void
  onConfirm: () => void
}) {
  return (
    <Modal
      open={open}
      onClose={onClose}
      title={title}
      description={description}
      footer={
        <div className="flex justify-end gap-2">
          <Button variant="outline" size="sm" onClick={onClose}>
            Batal
          </Button>
          <Button variant="destructive" size="sm" isLoading={pending} onClick={onConfirm}>
            {confirmLabel}
          </Button>
        </div>
      }
    >
      <p className="text-[12px] text-muted-foreground">
        Pastikan tidak ada pelanggan aktif yang masih memakai data ini.
      </p>
    </Modal>
  )
}
