/** Bentuk balasan seragam dari backend (pkg/httpx). */

export type Tier = 'free' | 'basic' | 'plus'
export type Role = 'user' | 'admin' | 'support'

export type ApiResponse<T> = {
  success: boolean
  message: string
  data?: T
  meta?: PaginationMeta
  error?: string
  details?: unknown
}

export type PaginationMeta = {
  page: number
  per_page: number
  total: number
  total_pages: number
  has_more?: boolean
  sort?: string
  order?: string
  /** Penanda total hanya perkiraan pada tabel besar; tampilkan dengan "~". */
  approx_total?: boolean
  /** Kursor halaman berikutnya pada daftar panjang. */
  next_cursor?: string
}

/**
 * TimePair menyajikan satu waktu dalam dua zona. Server yang menghitung,
 * bukan browser, supaya hasilnya sama di perangkat mana pun (plans/22).
 */
export type TimePair = {
  utc: string
  wib: string
  ny: string
}

export type User = {
  id: string
  name: string
  email: string
  role: Role
  tier: Tier
  /** Memutuskan apakah modal "masih di sini?" ditampilkan. */
  idle_warning: boolean
  email_verified_at: string | null
  created_at: string
  updated_at: string
}

export type LoginPayload = {
  email: string
  password: string
}

export type RegisterPayload = {
  name: string
  email: string
  password: string
}

export type LoginResult = {
  expires_at: string
  user: User
}

export type BrokerType = 'matchtrader'
export type AccountMode = 'manual' | 'auto'

export type BrokerAccount = {
  id: string
  broker_type: BrokerType
  label: string
  external_account_id: string
  mode: AccountMode
  risk_percent: number
  is_active: boolean
  is_frozen: boolean
  is_dry_run: boolean
  has_credential: boolean
  last_verified_at: string | null
  last_error: string
  connected: boolean
  balance: number
  equity: number
  free_margin: number
  last_synced_at: string | null
}

export type CreateAccountPayload = {
  broker_type: BrokerType
  label: string
  mode: AccountMode
  risk_percent: number
  is_dry_run: boolean
  token: string
  base_url: string
}

export type UpdateAccountPayload = {
  label?: string
  mode?: AccountMode
  risk_percent?: number
  is_active?: boolean
  is_dry_run?: boolean
}

export type StrategyVersion = 'ila-2.0' | 'ila-3.0'

export type Subscription = {
  id: string
  symbol: string
  strategy: StrategyVersion
  broker_account_id: string | null
  is_active: boolean
}

export type CreateSubscriptionPayload = {
  symbol: string
  strategy: StrategyVersion
  broker_account_id: string
}

export type StrategyStatus = 'active' | 'beta' | 'coming_soon'

export type Strategy = {
  version: StrategyVersion
  name: string
  status: StrategyStatus
  available: boolean
  allowed: boolean
  selected: boolean
  description: string
}

export type SignalStatus =
  | 'READY_TO_ENTRY'
  | 'ACKED'
  | 'REJECTED'
  | 'EXPIRED'
  | 'CANCELLED'
  | 'SUPERSEDED'
  | 'CONSUMED'
  | 'SKIPPED'

export type TradeDirection = 'BUY' | 'SELL'

export type Signal = {
  signal_id: string
  symbol: string
  strategy_version: StrategyVersion
  engine_version: string
  direction: TradeDirection
  entry_price: number
  stop_loss: number
  take_profit: number
  risk_reward: number
  status: SignalStatus
  summary: string
  next_action: string
  reject_reason: string
  published_at: string
  expires_at: string
  can_confirm: boolean
  delayed: boolean
}

export type PositionStatus = 'OPEN' | 'CLOSED_TP' | 'CLOSED_SL' | 'CLOSED_MANUAL' | 'UNTRACKED'

export type Position = {
  id: string
  symbol: string
  strategy_version: StrategyVersion
  direction: TradeDirection
  ticket_number: string
  lot_size: number
  open_price: number
  stop_loss: number
  take_profit: number
  exit_price: number
  profit_loss: number
  outcome_r: number
  risk_percent: number
  status: PositionStatus
  is_dry_run: boolean
  opened_at: string
  opened_at_pair: TimePair
  closed_at: string | null
  closed_at_pair: TimePair | null
  /**
   * false berarti hasil belum terbaca dari broker. Jangan tampilkan 0 —
   * gunakan garis "(—)" agar tidak menyesatkan (plans/19 C9).
   */
  pnl_known: boolean
}

/** Satu perangkat yang sedang masuk ke akun ini. */
export type Session = {
  id: string
  device_label: string
  user_agent: string
  last_seen_at: string
  last_seen_pair: TimePair
  created_at: string
  created_pair: TimePair
  expires_at: string
  expires_pair: TimePair
  /** true untuk perangkat yang sedang dipakai. */
  current: boolean
}

export type MarketState = 'OPEN' | 'CLOSED' | 'ROLLOVER' | 'UNKNOWN'

export type SystemStatus = {
  market: { state: MarketState; label: string; updated_at: string }
  engines: {
    strategy: StrategyVersion
    connected: boolean
    engine_version: string
    last_seen_at: string
  }[]
  brokers: { total: number; connected: number; any_error: boolean }
  daily: {
    day: string
    win_count: number
    loss_count: number
    total_trades: number
    max_trades: number
    open_count: number
    locked: boolean
    lock_reason: string
  }
  server_time: string
  daily_timezone: string
}

export type Trial = {
  used: boolean
  active: boolean
  duration_days: number
  days_remaining: number
  dry_run_only: boolean
  trial_started_at: string | null
  trial_expires_at: string | null
}

export type TelegramLink = {
  url: string
  expires_at: string
}

export type Approval = {
  signal_id: string
  symbol: string
  direction: TradeDirection
  strategy_version: StrategyVersion
  entry_price: number
  stop_loss: number
  take_profit: number
  risk_reward: number
  expires_at: string
  broker_account_id: string
  can_confirm: boolean
  blocked_reason: string
}

export type ApprovalResult = {
  signal_id: string
  status: SignalStatus
  message: string
}

/** Posisi terakhir alur mulai; dipakai agar bisa dilanjutkan. */
export type OnboardingState = {
  step: number
  done: boolean
  total_steps: number
  steps: string[]
  has_broker: boolean
  email_verified: boolean
  telegram_linked: boolean
  next_action: string
}

/** Keadaan langganan untuk banner dashboard. */
export type SubscriptionBanner = {
  state: 'normal' | 'ending' | 'grace' | 'expired'
  title: string
  message: string
  expires_at?: string
  days_left: number
  level: 'info' | 'warning' | 'critical'
}
