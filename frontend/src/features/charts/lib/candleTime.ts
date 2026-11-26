// Alias mengikuti resolusi market; 1M adalah satu menit, bukan satu bulan.
const TIMEFRAME_ALIASES: Record<string, string> = {
  M1: 'M1',
  '1M': 'M1',
  '1': 'M1',
  M3: 'M3',
  '3M': 'M3',
  '3': 'M3',
  M5: 'M5',
  '5M': 'M5',
  '5': 'M5',
  M15: 'M15',
  '15M': 'M15',
  '15': 'M15',
  M30: 'M30',
  '30M': 'M30',
  '30': 'M30',
  H1: 'H1',
  '1H': 'H1',
  '60M': 'H1',
  '60': 'H1',
  H2: 'H2',
  '2H': 'H2',
  '120M': 'H2',
  '120': 'H2',
  H4: 'H4',
  '4H': 'H4',
  '240M': 'H4',
  '240': 'H4',
  D1: 'D1',
  D: 'D1',
  '1D': 'D1',
  DAILY: 'D1',
  W1: 'W1',
  W: 'W1',
  '1W': 'W1',
  WEEKLY: 'W1',
  MN: 'MN',
  MN1: 'MN',
  MONTHLY: 'MN',
  '1MO': 'MN',
  Q: 'Q',
  Q1: 'Q',
  QUARTERLY: 'Q',
  '3MO': 'Q',
}

export function normalizeTimeframe(timeframe: string): string | undefined {
  const key = timeframe.trim().toUpperCase()
  return Object.hasOwn(TIMEFRAME_ALIASES, key) ? TIMEFRAME_ALIASES[key] : undefined
}

/** Akhir eksklusif candle, berdasarkan timeframe eksplisit, bukan jarak bar yang bisa bolong. */
export function candleEnd(time: number, timeframe?: string): number | undefined {
  if (!Number.isFinite(time) || !timeframe) return undefined
  const tf = normalizeTimeframe(timeframe)
  if (!tf) return undefined
  const date = new Date(time * 1000)
  if (tf === 'MN' || tf === 'Q') {
    date.setUTCMonth(date.getUTCMonth() + (tf === 'Q' ? 3 : 1), 1)
    return date.getTime() / 1000
  }
  if (tf === 'D1' || tf === 'W1') {
    date.setUTCDate(date.getUTCDate() + (tf === 'W1' ? 7 : 1))
    return date.getTime() / 1000
  }
  return time + Number(tf.slice(1)) * (tf.startsWith('H') ? 3600 : 60)
}

/** Menyelaraskan waktu eksekusi ke bar yang tersedia; akhir candle terakhir eksklusif. */
export function snapKeBar(time: number, barTimes: number[], lastBarEnd?: number): number | null {
  if (!Number.isFinite(time)) return null
  const first = barTimes[0]
  const last = barTimes.at(-1)
  if (first === undefined || last === undefined || time < first) return null
  if (time > last && (lastBarEnd === undefined || time >= lastBarEnd)) return null
  let lo = 0
  let hi = barTimes.length - 1
  while (lo <= hi) {
    const mid = (lo + hi) >> 1
    const current = barTimes[mid]
    if (current === undefined) break
    if (current === time) return current
    if (current < time) lo = mid + 1
    else hi = mid - 1
  }
  return barTimes[hi] ?? null
}
