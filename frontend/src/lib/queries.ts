import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  accountApi,
  pairApi,
  riskApi,
  securityApi,
  sessionApi,
  strategyApi,
  strategySwitchApi,
  subscriptionApi,
  systemApi,
  telegramApi,
  trialApi,
} from '@/lib/api'
import { tradingApi } from '@/lib/api'
import type {
  CreateAccountPayload,
  CreateSubscriptionPayload,
  UpdateAccountPayload,
} from '@/types/api'

/** Kunci query dipusatkan agar invalidasi tidak salah sasaran. */
export const queryKeys = {
  accounts: ['accounts'] as const,
  subscriptions: ['subscriptions'] as const,
  strategies: ['strategies'] as const,
  strategySwitch: (symbol: string) => ['strategy-switch', symbol] as const,
  signals: (limit: number) => ['signals', limit] as const,
  positions: (limit: number) => ['positions', limit] as const,
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
    refetchInterval: 30_000,
  })
}

export function usePositions(limit = 100) {
  return useQuery({
    queryKey: queryKeys.positions(limit),
    queryFn: () => tradingApi.positions(limit),
    refetchInterval: 30_000,
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
  return useMutation({ mutationFn: telegramApi.unlink })
}
