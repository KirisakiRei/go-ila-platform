import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  accountApi,
  securityApi,
  sessionApi,
  strategyApi,
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
  signals: (limit: number) => ['signals', limit] as const,
  positions: (limit: number) => ['positions', limit] as const,
  systemStatus: ['system', 'status'] as const,
  trial: ['trial'] as const,
  sessions: ['me', 'sessions'] as const,
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
