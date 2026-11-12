import type { StrategyVersion } from '@/types/api'

/**
 * Tahap ringkas satu slot timeframe untuk kartu overview.
 *
 * Kartu overview menampilkan empat slot berdampingan, jadi label tahap harus
 * muat dalam satu kata. Label panjang (`Pantau key level HTF`) hanya muat di
 * halaman detail, bukan di sini.
 */
export type SlotStage = {
  /** Label pendek siap tampil, mis. `Sweep`, `CISD`, `Ready`. */
  label: string
  /**
   * Urutan kedekatan ke entry. Angka KECIL berarti lebih dekat.
   * Dipakai memilih slot yang diberi aksen, bukan untuk menyimpulkan arah:
   * tiap pasangan timeframe bergerak sendiri-sendiri.
   */
  rank: number
  /** True hanya saat setup sudah boleh dieksekusi. */
  ready: boolean
}

/** Tahap saat slot belum punya sinyal sama sekali. */
const TANPA_SINYAL: SlotStage = { label: 'Scan', rank: 99, ready: false }

/**
 * Tahap ILA 2.0: Sweep -> Candle -> CISD -> Ready.
 *
 * Urutan mengikuti aturan trading engine, bukan urutan abjad, supaya angka
 * `rank` benar-benar bermakna "makin dekat ke entry".
 */
const ILA2_STAGE: Record<string, SlotStage> = {
  WAIT_HTF_SWEEP: { label: 'Sweep', rank: 40, ready: false },
  WAIT_HTF_CANDLE_CLOSE: { label: 'Candle', rank: 30, ready: false },
  WAIT_LTF_CISD: { label: 'CISD', rank: 20, ready: false },
  READY_TO_ENTRY: { label: 'Ready', rank: 0, ready: true },
  INVALIDATED: { label: 'Batal', rank: 90, ready: false },
}

/**
 * Tahap ILA 3.0: Key Level -> Reversal -> CISD 1 -> Ready 1 ->
 * Expansi 1 -> Retrace -> CISD 2 -> Ready 2.
 *
 * Dua titik entry berbagi cukup satu label `Ready`, karena backend sudah
 * memetakan READY_ENTRY_1/2 ke READY_TO_ENTRY. Entri READY_ENTRY_* tetap
 * disediakan sebagai jaring pengaman bila engine mengirim nilai aslinya.
 */
const ILA3_STAGE: Record<string, SlotStage> = {
  WAIT_ILA3_HTF_KEY_LEVEL: { label: 'Key Lvl', rank: 70, ready: false },
  WAIT_ILA3_MTF_REVERSAL: { label: 'Reversal', rank: 60, ready: false },
  WAIT_ILA3_LTF_CISD_1: { label: 'CISD 1', rank: 50, ready: false },
  READY_ENTRY_1: { label: 'Ready', rank: 0, ready: true },
  ILA3_EXPANSION_1_ACTIVE: { label: 'Expansi 1', rank: 10, ready: false },
  WAIT_ILA3_MTF_FVG_RETR: { label: 'Retrace', rank: 45, ready: false },
  WAIT_ILA3_LTF_CISD_2: { label: 'CISD 2', rank: 40, ready: false },
  READY_ENTRY_2: { label: 'Ready', rank: 0, ready: true },
  ILA3_EXPANSION_2_ACTIVE: { label: 'Expansi 2', rank: 10, ready: false },
  ILA3_TARGET_REACHED: { label: 'Target', rank: 5, ready: false },
  // Status siap yang sudah dinormalkan backend.
  READY_TO_ENTRY: { label: 'Ready', rank: 0, ready: true },
  OPPORTUNITY: { label: 'Peluang', rank: 5, ready: false },
  INVALIDATED: { label: 'Batal', rank: 90, ready: false },
}

function cariStage(tabel: Record<string, SlotStage>, status: string, fallback: SlotStage): SlotStage {
  const kunci = (status || '').toUpperCase().trim()
  if (!kunci) return fallback
  return tabel[kunci] ?? fallback
}

/** Tahap satu slot ILA 2.0 dari status engine. */
export function stageILA2(status: string): SlotStage {
  return cariStage(ILA2_STAGE, status, TANPA_SINYAL)
}

/** Tahap satu slot ILA 3.0 dari status engine. */
export function stageILA3(status: string): SlotStage {
  return cariStage(ILA3_STAGE, status, TANPA_SINYAL)
}

/** Memilih tabel tahap sesuai metode. */
export function stageOf(strategy: StrategyVersion, status: string): SlotStage {
  return strategy === 'ila-3.0' ? stageILA3(status) : stageILA2(status)
}

/**
 * Mengembalikan key slot yang paling dekat ke entry.
 *
 * Hanya dipakai untuk memberi aksen pada kartu. Slot tanpa sinyal diabaikan
 * agar tidak ada yang tersorot padahal belum ada apa pun untuk dilihat.
 * Bila tidak ada satu pun slot yang berjalan, hasilnya `null`.
 */
export function slotPalingDekat(
  slots: Array<{ key: string; latestSignal?: { status: string } }>,
  strategy: StrategyVersion,
): string | null {
  let terbaik: string | null = null
  let rankTerbaik = Number.POSITIVE_INFINITY
  let jumlahTerbaik = 0

  for (const slot of slots) {
    if (!slot.latestSignal) continue
    const stage = stageOf(strategy, slot.latestSignal.status)
    if (stage.rank >= 99) continue
    if (stage.rank < rankTerbaik) {
      rankTerbaik = stage.rank
      terbaik = slot.key
      jumlahTerbaik = 1
    } else if (stage.rank === rankTerbaik) {
      jumlahTerbaik += 1
    }
  }

  // Bila beberapa slot berada di tahap yang sama, tidak ada yang lebih dekat.
  // Menyorot salah satunya secara acak akan menyesatkan, jadi tidak ada aksen.
  if (jumlahTerbaik > 1) return null
  return terbaik
}
