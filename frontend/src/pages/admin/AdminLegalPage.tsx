import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
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
  PageHeader,
  Skeleton,
} from '@/components/ui'
import { request } from '@/lib/http'
import { formatDateTime } from '@/lib/format'

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

const SLUGS: { slug: string; label: string }[] = [
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
 * Dokumen yang sudah terbit tidak pernah diubah: perubahan selalu menjadi
 * versi baru yang menerbitkan tanggal berlaku sendiri. Dengan begitu
 * persetujuan yang pernah diberikan pengguna tetap dapat ditelusuri ke versi
 * yang benar-benar mereka baca.
 */
export default function AdminLegalPage() {
  const [slug, setSlug] = useState(SLUGS[0]?.slug ?? 'tos')

  return (
    <>
      <PageHeader
        title="Dokumen Legal"
        description="Menyusun draf versi baru dan menerbitkannya. Versi terbit tidak diubah; perubahan selalu menjadi versi baru."
      />
      <div className="grid gap-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Pilih dokumen</CardTitle>
            <CardDescription className="text-xs">
              Draf yang diterbitkan langsung tampil di situs publik setelah disetujui pengguna bila
              ditandai wajib disetujui ulang.
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

function LegalEditor({ slug }: { slug: string }) {
  const queryClient = useQueryClient()
  const [notice, setNotice] = useState<string | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'legal', slug],
    queryFn: () => request<LegalVersion[]>(`${ADMIN}/legal`, { query: { slug } }),
  })

  const versions = data ?? []
  const published = versions.filter((item) => item.published)
  const latest = published[published.length - 1]
  const draft = versions.find((item) => !item.published)

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
          title: latest?.title ?? '',
          body_md: latest?.body_md ?? '',
          require_reaccept: false,
        },
      }),
    onSuccess: () => {
      setNotice('Draf versi baru dibuat dari versi terbit terakhir. Sunting isinya lalu terbitkan.')
      invalidate()
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal membuat draf.'),
  })

  const publish = useMutation({
    mutationFn: (id: string) =>
      request<LegalVersion>(`${ADMIN}/legal/${id}/publish`, {
        method: 'POST',
        body: { reason: 'Pembaruan naskah agar terbaca publik' },
      }),
    onSuccess: () => {
      setNotice('Versi baru diterbitkan dan sekarang tampil di situs publik.')
      invalidate()
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal menerbitkan.'),
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Riwayat versi</CardTitle>
        <CardDescription className="text-xs">
          Versi terbit dipertahankan sebagai arsip sehingga persetujuan lama tetap dapat ditelusuri.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {notice && <Alert>{notice}</Alert>}
        {isLoading && <Skeleton className="h-24 w-full" />}

        {!isLoading && versions.length === 0 && (
          <EmptyState
            title="Belum ada dokumen"
            description="Dokumen ini belum pernah diterbitkan."
          />
        )}

        {versions.map((item) => (
          <div
            key={`${item.slug}-${item.version}`}
            className="border-b border-border pb-3 last:border-0"
          >
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <span className="text-[13px] font-medium">Versi {item.version}</span>
                <Badge variant={item.published ? 'outline' : 'default'}>
                  {item.published ? 'Terbit' : 'Draf'}
                </Badge>
                {item.require_reaccept && <Badge variant="solid">Setujui ulang</Badge>}
              </div>
              <span className="text-[11px] text-muted-foreground">
                {item.published_at
                  ? `Terbit ${formatDateTime(item.published_at)}`
                  : 'Belum diterbitkan'}
              </span>
            </div>
            <p className="mt-1 text-[11px] text-muted-foreground">
              Berlaku sejak {formatDateTime(item.effective_at ?? '')}
            </p>
            {!item.published && (
              <Button
                size="sm"
                className="mt-2"
                disabled={publish.isPending}
                onClick={() => item.id && publish.mutate(item.id)}
              >
                Terbitkan versi {item.version}
              </Button>
            )}
          </div>
        ))}

        <div className="space-y-2 border-t border-border pt-4">
          <Label>Versi baru</Label>
          <p className="text-xs text-muted-foreground">
            Draf dibuat dari salinan versi terbit terakhir, sehingga Anda hanya perlu menyunting
            bagian yang berubah.
          </p>
          <Button
            size="sm"
            variant="outline"
            disabled={createDraft.isPending || !latest || Boolean(draft)}
            onClick={() => createDraft.mutate()}
          >
            {draft ? 'Sudah ada draf — terbitkan dulu' : 'Buat draf versi baru'}
          </Button>
        </div>

        {draft && <DraftEditor draft={draft} onSaved={invalidate} />}
      </CardContent>
    </Card>
  )
}

/** DraftEditor menyunting isi draf; versi terbit tidak bisa disunting. */
function DraftEditor({ draft, onSaved }: { draft: LegalVersion; onSaved: () => void }) {
  const [body, setBody] = useState(draft.body_md ?? '')
  const [title, setTitle] = useState(draft.title)
  const [notice, setNotice] = useState<string | null>(null)

  const save = useMutation({
    mutationFn: () =>
      request<LegalVersion>(`${ADMIN}/legal/${draft.id}`, {
        method: 'PATCH',
        body: { slug: draft.slug, title, body_md: body, require_reaccept: draft.require_reaccept },
      }),
    onSuccess: () => {
      setNotice('Draf tersimpan.')
      onSaved()
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal menyimpan draf.'),
  })

  return (
    <div className="mt-4 rounded-lg border border-border p-4">
      <div className="flex items-center justify-between">
        <p className="text-[13px] font-medium">Sunting draf versi {draft.version}</p>
        <span className="text-[11px] text-muted-foreground">{body.length} karakter</span>
      </div>
      {notice && <Alert className="mt-2">{notice}</Alert>}
      <div className="mt-3">
        <Label htmlFor="legal-draft-title">Judul dokumen</Label>
        <Input
          id="legal-draft-title"
          value={title}
          onChange={(event) => setTitle(event.target.value)}
          className="mt-1"
        />
      </div>
      <div className="mt-3">
        <Label htmlFor="legal-draft-body">Isi (markdown)</Label>
        <textarea
          id="legal-draft-body"
          value={body}
          rows={18}
          onChange={(event) => setBody(event.target.value)}
          className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 font-mono text-xs"
        />
      </div>
      <Button
        size="sm"
        className="mt-3"
        disabled={save.isPending || !title.trim() || body.trim().length < 20}
        onClick={() => save.mutate()}
      >
        {save.isPending ? 'Menyimpan…' : 'Simpan draf'}
      </Button>
    </div>
  )
}
