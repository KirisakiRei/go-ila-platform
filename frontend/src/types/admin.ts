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
  /** Akun superadmin utama: tidak boleh dihapus dari panel. */
  is_protected?: boolean
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

/** AdminInvoiceItem adalah tagihan lintas pengguna di panel admin. */
export type AdminInvoiceItem = {
  id: string
  number: string
  user_id: string
  user_email: string
  plan_code: string
  amount_idr: number
  discount_idr: number
  credit_idr: number
  total_idr: number
  status: 'pending' | 'paid' | 'expired' | 'cancelled' | 'refunded'
  coupon_code?: string
  gateway?: string
  period_start: TimePair
  period_end: TimePair
  created_at: TimePair
  paid_at?: TimePair
  expires_at: TimePair
}

/** AdminTrialItem adalah satu baris halaman uji coba admin. */
export type AdminTrialItem = {
  user_id: string
  email: string
  name: string
  tier: string
  trial_used: boolean
  status: 'active' | 'ended' | 'never'
  started_at?: TimePair
  expires_at?: TimePair
  days_remaining: number
  tier_during_trial: string
  account_limit: number
}

/** CreditCodeItem adalah voucher saldo yang bisa ditukar pengguna. */
export type CreditCodeItem = {
  id: string
  code: string
  amount_idr: number
  max_redemptions: number
  max_per_user: number
  starts_at: TimePair
  ends_at: TimePair
  is_active: boolean
  redeemed: number
  status: 'active' | 'scheduled' | 'expired' | 'exhausted' | 'disabled'
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

/** Payload pembuatan akun pelanggan baru oleh admin. */
export type AdminCreateUserPayload = {
  name: string
  email: string
  username?: string
  password: string
  tier: Tier
}

// --- Katalog pair -----------------------------------------------------------

/** Satu pair pada katalog admin, lengkap dengan jumlah pelanggan aktif. */
export type AdminPairItem = {
  id: string
  symbol: string
  display_name: string
  category_id: string
  category_key: string
  group_id?: string
  group_key?: string
  aliases?: string
  correlated_symbols?: string
  sort_order: number
  is_enabled: boolean
  description?: string
  subscribers: number
}

/** Sub-kategori (grup) di dalam sebuah kategori pair. */
export type AdminPairGroup = {
  id: string
  key: string
  label: string
  sort_order: number
  is_enabled: boolean
}

/** Kategori pair beserta grup dan pair di dalamnya. */
export type AdminPairCategory = {
  id: string
  key: string
  label: string
  sort_order: number
  is_enabled: boolean
  groups: AdminPairGroup[]
  pairs: AdminPairItem[]
}

/** Seluruh katalog pair untuk panel admin. */
export type AdminPairCatalog = {
  categories: AdminPairCategory[]
}

// --- RBAC -------------------------------------------------------------------

/** Satu role RBAC beserta permission yang dipegangnya. */
export type AdminRole = {
  id: string
  code: string
  name: string
  description: string
  /** Role bawaan sistem tidak dapat diubah atau dihapus. */
  is_system: boolean
  editable: boolean
  permissions: string[]
  user_count: number
  created_at: string
}

/** Satu kewenangan di katalog permission. */
export type AdminPermission = {
  key: string
  group: string
  label: string
  risk: 'rendah' | 'sedang' | 'tinggi'
  /** false berarti pelaku tidak boleh memberikan izin ini. */
  grantable: boolean
}

/** Katalog permission yang dikelompokkan untuk tampilan centang bergrup. */
export type AdminPermissionGroup = {
  group: string
  permissions: AdminPermission[]
}

/** Detail role beserta katalog permission yang bisa dipilih. */
export type AdminRoleDetail = AdminRole & {
  available_permissions: AdminPermission[]
}

// --- Sinyal admin -----------------------------------------------------------

/** Satu sinyal dari sudut pandang admin. */
export type AdminSignalItem = {
  signal_id: string
  symbol: string
  strategy_version: string
  engine_version: string
  direction: string
  entry_price: number
  stop_loss: number
  take_profit: number
  risk_reward: number
  status: string
  summary: string
  next_action: string
  published_at: TimePair
  expires_at: TimePair
  cancel_reason?: string
  /** Jumlah user yang menerima sinyal ini. */
  recipients: number
  /** Jumlah user yang benar-benar entry. */
  executed: number
}

/** Nasib sinyal pada satu user. */
export type AdminSignalRecipient = {
  user_id: string
  user_name: string
  user_email: string
  broker_account_id?: string
  status: string
  reject_reason?: string
  position_id?: string
  decided_at?: TimePair
  created_at: TimePair
}

/** Satu perubahan status sinyal. */
export type AdminSignalEvent = {
  user_id: string
  from_status?: string
  to_status: string
  reason?: string
  created_at: TimePair
}

/** Klaim eksekusi satu sinyal pada satu akun. */
export type AdminSignalExecution = {
  user_id: string
  broker_account_id: string
  status: string
  created_at: TimePair
}

/** Perjalanan lengkap satu sinyal. */
export type AdminSignalTrace = {
  signal: AdminSignalItem
  recipients: AdminSignalRecipient[]
  events: AdminSignalEvent[]
  executions: AdminSignalExecution[]
  positions: AdminPositionItem[]
}
