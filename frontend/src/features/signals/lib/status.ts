import type { SignalStatus } from '@/types/api'

/** Label status dalam bahasa manusia. */
export const SIGNAL_LABEL: Record<SignalStatus, string> = {
  READY_TO_ENTRY: 'Siap entry',
  WAIT_HTF_SWEEP: 'Pantau HTF',
  WAIT_HTF_CANDLE_CLOSE: 'Konfirmasi HTF',
  WAIT_LTF_CISD: 'Menunggu CISD',
  INVALIDATED: 'Struktur batal',
  ACKED: 'Diterima',
  REJECTED: 'Tertahan',
  OPPORTUNITY: 'Peluang entry',
  EXPIRED: 'Kedaluwarsa',
  CANCELLED: 'Dibatalkan',
  SUPERSEDED: 'Digantikan',
  CONSUMED: 'Sudah dieksekusi',
  SKIPPED: 'Dilewati',
}

/**
 * Bobot visual per status. Karena palet monokrom, status dibedakan lewat
 * gaya garis: `solid` paling menonjol, `muted` paling tenang.
 */
export const SIGNAL_TONE: Record<SignalStatus, 'solid' | 'outline' | 'neutral' | 'muted'> = {
  READY_TO_ENTRY: 'solid',
  WAIT_HTF_SWEEP: 'neutral',
  WAIT_HTF_CANDLE_CLOSE: 'outline',
  WAIT_LTF_CISD: 'outline',
  INVALIDATED: 'muted',
  ACKED: 'outline',
  REJECTED: 'outline',
  // Peluang diberi bobot yang lebih menonjol daripada penolakan: ini setup yang
  // masih bisa ditindaklanjuti user.
  OPPORTUNITY: 'solid',
  CONSUMED: 'outline',
  EXPIRED: 'muted',
  CANCELLED: 'muted',
  SUPERSEDED: 'muted',
  SKIPPED: 'muted',
}

/** Status yang menandakan sinyal sudah tidak bisa ditindaklanjuti. */
export function isFinal(status: SignalStatus): boolean {
  return ['EXPIRED', 'CANCELLED', 'SUPERSEDED', 'SKIPPED', 'CONSUMED', 'INVALIDATED'].includes(status)
}
