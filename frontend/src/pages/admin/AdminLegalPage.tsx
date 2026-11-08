import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { BookOpen, Edit3, Sparkles } from 'lucide-react'
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

const SLUGS: { slug: LegalSlug; label: string }[] = [
  { slug: 'tos', label: 'Syarat & Ketentuan' },
  { slug: 'privacy', label: 'Kebijakan Privasi' },
  { slug: 'risk', label: 'Disclaimer Risiko' },
  { slug: 'refund', label: 'Pengembalian Dana' },
  { slug: 'fair_use', label: 'Penggunaan Wajar' },
]

const ADMIN = '/admin'

/**
 * AdminLegalPage mengelola versi dokumen legal.
 *
 * Menampilkan pratinjau naskah terbit aktif, riwayat versi, serta penyuntingan draf
 * dalam bentuk modal terfokus dengan dukungan template standar sistem.
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
          <div className="mb-2">
            <p className="text-base font-semibold text-foreground">{activeTitle}</p>
          </div>
          <div className="max-h-80 overflow-y-auto rounded-lg border border-border bg-muted/20 p-4 font-mono text-xs leading-relaxed whitespace-pre-wrap">
            {activeBody}
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
          <DraftModalForm
            draft={draft}
            fallbackBody={fallback.body}
            fallbackTitle={fallback.title}
            onSaved={invalidate}
            onPublish={(id) => publish.mutate(id)}
            isPublishing={publish.isPending}
            onClose={() => setModalOpen(false)}
          />
        </Modal>
      )}
    </div>
  )
}

function DraftModalForm({
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
  onPublish: (id: string) => void
  isPublishing: boolean
  onClose: () => void
}) {
  const [body, setBody] = useState(draft.body_md?.trim() ? draft.body_md : fallbackBody)
  const [title, setTitle] = useState(draft.title || fallbackTitle)
  const [requireReaccept, setRequireReaccept] = useState(draft.require_reaccept)
  const [notice, setNotice] = useState<string | null>(null)

  const save = useMutation({
    mutationFn: () =>
      request<LegalVersion>(`${ADMIN}/legal/${draft.id}`, {
        method: 'PATCH',
        body: { slug: draft.slug, title, body_md: body, require_reaccept: requireReaccept },
      }),
    onSuccess: () => {
      setNotice('Draf berhasil disimpan.')
      onSaved()
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal menyimpan draf.'),
  })

  const loadStandardTemplate = () => {
    setTitle(fallbackTitle)
    setBody(fallbackBody)
    setNotice('Naskah standar sistem berhasil dimuat ke formulir draf.')
  }

  return (
    <div className="space-y-4">
      {notice && <Alert>{notice}</Alert>}

      <div className="flex items-center justify-between gap-2">
        <span className="text-xs text-muted-foreground">{body.length} karakter</span>
        <Button
          type="button"
          size="sm"
          variant="outline"
          className="text-xs"
          onClick={loadStandardTemplate}
        >
          <Sparkles className="mr-1.5 h-3.5 w-3.5 text-yellow-500" />
          Muat Naskah Bersih Sistem
        </Button>
      </div>

      <div className="space-y-1.5">
        <Label htmlFor="legal-draft-title">Judul Dokumen</Label>
        <Input
          id="legal-draft-title"
          value={title}
          onChange={(event) => setTitle(event.target.value)}
        />
      </div>

      <div className="space-y-1.5">
        <Label htmlFor="legal-draft-body">Isi Naskah (Markdown)</Label>
        <textarea
          id="legal-draft-body"
          value={body}
          rows={16}
          onChange={(event) => setBody(event.target.value)}
          className="w-full rounded-md border border-input bg-background p-3 font-mono text-xs leading-relaxed outline-none focus:ring-1 focus:ring-ring"
        />
      </div>

      <div className="flex items-center gap-2 pt-2">
        <input
          type="checkbox"
          id="require-reaccept"
          checked={requireReaccept}
          onChange={(e) => setRequireReaccept(e.target.checked)}
          className="rounded border-input"
        />
        <Label htmlFor="require-reaccept" className="text-xs cursor-pointer">
          Wajibkan pengguna menyetujui ulang naskah ini setelah diterbitkan
        </Label>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-border pt-4">
        <Button variant="ghost" size="sm" type="button" onClick={onClose}>
          Tutup
        </Button>
        <div className="flex items-center gap-2">
          <Button
            size="sm"
            variant="outline"
            disabled={save.isPending || !title.trim() || body.trim().length < 20}
            onClick={() => save.mutate()}
          >
            {save.isPending ? 'Menyimpan…' : 'Simpan Draf'}
          </Button>
          <Button
            size="sm"
            disabled={isPublishing || !draft.id || !title.trim() || body.trim().length < 20}
            onClick={() => draft.id && onPublish(draft.id)}
          >
            {isPublishing ? 'Menerbitkan…' : 'Terbitkan Sekarang'}
          </Button>
        </div>
      </div>
    </div>
  )
}
