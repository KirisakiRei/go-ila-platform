import { useId, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ChevronDown, ChevronUp, Plus, Trash2 } from 'lucide-react'
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
 * AdminPageContentPage mengelola isi halaman publik.
 *
 * Halaman diubah sebagai satu kesatuan: panel selalu mengirim keadaan akhir,
 * sehingga urutan dan penghapusan bagian yang terlihat di layar ini persis
 * dengan yang dirender situs publik.
 */
export default function AdminPageContentPage() {
  const [slug, setSlug] = useState(MANAGED_PAGES[0]?.slug ?? 'fitur')
  const meta = MANAGED_PAGES.find((entry) => entry.slug === slug) ?? MANAGED_PAGES[0]

  return (
    <>
      <PageHeader
        title="Konten Halaman"
        description="Menyunting isi halaman publik tanpa deploy ulang. Perubahan langsung tampil setelah disimpan."
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

/** PageEditor menyunting satu halaman; dirender ulang setiap ganti halaman. */
function PageEditor({ slug }: { slug: string }) {
  const queryClient = useQueryClient()
  const [notice, setNotice] = useState<string | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'cms', 'page', slug],
    queryFn: () => adminCmsApi.page(slug),
  })

  // Draft dibangun saat data tiba dan direset setiap ganti halaman. Editor
  // dirender ulang (lihat `key` di pemanggil) sehingga `stamp` selalu berganti
  // bersama `slug`; pola ini menghindari setState di dalam effect.
  const [draft, setDraft] = useState<{ slug: string; value: PageContentPayload | null }>({
    slug,
    value: null,
  })

  if (data && draft.slug !== slug) {
    setDraft({ slug, value: draftFromPage(data) })
  }

  // Data yang baru dimuat mengisi draft hanya bila belum ada isi tersunting.
  if (data && draft.slug === slug && draft.value === null) {
    setDraft({ slug, value: draftFromPage(data) })
  }

  const save = useMutation({
    mutationFn: (payload: PageContentPayload) => adminCmsApi.savePage(slug, payload),
    onSuccess: (saved) => {
      setNotice('Halaman disimpan dan langsung tampil di situs publik.')
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
      <Card>
        <CardContent className="pt-6">
          <Skeleton className="h-40 w-full" />
        </CardContent>
      </Card>
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
    patch({
      sections: [
        ...editor.sections,
        {
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
        },
      ],
    })
  }

  const removeSection = (index: number) => {
    patch({ sections: editor.sections.filter((_, current) => current !== index) })
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
    // Urutan tampil mengikuti posisi di layar, bukan nomor yang diketik.
    patch({
      sections: sections.map((section, position) => ({ ...section, sort_order: position + 1 })),
    })
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
  }

  const removeItem = (sectionIndex: number, itemIndex: number) => {
    const section = editor.sections[sectionIndex]
    if (!section) return
    patchSection(sectionIndex, {
      items: section.items.filter((_, current) => current !== itemIndex),
    })
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <CardTitle className="text-sm">Halaman {slug}</CardTitle>
            <CardDescription className="text-xs">
              {data?.updated_at
                ? `Terakhir diubah ${formatDateTime(data.updated_at)}`
                : 'Belum pernah disimpan'}
            </CardDescription>
          </div>
          <div className="flex items-center gap-2">
            <Badge variant={editor.is_published ? 'outline' : 'default'}>
              {editor.is_published ? 'Terbit' : 'Draf'}
            </Badge>
            <Button
              size="sm"
              variant="outline"
              onClick={() => patch({ is_published: !editor.is_published })}
            >
              {editor.is_published ? 'Jadikan draf' : 'Terbitkan'}
            </Button>
          </div>
        </div>
      </CardHeader>

      <CardContent className="space-y-6">
        {notice && <Alert>{notice}</Alert>}

        <div className="grid gap-3 sm:grid-cols-2">
          <Field
            label="Label di atas judul"
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
        <div className="grid gap-3 sm:grid-cols-2">
          <Field
            label="Judul mesin pencari"
            value={editor.meta_title}
            onChange={(value) => patch({ meta_title: value })}
          />
          <Field
            label="Deskripsi mesin pencari"
            value={editor.meta_description}
            onChange={(value) => patch({ meta_description: value })}
          />
        </div>

        <div className="space-y-4 border-t border-border pt-4">
          <div className="flex items-center justify-between">
            <p className="text-[13px] font-medium">Bagian halaman</p>
            <Button size="sm" variant="outline" onClick={addSection}>
              <Plus className="mr-1 h-3.5 w-3.5" aria-hidden="true" />
              Tambah bagian
            </Button>
          </div>

          {editor.sections.length === 0 && (
            <p className="rounded-md border border-dashed border-border p-4 text-[13px] text-muted-foreground">
              Halaman ini belum punya bagian. Tambahkan bagian agar isinya tampil.
            </p>
          )}

          {editor.sections.map((section, index) => (
            <SectionEditor
              key={`${section.key}-${index}`}
              section={section}
              index={index}
              total={editor.sections.length}
              onPatch={(changes) => patchSection(index, changes)}
              onRemove={() => removeSection(index)}
              onMove={(direction) => moveSection(index, direction)}
              onAddItem={() => addItem(index)}
              onRemoveItem={(itemIndex) => removeItem(index, itemIndex)}
              onPatchItem={(itemIndex, changes) => patchItem(index, itemIndex, changes)}
            />
          ))}
        </div>

        <div className="flex flex-wrap items-center gap-2 border-t border-border pt-4">
          <Button size="sm" disabled={save.isPending} onClick={() => save.mutate(editor)}>
            {save.isPending ? 'Menyimpan…' : 'Simpan halaman'}
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => data && setEditor(draftFromPage(data))}
          >
            Batalkan perubahan
          </Button>
          <span className="text-xs text-muted-foreground">
            Perubahan baru tampil di situs setelah disimpan.
          </span>
        </div>
      </CardContent>
    </Card>
  )
}

function SectionEditor({
  section,
  index,
  total,
  onPatch,
  onRemove,
  onMove,
  onAddItem,
  onRemoveItem,
  onPatchItem,
}: {
  section: PageSection
  index: number
  total: number
  onPatch: (changes: Partial<PageSection>) => void
  onRemove: () => void
  onMove: (direction: -1 | 1) => void
  onAddItem: () => void
  onRemoveItem: (itemIndex: number) => void
  onPatchItem: (itemIndex: number, changes: Partial<PageItem>) => void
}) {
  const needsItems = section.kind !== 'prose' && section.kind !== 'callout'
  const needsBody = section.kind === 'prose' || section.kind === 'callout'
  const itemLabel = section.kind === 'faq' ? 'Pertanyaan' : 'Judul butir'

  return (
    <div className="rounded-lg border border-border p-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Badge variant="outline">{String(index + 1).padStart(2, '0')}</Badge>
          <span className="text-[13px] font-medium">{section.title || section.key}</span>
          <span className="text-[11px] uppercase tracking-wider text-muted-foreground">
            {KIND_LABEL[section.kind]}
          </span>
          {!section.is_published && <Badge variant="default">Disembunyikan</Badge>}
        </div>
        <div className="flex items-center gap-1">
          <Button size="sm" variant="ghost" disabled={index === 0} onClick={() => onMove(-1)}>
            <ChevronUp className="h-3.5 w-3.5" aria-hidden="true" />
          </Button>
          <Button
            size="sm"
            variant="ghost"
            disabled={index === total - 1}
            onClick={() => onMove(1)}
          >
            <ChevronDown className="h-3.5 w-3.5" aria-hidden="true" />
          </Button>
          <Button
            size="sm"
            variant="ghost"
            onClick={() => onPatch({ is_published: !section.is_published })}
          >
            {section.is_published ? 'Sembunyikan' : 'Tampilkan'}
          </Button>
          <Button size="sm" variant="ghost" onClick={onRemove}>
            <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
          </Button>
        </div>
      </div>

      <div className="mt-3 grid gap-3 sm:grid-cols-2">
        <Field
          label="Kunci bagian (anchor)"
          value={section.key}
          onChange={(value) => onPatch({ key: value })}
        />
        <div>
          <Label htmlFor={`kind-${section.key}-${index}`}>Jenis bagian</Label>
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
          label="Label kecil"
          value={section.eyebrow}
          onChange={(value) => onPatch({ eyebrow: value })}
        />
      </div>

      <div className="mt-3">
        <TextArea
          label="Pengantar bagian"
          rows={2}
          value={section.lead}
          onChange={(value) => onPatch({ lead: value })}
        />
      </div>

      {needsBody && (
        <div className="mt-3">
          <TextArea
            label="Isi (markdown: **tebal**, - daftar, ## subjudul)"
            rows={6}
            value={section.body}
            onChange={(value) => onPatch({ body: value })}
          />
        </div>
      )}

      {needsItems && (
        <div className="mt-4 space-y-2">
          <div className="flex items-center justify-between">
            <p className="text-[12px] font-medium text-muted-foreground">
              Butir ({section.items.length})
            </p>
            <Button size="sm" variant="outline" onClick={onAddItem}>
              <Plus className="mr-1 h-3.5 w-3.5" aria-hidden="true" />
              Tambah butir
            </Button>
          </div>

          {section.items.map((item, itemIndex) => (
            <div key={`${item.label}-${itemIndex}`} className="rounded-md border border-border p-3">
              <div className="flex items-start justify-between gap-2">
                <p className="text-[11px] uppercase tracking-wider text-muted-foreground">
                  {itemLabel} {itemIndex + 1}
                </p>
                <Button size="sm" variant="ghost" onClick={() => onRemoveItem(itemIndex)}>
                  <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                </Button>
              </div>
              <div className="mt-2 grid gap-2 sm:grid-cols-2">
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
                  label="Batasan / catatan (opsional)"
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
                    label="Tautan"
                    value={item.href}
                    onChange={(value) => onPatchItem(itemIndex, { href: value })}
                  />
                </div>
              </div>
            </div>
          ))}
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
    <div>
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="mt-1"
      />
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
    <div>
      <Label htmlFor={id}>{label}</Label>
      <textarea
        id={id}
        value={value}
        rows={rows}
        onChange={(event) => onChange(event.target.value)}
        className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
      />
    </div>
  )
}
