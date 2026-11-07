import { request } from './http'
import type {
  Approval,
  ApprovalResult,
  BrokerAccount,
  CreateAccountPayload,
  CreateSubscriptionPayload,
  Position,
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
