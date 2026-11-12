import type { Signal } from '@/types/api'
import { getIla3Mtf } from './status'

export type TimeframeSlot = {
  key: string
  label: string
  cycle?: string
  htf: string
  mtf?: string
  ltf: string
  latestSignal?: Signal
  status: 'READY' | 'WAIT_CISD' | 'WAIT_SWEEP' | 'SCANNING' | 'INVALIDATED'
  statusLabel: string
  statusTone: 'solid' | 'outline' | 'neutral' | 'muted'
}

/** 4 Pasangan Timeframe Baku untuk ILA 2.0 */
export const ILA2_SLOTS: Array<{ htf: string; ltf: string; label: string }> = [
  { htf: 'W', ltf: 'H4', label: 'W · H4' },
  { htf: 'D1', ltf: 'H1', label: 'D1 · H1' },
  { htf: 'H4', ltf: 'M15', label: 'H4 · M15' },
  { htf: 'H1', ltf: 'M5', label: 'H1 · M5' },
]

/** 4 Siklus Quarterly Baku untuk ILA 3.0 */
export const ILA3_SLOTS: Array<{ htf: string; mtf: string; ltf: string; cycle: string; label: string }> = [
  { htf: 'W', mtf: '4H', ltf: '1H', cycle: 'Monthly Cycle', label: 'W · 4H · 1H' },
  { htf: 'D', mtf: '4H', ltf: '15M', cycle: 'Weekly Cycle', label: 'D · 4H · 15M' },
  { htf: '4H', mtf: '1H', ltf: '5M', cycle: 'Daily Cycle', label: '4H · 1H · 5M' },
  { htf: '1H', mtf: '15M', ltf: '5M', cycle: '90M Cycle', label: '1H · 15M · 5M' },
]

function normTF(s: string): string {
  return (s || '').toUpperCase().replace(/[\s\-_]+/g, '')
}

function matchTF(a: string, b: string): boolean {
  const na = normTF(a)
  const nb = normTF(b)
  if (na === nb) return true
  // Handle alias (D vs D1, 4H vs H4, 15M vs M15)
  if ((na === 'D' || na === 'D1' || na === '1D') && (nb === 'D' || nb === 'D1' || nb === '1D')) return true
  if ((na === 'W' || na === 'W1' || na === '1W') && (nb === 'W' || nb === 'W1' || nb === '1W')) return true
  if ((na === '4H' || na === 'H4') && (nb === '4H' || nb === 'H4')) return true
  if ((na === '1H' || na === 'H1') && (nb === '1H' || nb === 'H1')) return true
  if ((na === '15M' || na === 'M15') && (nb === '15M' || nb === 'M15')) return true
  if ((na === '5M' || na === 'M5') && (nb === '5M' || nb === 'M5')) return true
  return false
}

/**
 * mapSignalsToSlots memetakan daftar sinyal dari server ke dalam slot pasangan timeframe yang tepat.
 * Setiap slot memegang sinyal terbarunya sendiri sehingga tidak saling menimpa.
 */
/**
 * waktuTerbaru mengembalikan patokan waktu terakhir baris disentuh.
 * Baris radar memakai identitas tetap sehingga `published_at`-nya bisa jauh
 * lebih lama daripada sinyal nyata di slot yang sama; `updated_at` yang
 * menunjukkan bahwa radar masih bergerak.
 */
function waktuTerbaru(s: Signal): number {
  return new Date(s.updated_at || s.published_at).getTime()
}

/**
 * pilihTerbaru mengambil sinyal yang paling akhir diperbarui di antara kandidat
 * satu slot. Tanpa ini, urutan dari API (published_at DESC) membuat sinyal lama
 * yang sudah EXPIRED selalu menang atas baris radar yang masih aktif.
 */
function pilihTerbaru(kandidat: Signal[]): Signal | undefined {
  if (kandidat.length === 0) return undefined
  return kandidat.reduce((a, b) => (waktuTerbaru(b) > waktuTerbaru(a) ? b : a))
}

export function mapSignalsToSlots(
  signals: Signal[],
  strategy: 'ila-2.0' | 'ila-3.0',
): TimeframeSlot[] {
  const isILA3 = strategy === 'ila-3.0'

  if (isILA3) {
    return ILA3_SLOTS.map((slot) => {
      // Cari sinyal terbaru yang cocok dengan slot ini
      const matched = pilihTerbaru(
        signals.filter(
          (s) =>
            s.strategy_version === 'ila-3.0' &&
            matchTF(s.htf, slot.htf) &&
            matchTF(s.ltf, slot.ltf),
        ),
      )

      return buildSlot({
        key: `${slot.htf}-${slot.ltf}`,
        label: slot.label,
        cycle: slot.cycle,
        htf: slot.htf,
        mtf: slot.mtf,
        ltf: slot.ltf,
        signal: matched,
      })
    })
  }

  // ILA 2.0
  return ILA2_SLOTS.map((slot) => {
    const matched = pilihTerbaru(
      signals.filter(
        (s) =>
          s.strategy_version === 'ila-2.0' &&
          matchTF(s.htf, slot.htf) &&
          matchTF(s.ltf, slot.ltf),
      ),
    )

    return buildSlot({
      key: `${slot.htf}-${slot.ltf}`,
      label: slot.label,
      htf: slot.htf,
      ltf: slot.ltf,
      signal: matched,
    })
  })
}

function buildSlot({
  key,
  label,
  cycle,
  htf,
  mtf,
  ltf,
  signal,
}: {
  key: string
  label: string
  cycle?: string
  htf: string
  mtf?: string
  ltf: string
  signal?: Signal
}): TimeframeSlot {
  if (!signal) {
    return {
      key,
      label,
      cycle,
      htf,
      mtf,
      ltf,
      status: 'SCANNING',
      statusLabel: 'Scanning',
      statusTone: 'muted',
    }
  }

  const s = signal.status.toUpperCase()

  if (s === 'READY_TO_ENTRY' || s === 'OPPORTUNITY' || s.startsWith('READY_ENTRY')) {
    return {
      key,
      label,
      cycle,
      htf,
      mtf: mtf || (signal.strategy_version === 'ila-3.0' ? getIla3Mtf(signal.htf, signal.ltf) : undefined),
      ltf,
      latestSignal: signal,
      status: 'READY',
      statusLabel: 'Ready Setup',
      statusTone: 'solid',
    }
  }

  if (s.includes('CISD') || s === 'WAIT_LTF_CISD') {
    return {
      key,
      label,
      cycle,
      htf,
      mtf: mtf || (signal.strategy_version === 'ila-3.0' ? getIla3Mtf(signal.htf, signal.ltf) : undefined),
      ltf,
      latestSignal: signal,
      status: 'WAIT_CISD',
      statusLabel: 'Wait CISD',
      statusTone: 'outline',
    }
  }

  if (s.includes('SWEEP') || s.includes('KEY_LEVEL') || s.includes('MTF')) {
    return {
      key,
      label,
      cycle,
      htf,
      mtf: mtf || (signal.strategy_version === 'ila-3.0' ? getIla3Mtf(signal.htf, signal.ltf) : undefined),
      ltf,
      latestSignal: signal,
      status: 'WAIT_SWEEP',
      statusLabel: 'Wait Sweep',
      statusTone: 'neutral',
    }
  }

  if (s === 'INVALIDATED' || s === 'CANCELLED') {
    return {
      key,
      label,
      cycle,
      htf,
      mtf: mtf || (signal.strategy_version === 'ila-3.0' ? getIla3Mtf(signal.htf, signal.ltf) : undefined),
      ltf,
      latestSignal: signal,
      status: 'INVALIDATED',
      statusLabel: 'Invalidated',
      statusTone: 'muted',
    }
  }

  return {
    key,
    label,
    cycle,
    htf,
    mtf: mtf || (signal.strategy_version === 'ila-3.0' ? getIla3Mtf(signal.htf, signal.ltf) : undefined),
    ltf,
    latestSignal: signal,
    status: 'SCANNING',
    statusLabel: 'Scanning',
    statusTone: 'muted',
  }
}

/**
 * getActiveCallEntries menyaring semua sinyal yang saat ini READY_TO_ENTRY atau OPPORTUNITY,
 * diurutkan berdasarkan waktu terbit paling baru.
 */
export function getActiveCallEntries(signals: Signal[]): Signal[] {
  return signals
    .filter((s) => s.status === 'READY_TO_ENTRY' || s.status === 'OPPORTUNITY')
    .sort((a, b) => new Date(b.published_at).getTime() - new Date(a.published_at).getTime())
}

/**
 * setTimeframeDipantau menghitung pasangan timeframe yang sedang dipantau
 * (watch) untuk satu metode, berdasarkan matriks switch yang dikirim server.
 *
 * Hasil `null` berarti "belum bisa dipastikan": user belum punya akun, atau
 * metodenya tidak berjalan. Pemanggil harus memperlakukannya sebagai "tidak
 * tahu", BUKAN "semua mati" — kalau salah, radar yang sebenarnya berjalan akan
 * tampil seperti dimatikan.
 *
 * Hasil himpunan kosong berarti semua switch memang dimatikan.
 */
export function setTimeframeDipantau(
  matrix: { groups?: Array<{ strategy: string; accounts: Array<{ pairs: Array<{ htf: string; ltf: string; watch: boolean }> }> }> } | null | undefined,
  strategy: 'ila-2.0' | 'ila-3.0',
): Set<string> | null {
  const group = matrix?.groups?.find((g) => g.strategy === strategy)
  if (!group || group.accounts.length === 0) return null

  const aktif = new Set<string>()
  for (const account of group.accounts) {
    for (const pair of account.pairs) {
      if (!pair.watch) continue
      aktif.add(`${normTF(pair.htf)}|${normTF(pair.ltf)}`)
    }
  }
  return aktif
}

/**
 * timeframeSlotDipantau menandai apakah satu slot benar-benar dipindai engine.
 *
 * `aktif` bernilai null berarti status belum diketahui, sehingga slot dianggap
 * dipantau agar tampilan tidak menuduh user mematikan sesuatu.
 */
export function timeframeSlotDipantau(
  aktif: Set<string> | null,
  slot: { htf: string; ltf: string },
): boolean {
  if (aktif === null) return true
  return aktif.has(`${normTF(slot.htf)}|${normTF(slot.ltf)}`)
}

/**
 * semuaTimeframeMati menandai kondisi khusus "tidak ada satu pun timeframe
 * yang dipantau". Hanya bernilai true bila status benar-benar diketahui.
 */
export function semuaTimeframeMati(aktif: Set<string> | null): boolean {
  return aktif !== null && aktif.size === 0
}
