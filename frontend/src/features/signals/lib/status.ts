import type { SignalStatus } from '@/types/api'

/** Label status dalam bahasa manusia. */
export const SIGNAL_LABEL: Record<SignalStatus, string> = {
  READY_TO_ENTRY: 'Siap entry',
  ACKED: 'Diterima',
  REJECTED: 'Tertahan',
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
  ACKED: 'outline',
  REJECTED: 'outline',
  CONSUMED: 'outline',
  EXPIRED: 'muted',
  CANCELLED: 'muted',
  SUPERSEDED: 'muted',
  SKIPPED: 'muted',
}

/** Status yang menandakan sinyal sudah tidak bisa ditindaklanjuti. */
export function isFinal(status: SignalStatus): boolean {
  return ['EXPIRED', 'CANCELLED', 'SUPERSEDED', 'SKIPPED', 'CONSUMED'].includes(status)
}
