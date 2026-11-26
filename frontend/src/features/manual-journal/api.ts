import { downloadFile, request } from '@/lib/http'
import { compressChartImage } from './image'
import type {
  AccountInput,
  CapitalInput,
  CapitalMovement,
  CSVPreview,
  CSVConfirm,
  JournalAccount,
  JournalReview,
  JournalReviewSettings,
  JournalFilters,
  JournalInsight,
  JournalMeta,
  JournalScreenshot,
  JournalStats,
  JournalTrade,
  TradeInput,
} from './types'

function normalizeTrade(trade: JournalTrade): JournalTrade {
  return {
    ...trade,
    tags: trade.tags ?? [],
    emotions: trade.emotions ?? [],
    mistakes: trade.mistakes ?? [],
    checklist: trade.checklist ?? [],
    tradingview_urls: trade.tradingview_urls ?? [],
  }
}
const root = '/manual-journal'
type List<T> = { data: T[]; meta?: JournalMeta }

async function list<T>(
  path: string,
  query: Record<string, string | number | undefined>,
  signal?: AbortSignal,
): Promise<List<T>> {
  // The shared transport unwraps data; fetch bounded pages and reject rather than silently truncate.
  const data: T[] = []
  for (let page = 1; page <= 101; page++) {
    const batch = await request<T[]>(path, { query: { ...query, page, per_page: 100 }, signal })
    if (data.length + batch.length > 10000)
      throw new Error('Lebih dari 10.000 catatan. Persempit periode atau pilih jurnal.')
    data.push(...batch)
    if (batch.length < 100) return { data }
  }
  throw new Error('Terlalu banyak catatan. Persempit filter.')
}
export const journalApi = {
  review: (account_id: string, signal?: AbortSignal) =>
    request<JournalReview>(`${root}/review`, { query: { account_id }, signal }),
  reviewSettings: (account_id?: string, signal?: AbortSignal) =>
    request<JournalReviewSettings>(`${root}/review/settings`, { query: { account_id }, signal }),
  saveReviewSettings: (ai_enabled: boolean, account_id?: string) =>
    request<JournalReviewSettings>(`${root}/review/settings`, {
      method: 'PUT',
      query: { account_id },
      body: { ai_enabled },
    }),
  accounts: async (signal?: AbortSignal) => {
    const result = await list<JournalAccount>(`${root}/accounts`, {}, signal)
    return {
      ...result,
      data: result.data.map((account) => ({ ...account, checklist: account.checklist ?? [] })),
    }
  },
  saveAccount: (input: AccountInput, id?: string) =>
    request<JournalAccount>(`${root}/accounts${id ? `/${id}` : ''}`, {
      method: id ? 'PUT' : 'POST',
      body: {
        name: input.name,
        broker: input.broker,
        currency: input.currency,
        initial_capital: input.initial_capital,
        initial_date: input.initial_date,
        notes: input.notes,
        archived: input.archived,
        checklist: input.checklist,
      },
    }),
  archiveAccount: (id: string) => request<void>(`${root}/accounts/${id}`, { method: 'DELETE' }),
  deleteAccount: (id: string) =>
    request<void>(`${root}/accounts/${id}`, { method: 'DELETE', query: { permanent: 'true' } }),
  entries: async (filters: JournalFilters, signal?: AbortSignal) => {
    const result = await list<JournalTrade>(`${root}/entries`, { ...filters }, signal)
    return { ...result, data: result.data.map(normalizeTrade) }
  },
  saveEntry: async (input: TradeInput, id?: string) =>
    normalizeTrade(
      await request<JournalTrade>(`${root}/entries${id ? `/${id}` : ''}`, {
        method: id ? 'PUT' : 'POST',
        body: Object.fromEntries(
          [
            'account_id',
            'status',
            'asset',
            'side',
            'opened_at',
            'closed_at',
            'entry_price',
            'stop_loss',
            'take_profit',
            'exit_price',
            'size',
            'size_unit',
            'net_pnl',
            'risk_amount',
            'capital_snapshot',
            'setup',
            'tags',
            'emotions',
            'mistakes',
            'session',
            'checklist',
            'notes',
            'tradingview_urls',
          ].map((key) => [key, input[key as keyof TradeInput]]),
        ),
      }),
    ),
  deleteEntry: (id: string) => request<void>(`${root}/entries/${id}`, { method: 'DELETE' }),
  capital: (account_id: string, signal?: AbortSignal) =>
    list<CapitalMovement>(`${root}/capital`, { account_id }, signal),
  saveCapital: (input: CapitalInput, id?: string) =>
    request<CapitalMovement>(`${root}/capital${id ? `/${id}` : ''}`, {
      method: id ? 'PUT' : 'POST',
      body: {
        account_id: input.account_id,
        kind: input.kind,
        amount: input.amount,
        date: input.date,
        notes: input.notes,
      },
    }),
  deleteCapital: (id: string) => request<void>(`${root}/capital/${id}`, { method: 'DELETE' }),
  stats: (filters: JournalFilters, signal?: AbortSignal) =>
    request<JournalStats>(`${root}/stats`, { query: { ...filters }, signal }),
  insights: (filters: JournalFilters, signal?: AbortSignal) =>
    request<JournalInsight[]>(`${root}/insights`, { query: { ...filters }, signal }),
  preview: (csv: string) =>
    request<CSVPreview>(`${root}/import/preview`, { method: 'POST', body: { csv } }),
  import: (input: CSVConfirm) =>
    request<{ count: number }>(`${root}/import/confirm`, { method: 'POST', body: input }),
  export: (filters: JournalFilters) => downloadFile(`${root}/export`, { ...filters }),
  screenshots: (id: string) => request<JournalScreenshot[]>(`${root}/entries/${id}/screenshots`),
  deleteScreenshot: (id: string, sid: string) =>
    request<void>(`${root}/entries/${id}/screenshots/${sid}`, { method: 'DELETE' }),
  screenshotUrl: (id: string, sid: string) =>
    `/api/v1${root}/entries/${encodeURIComponent(id)}/screenshots/${encodeURIComponent(sid)}`,
  async uploadScreenshot(id: string, file: File) {
    const ready = await compressChartImage(file)
    const body = new FormData()
    body.append('file', ready)
    return request<JournalScreenshot>(`${root}/entries/${encodeURIComponent(id)}/screenshots`, {
      method: 'POST',
      body,
      timeoutMs: 30000,
    })
  },
}
