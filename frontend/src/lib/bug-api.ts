import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { request } from './http'

/** Batas lampiran HARUS sama dengan `request_body max_size` di Caddyfile. */
export const BUG_MAX_LAMPIRAN_BYTE = 1 << 20
export const BUG_MAKS_LAMPIRAN_MB = 1

export type BugStatus = 'baru' | 'ditinjau' | 'selesai' | 'ditolak'

/** Satu baris riwayat laporan milik pengguna. */
export type BugItem = {
  id: string
  category: string
  title: string
  status: BugStatus
  admin_note: string
  created_at: string
  updated_at: string
}

export type BugAttachmentItem = {
  id: string
  file_name: string
  mime_type: string
  size_bytes: number
  from_admin?: boolean
}

/** Laporan lengkap: dipakai daftar milik sendiri dan panel admin. */
export type BugDetail = BugItem & {
  body: string
  page_url: string
  user_agent: string
  user_id: string
  attachments: BugAttachmentItem[]
}

export type BugPayload = {
  category: string
  title: string
  body: string
  pageUrl: string
  attachment?: File | null
}

/**
 * kirim membentuk multipart dari formulir.
 *
 * `page_url` diisi otomatis: tanpa itu, admin harus menebak halaman mana yang
 * dimaksud pelapor, dan reproduksi bug menjadi jauh lebih lambat.
 */
async function kirim(payload: BugPayload): Promise<BugDetail> {
  const form = new FormData()
  form.append('category', payload.category)
  form.append('title', payload.title)
  form.append('body', payload.body)
  form.append('page_url', payload.pageUrl)
  if (payload.attachment) form.append('attachment', payload.attachment)

  // `body: form` membuat lapisan request tidak memasang Content-Type JSON.
  return request<BugDetail>('/bug-reports', { method: 'POST', body: form })
}

export const bugApi = {
  create: kirim,
  listMine: () => request<BugItem[]>('/bug-reports'),
  categories: () => request<string[]>('/bug-reports/categories'),
  /**
   * Unduh lampiran. Path langsung dipakai tautan `<a href>` sehingga cookie
   * sesi ikut terkirim tanpa perlu fetch manual.
   *
   * Prefix /api/v1 wajib ditulis di sini: tautan ini tidak melewati buildUrl,
   * jadi tanpa prefix browser membuka halaman aplikasi dan hasilnya 404.
   */
  attachmentUrl: (reportID: string, attachmentID: string) =>
    `/api/v1/bug-reports/${reportID}/attachments/${attachmentID}`,
  detail: (id: string) => request<BugDetail>(`/bug-reports/${id}`),
  adminReply: (id: string, payload: { note: string; attachment?: File | null }) => {
    const form = new FormData()
    form.append('note', payload.note)
    if (payload.attachment) form.append('attachment', payload.attachment)
    return request<void>(`/admin/bug-reports/${id}/reply`, { method: 'POST', body: form })
  },

  adminList: (query?: { status?: string; category?: string }) =>
    request<BugDetail[]>('/admin/bug-reports', { query }),
  adminUpdate: (id: string, payload: { status: BugStatus; note?: string }) =>
    request<void>(`/admin/bug-reports/${id}`, { method: 'PATCH', body: payload }),
}

export function useBugCategories() {
  return useQuery({
    queryKey: ['bug', 'categories'],
    queryFn: bugApi.categories,
    staleTime: 5 * 60_000,
  })
}

export function useMyBugReports() {
  return useQuery({ queryKey: ['bug', 'mine'], queryFn: bugApi.listMine })
}

/** useKirimLaporanBug mengirim laporan lalu menyegarkan riwayat. */
export function useKirimLaporanBug() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: bugApi.create,
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['bug', 'mine'] }),
  })
}

/** useBugDetail membaca satu laporan milik pengguna untuk halaman detail. */
export function useBugDetail(id: string | undefined) {
  return useQuery({
    queryKey: ['bug', 'detail', id],
    queryFn: () => bugApi.detail(id as string),
    enabled: Boolean(id),
  })
}

/** useBalasBug menyimpan tanggapan tim beserta lampiran opsional. */
export function useBalasBug() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...payload }: { id: string; note: string; attachment?: File | null }) =>
      bugApi.adminReply(id, payload),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['bug'] }),
  })
}
