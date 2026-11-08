import { request } from './http'
import type {
  AccountPairSelectionResponse,
  Approval,
  ApprovalResult,
  BrokerAccount,
  CreateAccountPayload,
  CreateSubscriptionPayload,
  PairCatalogResponse,
  Position,
  Session,
  Signal,
  Strategy,
  Subscription,
  SystemStatus,
  TelegramLink,
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
}

export const systemApi = {
  status: () => request<SystemStatus>('/system/status'),
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
}
