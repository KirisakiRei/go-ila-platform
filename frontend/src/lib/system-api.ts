import { request } from './http'
import type {
  AnnouncementItem,
  BrokerHealth,
  EngineOverview,
  FaqItem,
  MaintenanceState,
  SettingRow,
  SupportNoteItem,
  SystemOverview,
} from '@/types/system'

/** Lampiran sesi admin memakai cookie sendiri, jadi path berbeda dari sesi user. */
const ADMIN = '/admin'

/** Flag sistem: membaca, mengubah, dan mengembalikan ke nilai bawaan. */
export const adminSystemApi = {
  overview: () => request<SystemOverview>(`${ADMIN}/system`),
  settings: () => request<SettingRow[]>(`${ADMIN}/system/settings`),
  updateSetting: (key: string, payload: { value: string; reason: string; confirm: string }) =>
    request<SettingRow>(`${ADMIN}/system/settings/${key}`, { method: 'PUT', body: payload }),
  resetSetting: (key: string, payload: { reason: string; confirm: string }) =>
    request<void>(`${ADMIN}/system/settings/${key}`, { method: 'DELETE', body: payload }),
  changeMode: (payload: {
    mode: string
    reason: string
    expected_end?: string
    confirm?: string
  }) => request<MaintenanceState>(`${ADMIN}/system/mode`, { method: 'PUT', body: payload }),
  supportNotes: (userID: string) =>
    request<SupportNoteItem[]>(`${ADMIN}/users/${userID}/support-notes`),
  addSupportNote: (userID: string, payload: { body: string; channel?: string; is_pinned?: boolean }) =>
    request<SupportNoteItem>(`${ADMIN}/users/${userID}/support-notes`, { method: 'POST', body: payload }),
}

/** Konten: FAQ dan pengumuman diubah tanpa deploy ulang. */
export const adminCmsApi = {
  faq: () => request<FaqItem[]>(`${ADMIN}/faq`),
  createFaq: (payload: Record<string, unknown>) =>
    request<FaqItem>(`${ADMIN}/faq`, { method: 'POST', body: payload }),
  updateFaq: (id: string, payload: Record<string, unknown>) =>
    request<FaqItem>(`${ADMIN}/faq/${id}`, { method: 'PATCH', body: payload }),
  deleteFaq: (id: string) => request<void>(`${ADMIN}/faq/${id}`, { method: 'DELETE' }),

  announcements: () => request<AnnouncementItem[]>(`${ADMIN}/announcements`),
  createAnnouncement: (payload: Record<string, unknown>) =>
    request<AnnouncementItem>(`${ADMIN}/announcements`, { method: 'POST', body: payload }),
  updateAnnouncement: (id: string, payload: Record<string, unknown>) =>
    request<AnnouncementItem>(`${ADMIN}/announcements/${id}`, { method: 'PATCH', body: payload }),
  deleteAnnouncement: (id: string) =>
    request<void>(`${ADMIN}/announcements/${id}`, { method: 'DELETE' }),
}

/** Konten publik: dibaca halaman bantuan dan dashboard tanpa sesi. */
export const publicContentApi = {
  faq: () => request<FaqItem[]>('/faq'),
  announcements: () => request<AnnouncementItem[]>('/announcements'),
  status: () => request<PublicStatus>('/status'),
}

export type PublicStatus = {
  mode: string
  message: string
  expected_end?: string
  server_time: string
  time_pair: { wib: string; ny: string }
}

export type { AnnouncementItem, BrokerHealth, EngineOverview, FaqItem, MaintenanceState, SettingRow, SupportNoteItem, SystemOverview }
