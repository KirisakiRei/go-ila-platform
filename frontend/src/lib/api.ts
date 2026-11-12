import { request } from './http'
import type {
  AccountPairSelectionResponse,
  Approval,
  ApprovalIssue,
  ApprovalResult,
  BrokerAccount,
  CreateAccountPayload,
  CreateSubscriptionPayload,
  JournalResponse,
  MarketCandlesResponse,
  PairCatalogResponse,
  Position,
  RiskSimRequest,
  RiskSimResult,
  Session,
  Signal,
  Strategy,
  StrategySwitchMatrix,
  StrategyVersion,
  Subscription,
  SystemStatus,
  TelegramLink,
  TimeframeSwitchItem,
  TimeframeSwitchMatrix,
  Trial,
  UpdateAccountPayload,
} from '@/types/api'

/** Akun broker. Token kredensial hanya dikirim sekali saat dibuat. */
export const accountApi = {
  list: () => request<BrokerAccount[]>('/accounts'),
  create: (payload: CreateAccountPayload) =>
    request<BrokerAccount>('/accounts', { method: 'POST', body: payload }),
  update: (id: string, payload: UpdateAccountPayload) =>
    request<BrokerAccount>(`/accounts/${id}`, { method: 'PATCH', body: payload }),
  remove: (id: string) => request<void>(`/accounts/${id}`, { method: 'DELETE' }),
  ctraderAuthUrl: () => request<{ auth_url: string }>('/accounts/ctrader/auth-url'),
  ctraderCallback: (payload: { code: string; mode?: string; risk_percent?: number }) =>
    request<{ total_connected: number; accounts: BrokerAccount[] }>('/accounts/ctrader/callback', {
      method: 'POST',
      body: payload,
    }),
  pairs: (id: string) => request<AccountPairSelectionResponse>(`/accounts/${id}/pairs`),
  updatePairs: (id: string, symbols: string[]) =>
    request<{ symbols: string[] }>(`/accounts/${id}/pairs`, {
      method: 'PUT',
      body: { symbols },
    }),
}

export const pairApi = {
  catalog: () => request<PairCatalogResponse>('/pairs'),
}

/** Simulasi risiko: satu tempat hitungan lot, dipakai pratinjau di detail akun. */
export const riskApi = {
  simulate: (payload: RiskSimRequest) =>
    request<RiskSimResult>('/calc/risk-sim', { method: 'POST', body: payload }),
}

/**
 * Saklar metode analisis per akun × pair.
 *
 * Pair pantauan bersifat global, sedangkan metode yang benar-benar dieksekusi
 * ditentukan per akun lewat saklar ini.
 */
export const strategySwitchApi = {
  matrix: (symbol: string) =>
    request<StrategySwitchMatrix>(`/pairs/${encodeURIComponent(symbol)}/strategies`),
  set: (accountID: string, symbol: string, strategies: Record<string, boolean>) =>
    request<StrategySwitchMatrix>(
      `/accounts/${accountID}/pairs/${encodeURIComponent(symbol)}/strategies`,
      { method: 'PUT', body: { strategies } },
    ),
}

/**
 * Saklar pasangan timeframe per akun × pair.
 *
 * Pasangan timeframe milik aturan trading ILA (pakem engine); endpoint ini hanya
 * mencatat mana yang dipantau dan mana yang dieksekusi otomatis.
 */
export const timeframeSwitchApi = {
  matrix: (symbol: string) =>
    request<TimeframeSwitchMatrix>(`/pairs/${encodeURIComponent(symbol)}/timeframes`),
  set: (
    accountID: string,
    symbol: string,
    strategy: StrategyVersion,
    pairs: TimeframeSwitchItem[],
  ) =>
    request<TimeframeSwitchMatrix>(
      `/accounts/${accountID}/pairs/${encodeURIComponent(symbol)}/timeframes`,
      { method: 'PUT', body: { strategy, pairs } },
    ),
}

export const subscriptionApi = {
  list: () => request<Subscription[]>('/subscriptions'),
  create: (payload: CreateSubscriptionPayload) =>
    request<Subscription>('/subscriptions', { method: 'POST', body: payload }),
  remove: (id: string) => request<void>(`/subscriptions/${id}`, { method: 'DELETE' }),
}

export const strategyApi = {
  list: () => request<Strategy[]>('/strategies'),
}

export const tradingApi = {
  signals: (limit = 50) => request<Signal[]>('/signals', { query: { limit } }),
  positions: (limit = 100) => request<Position[]>('/positions', { query: { limit } }),
  /**
   * Jurnal eksekusi: posisi, ringkasan, dan peluang yang terlewat.
   *
   * Rentang dikirim sebagai `days` bila pengguna tidak memilih tanggal,
   * supaya server yang menentukan batas hari pada zona hari akuntansi.
   */
  journal: (params: {
    days?: number
    from?: string
    to?: string
    strategy?: string
    symbol?: string
    limit?: number
    skippedLimit?: number
  }) =>
    request<JournalResponse>('/journal', {
      query: {
        days: params.days,
        from: params.from || undefined,
        to: params.to || undefined,
        strategy: params.strategy || undefined,
        symbol: params.symbol || undefined,
        limit: params.limit,
        skipped_limit: params.skippedLimit,
      },
    }),
}

/**
 * Data candle untuk chart dashboard.
 *
 * Feed tidak dikirim dari klien: server memakai feed milik pair yang dipilih
 * admin, lalu mengembalikan feed yang benar-benar dipakai.
 */
export const marketApi = {
  candles: (symbol: string, tf: string, limit = 200) =>
    request<MarketCandlesResponse>('/market/candles', {
      query: { symbol, tf, limit },
    }),
}

export const systemApi = {
  status: () => request<SystemStatus>('/system/status'),
}

export const legalApi = {
  me: () =>
    request<{
      documents: {
        slug: string
        title: string
        current_version: number
        accepted_version: number
        accepted: boolean
        require_reaccept: boolean
      }[]
      pending: { slug: string; title: string; required: boolean }[]
      blocked: boolean
    }>('/me/legal'),
  accept: (slug: string, version = 0) =>
    request<void>(`/me/legal/${encodeURIComponent(slug)}/accept`, {
      method: 'POST',
      body: { version },
    }),
}

export const trialApi = {
  get: () => request<Trial>('/trial'),
  start: () => request<Trial>('/trial', { method: 'POST' }),
}

/** Perangkat yang sedang masuk dan pencabutannya. */
export const sessionApi = {
  list: () => request<Session[]>('/me/sessions'),
  revoke: (id: string) => request<{ revoked: boolean }>(`/me/sessions/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  }),
}

/** Keamanan akun: ganti password dan cabut semua sesi. */
export const securityApi = {
  changePassword: (payload: { current_password: string; new_password: string }) =>
    request<{ re_login_required: boolean }>('/me/password', { method: 'POST', body: payload }),
  logoutAll: () => request<{ logged_out: boolean }>('/me/logout-all', { method: 'POST' }),
}

export const telegramApi = {
  link: () => request<TelegramLink>('/telegram/link', { method: 'POST' }),
  unlink: () => request<void>('/telegram/link', { method: 'DELETE' }),
}

/** Deep-link konfirmasi entry dari bot Telegram. */
export const approvalApi = {
  get: (token: string) => request<Approval>(`/approve/${encodeURIComponent(token)}`),
  confirm: (token: string) =>
    request<ApprovalResult>(`/approve/${encodeURIComponent(token)}/confirm`, {
      method: 'POST',
    }),
  decline: (token: string) =>
    request<ApprovalResult>(`/approve/${encodeURIComponent(token)}/decline`, {
      method: 'POST',
    }),
  /**
   * Menerbitkan token untuk tombol eksekusi di dashboard.
   *
   * Token sengaja tetap dipakai walau tombol berada di dalam aplikasi: dengan
   * begitu tombol dan tautan Telegram melewati alur yang sama, termasuk
   * pemeriksaan ulang seluruh syarat trading saat benar-benar ditekan.
   */
  issue: (signalID: string) =>
    request<ApprovalIssue>(`/approve/signals/${encodeURIComponent(signalID)}`, {
      method: 'POST',
    }),
}
