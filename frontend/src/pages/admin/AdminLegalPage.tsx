import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ArrowDown, ArrowUp, BookOpen, Edit3, Eye, Plus, Sparkles, Trash2 } from 'lucide-react'
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
  Skeleton,
} from '@/components/ui'
import { request } from '@/lib/http'
import { formatDateTime } from '@/lib/format'
import { LEGAL_DOCUMENTS, type LegalSlug } from '@/pages/public/legal/documents'
import { MarkdownView } from '@/pages/public/legal/MarkdownView'

/** Satu versi dokumen legal, termasuk draf yang belum terbit. */
type LegalVersion = {
  id?: string
  slug: string
  version: number
  title: string
  body_md?: string
  require_reaccept: boolean
  published: boolean
  effective_at?: string
  published_at?: string
}

/** Satu bagian (pasal/klausul) pada editor terstruktur. */
type LegalSection = {
  id: string
  heading: string
  content: string
}

const SLUGS: { slug: LegalSlug; label: string }[] = [
  { slug: 'tos', label: 'Syarat & Ketentuan' },
  { slug: 'privacy', label: 'Kebijakan Privasi' },
  { slug: 'risk', label: 'Disclaimer Risiko' },
  { slug: 'refund', label: 'Pengembalian Dana' },
  { slug: 'fair_use', label: 'Penggunaan Wajar' },
]

const ADMIN = '/admin'

let sectionSeq = 0
function nextSectionId(): string {
  sectionSeq += 1
  return `legal-bagian-${sectionSeq}`
}

/**
 * parseSections memecah markdown menjadi bagian berdasarkan judul `##`.
 *
 * Teks sebelum judul pertama dipertahankan sebagai bagian pembuka (tanpa judul),
 * sehingga naskah lama yang diawali paragraf tetap utuh saat disunting.
 */
function parseSections(markdown: string): LegalSection[] {
  const lines = markdown.replace(/\r\n/g, '\n').split('\n')
  const sections: LegalSection[] = []

  let heading = ''
  let buffer: string[] = []

  const flush = () => {
    const content = buffer.join('\n').trim()
    if (heading || content) {
      sections.push({ id: nextSectionId(), heading, content })
    }
    heading = ''
    buffer = []
  }

  for (const line of lines) {
    const match = /^##\s+(.+?)\s*$/.exec(line)
    if (match) {
      flush()
      heading = match[1] ?? ''
      continue
    }
    buffer.push(line)
  }
  flush()

  return sections
}

/**
 * serializeSections mengembalikan bagian terstruktur ke markdown (`##`).
 *
 * Penyimpanan tetap markdown agar kontrak server, riwayat versi, dan pencatatan
 * persetujuan tidak berubah; editor hanya memberi bentuk yang lebih terarah.
 */
function serializeSections(sections: LegalSection[]): string {
  const blocks = sections
    .map((section) => {
      const heading = section.heading.trim()
      const content = section.content.trim()
      if (!heading) return content
      return content ? `## ${heading}\n\n${content}` : `## ${heading}`
    })
    .filter((block) => block.trim() !== '')

  return blocks.length > 0 ? `${blocks.join('\n\n')}\n` : ''
}

/** toDateInputValue mengubah RFC3339 dari server menjadi nilai <input type="date">. */
function toDateInputValue(value?: string): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toISOString().slice(0, 10)
}

/** fromDateInputValue mengubah tanggal lokal menjadi RFC3339 UTC (awal hari). */
function fromDateInputValue(value: string): string | undefined {
  if (!value) return undefined
  const date = new Date(`${value}T00:00:00Z`)
  if (Number.isNaN(date.getTime())) return undefined
  return date.toISOString()
}

/**
 * AdminLegalPage mengelola versi dokumen legal.
 *
 * Menampilkan pratinjau naskah terbit aktif, riwayat versi, serta penyuntingan
 * draf terstruktur per bagian (judul pasal + isi) dengan pratinjau langsung.
 */
export default function AdminLegalPage() {
  const [slug, setSlug] = useState<LegalSlug>(SLUGS[0]?.slug ?? 'tos')

  return (
    <>
      <PageHeader
        title="Dokumen Legal"
        description="Kelola draf versi baru dan terbitkan. Dokumen terbit tidak diubah; perubahan selalu menjadi versi baru demi kepatuhan hukum."
      />
      <div className="grid gap-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Pilih jenis dokumen</CardTitle>
            <CardDescription className="text-xs">
              Pilih dokumen legal yang ingin diperiksa pratinjaunya atau dibuatkan draf versi barunya.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex flex-wrap gap-2">
              {SLUGS.map((entry) => (
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

        <LegalEditor key={slug} slug={slug} />
      </div>
    </>
  )
}

function LegalEditor({ slug }: { slug: LegalSlug }) {
  const queryClient = useQueryClient()
  const [notice, setNotice] = useState<string | null>(null)
  const [modalOpen, setModalOpen] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'legal', slug],
    queryFn: () => request<LegalVersion[]>(`${ADMIN}/legal`, { query: { slug } }),
  })

  const fallback = LEGAL_DOCUMENTS[slug]
  const versions = data ?? []
  const published = versions.filter((item) => item.published)
  const latest = published[published.length - 1]
  const draft = versions.find((item) => !item.published)

  const activeBody = latest?.body_md?.trim() ? latest.body_md : fallback.body
  const activeTitle = latest?.title || fallback.title

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['admin', 'legal', slug] })
    void queryClient.invalidateQueries({ queryKey: ['legal'] })
  }

  const createDraft = useMutation({
    mutationFn: () =>
      request<LegalVersion>(`${ADMIN}/legal`, {
        method: 'POST',
        body: {
          slug,
          title: latest?.title ?? fallback.title,
          body_md: latest?.body_md?.trim() ? latest.body_md : fallback.body,
          require_reaccept: false,
        },
      }),
    onSuccess: () => {
      setNotice('Draf versi baru dibuat. Silakan sunting isinya di modal lalu terbitkan.')
      setModalOpen(true)
      invalidate()
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal membuat draf.'),
  })

  const publish = useMutation({
    mutationFn: (id: string) =>
      request<LegalVersion>(`${ADMIN}/legal/${id}/publish`, {
        method: 'POST',
        body: { reason: 'Pembaruan naskah resmi disetujui admin' },
      }),
    onSuccess: () => {
      setNotice('Versi baru berhasil diterbitkan dan langsung tayang di situs publik.')
      setModalOpen(false)
      invalidate()
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal menerbitkan.'),
  })

  return (
    <div className="grid gap-4">
      {notice && <Alert>{notice}</Alert>}

      {/* Pratinjau Naskah Terbit Aktif */}
      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <BookOpen className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
              <CardTitle className="text-sm">Naskah Terbit Saat Ini (Aktif di Publik)</CardTitle>
            </div>
            <div className="flex items-center gap-2">
              <Badge variant={latest ? 'outline' : 'default'}>
                {latest ? `Versi ${latest.version}` : 'Versi Standar'}
              </Badge>
              {draft ? (
                <Button size="sm" onClick={() => setModalOpen(true)}>
                  <Edit3 className="mr-1.5 h-3.5 w-3.5" />
                  Buka Draf Versi {draft.version}
                </Button>
              ) : (
                <Button
                  size="sm"
                  disabled={createDraft.isPending}
                  onClick={() => createDraft.mutate()}
                >
                  <Edit3 className="mr-1.5 h-3.5 w-3.5" />
                  Buat Draf Versi Baru
                </Button>
              )}
            </div>
          </div>
          <CardDescription className="text-xs">
            {latest?.published_at
              ? `Diterbitkan pada ${formatDateTime(latest.published_at)} (berlaku sejak ${formatDateTime(latest.effective_at ?? '')})`
              : 'Menampilkan naskah standar bawaan sistem.'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <p className="mb-3 text-base font-semibold text-foreground">{activeTitle}</p>
          <div className="max-h-80 overflow-y-auto rounded-lg border border-border bg-white px-5 py-4">
            <MarkdownView source={activeBody} />
          </div>
        </CardContent>
      </Card>

      {/* Riwayat Versi Dokumen */}
      <Card>
        <CardHeader>
          <CardTitle className="text-sm">Riwayat Versi</CardTitle>
          <CardDescription className="text-xs">
            Seluruh versi tersimpan abadi sebagai rekaman legal untuk audit kepatuhan.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {isLoading && <Skeleton className="h-24 w-full" />}

          {!isLoading && versions.length === 0 && (
            <EmptyState
              title="Belum ada versi di basis data"
              description="Naskah saat ini menggunakan template standar bawaan sistem."
            />
          )}

          {versions.map((item) => (
            <div
              key={`${item.slug}-${item.version}`}
              className="flex flex-wrap items-center justify-between gap-2 border-b border-border pb-3 last:border-0"
            >
              <div>
                <div className="flex items-center gap-2">
                  <span className="text-[13px] font-medium">Versi {item.version}</span>
                  <Badge variant={item.published ? 'outline' : 'default'}>
                    {item.published ? 'Terbit' : 'Draf'}
                  </Badge>
                  {item.require_reaccept && <Badge variant="solid">Setujui ulang</Badge>}
                </div>
                <p className="mt-1 text-[11px] text-muted-foreground">
                  {item.published_at
                    ? `Terbit ${formatDateTime(item.published_at)} • Berlaku ${formatDateTime(item.effective_at ?? '')}`
                    : 'Draf belum diterbitkan'}
                </p>
              </div>

              {!item.published && (
                <div className="flex items-center gap-2">
                  <Button size="sm" variant="outline" onClick={() => setModalOpen(true)}>
                    Sunting Draf
                  </Button>
                  <Button
                    size="sm"
                    disabled={publish.isPending}
                    onClick={() => item.id && publish.mutate(item.id)}
                  >
                    Terbitkan Sekarang
                  </Button>
                </div>
              )}
            </div>
          ))}
        </CardContent>
      </Card>

      {/* Modal Penyuntingan Draf Versi Baru */}
      {draft && (
        <Modal
          open={modalOpen}
          onClose={() => setModalOpen(false)}
          title={`Sunting Draf Dokumen: ${draft.title || fallback.title} (Versi ${draft.version})`}
          description="Perubahan di draf tidak akan mempengaruhi tampilan publik sebelum Anda menekan tombol 'Terbitkan'."
          size="2xl"
        >
          <DraftSectionEditor
            draft={draft}
            fallbackBody={fallback.body}
            fallbackTitle={fallback.title}
            onSaved={invalidate}
            onPublish={(id) => publish.mutateAsync(id)}
            isPublishing={publish.isPending}
            onClose={() => setModalOpen(false)}
          />
        </Modal>
      )}
    </div>
  )
}

/**
 * DraftSectionEditor menyunting naskah sebagai daftar bagian, bukan teks
 * markdown mentah. Setiap perubahan dirangkai kembali menjadi `body_md` saat
 * disimpan, sehingga server dan halaman publik tidak perlu tahu bentuk editor.
 */
function DraftSectionEditor({
  draft,
  fallbackBody,
  fallbackTitle,
  onSaved,
  onPublish,
  isPublishing,
  onClose,
}: {
  draft: LegalVersion
  fallbackBody: string
  fallbackTitle: string
  onSaved: () => void
  onPublish: (id: string) => Promise<unknown>
  isPublishing: boolean
  onClose: () => void
}) {
  const sumberAwal = draft.body_md?.trim() ? draft.body_md : fallbackBody

  const [title, setTitle] = useState(draft.title || fallbackTitle)
  const [sections, setSections] = useState<LegalSection[]>(() => {
    const parsed = parseSections(sumberAwal)
    return parsed.length > 0 ? parsed : [{ id: nextSectionId(), heading: '', content: '' }]
  })
  const [requireReaccept, setRequireReaccept] = useState(draft.require_reaccept)
  const [effectiveDate, setEffectiveDate] = useState(toDateInputValue(draft.effective_at))
  const [notice, setNotice] = useState<{ tone: 'info' | 'error'; message: string } | null>(null)
  const [showPreview, setShowPreview] = useState(false)
  const [saving, setSaving] = useState(false)
  const [publishing, setPublishing] = useState(false)

  const markdown = useMemo(() => serializeSections(sections), [sections])

  const ubahBagian = (id: string, patch: Partial<Pick<LegalSection, 'heading' | 'content'>>) => {
    setSections((current) =>
      current.map((section) => (section.id === id ? { ...section, ...patch } : section)),
    )
  }

  const geser = (index: number, arah: -1 | 1) => {
    setSections((current) => {
      const target = index + arah
      if (target < 0 || target >= current.length) return current
      const next = [...current]
      const [item] = next.splice(index, 1)
      if (!item) return current
      next.splice(target, 0, item)
      return next
    })
  }

  const hapusBagian = (id: string) => {
    setSections((current) => {
      const next = current.filter((section) => section.id !== id)
      return next.length > 0 ? next : [{ id: nextSectionId(), heading: '', content: '' }]
    })
  }

  const tambahBagian = () => {
    setSections((current) => [...current, { id: nextSectionId(), heading: '', content: '' }])
  }

  const muatNaskahSistem = () => {
    const parsed = parseSections(fallbackBody)
    setTitle(fallbackTitle)
    setSections(parsed.length > 0 ? parsed : [{ id: nextSectionId(), heading: '', content: '' }])
    setNotice({ tone: 'info', message: 'Naskah standar sistem dimuat ke formulir draf.' })
  }

  const valid = title.trim().length > 0 && markdown.trim().length >= 20

  const simpanDraf = async () => {
    const effectiveAt = fromDateInputValue(effectiveDate)
    const body: Record<string, unknown> = {
      slug: draft.slug,
      title: title.trim(),
      body_md: markdown,
      require_reaccept: requireReaccept,
    }
    // Kosong berarti "ikut waktu draf dibuat"; jangan menimpa dengan string kosong.
    if (effectiveAt) body.effective_at = effectiveAt

    return request<LegalVersion>(`${ADMIN}/legal/${draft.id}`, {
      method: 'PATCH',
      body,
    })
  }

  const handleSimpan = async () => {
    setSaving(true)
    setNotice(null)
    try {
      await simpanDraf()
      setNotice({ tone: 'info', message: 'Draf berhasil disimpan.' })
      onSaved()
    } catch (error) {
      setNotice({
        tone: 'error',
        message: error instanceof Error ? error.message : 'Gagal menyimpan draf.',
      })
    } finally {
      setSaving(false)
    }
  }

  const handleTerbitkan = async () => {
    if (!draft.id) return
    setPublishing(true)
    setNotice(null)
    try {
      // Simpan dulu agar suntingan terakhir tidak hilang saat diterbitkan.
      await simpanDraf()
      await onPublish(draft.id)
      onSaved()
    } catch (error) {
      setNotice({
        tone: 'error',
        message: error instanceof Error ? error.message : 'Gagal menerbitkan dokumen.',
      })
    } finally {
      setPublishing(false)
    }
  }

  return (
    <div className="space-y-5">
      {notice && <Alert variant={notice.tone === 'error' ? 'destructive' : 'default'}>{notice.message}</Alert>}

      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="text-xs text-muted-foreground">
          {sections.length} bagian • {markdown.length} karakter
        </span>
        <div className="flex flex-wrap items-center gap-2">
          <Button
            type="button"
            size="sm"
            variant="outline"
            className="text-xs"
            onClick={() => setShowPreview((value) => !value)}
          >
            <Eye className="mr-1.5 h-3.5 w-3.5" />
            {showPreview ? 'Sembunyikan Pratinjau' : 'Pratinjau Publik'}
          </Button>
          <Button
            type="button"
            size="sm"
            variant="outline"
            className="text-xs"
            onClick={muatNaskahSistem}
          >
            <Sparkles className="mr-1.5 h-3.5 w-3.5 text-yellow-500" />
            Muat Naskah Sistem
          </Button>
        </div>
      </div>

      <div className="grid gap-3 sm:grid-cols-2">
        <div className="space-y-1.5">
          <Label htmlFor="legal-draft-title">Judul Dokumen</Label>
          <Input
            id="legal-draft-title"
            value={title}
            onChange={(event) => setTitle(event.target.value)}
          />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="legal-effective">Berlaku Sejak</Label>
          <Input
            id="legal-effective"
            type="date"
            value={effectiveDate}
            onChange={(event) => setEffectiveDate(event.target.value)}
          />
          <p className="text-[11px] text-muted-foreground">
            Kosongkan bila ingin mengikuti tanggal draf dibuat.
          </p>
        </div>
      </div>

      <div className="flex items-center gap-2">
        <input
          type="checkbox"
          id="require-reaccept"
          checked={requireReaccept}
          onChange={(event) => setRequireReaccept(event.target.checked)}
          className="rounded border-input"
        />
        <Label htmlFor="require-reaccept" className="cursor-pointer text-xs">
          Wajibkan pengguna menyetujui ulang naskah ini setelah diterbitkan
        </Label>
      </div>

      {/* Daftar bagian: judul pasal + isi, dapat digeser dan dihapus. */}
      <div className="space-y-3">
        {sections.map((section, index) => (
          <div key={section.id} className="rounded-xl border border-border bg-muted/20 p-3">
            <div className="mb-2 flex items-center justify-between gap-2">
              <span className="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">
                {section.heading.trim() ? `Bagian ${index + 1}` : index === 0 ? 'Pembuka' : `Bagian ${index + 1}`}
              </span>
              <div className="flex items-center gap-1">
                <Button
                  type="button"
                  size="sm"
                  variant="ghost"
                  disabled={index === 0}
                  onClick={() => geser(index, -1)}
                  aria-label={`Naikkan bagian ${index + 1}`}
                >
                  <ArrowUp className="h-3.5 w-3.5" aria-hidden="true" />
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="ghost"
                  disabled={index === sections.length - 1}
                  onClick={() => geser(index, 1)}
                  aria-label={`Turunkan bagian ${index + 1}`}
                >
                  <ArrowDown className="h-3.5 w-3.5" aria-hidden="true" />
                </Button>
                <Button
                  type="button"
                  size="sm"
                  variant="ghost"
                  className="text-destructive hover:bg-destructive/10"
                  onClick={() => hapusBagian(section.id)}
                  aria-label={`Hapus bagian ${index + 1}`}
                >
                  <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                </Button>
              </div>
            </div>

            <Input
              value={section.heading}
              placeholder="Judul pasal, mis. Ruang Lingkup Layanan"
              onChange={(event) => ubahBagian(section.id, { heading: event.target.value })}
            />

            <textarea
              value={section.content}
              rows={6}
              placeholder="Isi pasal. Baris kosong memisahkan paragraf; awali '-' untuk daftar dan '> [!catatan]' untuk catatan."
              onChange={(event) => ubahBagian(section.id, { content: event.target.value })}
              className="mt-2 w-full rounded-md border border-input bg-background p-3 font-mono text-xs leading-relaxed outline-none focus:ring-1 focus:ring-ring"
            />
          </div>
        ))}

        <Button type="button" size="sm" variant="outline" onClick={tambahBagian}>
          <Plus className="mr-1.5 h-3.5 w-3.5" />
          Tambah Bagian
        </Button>
      </div>

      {showPreview && (
        <div className="space-y-2">
          <p className="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">
            Pratinjau Publik
          </p>
          <div className="max-h-96 overflow-y-auto rounded-xl border border-border bg-white px-5 py-4">
            <MarkdownView source={markdown} />
          </div>
        </div>
      )}

      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-border pt-4">
        <Button variant="ghost" size="sm" type="button" onClick={onClose}>
          Tutup
        </Button>
        <div className="flex items-center gap-2">
          <Button
            size="sm"
            variant="outline"
            disabled={saving || publishing || !valid}
            onClick={handleSimpan}
          >
            {saving ? 'Menyimpan…' : 'Simpan Draf'}
          </Button>
          <Button
            size="sm"
            disabled={saving || publishing || isPublishing || !draft.id || !valid}
            onClick={handleTerbitkan}
          >
            {publishing || isPublishing ? 'Menerbitkan…' : 'Simpan & Terbitkan'}
          </Button>
        </div>
      </div>
    </div>
  )
}
