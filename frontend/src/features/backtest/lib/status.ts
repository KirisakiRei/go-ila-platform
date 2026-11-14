/**
 * Label dan status satu run backtest, dipakai bersama oleh halaman riwayat,
 * halaman run, dan halaman kerja.
 */
import type { BacktestRun, BacktestStatus } from '@/types/api'
import { formatTanggalPanjang } from './tanggal'

export type NadaStatus = 'neutral' | 'solid' | 'outline' | 'muted'

export const STATUS_RUN: Record<BacktestStatus, { teks: string; nada: NadaStatus }> = {
  queued: { teks: 'Antrean', nada: 'muted' },
  running: { teks: 'Berjalan', nada: 'outline' },
  done: { teks: 'Selesai', nada: 'solid' },
  failed: { teks: 'Gagal', nada: 'muted' },
}

/** statusRun mengembalikan label untuk status apa pun, termasuk yang tak dikenal. */
export function statusRun(status: BacktestStatus): { teks: string; nada: NadaStatus } {
  return STATUS_RUN[status] ?? { teks: status, nada: 'muted' }
}

/** labelRingkas menulis judul satu run: pair, pasangan timeframe, rentang. */
export function labelRingkas(
  run: Pick<BacktestRun, 'symbol' | 'htf' | 'ltf' | 'dari' | 'sampai'>,
): string {
  return `${run.symbol} ${run.htf} ke ${run.ltf}, ${formatTanggalPanjang(run.dari)} sampai ${formatTanggalPanjang(run.sampai)}`
}
