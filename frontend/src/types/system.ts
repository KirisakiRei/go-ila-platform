/** Tipe untuk pengaturan sistem, kesehatan, dan konten (plans/11 §11–13). */

export type SettingRow = {
  key: string
  label: string
  help: string
  kind: 'bool' | 'int'
  value: string
  source: 'database' | 'env'
  danger: boolean
  min?: number
  max?: number
}

export type EngineOverview = {
  strategy: string
  connected: boolean
  engine_version?: string
  market_state?: string
  last_seen_at?: string
  last_error?: string
  stale: boolean
}

export type RunnerOverview = {
  connected: boolean
  version?: string
  last_seen?: string
  stale: boolean
}

export type BrokerHealth = {
  total: number
  connected: number
  error_count: number
  frozen: number
}

export type MaintenanceState = {
  mode: 'normal' | 'read-only' | 'maintenance'
  label: string
  reason: string
  expected_end?: string
  changed_at: string
  changed_by?: string
}

export type SystemOverview = {
  mode: MaintenanceState
  settings: SettingRow[]
  engines: EngineOverview[]
  runner: RunnerOverview
  broker_health: BrokerHealth
  announcements: number
  last_engine_seen?: string
  server_time: string
  server_time_pair: { wib: string; ny: string }
}

export type SupportNoteItem = {
  id: string
  user_id: string
  actor_id: string
  channel: string
  body: string
  is_pinned: boolean
  created_at: string
}

export type FaqItem = {
  id: string
  category: string
  question: string
  answer_md: string
  sort_order: number
  is_active: boolean
  updated_at: string
}

/** Satu butir bagian halaman: kartu, langkah, pertanyaan, atau sel tabel. */
export type PageItem = {
  label: string
  value: string
  note: string
  icon: string
  href: string
  sort_order: number
}

/** Jenis bagian yang punya bentuk render tetap di antarmuka. */
export type PageSectionKind =
  'prose' | 'cards' | 'steps' | 'faq' | 'table' | 'callout' | 'features' | 'stats'

/** Satu bagian halaman publik. */
export type PageSection = {
  key: string
  kind: PageSectionKind
  eyebrow: string
  title: string
  lead: string
  body: string
  layout: string
  sort_order: number
  is_published: boolean
  items: PageItem[]
}

/** Isi lengkap satu halaman publik. */
export type PageContent = {
  slug: string
  eyebrow: string
  title: string
  lead: string
  meta_title: string
  meta_description: string
  is_published: boolean
  updated_at: string
  sections: PageSection[]
}

/** Bentuk kiriman saat admin menyimpan halaman. */
export type PageContentPayload = {
  eyebrow: string
  title: string
  lead: string
  meta_title: string
  meta_description: string
  is_published: boolean
  sections: PageSection[]
}

export type AnnouncementItem = {
  id: string
  title: string
  body: string
  level: 'info' | 'warning' | 'critical'
  starts_at: string
  ends_at?: string
  is_active: boolean
  created_at: string
}

/** Jadwal pasar dalam dua zona: sesi mengikuti NY, hari akuntansi WIB. */
export type MarketWindow = {
  open: boolean
  label: string
  next_change: string
  next_label: string
  next_change_pair: { wib: string; ny: string; utc: string }
}

export type PublicStatus = {
  mode: string
  message: string
  expected_end?: string
  server_time: string
  time_pair: { wib: string; ny: string; utc: string }
  market: MarketWindow
  /**
   * Platform sedang tertutup untuk pendaftar mandiri: pendaftaran, uji coba,
   * dan pembayaran tidak ditawarkan. Akun baru hanya dari undangan admin.
   */
  invite_only: boolean
}
