import type { JournalFilters, JournalTrade, TradeInput } from './types'

export function formatAmount(value: number | string, currency: string) {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: /^[A-Z]{3}$/.test(currency) ? currency : 'USD',
    maximumFractionDigits: 4,
  }).format(Number(value))
}

export const wibDay = (date: string) =>
  new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Jakarta',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(new Date(date))
export const wibTime = (date: string) =>
  new Intl.DateTimeFormat('id-ID', {
    timeZone: 'Asia/Jakarta',
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(date))
export const localWib = (date: string | null) =>
  date ? new Date(new Date(date).getTime() + 7 * 3600000).toISOString().slice(0, 16) : ''
export const fromWib = (date: string) => (date ? new Date(`${date}:00+07:00`).toISOString() : null)
export const tradeDay = (trade: JournalTrade) =>
  wibDay((trade.status === 'closed' ? trade.closed_at : trade.opened_at) || trade.created_at)
export const splitList = (text: string) => [
  ...new Set(
    text
      .split(/\r?\n/)
      .map((s) => s.trim())
      .filter(Boolean),
  ),
]
export const safeLink = (url: string) => {
  try {
    const parsed = new URL(url)
    return (
      parsed.protocol === 'https:' &&
      !parsed.port &&
      !parsed.username &&
      !parsed.password &&
      (parsed.hostname === 'tradingview.com' || parsed.hostname.endsWith('.tradingview.com'))
    )
  } catch {
    return false
  }
}
export function filterTrades(trades: JournalTrade[], filters: JournalFilters) {
  return trades.filter((t) => {
    const day = tradeDay(t)
    return (
      (!filters.account_id || t.account_id === filters.account_id) &&
      (!filters.status || t.status === filters.status) &&
      (!filters.from || day >= filters.from) &&
      (!filters.to || day <= filters.to) &&
      t.asset.toLowerCase().includes(filters.asset.toLowerCase()) &&
      t.setup.toLowerCase().includes(filters.setup.toLowerCase()) &&
      (!filters.tag ||
        t.tags.some((tag) => tag.toLowerCase().includes(filters.tag.toLowerCase()))) &&
      (!filters.emotion || t.emotions.some((emotion) => emotion === filters.emotion)) &&
      (!filters.mistake || t.mistakes.some((mistake) => mistake === filters.mistake))
    )
  })
}
export function validateDecimal(value: string, decimals = 4, signed = false) {
  return (
    new RegExp(`^${signed ? '-?' : ''}\\d+(?:\\.\\d{1,${decimals}})?$`).test(value) &&
    Math.abs(Number(value)) <= 1_000_000_000
  )
}
export function validateTrade(trade: TradeInput) {
  if (!trade.account_id) return 'Pilih akun jurnal terlebih dahulu.'
  if (
    trade.status !== 'draft' &&
    (!trade.asset.trim() ||
      !trade.side ||
      !trade.opened_at ||
      trade.entry_price === null ||
      trade.size === null ||
      !trade.size_unit.trim())
  )
    return 'Trade open/closed membutuhkan simbol, arah, waktu masuk, harga entry, size dan satuannya.'
  if (trade.status === 'closed' && (!trade.closed_at || trade.net_pnl === null))
    return 'Trade closed membutuhkan waktu keluar dan net P&L aktual.'
  if (trade.status !== 'closed' && (trade.closed_at || trade.net_pnl !== null))
    return 'Waktu keluar dan net P&L hanya diisi pada trade closed.'
  for (const date of [trade.opened_at, trade.closed_at]) {
    if (
      date &&
      (!Number.isFinite(Date.parse(date)) ||
        new Date(date).getUTCFullYear() < 1900 ||
        new Date(date).getUTCFullYear() > 2200)
    )
      return 'Waktu harus valid dalam rentang 1900–2200.'
  }
  if (
    trade.closed_at &&
    (!trade.opened_at || Date.parse(trade.closed_at) < Date.parse(trade.opened_at))
  )
    return 'Waktu keluar tidak boleh sebelum waktu masuk.'
  for (const key of [
    'entry_price',
    'exit_price',
    'stop_loss',
    'take_profit',
    'size',
    'risk_amount',
    'capital_snapshot',
    'net_pnl',
  ] as const) {
    const value = trade[key]
    if (
      value !== null &&
      ['entry_price', 'exit_price', 'stop_loss', 'take_profit', 'size'].includes(key) &&
      Number(value) === 0
    )
      return `${key} harus lebih dari nol.`
    if (
      value !== null &&
      !validateDecimal(
        value,
        ['risk_amount', 'capital_snapshot', 'net_pnl'].includes(key) ? 4 : 8,
        key === 'net_pnl',
      )
    )
      return `${key}: angka desimal tidak valid (maksimal ${['risk_amount', 'capital_snapshot', 'net_pnl'].includes(key) ? 4 : 8} desimal; batas 1 miliar).`
  }
  if (trade.tradingview_urls.some((url) => !safeLink(url)))
    return 'Gunakan tautan HTTPS TradingView yang valid.'
  if (
    [trade.tags, trade.emotions, trade.mistakes, trade.tradingview_urls, trade.checklist].some(
      (list) => list.length > 30,
    )
  )
    return 'Maksimal 30 item per daftar.'
  return ''
}
export function emptyTrade(accountId: string): TradeInput {
  return {
    account_id: accountId,
    status: 'draft',
    asset: '',
    side: 'long',
    opened_at: null,
    closed_at: null,
    entry_price: null,
    exit_price: null,
    stop_loss: null,
    take_profit: null,
    size: null,
    size_unit: '',
    net_pnl: null,
    risk_amount: null,
    capital_snapshot: null,
    setup: '',
    tags: [],
    emotions: [],
    mistakes: [],
    session: '',
    checklist: [],
    notes: '',
    tradingview_urls: [],
  }
}
