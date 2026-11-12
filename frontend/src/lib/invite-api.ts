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
