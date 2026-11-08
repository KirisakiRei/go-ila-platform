import { request } from './http'
import type {
  AnnouncementItem,
  PublicStatus,
  BrokerHealth,
  EngineOverview,
  FaqItem,
  MaintenanceState,
  PageContent,
  PageContentPayload,
  RunnerOverview,
  SettingRow,
  SupportNoteItem,
  SystemOverview,
} from '@/types/system'

/** Lampiran sesi admin memakai cookie sendiri, jadi path berbeda dari sesi user. */
const ADMIN = '/admin'

/** Flag sistem: membaca, mengubah, dan mengembalikan ke nilai bawaan. */
export const adminSystemApi = {
  overview: () => request<SystemOverview>(`${ADMIN}/system`),
  engines: () => request<EngineOverview[]>(`${ADMIN}/system/engine`),
  runner: () => request<RunnerOverview>(`${ADMIN}/system/runner`),
  brokers: () => request<BrokerHealth>(`${ADMIN}/system/brokers`),
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
  addSupportNote: (
    userID: string,
    payload: { body: string; channel?: string; is_pinned?: boolean },
  ) =>
    request<SupportNoteItem>(`${ADMIN}/users/${userID}/support-notes`, {
      method: 'POST',
      body: payload,
    }),
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

  pages: () => request<PageContent[]>(`${ADMIN}/pages`),
  page: (slug: string) => request<PageContent>(`${ADMIN}/pages/${slug}`),
  savePage: (slug: string, payload: PageContentPayload) =>
    request<PageContent>(`${ADMIN}/pages/${slug}`, { method: 'PUT', body: payload }),
}

/** Konten publik: dibaca halaman bantuan dan dashboard tanpa sesi. */
export const publicContentApi = {
  faq: () => request<FaqItem[]>('/faq'),
  announcements: () => request<AnnouncementItem[]>('/announcements'),
  status: () => request<PublicStatus>('/status'),
  /** Isi halaman publik dikelola admin; halaman menampilkan keadaan kosong bila belum ada. */
  page: (slug: string) => request<PageContent>(`/pages/${slug}`),
}

export type {
  AnnouncementItem,
  BrokerHealth,
  EngineOverview,
  FaqItem,
  MaintenanceState,
  PublicStatus,
  RunnerOverview,
  SettingRow,
  SupportNoteItem,
  SystemOverview,
}
