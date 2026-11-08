import { useId, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  ChevronDown,
  ChevronRight,
  ChevronUp,
  Eye,
  EyeOff,
  FolderTree,
  LayoutTemplate,
  Layers,
  Plus,
  Trash2,
} from 'lucide-react'
import {
  Alert,
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Input,
  Label,
  PageHeader,
  Skeleton,
} from '@/components/ui'
import { adminCmsApi } from '@/lib/system-api'
import { formatDateTime } from '@/lib/format'
import { cn } from '@/lib/utils'
import type {
  PageContent,
  PageContentPayload,
  PageItem,
  PageSection,
  PageSectionKind,
} from '@/types/system'

/** Halaman publik yang isinya dapat disunting dari panel. */
const MANAGED_PAGES: { slug: string; label: string; note: string }[] = [
  {
    slug: 'fitur',
    label: 'Fitur',
    note: 'Penjelasan kemampuan dan batasannya. Bagian "kemampuan" memakai jenis Fitur agar catatan batas ikut tampil.',
  },
  {
    slug: 'harga',
    label: 'Harga',
    note: 'Judul, pengantar, FAQ pembayaran, dan disclaimer. Harga paket sendiri diatur di menu Paket & Harga.',
  },
  {
    slug: 'tentang',
    label: 'Tentang',
    note: 'Menjelaskan siapa kami dan apa yang tidak kami janjikan. Bagian itu sengaja berjenis Peringatan.',
  },
  {
    slug: 'bantuan',
    label: 'Pusat Bantuan',
    note: 'Topik panduan dan jalur kontak. Isi tiap topik diatur terpisah.',
  },
  {
    slug: 'legal',
    label: 'Legal Center',
    note: 'Ringkasan dokumen di halaman hub. Isi dokumen legal sendiri di menu Dokumen Legal.',
  },
]

const KIND_LABEL: Record<PageSectionKind, string> = {
  prose: 'Prosa (markdown)',
  cards: 'Kartu',
  steps: 'Langkah bernomor',
  faq: 'Tanya-jawab',
  table: 'Tabel',
  callout: 'Peringatan',
  features: 'Kemampuan + batasan',
  stats: 'Angka ringkas',
}

const KINDS = Object.keys(KIND_LABEL) as PageSectionKind[]

/** draftFromPage menyalin isi server ke keadaan yang bisa disunting. */
function draftFromPage(page: PageContent): PageContentPayload {
  return {
    eyebrow: page.eyebrow,
    title: page.title,
    lead: page.lead,
    meta_title: page.meta_title,
    meta_description: page.meta_description,
    is_published: page.is_published,
    sections: page.sections.map((section) => ({
      ...section,
      items: section.items.map((item) => ({ ...item })),
    })),
  }
}

/**
 * AdminPageContentPage mengelola isi halaman publik dengan format Tree Hierarchy.
 *
 * Mengadopsi struktur Notion: panel kiri menampilkan pohon struktur (Metadata Halaman
 * -> Bagian / Sections -> Butir / Items), dan panel kanan menampilkan editor terfokus.
 */
export default function AdminPageContentPage() {
  const [slug, setSlug] = useState(MANAGED_PAGES[0]?.slug ?? 'fitur')
  const meta = MANAGED_PAGES.find((entry) => entry.slug === slug) ?? MANAGED_PAGES[0]

  return (
    <>
      <PageHeader
        title="Konten Halaman"
        description="Menyunting isi halaman publik tanpa deploy ulang. Menggunakan format struktur pohon (Tree)."
      />
      <div className="grid gap-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Pilih halaman</CardTitle>
            <CardDescription className="text-xs">{meta?.note}</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              {MANAGED_PAGES.map((entry) => (
                <Button
                  key={entry.slug}
                  size="sm"
                  variant={entry.slug === slug ? 'default' : 'outline'}
                  onClick={() => setSlug(entry.slug)}
                >
                  {entry.label}
                </Button>
              ))}
            </div>
          </CardContent>
        </Card>

        <PageEditor key={slug} slug={slug} />
      </div>
    </>
  )
}

type SelectedNode = { type: 'page' } | { type: 'section'; index: number }

/** PageEditor menyunting satu halaman dengan tata letak Tree + Focused Editor. */
function PageEditor({ slug }: { slug: string }) {
  const queryClient = useQueryClient()
  const [notice, setNotice] = useState<string | null>(null)
  const [selectedNode, setSelectedNode] = useState<SelectedNode>({ type: 'page' })
  const [expandedSections, setExpandedSections] = useState<Record<number, boolean>>({})

  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'cms', 'page', slug],
    queryFn: () => adminCmsApi.page(slug),
  })

  const [draft, setDraft] = useState<{ slug: string; value: PageContentPayload | null }>({
    slug,
    value: null,
  })

  if (data && draft.slug !== slug) {
    setDraft({ slug, value: draftFromPage(data) })
  }

  if (data && draft.slug === slug && draft.value === null) {
    setDraft({ slug, value: draftFromPage(data) })
  }

  const save = useMutation({
    mutationFn: (payload: PageContentPayload) => adminCmsApi.savePage(slug, payload),
    onSuccess: (saved) => {
      setNotice('Halaman berhasil disimpan dan langsung aktif di situs publik.')
      setDraft({ slug, value: draftFromPage(saved) })
      void queryClient.invalidateQueries({ queryKey: ['admin', 'cms', 'page', slug] })
      void queryClient.invalidateQueries({ queryKey: ['public', 'page', slug] })
    },
    onError: (error) =>
      setNotice(error instanceof Error ? error.message : 'Gagal menyimpan halaman.'),
  })

  const editor = draft.value
  const sectionCount = editor?.sections.length ?? 0

  if (isLoading || !editor) {
    return (
      <div className="grid gap-4 lg:grid-cols-12">
        <div className="lg:col-span-4">
          <Card className="h-full">
            <CardHeader className="pb-3">
              <Skeleton className="h-5 w-32" />
              <Skeleton className="h-4 w-48 mt-1" />
            </CardHeader>
            <CardContent className="space-y-3 pt-2">
              <Skeleton className="h-10 w-full" />
              <div className="space-y-2 pt-2">
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-8 w-full" />
              </div>
            </CardContent>
          </Card>
        </div>
        <div className="lg:col-span-8">
          <Card>
            <CardHeader>
              <Skeleton className="h-5 w-40" />
              <Skeleton className="h-4 w-60 mt-1" />
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Skeleton className="h-4 w-20" />
                <Skeleton className="h-10 w-full" />
              </div>
              <div className="space-y-2">
                <Skeleton className="h-4 w-28" />
                <Skeleton className="h-10 w-full" />
              </div>
              <div className="space-y-2">
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-24 w-full" />
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    )
  }

  const setEditor = (value: PageContentPayload) => setDraft({ slug, value })
  const patch = (changes: Partial<PageContentPayload>) => setEditor({ ...editor, ...changes })

  const patchSection = (index: number, changes: Partial<PageSection>) => {
    const sections = editor.sections.map((section, current) =>
      current === index ? { ...section, ...changes } : section,
    )
    patch({ sections })
  }

  const patchItem = (sectionIndex: number, itemIndex: number, changes: Partial<PageItem>) => {
    const sections = editor.sections.map((section, current) => {
      if (current !== sectionIndex) return section
      return {
        ...section,
        items: section.items.map((item, inner) =>
          inner === itemIndex ? { ...item, ...changes } : item,
        ),
      }
    })
    patch({ sections })
  }

  const addSection = () => {
    const key = `bagian-${sectionCount + 1}`
    const newSection: PageSection = {
      key,
      kind: 'prose',
      eyebrow: '',
      title: 'Bagian baru',
      lead: '',
      body: '',
      layout: '',
      sort_order: sectionCount + 1,
      is_published: true,
      items: [],
    }
    patch({
      sections: [...editor.sections, newSection],
    })
    setSelectedNode({ type: 'section', index: editor.sections.length })
  }

  const removeSection = (index: number) => {
    patch({ sections: editor.sections.filter((_, current) => current !== index) })
    if (selectedNode.type === 'section' && selectedNode.index === index) {
      setSelectedNode({ type: 'page' })
    }
  }

  const moveSection = (index: number, direction: -1 | 1) => {
    const target = index + direction
    if (target < 0 || target >= editor.sections.length) return
    const sections = [...editor.sections]
    const swapped = sections[index]
    const other = sections[target]
    if (!swapped || !other) return
    sections[index] = other
    sections[target] = swapped
    patch({
      sections: sections.map((section, position) => ({ ...section, sort_order: position + 1 })),
    })
    if (selectedNode.type === 'section' && selectedNode.index === index) {
      setSelectedNode({ type: 'section', index: target })
    }
  }

  const addItem = (sectionIndex: number) => {
    const section = editor.sections[sectionIndex]
    if (!section) return
    patchSection(sectionIndex, {
      items: [
        ...section.items,
        {
          label: 'Butir baru',
          value: '',
          note: '',
          icon: '',
          href: '',
          sort_order: section.items.length + 1,
        },
      ],
    })
    setExpandedSections((prev) => ({ ...prev, [sectionIndex]: true }))
  }

  const removeItem = (sectionIndex: number, itemIndex: number) => {
    const section = editor.sections[sectionIndex]
    if (!section) return
    patchSection(sectionIndex, {
      items: section.items.filter((_, current) => current !== itemIndex),
    })
  }

  const toggleExpand = (index: number) => {
    setExpandedSections((prev) => ({ ...prev, [index]: !prev[index] }))
  }

  const activeSection =
    selectedNode.type === 'section' ? editor.sections[selectedNode.index] : null

  return (
    <div className="space-y-4">
      {notice && <Alert>{notice}</Alert>}

      <div className="grid gap-4 lg:grid-cols-12">
        {/* Kolom Kiri: Tree Navigation Hierarchy ala Notion */}
        <div className="lg:col-span-4">
          <Card className="h-full">
            <CardHeader className="pb-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <FolderTree className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
                  <CardTitle className="text-sm">Struktur Halaman</CardTitle>
                </div>
                <Badge variant={editor.is_published ? 'outline' : 'default'}>
                  {editor.is_published ? 'Terbit' : 'Draf'}
                </Badge>
              </div>
              <CardDescription className="text-xs">
                {data?.updated_at
                  ? `Diperbarui ${formatDateTime(data.updated_at)}`
                  : 'Belum disimpan'}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-1 p-2">
              {/* Node Utama: Metadata Halaman */}
              <button
                type="button"
                onClick={() => setSelectedNode({ type: 'page' })}
                className={cn(
                  'flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-[13px] transition-colors',
                  selectedNode.type === 'page'
                    ? 'bg-muted font-semibold text-foreground shadow-sm'
                    : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground',
                )}
              >
                <LayoutTemplate className="h-4 w-4 shrink-0 text-muted-foreground" />
                <span className="truncate">Informasi & Header Halaman</span>
              </button>

              {/* Header Bagian */}
              <div className="pt-3 pb-1">
                <p className="px-3 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                  Daftar Bagian ({editor.sections.length})
                </p>
              </div>

              {/* Node-node Bagian */}
              <div className="space-y-1">
                {editor.sections.map((section, index) => {
                  const isSelected =
                    selectedNode.type === 'section' && selectedNode.index === index
                  const isExpanded = expandedSections[index] ?? true

                  return (
                    <div key={`${section.key}-${index}`} className="group flex flex-col">
                      <div
                        className={cn(
                          'flex items-center justify-between rounded-lg px-2 py-1.5 transition-colors',
                          isSelected
                            ? 'bg-muted font-medium text-foreground'
                            : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground',
                        )}
                      >
                        <div
                          className="flex min-w-0 flex-1 cursor-pointer items-center gap-1.5"
                          onClick={() => setSelectedNode({ type: 'section', index })}
                        >
                          {section.items.length > 0 ? (
                            <button
                              type="button"
                              onClick={(e) => {
                                e.stopPropagation()
                                toggleExpand(index)
                              }}
                              className="rounded p-0.5 hover:bg-background"
                            >
                              {isExpanded ? (
                                <ChevronDown className="h-3 w-3" />
                              ) : (
                                <ChevronRight className="h-3 w-3" />
                              )}
                            </button>
                          ) : (
                            <Layers className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                          )}
                          <span className="truncate text-[13px]">
                            {section.title || section.key}
                          </span>
                        </div>

                        <div className="flex shrink-0 items-center gap-1">
                          <Badge variant="outline" className="text-[10px] py-0 px-1.5">
                            {section.kind}
                          </Badge>
                          {!section.is_published && (
                            <EyeOff className="h-3 w-3 text-muted-foreground" aria-hidden="true" />
                          )}
                        </div>
                      </div>

                      {/* Sub-node Butir Items */}
                      {isExpanded && section.items.length > 0 && (
                        <div className="mt-0.5 ml-5 space-y-0.5 border-l border-border pl-2">
                          {section.items.map((item, itemIndex) => (
                            <div
                              key={`${item.label}-${itemIndex}`}
                              onClick={() => setSelectedNode({ type: 'section', index })}
                              className="cursor-pointer truncate rounded px-2 py-1 text-[11px] text-muted-foreground hover:bg-muted/40 hover:text-foreground"
                            >
                              • {item.label || `Butir ${itemIndex + 1}`}
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  )
                })}
              </div>

              <div className="pt-3">
                <Button size="sm" variant="outline" className="w-full text-xs" onClick={addSection}>
                  <Plus className="mr-1.5 h-3.5 w-3.5" />
                  Tambah Bagian
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Kolom Kanan: Focused Editor Pane */}
        <div className="lg:col-span-8">
          <Card>
            <CardHeader className="border-b border-border pb-4">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <CardTitle className="text-sm">
                    {selectedNode.type === 'page'
                      ? `Informasi Utama Halaman ${slug}`
                      : `Sunting Bagian: ${activeSection?.title || activeSection?.key || ''}`}
                  </CardTitle>
                  <CardDescription className="text-xs">
                    {selectedNode.type === 'page'
                      ? 'Atur judul, pengantar, dan optimasi mesin pencari (SEO) halaman ini.'
                      : `Konfigurasi konten bagian jenis ${activeSection ? KIND_LABEL[activeSection.kind] : ''}.`}
                  </CardDescription>
                </div>

                <div className="flex items-center gap-2">
                  {selectedNode.type === 'page' ? (
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => patch({ is_published: !editor.is_published })}
                    >
                      {editor.is_published ? 'Jadikan draf' : 'Terbitkan halaman'}
                    </Button>
                  ) : activeSection ? (
                    <div className="flex items-center gap-1">
                      <Button
                        size="sm"
                        variant="ghost"
                        disabled={selectedNode.index === 0}
                        onClick={() => moveSection(selectedNode.index, -1)}
                        title="Pindah ke atas"
                      >
                        <ChevronUp className="h-4 w-4" />
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        disabled={selectedNode.index === editor.sections.length - 1}
                        onClick={() => moveSection(selectedNode.index, 1)}
                        title="Pindah ke bawah"
                      >
                        <ChevronDown className="h-4 w-4" />
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() =>
                          patchSection(selectedNode.index, {
                            is_published: !activeSection.is_published,
                          })
                        }
                      >
                        {activeSection.is_published ? (
                          <Eye className="h-4 w-4 text-foreground" />
                        ) : (
                          <EyeOff className="h-4 w-4 text-muted-foreground" />
                        )}
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        className="text-destructive hover:bg-destructive/10"
                        onClick={() => removeSection(selectedNode.index)}
                        title="Hapus bagian"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  ) : null}
                </div>
              </div>
            </CardHeader>

            <CardContent className="pt-6">
              {/* Form Editor Halaman (Root) */}
              {selectedNode.type === 'page' && (
                <div className="space-y-4">
                  <div className="grid gap-3 sm:grid-cols-2">
                    <Field
                      label="Label di atas judul (Eyebrow)"
                      value={editor.eyebrow}
                      onChange={(value) => patch({ eyebrow: value })}
                    />
                    <Field
                      label="Judul halaman"
                      value={editor.title}
                      onChange={(value) => patch({ title: value })}
                    />
                  </div>
                  <TextArea
                    label="Paragraf pembuka"
                    rows={3}
                    value={editor.lead}
                    onChange={(value) => patch({ lead: value })}
                  />

                  <div className="border-t border-border pt-4">
                    <p className="text-[13px] font-semibold">Pengaturan Mesin Pencari (SEO)</p>
                    <div className="mt-3 grid gap-3 sm:grid-cols-2">
                      <Field
                        label="Judul mesin pencari (Meta Title)"
                        value={editor.meta_title}
                        onChange={(value) => patch({ meta_title: value })}
                      />
                      <Field
                        label="Deskripsi mesin pencari (Meta Description)"
                        value={editor.meta_description}
                        onChange={(value) => patch({ meta_description: value })}
                      />
                    </div>
                  </div>
                </div>
              )}

              {/* Form Editor Section Terfokus */}
              {selectedNode.type === 'section' && activeSection && (
                <SectionFocusedEditor
                  section={activeSection}
                  index={selectedNode.index}
                  onPatch={(changes) => patchSection(selectedNode.index, changes)}
                  onAddItem={() => addItem(selectedNode.index)}
                  onRemoveItem={(itemIndex) => removeItem(selectedNode.index, itemIndex)}
                  onPatchItem={(itemIndex, changes) =>
                    patchItem(selectedNode.index, itemIndex, changes)
                  }
                />
              )}
            </CardContent>
          </Card>

          {/* Action Bar Simpan Perubahan */}
          <div className="mt-4 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border bg-card p-4 shadow-sm">
            <div className="flex items-center gap-2">
              <Button size="sm" disabled={save.isPending} onClick={() => save.mutate(editor)}>
                {save.isPending ? 'Menyimpan…' : 'Simpan Halaman'}
              </Button>
              <Button
                size="sm"
                variant="ghost"
                onClick={() => data && setEditor(draftFromPage(data))}
              >
                Batalkan Perubahan
              </Button>
            </div>
            <span className="text-xs text-muted-foreground">
              Perubahan langsung tampil di situs publik setelah disimpan.
            </span>
          </div>
        </div>
      </div>
    </div>
  )
}

/** SectionFocusedEditor menampilkan formulir untuk satu bagian yang aktif dipilih. */
function SectionFocusedEditor({
  section,
  index,
  onPatch,
  onAddItem,
  onRemoveItem,
  onPatchItem,
}: {
  section: PageSection
  index: number
  onPatch: (changes: Partial<PageSection>) => void
  onAddItem: () => void
  onRemoveItem: (itemIndex: number) => void
  onPatchItem: (itemIndex: number, changes: Partial<PageItem>) => void
}) {
  const needsItems = section.kind !== 'prose' && section.kind !== 'callout'
  const needsBody = section.kind === 'prose' || section.kind === 'callout'
  const itemLabel = section.kind === 'faq' ? 'Pertanyaan' : 'Judul butir'

  return (
    <div className="space-y-4">
      <div className="grid gap-3 sm:grid-cols-2">
        <Field
          label="Kunci bagian (anchor url, mis: fitur-utama)"
          value={section.key}
          onChange={(value) => onPatch({ key: value })}
        />
        <div>
          <Label htmlFor={`kind-${section.key}-${index}`}>Jenis tampilan bagian</Label>
          <select
            id={`kind-${section.key}-${index}`}
            value={section.kind}
            onChange={(event) => onPatch({ kind: event.target.value as PageSectionKind })}
            className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
          >
            {KINDS.map((kind) => (
              <option key={kind} value={kind}>
                {KIND_LABEL[kind]}
              </option>
            ))}
          </select>
        </div>
        <Field
          label="Judul bagian"
          value={section.title}
          onChange={(value) => onPatch({ title: value })}
        />
        <Field
          label="Label kecil (Eyebrow)"
          value={section.eyebrow}
          onChange={(value) => onPatch({ eyebrow: value })}
        />
      </div>

      <TextArea
        label="Pengantar bagian (Lead)"
        rows={2}
        value={section.lead}
        onChange={(value) => onPatch({ lead: value })}
      />

      {needsBody && (
        <TextArea
          label="Isi konten (markdown: **tebal**, - butir daftar, ## subjudul)"
          rows={7}
          value={section.body}
          onChange={(value) => onPatch({ body: value })}
        />
      )}

      {needsItems && (
        <div className="border-t border-border pt-4">
          <div className="flex items-center justify-between pb-3">
            <div>
              <p className="text-[13px] font-semibold">Daftar Butir Konten ({section.items.length})</p>
              <p className="text-xs text-muted-foreground">
                Elemen-elemen kartu, tabel, atau pertanyaan FAQ di dalam bagian ini.
              </p>
            </div>
            <Button size="sm" variant="outline" onClick={onAddItem}>
              <Plus className="mr-1.5 h-3.5 w-3.5" aria-hidden="true" />
              Tambah Butir
            </Button>
          </div>

          {section.items.length === 0 && (
            <p className="rounded-lg border border-dashed border-border p-4 text-center text-xs text-muted-foreground">
              Belum ada butir. Klik &quot;Tambah Butir&quot; untuk menambahkan elemen kartu atau tanya-jawab.
            </p>
          )}

          <div className="space-y-3">
            {section.items.map((item, itemIndex) => (
              <div key={`${item.label}-${itemIndex}`} className="rounded-lg border border-border p-3.5">
                <div className="flex items-center justify-between pb-2">
                  <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                    {itemLabel} {itemIndex + 1}
                  </span>
                  <Button
                    size="sm"
                    variant="ghost"
                    className="h-7 text-destructive hover:bg-destructive/10"
                    onClick={() => onRemoveItem(itemIndex)}
                  >
                    <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                  </Button>
                </div>
                <div className="grid gap-2 sm:grid-cols-2">
                  <Field
                    label={itemLabel}
                    value={item.label}
                    onChange={(value) => onPatchItem(itemIndex, { label: value })}
                  />
                  <Field
                    label={section.kind === 'faq' ? 'Jawaban' : 'Isi butir'}
                    value={item.value}
                    onChange={(value) => onPatchItem(itemIndex, { value })}
                  />
                  <Field
                    label="Catatan / Batasan (opsional)"
                    value={item.note}
                    onChange={(value) => onPatchItem(itemIndex, { note: value })}
                  />
                  <div className="grid grid-cols-2 gap-2">
                    <Field
                      label="Ikon"
                      value={item.icon}
                      onChange={(value) => onPatchItem(itemIndex, { icon: value })}
                    />
                    <Field
                      label="Tautan (href)"
                      value={item.href}
                      onChange={(value) => onPatchItem(itemIndex, { href: value })}
                    />
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

function Field({
  label,
  value,
  onChange,
}: {
  label: string
  value: string
  onChange: (value: string) => void
}) {
  const id = useId()
  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={id}>{label}</Label>
      <Input id={id} value={value} onChange={(event) => onChange(event.target.value)} />
    </div>
  )
}

function TextArea({
  label,
  value,
  rows,
  onChange,
}: {
  label: string
  value: string
  rows: number
  onChange: (value: string) => void
}) {
  const id = useId()
  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={id}>{label}</Label>
      <textarea
        id={id}
        rows={rows}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm leading-relaxed outline-none focus:ring-1 focus:ring-ring"
      />
    </div>
  )
}
