import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  accountApi,
  approvalApi,
  backtestApi,
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
  BacktestCreatePayload,
  CreateAccountPayload,
  CreateSubscriptionPayload,
  JournalEvalStatus,
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
  backtestCoverage: (symbol: string) => ['backtest', 'coverage', symbol] as const,
  backtestQuota: ['backtest', 'quota'] as const,
  backtestRun: (id: string) => ['backtest', 'run', id] as const,
  backtestTrades: (id: string) => ['backtest', 'trades', id] as const,
  backtestTrade: (id: string, seq: number) => ['backtest', 'trade', id, seq] as const,
  backtestHistory: ['backtest', 'history'] as const,
  /** Satu halaman riwayat. Awalan yang sama membuat penghapusan run menyegarkan semua halaman. */
  backtestHistoryPage: (limit: number, offset: number) =>
    ['backtest', 'history', limit, offset] as const,
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
  evalStatus?: JournalEvalStatus | ''
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

/**
 * Cakupan data historis satu pair.
 *
 * Dibaca saat pengguna memilih pair, bukan saat halaman dibuka: pemeriksaan
 * pertama menghubungi feed empat kali sehingga tidak boleh berjalan untuk
 * setiap pair di katalog sekaligus. Server menyimpan hasilnya sampai tengah
 * malam WIB, jadi membuka pair yang sama lagi hampir seketika.
 */
export function useBacktestCoverage(symbol: string | null) {
  return useQuery({
    queryKey: queryKeys.backtestCoverage(symbol ?? ''),
    queryFn: () => backtestApi.coverage(symbol ?? ''),
    enabled: Boolean(symbol),
    staleTime: 30 * 60_000,
    retry: 1,
    placeholderData: keepPreviousData,
  })
}

/** Sisa jatah backtest hari ini. */
export function useBacktestQuota() {
  return useQuery({
    queryKey: queryKeys.backtestQuota,
    queryFn: backtestApi.quota,
    staleTime: 60_000,
  })
}

/**
 * Satu run backtest. Selama statusnya belum selesai, hook ini memeriksa ulang
 * secara berkala; setelah selesai, polling berhenti sendiri.
 */
export function useBacktestRun(id: string | null) {
  return useQuery({
    queryKey: queryKeys.backtestRun(id ?? ''),
    queryFn: () => backtestApi.detail(id ?? ''),
    enabled: Boolean(id),
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status === 'queued' || status === 'running' ? 2_000 : false
    },
  })
}

/** Daftar entry sebuah run. */
export function useBacktestTrades(id: string | null, aktif: boolean) {
  return useQuery({
    queryKey: queryKeys.backtestTrades(id ?? ''),
    queryFn: async () => (await backtestApi.trades(id ?? '', 200, 0)).data,
    enabled: Boolean(id) && aktif,
    staleTime: 5 * 60_000,
  })
}

/** Detail satu entry beserta candle dan levelnya untuk chart. */
export function useBacktestTrade(id: string | null, seq: number | null) {
  return useQuery({
    queryKey: queryKeys.backtestTrade(id ?? '', seq ?? 0),
    queryFn: () => backtestApi.trade(id ?? '', seq ?? 0),
    enabled: Boolean(id) && Boolean(seq) && (seq ?? 0) > 0,
    staleTime: 10 * 60_000,
  })
}

/**
 * Riwayat run milik pengguna.
 *
 * Halaman dan jumlah per halaman ikut jadi bagian kunci query: berpindah
 * halaman harus memuat data lain, dan halaman yang pernah dibuka tetap
 * tersimpan di cache supaya kembali ke sana tidak meminta ulang ke server.
 */
export function useBacktestHistory(limit = 20, offset = 0) {
  return useQuery({
    queryKey: queryKeys.backtestHistoryPage(limit, offset),
    queryFn: async () => {
      const balasan = await backtestApi.history(limit, offset)
      return { daftar: balasan.data, total: balasan.meta?.total ?? balasan.data.length }
    },
    staleTime: 30_000,
  })
}

/** Membuat backtest baru. */
export function useCreateBacktest() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (payload: BacktestCreatePayload) => backtestApi.create(payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.backtestQuota })
      void queryClient.invalidateQueries({ queryKey: queryKeys.backtestHistory })
    },
  })
}

/** Menghapus satu run. */
export function useDeleteBacktest() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => backtestApi.remove(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.backtestHistory })
      // Run yang dihapus tidak boleh terlihat lagi di halaman detailnya.
      void queryClient.invalidateQueries({ queryKey: ['backtest', 'run'] })
    },
  })
}
