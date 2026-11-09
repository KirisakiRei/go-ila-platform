import { request, requestList } from './http'
import type { TimePair } from '@/types/api'
import type { PublicPlan } from '@/types/admin'
import type {
  AdminRole,
  AdminRoleDetail,
  AdminPermissionGroup,
  AdminAccountItem,
  AdminInvoiceItem,
  AdminTrialItem,
  AdminPairCatalog,
  AdminPairCategory,
  AdminPairGroup,
  AdminPairItem,
  CreditCodeItem,
  AdminLegalItem,
  AdminNoteItem,
  AdminPositionItem,
  AdminSignalItem,
  AdminSignalTrace,
  AdminUserDetail,
  AdminUserListItem,
  AuditCatalogEntry,
  AuditEntry,
  CouponItem,
  CouponReport,
  CreditSummary,
  CustomerReport,
  OperationReport,
  RevenueReport,
  UserActionPayload,
} from '@/types/admin'

/** Envelope daftar: data dan meta dikirim terpisah oleh backend. */

/** Lampiran sesi admin memakai cookie sendiri, jadi path berbeda dari sesi user. */
const ADMIN = '/admin'

/** Identitas admin yang dikirim server pada login dan profil. */
export type AdminIdentity = {
  id: string
  name: string
  email: string
  role: string
}

/** Hasil login admin: sesi terbit, atau perlu kode dua faktor. */
export type AdminLoginResult = {
  step?: 'complete' | 'two_factor_required'
  challenge_id?: string
  expires_in_seconds?: number
  expires_at: string
  user: AdminIdentity
  role_code: string
  permissions: string[]
}

/** Profil sesi admin beserta role RBAC-nya. */
export type AdminProfile = {
  user: AdminIdentity
  role: { code: string; name: string; is_superadmin: boolean }
  permissions: string[]
}

export const adminAuthApi = {
  /**
   * Masuk ke pintu admin. Server hanya menerima akun ber-role admin dan
   * menerbitkan cookie sesi admin yang terpisah dari sesi pelanggan.
   */
  login: (payload: { identifier: string; password: string }) =>
    request<AdminLoginResult>(`${ADMIN}/auth/login`, { method: 'POST', body: payload }),
  verify2fa: (payload: { challenge_id: string; code: string }) =>
    request<AdminLoginResult>(`${ADMIN}/auth/verify-2fa`, { method: 'POST', body: payload }),
  me: () => request<AdminProfile>(`${ADMIN}/auth/me`),
  /** Memperpanjang cookie sesi; dipanggil hanya saat ada interaksi nyata. */
  heartbeat: () =>
    request<{ expires_at: string; idle_minutes: number }>(`${ADMIN}/auth/heartbeat`, {
      method: 'POST',
      body: {},
    }),
  logout: () => request<void>(`${ADMIN}/auth/logout`, { method: 'POST', body: {} }),
}

export const adminUserApi = {
  list: (query: Record<string, string | number | boolean | undefined>) =>
    requestList<AdminUserListItem>(`${ADMIN}/users`, { query }),
  detail: (id: string) => request<AdminUserDetail>(`${ADMIN}/users/${id}`),
  accounts: (id: string) => request<AdminAccountItem[]>(`${ADMIN}/users/${id}/accounts`),
  positions: (id: string) => request<AdminPositionItem[]>(`${ADMIN}/users/${id}/positions`),
  legal: (id: string) => request<AdminLegalItem[]>(`${ADMIN}/users/${id}/legal`),
  notes: (id: string) => request<AdminNoteItem[]>(`${ADMIN}/users/${id}/notes`),
  addNote: (id: string, body: string) =>
    request<void>(`${ADMIN}/users/${id}/notes`, { method: 'POST', body: { body } }),

  suspend: (id: string, payload: UserActionPayload) =>
    request<void>(`${ADMIN}/users/${id}/suspend`, { method: 'POST', body: payload }),
  unsuspend: (id: string) =>
    request<void>(`${ADMIN}/users/${id}/unsuspend`, { method: 'POST', body: {} }),
  ban: (id: string, payload: UserActionPayload) =>
    request<void>(`${ADMIN}/users/${id}/ban`, { method: 'POST', body: payload }),
  unban: (id: string) => request<void>(`${ADMIN}/users/${id}/unban`, { method: 'POST', body: {} }),
  remove: (id: string, payload: UserActionPayload) =>
    request<void>(`${ADMIN}/users/${id}/delete`, { method: 'POST', body: payload }),
  changeTier: (id: string, payload: { tier: string; reason: string; days?: number }) =>
    request<void>(`${ADMIN}/users/${id}/tier`, { method: 'PATCH', body: payload }),

  freezeAccount: (accountID: string, reason: string) =>
    request<void>(`${ADMIN}/accounts/${accountID}/freeze`, { method: 'POST', body: { reason } }),
  unfreezeAccount: (accountID: string) =>
    request<void>(`${ADMIN}/accounts/${accountID}/unfreeze`, { method: 'POST', body: {} }),

  credit: (id: string) => request<CreditSummary>(`${ADMIN}/users/${id}/credit`),
  adjustCredit: (id: string, payload: { amount_idr: number; reason: string }) =>
    request<CreditSummary>(`${ADMIN}/users/${id}/credit`, { method: 'POST', body: payload }),
}

export const adminAuditApi = {
  list: (query: Record<string, string | number | boolean | undefined>) =>
    requestList<AuditEntry>(`${ADMIN}/audit`, { query }),
  detail: (id: string) => request<AuditEntry>(`${ADMIN}/audit/${id}`),
  catalog: () => request<AuditCatalogEntry[]>(`${ADMIN}/audit-catalog`),
}

/**
 * RBAC: role, katalog permission, dan penugasan role ke akun admin.
 *
 * PATCH role mengirim SELURUH daftar permission, bukan selisih, karena server
 * mengganti isi role secara utuh. Mengirim tambahan saja akan mencabut yang
 * tidak disebut.
 */
export const adminRoleApi = {
  list: () => request<AdminRole[]>(`${ADMIN}/roles`),
  detail: (id: string) => request<AdminRoleDetail>(`${ADMIN}/roles/${id}`),
  permissions: () => request<AdminPermissionGroup[]>(`${ADMIN}/permissions`),
  create: (payload: { code: string; name: string; description?: string; permissions: string[] }) =>
    request<AdminRole>(`${ADMIN}/roles`, { method: 'POST', body: payload }),
  updatePermissions: (id: string, permissions: string[]) =>
    request<AdminRole>(`${ADMIN}/roles/${id}`, { method: 'PATCH', body: { permissions } }),
  remove: (id: string) => request<void>(`${ADMIN}/roles/${id}`, { method: 'DELETE' }),
  assign: (userId: string, roleId: string | null) =>
    request<void>(`${ADMIN}/users/${userId}/role`, { method: 'PATCH', body: { role_id: roleId } }),
}

/**
 * Katalog pair dikelola admin: kategori → grup → pair.
 *
 * Pair yang dipakai user pada halaman Pair Pantauan hanya boleh berasal dari
 * katalog ini, sehingga penulisan simbol bebas tidak pernah terjadi.
 */
export const adminPairApi = {
  catalog: () => request<AdminPairCatalog>(`${ADMIN}/pairs`),

  createPair: (payload: Record<string, unknown>) =>
    request<AdminPairItem>(`${ADMIN}/pairs`, { method: 'POST', body: payload }),
  updatePair: (id: string, payload: Record<string, unknown>) =>
    request<void>(`${ADMIN}/pairs/${id}`, { method: 'PATCH', body: payload }),
  deletePair: (id: string) => request<void>(`${ADMIN}/pairs/${id}`, { method: 'DELETE' }),

  createCategory: (payload: Record<string, unknown>) =>
    request<AdminPairCategory>(`${ADMIN}/pair-categories`, { method: 'POST', body: payload }),
  updateCategory: (id: string, payload: Record<string, unknown>) =>
    request<void>(`${ADMIN}/pair-categories/${id}`, { method: 'PATCH', body: payload }),
  deleteCategory: (id: string) =>
    request<void>(`${ADMIN}/pair-categories/${id}`, { method: 'DELETE' }),

  createGroup: (payload: Record<string, unknown>) =>
    request<AdminPairGroup>(`${ADMIN}/pair-groups`, { method: 'POST', body: payload }),
  updateGroup: (id: string, payload: Record<string, unknown>) =>
    request<void>(`${ADMIN}/pair-groups/${id}`, { method: 'PATCH', body: payload }),
  deleteGroup: (id: string) => request<void>(`${ADMIN}/pair-groups/${id}`, { method: 'DELETE' }),
}

export const adminPlanApi = {
  list: () => request<PublicPlan[]>(`${ADMIN}/plans`),
  update: (code: string, payload: Record<string, unknown>) =>
    request<void>(`${ADMIN}/plans/${code}`, { method: 'PATCH', body: payload }),
  setGuard: (code: string, payload: { min_price_idr: number; reason: string }) =>
    request<void>(`${ADMIN}/plans/${code}/guard`, { method: 'PUT', body: payload }),
}

export const adminCouponApi = {
  list: (query?: Record<string, string | undefined>) =>
    request<CouponItem[]>(`${ADMIN}/coupons`, { query }),
  create: (payload: Record<string, unknown>) =>
    request<CouponItem>(`${ADMIN}/coupons`, { method: 'POST', body: payload }),
  update: (id: string, payload: Record<string, unknown>) =>
    request<void>(`${ADMIN}/coupons/${id}`, { method: 'PATCH', body: payload }),
  disable: (id: string, reason: string) =>
    request<void>(`${ADMIN}/coupons/${id}/disable`, { method: 'POST', body: { reason } }),
}

export const adminInvoiceApi = {
  list: (query?: Record<string, string | undefined>) =>
    request<AdminInvoiceItem[]>(`${ADMIN}/invoices`, { query }),
  markPaid: (id: string, payload: { amount_idr: number; note: string }) =>
    request<void>(`${ADMIN}/invoices/${id}/mark-paid`, { method: 'POST', body: payload }),
  refund: (
    id: string,
    payload: { payment_id: string; amount_idr: number; reason: string; gateway_ref?: string },
  ) => request<void>(`${ADMIN}/invoices/${id}/refund`, { method: 'POST', body: payload }),
}

export const adminCreditCodeApi = {
  list: (query?: Record<string, string | undefined>) =>
    request<CreditCodeItem[]>(`${ADMIN}/credit-codes`, { query }),
  create: (payload: Record<string, unknown>) =>
    request<CreditCodeItem>(`${ADMIN}/credit-codes`, { method: 'POST', body: payload }),
  disable: (id: string, reason: string) =>
    request<void>(`${ADMIN}/credit-codes/${id}/disable`, { method: 'POST', body: { reason } }),
}

export const adminTrialApi = {
  list: (query?: Record<string, string | undefined>) =>
    request<AdminTrialItem[]>(`${ADMIN}/trials`, { query }),
  extend: (id: string, payload: { days: number; reason: string }) =>
    request<AdminTrialItem>(`${ADMIN}/trials/${id}/extend`, { method: 'POST', body: payload }),
  end: (id: string, reason: string) =>
    request<void>(`${ADMIN}/trials/${id}/end`, {
      method: 'POST',
      body: { reason, category: 'other' },
    }),
}

/**
 * Panel sinyal admin: daftar sinyal terbit dan telusur perjalanannya.
 *
 * Rentang tanggal wajib agar tabel sinyal tidak dipindai seluruhnya; server
 * memakai bawaan 7 hari bila rentang tidak disebut.
 */
export const adminSignalApi = {
  list: (query: Record<string, string | number | boolean | undefined>) =>
    requestList<AdminSignalItem>(`${ADMIN}/signals`, { query }),
  trace: (id: string) => request<AdminSignalTrace>(`${ADMIN}/signals/${id}/trace`),
}

export const adminReportApi = {
  revenue: (from?: string, to?: string) =>
    request<RevenueReport>(`${ADMIN}/reports/revenue`, { query: { from, to } }),
  customers: (from?: string, to?: string) =>
    request<CustomerReport>(`${ADMIN}/reports/customers`, { query: { from, to } }),
  coupons: (from?: string, to?: string) =>
    request<CouponReport>(`${ADMIN}/reports/coupons`, { query: { from, to } }),
  operations: (from?: string, to?: string) =>
    request<OperationReport>(`${ADMIN}/reports/operations`, { query: { from, to } }),
}

/** Paket publik untuk halaman harga; tanpa sesi. */
export const publicPlanApi = {
  list: () => request<PublicPlan[]>('/billing/plans'),
}

export type { TimePair }
