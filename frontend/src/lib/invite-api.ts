import { useQuery } from '@tanstack/react-query'

import { request } from './http'

/** Balasan pemeriksaan tautan undangan. Tidak memuat alamat surel tujuan. */
export type InvitePublic = {
  valid: boolean
  tier: 'free' | 'basic' | 'plus'
  expires_at: string
  sisa_kuota: number
  email_terkunci: boolean
}

/** Satu baris daftar undangan di panel admin. Tidak memuat kode mentah. */
export type InviteItem = {
  id: string
  prefix: string
  email: string
  tier: 'free' | 'basic' | 'plus'
  max_uses: number
  used_count: number
  sisa_kuota: number
  expires_at: string
  status: 'aktif' | 'habis' | 'kedaluwarsa' | 'dicabut'
  note: string
  created_at: string
}

/**
 * Balasan pembuatan undangan. Code dan URL hanya muncul SEKALI di sini dan
 * tidak bisa diambil lagi: backend hanya menyimpan hash-nya.
 */
export type CreatedInvite = InviteItem & { code: string; url: string }

/** Payload publik form pengajuan akses selama mode invite-only. */
export type SubmitInviteRequestPayload = {
  name: string
  email: string
  contact?: string
  note?: string
  website?: string
}

/** Satu baris permohonan akses di panel admin. */
export type InviteRequestItem = {
  id: string
  name: string
  email: string
  contact: string
  note: string
  status: 'pending' | 'approved' | 'rejected'
  invite_id?: string
  reviewed_by?: string
  reviewed_at?: string
  review_note?: string
  created_at: string
}

export type InviteRequestList = {
  items: InviteRequestItem[]
  pending_count: number
}

export type ApprovedInviteRequestResult = {
  request: InviteRequestItem
  invite: CreatedInvite
  email_queued: boolean
}

export const inviteApi = {
  /** Pemeriksaan publik untuk halaman daftar; tidak memakai kuota. */
  periksa: (code: string) => request<InvitePublic>(`/invites/${encodeURIComponent(code)}`),
  list: (status?: string) =>
    request<InviteItem[]>('/admin/invites', { query: status ? { status } : undefined }),
  create: (payload: {
    tier?: string
    max_uses: number
    expires_at?: string
    email?: string
    note?: string
  }) => request<CreatedInvite>('/admin/invites', { method: 'POST', body: payload }),
  revoke: (id: string, reason: string) =>
    request<void>(`/admin/invites/${id}/revoke`, { method: 'POST', body: { reason } }),
}

export const inviteRequestApi = {
  /** Pengajuan publik dari beranda tanpa kredensial. */
  submit: (payload: SubmitInviteRequestPayload) =>
    request<void>('/invite-requests', { method: 'POST', body: payload }),
  /** Daftar permohonan akses untuk ditinjau admin. */
  list: (params?: { status?: string; page?: number; per_page?: number }) =>
    request<InviteRequestList>('/admin/invite-requests', {
      query: {
        ...(params?.status ? { status: params.status } : {}),
        ...(params?.page ? { page: params.page } : {}),
        ...(params?.per_page ? { per_page: params.per_page } : {}),
      },
    }),
  approve: (id: string, payload?: { tier?: string; expires_days?: number; note?: string }) =>
    request<ApprovedInviteRequestResult>(`/admin/invite-requests/${id}/approve`, {
      method: 'POST',
      body: payload ?? {},
    }),
  reject: (id: string, reason?: string) =>
    request<void>(`/admin/invite-requests/${id}/reject`, {
      method: 'POST',
      body: { reason: reason ?? '' },
    }),
}

/**
 * useInviteCheck memeriksa satu kode undangan.
 *
 * `retry: false` disengaja: kode tidak sah adalah jawaban biasa, bukan gangguan
 * jaringan, sehingga tidak perlu diulang. Hasilnya hanya untuk tampilan;
 * keputusan sebenarnya tetap diambil backend saat formulir dikirim.
 */
export function useInviteCheck(code: string | null) {
  return useQuery({
    queryKey: ['invite', code],
    queryFn: () => inviteApi.periksa(code ?? ''),
    enabled: Boolean(code),
    retry: false,
    staleTime: 30_000,
  })
}
