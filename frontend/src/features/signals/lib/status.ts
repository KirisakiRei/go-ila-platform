import type { SignalStatus } from '@/types/api'

/** Label status dalam bahasa manusia. */
export const SIGNAL_LABEL: Record<SignalStatus, string> = {
  READY_TO_ENTRY: 'Siap entry',
  WAIT_HTF_SWEEP: 'Pantau HTF',
  WAIT_HTF_CANDLE_CLOSE: 'Konfirmasi HTF',
  WAIT_LTF_CISD: 'Menunggu CISD',
  // Tahapan ILA 3.0. Tanpa label ini badge status tampil kosong karena engine
  // mengirim nilai baru yang belum dikenal frontend.
  WAIT_ILA3_HTF_KEY_LEVEL: 'Pantau key level HTF',
  WAIT_ILA3_MTF_REVERSAL: 'Menunggu reversal MTF',
  WAIT_ILA3_LTF_CISD_1: 'Menunggu CISD LTF (1)',
  ILA3_EXPANSION_1_ACTIVE: 'Ekspansi 1 aktif',
  WAIT_ILA3_MTF_FVG_RETR: 'Menunggu retr. FVG MTF',
  WAIT_ILA3_LTF_CISD_2: 'Menunggu CISD LTF (2)',
  ILA3_EXPANSION_2_ACTIVE: 'Ekspansi 2 aktif',
  ILA3_TARGET_REACHED: 'Target tercapai',
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
  WAIT_ILA3_HTF_KEY_LEVEL: 'neutral',
  WAIT_ILA3_MTF_REVERSAL: 'outline',
  WAIT_ILA3_LTF_CISD_1: 'outline',
  ILA3_EXPANSION_1_ACTIVE: 'outline',
  WAIT_ILA3_MTF_FVG_RETR: 'outline',
  WAIT_ILA3_LTF_CISD_2: 'outline',
  ILA3_EXPANSION_2_ACTIVE: 'outline',
  // Setup yang sudah mencapai target adalah pencapaian, bukan penundaan.
  ILA3_TARGET_REACHED: 'solid',
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

/** Mengembalikan nama siklus ILA 3.0 berdasarkan pasangan HTF & LTF */
export function getCycleBadge(strategy: string, htf: string, ltf: string): string | null {
  if (strategy !== 'ila-3.0') return null
  const h = (htf || '').toUpperCase().replace(/\s+/g, '')
  const l = (ltf || '').toUpperCase().replace(/\s+/g, '')
  if ((h === '4H' || h === 'H4') && (l === '5M' || l === 'M5')) return 'Daily · New Day'
  if ((h === 'D' || h === 'D1' || h === '1D') && (l === '15M' || l === 'M15')) return 'Weekly · New Week'
  if ((h === '1H' || h === 'H1') && (l === '5M' || l === 'M5')) return '90M · Intraday'
  if ((h === 'W' || h === 'W1' || h === '1W') && (l === '1H' || l === 'H1')) return 'Monthly · New Month'
  return null
}

/** Menentukan timeframe medium (MTF) dari ILA 3.0 berdasarkan HTF dan LTF */
export function getIla3Mtf(htf: string, ltf: string): string {
  const h = (htf || '').toUpperCase().replace(/\s+/g, '')
  const l = (ltf || '').toUpperCase().replace(/\s+/g, '')
  if ((h === '4H' || h === 'H4') && (l === '5M' || l === 'M5')) return '1H'
  if ((h === 'D' || h === 'D1' || h === '1D') && (l === '15M' || l === 'M15')) return '4H'
  if ((h === '1H' || h === 'H1') && (l === '5M' || l === 'M5')) return '15M'
  if ((h === 'W' || h === 'W1' || h === '1W') && (l === '1H' || l === 'H1')) return '4H'
  return 'MTF'
}

/** Menghitung progres langkah ILA 3.0 dari status sinyal */
export function getIla3Progress(status: string): { step: number; label: string } {
  const s = (status || '').toUpperCase()
  if (
    s === 'READY_TO_ENTRY' ||
    s === 'READY_ENTRY_1' ||
    s === 'READY_ENTRY_2' ||
    s === 'OPPORTUNITY' ||
    s === 'CONSUMED' ||
    s.includes('EXPANSION') ||
    s.includes('TARGET_REACHED')
  ) {
    return { step: 3, label: '3 / 3 Selesai (Setup Ready)' }
  }
  if (s.includes('LTF') || s.includes('CISD')) {
    return { step: 2, label: '2 / 3 Terkonfirmasi (LTF Trigger Phase)' }
  }
  if (s.includes('MTF') || s.includes('REVERSAL') || s.includes('RETR')) {
    return { step: 1, label: '1 / 3 Terkonfirmasi (MTF Alignment Phase)' }
  }
  return { step: 0, label: '0 / 3 Memantau (HTF Key Level Phase)' }
}
