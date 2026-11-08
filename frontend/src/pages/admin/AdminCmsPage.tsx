import { useState } from 'react'
import { Link } from 'react-router-dom'
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
import { adminCmsApi } from '@/lib/system-api'
import { formatDateTime } from '@/lib/format'
import type { AnnouncementItem, FaqItem } from '@/types/system'

/**
 * AdminCmsPage mengelola FAQ dan pengumuman.
 *
 * Konten disimpan di database sehingga bisa diubah tanpa deploy. Pengumuman
 * memiliki jendela waktu, jadi yang kedaluwarsa berhenti tampil sendiri
 * (plans/11 Fase J).
 */
export default function AdminCmsPage() {
  return (
    <>
      <PageHeader
        title="Konten"
        description="FAQ tampil di pusat bantuan; pengumuman tampil di dashboard pelanggan; isi halaman publik disunting di menu Halaman Publik."
      />
      <div className="mb-4">
        <Link
          to="/admin/konten/halaman"
          className="inline-flex items-center gap-1.5 text-sm font-medium underline underline-offset-4"
        >
          Buka pengelolaan isi halaman publik
        </Link>
      </div>
      <div className="grid gap-4">
        <FaqSection />
        <AnnouncementSection />
      </div>
    </>
  )
}

function FaqSection() {
  const queryClient = useQueryClient()
  const [notice, setNotice] = useState<string | null>(null)
  const [draft, setDraft] = useState({ category: '', question: '', answer_md: '' })

  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'cms', 'faq'],
    queryFn: adminCmsApi.faq,
  })

  const invalidate = () => void queryClient.invalidateQueries({ queryKey: ['admin', 'cms', 'faq'] })

  const create = useMutation({
    mutationFn: () => adminCmsApi.createFaq({ ...draft, is_active: true }),
    onSuccess: () => {
      setNotice('FAQ ditambahkan.')
      setDraft({ category: '', question: '', answer_md: '' })
      invalidate()
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal menambah FAQ.'),
  })

  const toggle = useMutation({
    mutationFn: (row: FaqItem) =>
      adminCmsApi.updateFaq(row.id, { ...row, is_active: !row.is_active }),
    onSuccess: invalidate,
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal mengubah FAQ.'),
  })

  const remove = useMutation({
    mutationFn: (id: string) => adminCmsApi.deleteFaq(id),
    onSuccess: () => {
      setNotice('FAQ dihapus.')
      invalidate()
    },
    onError: (error) => setNotice(error instanceof Error ? error.message : 'Gagal menghapus FAQ.'),
  })

  const rows: FaqItem[] = data ?? []

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">FAQ</CardTitle>
        <CardDescription className="text-xs">
          Yang dinonaktifkan tetap tersimpan agar bisa diaktifkan lagi nanti.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {notice && <Alert>{notice}</Alert>}
        {isLoading && <Skeleton className="h-24 w-full" />}
        {!isLoading && rows.length === 0 && (
          <EmptyState title="Belum ada FAQ" description="Tambahkan pertanyaan pertama di bawah." />
        )}

        {rows.map((row) => (
          <div
            key={row.id}
            className="flex flex-wrap items-start justify-between gap-2 border-b border-border pb-2 last:border-0"
          >
            <div className="min-w-0">
              <p className="text-[13px] font-medium">{row.question}</p>
              <p className="text-[11px] uppercase tracking-wider text-muted-foreground">
                {row.category}
              </p>
            </div>
            <div className="flex items-center gap-2">
              {!row.is_active && <Badge variant="default">Nonaktif</Badge>}
              <Button size="sm" variant="outline" onClick={() => toggle.mutate(row)}>
                {row.is_active ? 'Nonaktifkan' : 'Aktifkan'}
              </Button>
              <Button size="sm" variant="ghost" onClick={() => remove.mutate(row.id)}>
                Hapus
              </Button>
            </div>
          </div>
        ))}

        <div className="space-y-2 border-t border-border pt-4">
          <div className="grid gap-2 sm:grid-cols-2">
            <div>
              <Label htmlFor="faq-category">Kategori</Label>
              <Input
                id="faq-category"
                value={draft.category}
                onChange={(event) => setDraft({ ...draft, category: event.target.value })}
                placeholder="Memulai"
                className="mt-1"
              />
            </div>
            <div>
              <Label htmlFor="faq-question">Pertanyaan</Label>
              <Input
                id="faq-question"
                value={draft.question}
                onChange={(event) => setDraft({ ...draft, question: event.target.value })}
                className="mt-1"
              />
            </div>
          </div>
          <div>
            <Label htmlFor="faq-answer">Jawaban</Label>
            <textarea
              id="faq-answer"
              value={draft.answer_md}
              onChange={(event) => setDraft({ ...draft, answer_md: event.target.value })}
              rows={3}
              className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            />
          </div>
          <Button
            size="sm"
            disabled={create.isPending || !draft.category || !draft.question || !draft.answer_md}
            onClick={() => create.mutate()}
          >
            Tambah FAQ
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

function AnnouncementSection() {
  const queryClient = useQueryClient()
  const [notice, setNotice] = useState<string | null>(null)
  const [draft, setDraft] = useState({
    title: '',
    body: '',
    level: 'info' as 'info' | 'warning' | 'critical',
  })

  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'cms', 'announcements'],
    queryFn: adminCmsApi.announcements,
  })

  const invalidate = () =>
    void queryClient.invalidateQueries({ queryKey: ['admin', 'cms', 'announcements'] })

  const create = useMutation({
    mutationFn: () => adminCmsApi.createAnnouncement({ ...draft, is_active: true }),
    onSuccess: () => {
      setNotice('Pengumuman diterbitkan dan langsung tampil di dashboard pelanggan.')
      setDraft({ title: '', body: '', level: 'info' })
      invalidate()
    },
    onError: (error) =>
      setNotice(error instanceof Error ? error.message : 'Gagal menerbitkan pengumuman.'),
  })

  const toggle = useMutation({
    mutationFn: (row: AnnouncementItem) =>
      adminCmsApi.updateAnnouncement(row.id, { ...row, is_active: !row.is_active }),
    onSuccess: invalidate,
    onError: (error) =>
      setNotice(error instanceof Error ? error.message : 'Gagal mengubah pengumuman.'),
  })

  const rows: AnnouncementItem[] = data ?? []

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Pengumuman</CardTitle>
        <CardDescription className="text-xs">
          Hindari menyebut angka keuntungan: pengumuman dibaca semua pelanggan.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {notice && <Alert>{notice}</Alert>}
        {isLoading && <Skeleton className="h-24 w-full" />}
        {!isLoading && rows.length === 0 && (
          <EmptyState
            title="Belum ada pengumuman"
            description="Pengumuman hanya muncul di jendela waktunya."
          />
        )}

        {rows.map((row) => (
          <div
            key={row.id}
            className="flex flex-wrap items-start justify-between gap-2 border-b border-border pb-2 last:border-0"
          >
            <div className="min-w-0">
              <p className="text-[13px] font-medium">{row.title}</p>
              <p className="text-[11px] text-muted-foreground">
                {formatDateTime(row.starts_at)}
                {row.ends_at ? ` → ${formatDateTime(row.ends_at)}` : ' → tanpa batas akhir'}
              </p>
            </div>
            <div className="flex items-center gap-2">
              <Badge variant={row.level === 'critical' ? 'solid' : 'outline'}>{row.level}</Badge>
              <Button size="sm" variant="outline" onClick={() => toggle.mutate(row)}>
                {row.is_active ? 'Nonaktifkan' : 'Aktifkan'}
              </Button>
            </div>
          </div>
        ))}

        <div className="space-y-2 border-t border-border pt-4">
          <div className="grid gap-2 sm:grid-cols-2">
            <div>
              <Label htmlFor="ann-title">Judul</Label>
              <Input
                id="ann-title"
                value={draft.title}
                onChange={(event) => setDraft({ ...draft, title: event.target.value })}
                className="mt-1"
              />
            </div>
            <div>
              <Label htmlFor="ann-level">Tingkat</Label>
              <select
                id="ann-level"
                value={draft.level}
                onChange={(event) =>
                  setDraft({
                    ...draft,
                    level: event.target.value as 'info' | 'warning' | 'critical',
                  })
                }
                className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              >
                <option value="info">Info</option>
                <option value="warning">Perhatian</option>
                <option value="critical">Penting</option>
              </select>
            </div>
          </div>
          <div>
            <Label htmlFor="ann-body">Isi</Label>
            <textarea
              id="ann-body"
              value={draft.body}
              onChange={(event) => setDraft({ ...draft, body: event.target.value })}
              rows={3}
              className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            />
          </div>
          <Button
            size="sm"
            disabled={create.isPending || !draft.title || !draft.body}
            onClick={() => create.mutate()}
          >
            Terbitkan
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
