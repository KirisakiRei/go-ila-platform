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
