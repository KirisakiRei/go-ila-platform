import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  accountApi,
  approvalApi,
  marketApi,
  pairApi,
  riskApi,
  securityApi,
  sessionApi,
  strategyApi,
  strategySwitchApi,
  timeframeSwitchApi,
  subscriptionApi,
  systemApi,
  telegramApi,
  trialApi,
} from '@/lib/api'
import { tradingApi } from '@/lib/api'
import type {
  CreateAccountPayload,
  CreateSubscriptionPayload,
  StrategyVersion,
  TimeframeSwitchItem,
  UpdateAccountPayload,
} from '@/types/api'

/** Kunci query dipusatkan agar invalidasi tidak salah sasaran. */
export const queryKeys = {
  accounts: ['accounts'] as const,
  subscriptions: ['subscriptions'] as const,
  strategies: ['strategies'] as const,
  strategySwitch: (symbol: string) => ['strategy-switch', symbol] as const,
  timeframeSwitch: (symbol: string) => ['timeframe-switch', symbol] as const,
  signals: (limit: number) => ['signals', limit] as const,
  marketCandles: (symbol: string, timeframe: string, limit: number) =>
    ['market', 'candles', symbol, timeframe, limit] as const,
  positions: (limit: number) => ['positions', limit] as const,
  journal: (params: JournalParams) => ['journal', params] as const,
  systemStatus: ['system', 'status'] as const,
  trial: ['trial'] as const,
  sessions: ['me', 'sessions'] as const,
  accountPairs: (id: string) => ['accounts', id, 'pairs'] as const,
  pairCatalog: ['pairs', 'catalog'] as const,
}

export function useAccounts() {
  return useQuery({ queryKey: queryKeys.accounts, queryFn: accountApi.list })
}

export function useCreateAccount() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateAccountPayload) => accountApi.create(payload),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.accounts })
      void qc.invalidateQueries({ queryKey: queryKeys.systemStatus })
    },
  })
}

export function useUpdateAccount() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: UpdateAccountPayload }) =>
      accountApi.update(id, payload),
    onSuccess: () => void qc.invalidateQueries({ queryKey: queryKeys.accounts }),
  })
}

export function useDeleteAccount() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => accountApi.remove(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.accounts })
      void qc.invalidateQueries({ queryKey: queryKeys.systemStatus })
    },
  })
}

export function useAccountPairs(accountId: string) {
  return useQuery({
    queryKey: queryKeys.accountPairs(accountId),
    queryFn: () => accountApi.pairs(accountId),
    enabled: Boolean(accountId),
  })
}

export function useUpdateAccountPairs() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, symbols }: { id: string; symbols: string[] }) =>
      accountApi.updatePairs(id, symbols),
    onSuccess: (_, variables) => {
      void qc.invalidateQueries({ queryKey: queryKeys.accountPairs(variables.id) })
      void qc.invalidateQueries({ queryKey: queryKeys.subscriptions })
      void qc.invalidateQueries({ queryKey: queryKeys.accounts })
    },
  })
}

export function usePairCatalog() {
  return useQuery({
    queryKey: queryKeys.pairCatalog,
    queryFn: pairApi.catalog,
  })
}

/**
 * useRiskSimulation menghitung pratinjau risiko di server.
 *
 * Sengaja berupa mutation, bukan query: hasilnya bergantung pada isian formulir
 * yang belum tentu valid, dan tidak perlu di-cache.
 */
export function useRiskSimulation() {
  return useMutation({ mutationFn: riskApi.simulate })
}

/** Menerbitkan token konfirmasi untuk tombol eksekusi di dashboard. */
export function useIssueApproval() {
  return useMutation({ mutationFn: (signalID: string) => approvalApi.issue(signalID) })
}

/**
 * useConfirmApproval mengeksekusi sinyal lewat token.
 *
 * Seluruh syarat trading diperiksa ulang di Runner, jadi penolakan bisa muncul
 * di sini walau tombol sempat aktif.
 */
export function useConfirmApproval() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (token: string) => approvalApi.confirm(token),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.signals(100) })
      void qc.invalidateQueries({ queryKey: queryKeys.positions(100) })
    },
  })
}

/** useDeclineApproval menandai sinyal dilewati user. */
export function useDeclineApproval() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (token: string) => approvalApi.decline(token),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.signals(100) })
    },
  })
}

/** Matriks saklar pasangan timeframe per akun untuk satu pair. */
export function useTimeframeSwitchMatrix(symbol: string | null) {
  return useQuery({
    queryKey: queryKeys.timeframeSwitch(symbol ?? ''),
    queryFn: () => timeframeSwitchApi.matrix(symbol ?? ''),
    enabled: Boolean(symbol),
  })
}

/**
 * useSetTimeframeSwitch mengubah saklar satu akun untuk satu metode.
 *
 * Respons server berisi matriks terbaru, jadi cache langsung ditimpa agar
 * perubahan saklar terasa seketika.
 */
export function useSetTimeframeSwitch(symbol: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (vars: {
      accountID: string
      strategy: StrategyVersion
      pairs: TimeframeSwitchItem[]
    }) => timeframeSwitchApi.set(vars.accountID, symbol, vars.strategy, vars.pairs),
    onSuccess: (data) => {
      qc.setQueryData(queryKeys.timeframeSwitch(symbol), data)
    },
  })
}

/** Matriks saklar metode per akun untuk satu pair. */
export function useStrategySwitchMatrix(symbol: string | null) {
  return useQuery({
    queryKey: queryKeys.strategySwitch(symbol ?? ''),
    queryFn: () => strategySwitchApi.matrix(symbol ?? ''),
    enabled: Boolean(symbol),
  })
}

/**
 * useSetStrategySwitch mengubah saklar satu akun lalu menyegarkan matriks.
 *
 * Respons server sudah berisi matriks terbaru, jadi cache langsung ditimpa agar
 * saklar terasa seketika tanpa menunggu permintaan ulang.
 */
export function useSetStrategySwitch(symbol: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (vars: { accountID: string; strategies: Record<string, boolean> }) =>
      strategySwitchApi.set(vars.accountID, symbol, vars.strategies),
    onSuccess: (data) => {
      qc.setQueryData(queryKeys.strategySwitch(symbol), data)
    },
  })
}

export function useCTraderCallback() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (payload: { code: string; mode?: string; risk_percent?: number }) =>
      accountApi.ctraderCallback(payload),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.accounts })
      void qc.invalidateQueries({ queryKey: queryKeys.systemStatus })
    },
  })
}

export function useSubscriptions() {
  return useQuery({ queryKey: queryKeys.subscriptions, queryFn: subscriptionApi.list })
}

export function useCreateSubscription() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateSubscriptionPayload) => subscriptionApi.create(payload),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.subscriptions })
      void qc.invalidateQueries({ queryKey: queryKeys.strategies })
    },
  })
}

export function useDeleteSubscription() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => subscriptionApi.remove(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.subscriptions })
      void qc.invalidateQueries({ queryKey: queryKeys.strategies })
    },
  })
}

export function useStrategies() {
  return useQuery({ queryKey: queryKeys.strategies, queryFn: strategyApi.list })
}

export function useSignals(limit = 50) {
  return useQuery({
    queryKey: queryKeys.signals(limit),
    queryFn: () => tradingApi.signals(limit),
    refetchInterval: 15_000,
    // Daftar tetap tampil selama penyegaran berkala; tanpa ini setiap polling
    // sempat mengosongkan data dan memicu kedipan.
    placeholderData: keepPreviousData,
  })
}

/**
 * JournalParams adalah kunci permintaan jurnal.
 *
 * Dipakai juga sebagai bagian kunci cache, sehingga setiap kombinasi filter
 * mendapat hasilnya sendiri tanpa saling menimpa.
 */
export type JournalParams = {
  days?: number
  from?: string
  to?: string
  strategy?: string
  symbol?: string
  /** Halaman daftar peluang terlewat, dimulai dari 1. */
  skippedPage?: number
  /** Jumlah baris peluang per halaman. */
  skippedLimit?: number
}

/**
 * useJournal membaca jurnal eksekusi.
 *
 * Jurnal menyegarkan diri lebih lambat daripada daftar posisi: isinya
 * ringkasan dan riwayat, dan memuat ulang terlalu sering hanya menambah beban
 * tanpa informasi baru.
 */
export function useJournal(params: JournalParams) {
  return useQuery({
    queryKey: queryKeys.journal(params),
    queryFn: () =>
      tradingApi.journal({
        ...params,
        limit: 100,
        skippedLimit: params.skippedLimit ?? 20,
        skippedPage: params.skippedPage,
      }),
    refetchInterval: 60_000,
    placeholderData: keepPreviousData,
  })
}

export function usePositions(limit = 100) {
  return useQuery({
    queryKey: queryKeys.positions(limit),
    queryFn: () => tradingApi.positions(limit),
    refetchInterval: 30_000,
    placeholderData: keepPreviousData,
  })
}

/**
 * Candle untuk chart dashboard.
 *
 * Segarkan berkala dipakai sebagai pengganti streaming: cache server sudah
 * menahan beban feed, sehingga interval pendek tetap ringan.
 */
export function useMarketCandles(
  symbol: string | null,
  timeframe: string,
  limit = 200,
  enabled = true,
) {
  return useQuery({
    queryKey: queryKeys.marketCandles(symbol ?? '', timeframe, limit),
    queryFn: () => marketApi.candles(symbol ?? '', timeframe, limit),
    enabled: Boolean(symbol) && enabled,
    staleTime: 8_000,
    refetchInterval: 20_000,
    // Pertahankan candle sebelumnya saat timeframe berganti, supaya chart tidak
    // berkedip kembali ke skeleton di tengah pemakaian.
    placeholderData: keepPreviousData,
    retry: 1,
  })
}

export function useSystemStatus() {
  return useQuery({
    queryKey: queryKeys.systemStatus,
    queryFn: systemApi.status,
    refetchInterval: 30_000,
  })
}

export function useTrial() {
  return useQuery({ queryKey: queryKeys.trial, queryFn: trialApi.get })
}

export function useStartTrial() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: trialApi.start,
    onSuccess: (trial) => {
      qc.setQueryData(queryKeys.trial, trial)
      void qc.invalidateQueries({ queryKey: queryKeys.strategies })
    },
  })
}

export function useSessions() {
  return useQuery({ queryKey: queryKeys.sessions, queryFn: sessionApi.list })
}

export function useRevokeSession() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: sessionApi.revoke,
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.sessions }),
  })
}

export function useChangePassword() {
  return useMutation({ mutationFn: securityApi.changePassword })
}

export function useLogoutAll() {
  return useMutation({ mutationFn: securityApi.logoutAll })
}

export function useTelegramLink() {
  return useMutation({ mutationFn: telegramApi.link })
}

export function useTelegramUnlink() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: telegramApi.unlink,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['me', 'onboarding'] })
      queryClient.invalidateQueries({ queryKey: ['trial'] })
    },
  })
}
