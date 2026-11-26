import type { JournalFilters, JournalTrade, TradeInput } from './types'

export function formatAmount(value: number | string, currency: string) {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: /^[A-Z]{3}$/.test(currency) ? currency : 'USD',
    maximumFractionDigits: 4,
  }).format(Number(value))
}

export function formatSignedAmount(value: number | string, currency: string) {
  const num = Number(value)
  if (!Number.isFinite(num)) return '—'
  const formatted = formatAmount(Math.abs(num), currency)
  if (num > 0) return `+${formatted}`
  if (num < 0) return `−${formatted}`
  return formatted
}

export function pnlTone(value: number | string | null | undefined) {
  const num = Number(value)
  if (!Number.isFinite(num) || num === 0) return 'text-foreground'
  return num > 0 ? 'text-emerald-600 dark:text-emerald-400' : 'text-rose-600 dark:text-rose-400'
}

export function formatDuration(openedAt: string | null, closedAt: string | null) {
  if (!openedAt || !closedAt) return null
  const diffMs = Date.parse(closedAt) - Date.parse(openedAt)
  if (!Number.isFinite(diffMs) || diffMs < 0) return null
  const totalMinutes = Math.round(diffMs / 60000)
  if (totalMinutes < 60) return `${totalMinutes}m`
  const hours = Math.floor(totalMinutes / 60)
  const mins = totalMinutes % 60
  if (hours < 24) return mins > 0 ? `${hours}j ${mins}m` : `${hours}j`
  const days = Math.floor(hours / 24)
  const remHours = hours % 24
  return remHours > 0 ? `${days}h ${remHours}j` : `${days}h`
}

export interface TradePreviewMetrics {
  plannedRR: string | null
  realizedR: string | null
  riskPercent: string | null
  gainPercent: string | null
  holdDuration: string | null
}

export function computeTradePreview(trade: Pick<
  TradeInput,
  | 'side'
  | 'entry_price'
  | 'stop_loss'
  | 'take_profit'
  | 'net_pnl'
  | 'risk_amount'
  | 'capital_snapshot'
  | 'opened_at'
  | 'closed_at'
>): TradePreviewMetrics {
  const entry = trade.entry_price !== null ? Number(trade.entry_price) : NaN
  const sl = trade.stop_loss !== null ? Number(trade.stop_loss) : NaN
  const tp = trade.take_profit !== null ? Number(trade.take_profit) : NaN
  let plannedRR: string | null = null
  if (
    trade.side &&
    Number.isFinite(entry) &&
    Number.isFinite(sl) &&
    Number.isFinite(tp) &&
    entry > 0 &&
    sl > 0 &&
    tp > 0 &&
    sl !== entry
  ) {
    const riskDist = trade.side === 'short' ? sl - entry : entry - sl
    const rewardDist = trade.side === 'short' ? entry - tp : tp - entry
    if (riskDist > 0 && rewardDist > 0) {
      plannedRR = `1:${(rewardDist / riskDist).toFixed(2)}`
    }
  }

  const pnl = trade.net_pnl !== null ? Number(trade.net_pnl) : NaN
  const risk = trade.risk_amount !== null ? Number(trade.risk_amount) : NaN
  const cap = trade.capital_snapshot !== null ? Number(trade.capital_snapshot) : NaN

  let realizedR: string | null = null
  if (Number.isFinite(pnl) && Number.isFinite(risk) && risk > 0) {
    const r = pnl / risk
    realizedR = `${r > 0 ? '+' : r < 0 ? '−' : ''}${Math.abs(r).toFixed(2)}R`
  }

  const riskPercent =
    Number.isFinite(risk) && risk > 0 && Number.isFinite(cap) && cap > 0
      ? `${((risk / cap) * 100).toFixed(2)}%`
      : null

  let gainPercent: string | null = null
  if (Number.isFinite(pnl) && Number.isFinite(cap) && cap > 0) {
    const pct = (pnl / cap) * 100
    gainPercent = `${pct > 0 ? '+' : pct < 0 ? '−' : ''}${Math.abs(pct).toFixed(2)}%`
  }

  const holdDuration = formatDuration(trade.opened_at, trade.closed_at)
  return { plannedRR, realizedR, riskPercent, gainPercent, holdDuration }
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
export function mergeChipDraft(values: string[], raw: string, maxItems = 30) {
  const parts = raw
    .split(/\r?\n/)
    .map((s) => s.trim())
    .filter(Boolean)
  if (!parts.length) return values
  const next = [...values]
  for (const part of parts) {
    if (next.length >= maxItems) break
    if (!next.includes(part)) next.push(part)
  }
  return next
}
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
  const q = (filters.search ?? '').trim().toLowerCase()
  return trades.filter((t) => {
    const day = tradeDay(t)
    return (
      (!filters.account_id || t.account_id === filters.account_id) &&
      (!q ||
        t.asset.toLowerCase().includes(q) ||
        t.setup.toLowerCase().includes(q) ||
        t.notes.toLowerCase().includes(q)) &&
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
export function normalizeDecimalInput(raw: string | null, signed = false): string | null {
  if (raw === null) return null
  let value = raw.trim()
  if (!value) return null
  if (signed && value.startsWith('+')) {
    value = value.slice(1).trim()
  }
  // Convert a single comma decimal separator (e.g. "100,50" from Indonesian keyboards) to a dot.
  if (!value.includes('.') && (value.match(/,/g)?.length ?? 0) === 1) {
    value = value.replace(',', '.')
  }
  return value || null
}

export function validateDecimal(value: string, decimals = 4, signed = false) {
  return (
    new RegExp(`^${signed ? '-?' : ''}\\d+(?:\\.\\d{1,${decimals}})?$`).test(value) &&
    Math.abs(Number(value)) <= 1_000_000_000
  )
}

export type TradeFieldKey =
  | 'account_id'
  | 'asset'
  | 'side'
  | 'opened_at'
  | 'closed_at'
  | 'entry_price'
  | 'stop_loss'
  | 'take_profit'
  | 'exit_price'
  | 'size'
  | 'size_unit'
  | 'net_pnl'
  | 'risk_amount'
  | 'capital_snapshot'
  | 'tradingview_urls'
  | 'emotions'
  | 'mistakes'
  | 'tags'
  | 'checklist'

export const TRADE_FIELD_LABELS: Record<TradeFieldKey, string> = {
  account_id: 'Jurnal',
  asset: 'Pair / Aset',
  side: 'Arah posisi',
  opened_at: 'Waktu masuk',
  closed_at: 'Waktu keluar',
  entry_price: 'Harga entry',
  stop_loss: 'Stop loss',
  take_profit: 'Take profit',
  exit_price: 'Exit aktual',
  size: 'Lot / Size',
  size_unit: 'Satuan size',
  net_pnl: 'Net P&L',
  risk_amount: 'Risiko awal',
  capital_snapshot: 'Modal saat entry',
  tradingview_urls: 'Link TradingView',
  emotions: 'Psikologi / Emosi',
  mistakes: 'Mistake',
  tags: 'Tag',
  checklist: 'Pre-trade checklist',
}

const REVIEW_FIELD_KEYS = new Set<TradeFieldKey>([
  'risk_amount',
  'capital_snapshot',
  'tradingview_urls',
  'emotions',
  'mistakes',
  'tags',
  'checklist',
])

function decimalFieldError(value: string, decimals: number, signed: boolean): string {
  if (!signed && value.startsWith('-')) return 'Tidak boleh negatif.'
  if (/^\s*[+-]?\d+(?:\.\d+)?\s*$/.test(value)) {
    if (Math.abs(Number(value)) > 1_000_000_000) return 'Maksimal 1 miliar.'
    return `Maksimal ${decimals} angka di belakang koma.`
  }
  return `Gunakan angka valid (maks ${decimals} desimal, contoh: ${decimals === 4 ? '1000.50' : '3540.25'}).`
}

export function validateTradeFields(trade: TradeInput): {
  errors: Partial<Record<TradeFieldKey, string>>
  firstTab: 'core' | 'review'
  summary: string
} {
  const errors: Partial<Record<TradeFieldKey, string>> = {}
  const setErr = (key: TradeFieldKey, message: string) => {
    if (!errors[key]) errors[key] = message
  }

  if (!trade.account_id) setErr('account_id', 'Pilih jurnal terlebih dahulu.')

  if (trade.status !== 'draft') {
    if (!trade.asset.trim()) setErr('asset', 'Isi pair atau simbol aset.')
    if (!trade.side) setErr('side', 'Pilih arah Buy atau Sell.')
    if (!trade.opened_at) setErr('opened_at', 'Isi waktu masuk.')
    if (trade.entry_price === null) setErr('entry_price', 'Isi harga entry.')
    if (trade.size === null) setErr('size', 'Isi lot atau ukuran posisi.')
    if (!trade.size_unit.trim()) setErr('size_unit', 'Isi satuan (mis. lot).')
  }

  if (trade.status === 'closed') {
    if (!trade.closed_at) setErr('closed_at', 'Isi waktu keluar untuk trade closed.')
    if (trade.net_pnl === null) setErr('net_pnl', 'Isi net P&L aktual untuk trade closed.')
  } else {
    if (trade.closed_at) setErr('closed_at', 'Waktu keluar hanya diisi pada trade closed.')
    if (trade.net_pnl !== null) setErr('net_pnl', 'Net P&L hanya diisi pada trade closed.')
  }

  for (const [key, date] of [
    ['opened_at', trade.opened_at],
    ['closed_at', trade.closed_at],
  ] as const) {
    if (
      date &&
      (!Number.isFinite(Date.parse(date)) ||
        new Date(date).getUTCFullYear() < 1900 ||
        new Date(date).getUTCFullYear() > 2200)
    ) {
      setErr(key, 'Format waktu tidak valid (rentang 1900–2200).')
    }
  }

  if (
    trade.closed_at &&
    (!trade.opened_at || Date.parse(trade.closed_at) < Date.parse(trade.opened_at))
  ) {
    setErr('closed_at', 'Waktu keluar tidak boleh sebelum waktu masuk.')
  }

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
    if (value === null) continue
    const decimals = ['risk_amount', 'capital_snapshot', 'net_pnl'].includes(key) ? 4 : 8
    const signed = key === 'net_pnl'
    if (
      ['entry_price', 'exit_price', 'stop_loss', 'take_profit', 'size'].includes(key) &&
      validateDecimal(value, decimals, false) &&
      Number(value) === 0
    ) {
      setErr(key, 'Harus lebih dari 0.')
      continue
    }
    if (!validateDecimal(value, decimals, signed)) {
      setErr(key, decimalFieldError(value, decimals, signed))
    }
  }

  // Direction-aware price guards (Buy: SL < Entry < TP; Sell: TP < Entry < SL).
  const validNum = (key: 'entry_price' | 'stop_loss' | 'take_profit' | 'risk_amount' | 'capital_snapshot') => {
    const raw = trade[key]
    if (raw === null || errors[key]) return null
    const n = Number(raw)
    return Number.isFinite(n) ? n : null
  }
  const entryNum = validNum('entry_price')
  const slNum = validNum('stop_loss')
  const tpNum = validNum('take_profit')

  if (trade.side === 'long') {
    if (entryNum !== null && slNum !== null && slNum >= entryNum) {
      setErr('stop_loss', 'Untuk Buy, Stop loss harus lebih rendah dari Entry.')
    }
    if (entryNum !== null && tpNum !== null && tpNum <= entryNum) {
      setErr('take_profit', 'Untuk Buy, Take profit harus lebih tinggi dari Entry.')
    }
    if (slNum !== null && tpNum !== null && tpNum <= slNum) {
      setErr('take_profit', 'Untuk Buy, Take profit harus lebih tinggi dari Stop loss.')
    }
  } else if (trade.side === 'short') {
    if (entryNum !== null && slNum !== null && slNum <= entryNum) {
      setErr('stop_loss', 'Untuk Sell, Stop loss harus lebih tinggi dari Entry.')
    }
    if (entryNum !== null && tpNum !== null && tpNum >= entryNum) {
      setErr('take_profit', 'Untuk Sell, Take profit harus lebih rendah dari Entry.')
    }
    if (slNum !== null && tpNum !== null && tpNum >= slNum) {
      setErr('take_profit', 'Untuk Sell, Take profit harus lebih rendah dari Stop loss.')
    }
  }

  const riskNum = validNum('risk_amount')
  const capNum = validNum('capital_snapshot')
  if (riskNum !== null && capNum !== null && capNum > 0 && riskNum > capNum) {
    setErr('risk_amount', 'Risiko awal tidak boleh melebihi modal saat entry.')
  }

  if (trade.tradingview_urls.some((url) => !safeLink(url))) {
    setErr('tradingview_urls', 'Gunakan link HTTPS dari tradingview.com.')
  }

  for (const [key, list] of [
    ['emotions', trade.emotions],
    ['mistakes', trade.mistakes],
    ['tags', trade.tags],
    ['tradingview_urls', trade.tradingview_urls],
    ['checklist', trade.checklist],
  ] as const) {
    if (list.length > 30) {
      setErr(key, 'Maksimal 30 item.')
    }
  }

  const keys = Object.keys(errors) as TradeFieldKey[]
  if (!keys.length) {
    return { errors, firstTab: 'core', summary: '' }
  }

  const hasCoreError = keys.some((k) => !REVIEW_FIELD_KEYS.has(k))
  const firstTab: 'core' | 'review' = hasCoreError ? 'core' : 'review'

  const summary =
    keys.length === 1
      ? `${TRADE_FIELD_LABELS[keys[0]!]}: ${errors[keys[0]!]}`
      : `Periksa ${keys.length} kolom yang ditandai merah: ${keys
          .slice(0, 4)
          .map((k) => TRADE_FIELD_LABELS[k])
          .join(', ')}${keys.length > 4 ? ', dll.' : '.'}`

  return { errors, firstTab, summary }
}

export function validateTrade(trade: TradeInput) {
  return validateTradeFields(trade).summary
}

export function friendlyJournalError(err: unknown, fallback = 'Gagal memproses data. Coba lagi.'): string {
  const raw = err instanceof Error ? err.message : ''
  if (!raw) return fallback
  const lower = raw.toLowerCase()
  if (lower.includes('account is archived')) {
    return 'Jurnal ini sedang diarsipkan. Pulihkan dulu di Pengaturan untuk mengubah data.'
  }
  if (lower.includes('closed_at precedes opened_at')) {
    return 'Waktu keluar tidak boleh sebelum waktu masuk.'
  }
  if (lower.includes('price/size must be positive')) {
    return 'Harga dan lot/size harus lebih dari 0.'
  }
  if (lower.includes('long stop_loss') || lower.includes('short stop_loss') || lower.includes('long take_profit') || lower.includes('short take_profit')) {
    return 'Posisi Stop loss atau Take profit tidak sesuai dengan arah Buy/Sell dan harga Entry.'
  }
  if (lower.includes('risk_amount exceeds capital_snapshot')) {
    return 'Risiko awal tidak boleh melebihi modal saat entry.'
  }
  if (lower.includes('only https tradingview')) {
    return 'Gunakan link HTTPS dari tradingview.com.'
  }
  if (lower.includes('invalid decimal') || lower.includes('invalid number') || lower.includes('decimal')) {
    return 'Format angka tidak valid. Gunakan angka dengan titik desimal (maks 4 desimal untuk nominal).'
  }
  if (lower.includes('open/closed trade requires')) {
    return 'Trade open/closed wajib mengisi pair, arah, waktu masuk, harga entry, dan lot/size.'
  }
  if (lower.includes('closed trade requires')) {
    return 'Trade closed wajib mengisi waktu keluar dan net P&L.'
  }
  return raw
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
