export interface JournalReview {
  account_id: string
  date: string
  revision: string
  status: 'pending' | 'running' | 'ready'
  text: string
  source: 'ai' | 'rules' | ''
  updated_at: string
}
export interface JournalReviewSettings {
  account_id: string
  ai_enabled: boolean
  effective_ai_enabled: boolean
}

export type JournalStatus = 'draft' | 'open' | 'closed'
export interface JournalAccount {
  id: string
  name: string
  broker: string
  currency: string
  initial_capital: string
  initial_date: string
  current_realized_balance: string | null
  balance_complete: boolean
  unknown_closed_pnl: number
  notes: string
  archived: boolean
  checklist: string[]
}
export interface JournalTrade {
  id: string
  created_at: string
  account_id: string
  status: JournalStatus
  asset: string
  side: 'long' | 'short' | ''
  opened_at: string | null
  closed_at: string | null
  entry_price: string | null
  exit_price: string | null
  stop_loss: string | null
  take_profit: string | null
  size: string | null
  size_unit: string
  net_pnl: string | null
  risk_amount: string | null
  capital_snapshot: string | null
  setup: string
  tags: string[]
  emotions: string[]
  mistakes: string[]
  session: string
  checklist: { label: string; checked: boolean }[]
  notes: string
  tradingview_urls: string[]
}
export type TradeInput = Omit<JournalTrade, 'id' | 'created_at'>
export type AccountInput = Omit<
  JournalAccount,
  'id' | 'current_realized_balance' | 'balance_complete' | 'unknown_closed_pnl'
>
export interface CapitalInput {
  account_id: string
  kind: 'deposit' | 'withdrawal'
  amount: string
  date: string
  notes: string
}
export interface CapitalMovement extends CapitalInput {
  id: string
}
export interface JournalFilters {
  account_id: string
  search?: string
  from: string
  to: string
  status: JournalStatus | ''
  asset: string
  setup: string
  tag: string
  emotion: string
  mistake: string
}
export interface JournalMeta {
  current_page: number
  per_page: number
  total_items: number
  total_pages: number
}
export interface JournalGroup {
  trades: number
  wins: number
  net_pnl: string
}
export interface JournalStats {
  currency: string
  trades: number
  wins: number
  losses: number
  breakeven: number
  win_rate: string
  profit_factor: string | null
  average_win: string
  average_loss: string
  net_pnl: string
  max_drawdown: string
  curve: { id: string; date: string; pnl: string; cumulative: string; drawdown: string }[]
  calendar: Record<string, JournalGroup>
  breakdowns: Record<string, Record<string, JournalGroup>>
}
export interface JournalInsight {
  code: string
  message: string
  count: number
}
export interface CSVPreview {
  rows: TradeInput[]
  count: number
  token: string
  expires_at: string
}
export interface CSVConfirm {
  csv: string
  token: string
}
export interface JournalScreenshot {
  id: string
  entry_id: string
  file_name: string
  mime_type: string
  size_bytes: number
}
