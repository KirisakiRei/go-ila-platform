import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { journalApi } from './api'
import type { JournalFilters } from './types'

export const journalKey = ['manual-journal'] as const
export const reviewPollInterval = (review?: { status: string }) =>
  review?.status === 'pending' || review?.status === 'running'
    ? 5000
    : review?.status === 'ready'
      ? 60000
      : false
export function useJournalReview(accountId: string) {
  return useQuery({
    queryKey: [...journalKey, 'review', accountId],
    queryFn: ({ signal }) => journalApi.review(accountId, signal),
    enabled: !!accountId,
    refetchInterval: (query) => reviewPollInterval(query.state.data),
  })
}
export function useJournalReviewSettings(accountId?: string) {
  return useQuery({
    queryKey: [...journalKey, 'review-settings', accountId ?? 'global'],
    queryFn: ({ signal }) => journalApi.reviewSettings(accountId, signal),
  })
}
export function useJournalAccounts() {
  return useQuery({
    queryKey: [...journalKey, 'accounts'],
    queryFn: ({ signal }) => journalApi.accounts(signal),
  })
}
export function useJournalEntries(filters: JournalFilters, enabled: boolean) {
  return useQuery({
    queryKey: [...journalKey, 'entries', filters],
    queryFn: ({ signal }) => journalApi.entries(filters, signal),
    enabled,
  })
}
export function useJournalStats(filters: JournalFilters, enabled: boolean) {
  return useQuery({
    queryKey: [...journalKey, 'stats', filters],
    queryFn: ({ signal }) => journalApi.stats(filters, signal),
    enabled,
  })
}
export function useJournalInsights(filters: JournalFilters, enabled: boolean) {
  return useQuery({
    queryKey: [...journalKey, 'insights', filters],
    queryFn: ({ signal }) => journalApi.insights(filters, signal),
    enabled,
  })
}
export function useJournalMutation<T, R>(mutationFn: (variables: T) => Promise<R>) {
  const client = useQueryClient()
  return useMutation({
    mutationKey: journalKey,
    mutationFn,
    onSuccess: () => client.invalidateQueries({ queryKey: journalKey }),
  })
}
