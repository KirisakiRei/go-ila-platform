/** Tipe data panel admin. Semua waktu memakai TimePair dari server. */
import type { Tier, TimePair } from './api'

export type AdminUserListItem = {
  id: string
  name: string
  email: string
  tier: Tier
  status: 'active' | 'suspended' | 'banned' | 'deleted'
  role: string
  role_code?: string
  telegram_linked: boolean
  account_count: number
  created_at: TimePair
  last_login_at?: TimePair
  tier_expires_at?: TimePair
}

export type AdminSubscriptionBrief = {
  plan_code: Tier
  status: string
  period_start: TimePair
  period_end: TimePair
  cancel_at_period_end: boolean
}

export type AdminUserDetail = AdminUserListItem & {
  trial_used: boolean
  trial_expires_at?: TimePair
  grace_until?: TimePair
  suspended_at?: TimePair
  banned_at?: TimePair
  ban_reason?: string
  ban_category?: string
  internal_note?: string
  idle_warning: boolean
  notify_enabled: boolean
  two_factor_enabled: boolean
  subscription?: AdminSubscriptionBrief
}

export type AdminAccountItem = {
  id: string
  label: string
  broker_type: string
  external_account_id: string
  mode: string
  risk_mode: string
  risk_percent: number
  is_active: boolean
  is_frozen: boolean
  is_dry_run: boolean
  frozen_at?: TimePair
  frozen_reason?: string
  last_verified_at?: TimePair
  last_error?: string
}

export type AdminPositionItem = {
  id: string
  symbol: string
  direction: string
  strategy_version: string
  status: string
  open_price: number
  exit_price: number
  profit_loss: number
  outcome_r: number
  is_dry_run: boolean
  opened_at: TimePair
  closed_at?: TimePair
}

export type AdminLegalItem = {
  slug: string
  title: string
  version: string
  accepted_at: TimePair
  ip_hash: string
}

export type AdminNoteItem = {
  id: string
  author_id: string
  author_name: string
  body: string
  created_at: TimePair
}

export type AuditEntry = {
  id: string
  actor_id: string
  actor_type: string
  actor_email?: string
  action: string
  target_type: string
  target_id: string
  reason?: string
  before_state?: string
  after_state?: string
  request_id?: string
  created_at: TimePair
}

export type AuditCatalogEntry = {
  action: string
  group: string
  label: string
  sensitive: boolean
}

export type PublicPlan = {
  code: Tier
  name: string
  description: string
  price_idr: number
  interval: string
  promo?: string
  features: Record<string, string>
  ila3_coming_soon: boolean
  active: boolean
}

export type CouponItem = {
  id: string
  code: string
  description: string
  discount_type: 'percent' | 'fixed'
  discount_value: number
  max_discount_idr?: number
  min_amount_idr: number
  applies_to_plans: string
  first_purchase_only: boolean
  starts_at: TimePair
  ends_at: TimePair
  max_redemptions: number
  max_per_user: number
  is_active: boolean
  redeemed: number
  status: 'active' | 'scheduled' | 'expired' | 'exhausted' | 'disabled'
}

export type CreditEntry = {
  id: string
  delta_idr: number
  balance_after: number
  kind: string
  reason: string
  created_at: TimePair
}

export type CreditSummary = {
  user_id: string
  balance_idr: number
  history: CreditEntry[]
}

export type RevenueReport = {
  from: TimePair
  to: TimePair
  gross_idr: number
  refund_idr: number
  net_idr: number
  daily: { date: string; paid_idr: number; refund_idr: number; count: number }[]
}

export type CustomerReport = {
  from: TimePair
  to: TimePair
  new_users: number
  paid_users: number
  canceled: number
  trial_started: number
}

export type CouponReport = {
  from: TimePair
  to: TimePair
  items: { code: string; uses: number; discount_idr: number }[]
}

export type OperationReport = {
  from: TimePair
  to: TimePair
  positions_opened: number
  positions_closed: number
  positions_dry_run: number
  notifications_sent: number
  notifications_failed: number
}

/** Aksi berlapis terhadap pengguna. Alasan wajib, email wajib untuk ban/hapus. */
export type UserActionPayload = {
  reason: string
  category?: string
  confirm_email?: string
  acknowledge?: boolean
}
