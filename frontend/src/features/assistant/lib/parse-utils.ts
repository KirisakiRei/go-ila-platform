export function clampText(raw: unknown, maxLen: number): string | undefined {
  if (typeof raw !== 'string') return undefined
  const s = raw.trim()
  if (!s) return undefined
  return s.length > maxLen ? `${s.slice(0, maxLen - 1)}…` : s
}

export function isFiniteNum(v: unknown): v is number {
  return typeof v === 'number' && Number.isFinite(v)
}

/** Menerima angka Unix detik/ms atau string ISO-8601; menghasilkan Unix detik. */
export function parseUnixSec(raw: unknown): number | undefined {
  if (typeof raw === 'number' && Number.isFinite(raw) && raw > 1_000_000_000) {
    return Math.floor(raw)
  }
  if (typeof raw === 'string' && raw.trim()) {
    const ms = new Date(raw.trim()).getTime()
    if (Number.isFinite(ms) && ms > 0) {
      return Math.floor(ms / 1000)
    }
  }
  return undefined
}
