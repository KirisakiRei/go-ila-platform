import { formatNYTime } from '@/lib/format'
import type { Signal } from '@/types/api'

export type Ila2StepState = 'done' | 'active' | 'pending' | 'invalid'

export type Ila2Timeline = {
  completedSteps: number
  htf: {
    state: Ila2StepState
    title: string
    headline: string
    details: string[]
  }
  ltf: {
    state: Ila2StepState
    title: string
    erlStatus: string
    cisdStatus: string
    cisdLevel?: number
  }
  entry: {
    state: Ila2StepState
    confirmed: boolean
    headline: string
    note?: string
    direction?: string
    entryPrice?: number
    stopLoss?: number
    takeProfit?: number
    riskReward?: number
  }
}

function formatJamRingkas(iso?: string): string | null {
  if (!iso) return null
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return null
  const wib = new Intl.DateTimeFormat('id-ID', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone: 'Asia/Jakarta',
  }).format(date)
  const ny = formatNYTime(iso)
  return `${wib} WIB · ${ny}`
}

function extractMatch(text: string, regex: RegExp): string | null {
  const m = text.match(regex)
  return m?.[1]?.trim() || null
}

function cisdName(direction: string, details: string): string {
  if (/-CISD/i.test(details)) return '-CISD'
  if (/\+CISD/i.test(details)) return '+CISD'
  if (direction === 'SELL') return '-CISD'
  if (direction === 'BUY') return '+CISD'
  return 'CISD'
}

/**
 * buildIla2Timeline menurunkan 3 langkah progres New ILA 2.0 (HTF → LTF → Entry)
 * dari payload Signal untuk ditampilkan pada vertical stepper.
 *
 * shortcut: parses deterministic engine detail strings; upgrade if proto adds
 * structured cisd.condition or htf_sweep.key_level_kind fields.
 */
export function buildIla2Timeline(signal: Signal): Ila2Timeline {
  const status = (signal.status || '').toUpperCase()
  const htfLabel = signal.htf || 'HTF'
  const ltfLabel = signal.ltf || 'LTF'
  const htfDetails = signal.htf_sweep?.details ?? ''
  const cisdDetails = signal.cisd?.details ?? ''
  const summaryText = `${signal.summary ?? ''} ${signal.next_action ?? ''}`

  const isInvalid = status === 'INVALIDATED' || status === 'CANCELLED'
  const isEntryConfirmed =
    status === 'READY_TO_ENTRY' ||
    status === 'OPPORTUNITY' ||
    status === 'CONSUMED' ||
    (Boolean(signal.cisd?.detected) && signal.entry_price > 0 && !isInvalid)
  const isWaitLtf = status === 'WAIT_LTF_CISD'
  const isC2Running = status === 'WAIT_HTF_CANDLE_CLOSE' || Boolean(signal.htf_sweep?.sweep_in_progress)
  const isC2Valid = isWaitLtf || isEntryConfirmed || Boolean(signal.htf_sweep?.detected)

  // --- 1. HTF (C2 Closure & C3 Cycle) ---
  const htfTime =
    formatJamRingkas(signal.htf_sweep?.sweep_time) ??
    extractMatch(htfDetails, /di candle\s+([^()]+?)(?:\s*\(|$)/i)

  const acuanHarga = extractMatch(htfDetails, /\(acuan\s+([^)]+)\)/i)
  let keyLevelInfo: string | null = null
  if (/\bSIBI\b/i.test(htfDetails) || signal.htf_sweep?.fvg?.type?.toUpperCase() === 'SIBI') {
    keyLevelInfo = acuanHarga ? `Key level: IRL (SIBI · acuan ${acuanHarga})` : 'Key level: IRL (SIBI)'
  } else if (/\bBISI\b/i.test(htfDetails) || signal.htf_sweep?.fvg?.type?.toUpperCase() === 'BISI') {
    keyLevelInfo = acuanHarga ? `Key level: IRL (BISI · acuan ${acuanHarga})` : 'Key level: IRL (BISI)'
  } else if (/swing high/i.test(htfDetails)) {
    keyLevelInfo = acuanHarga
      ? `Key level: ERL (Swing High · acuan ${acuanHarga})`
      : 'Key level: ERL (Swing High)'
  } else if (/swing low/i.test(htfDetails)) {
    keyLevelInfo = acuanHarga
      ? `Key level: ERL (Swing Low · acuan ${acuanHarga})`
      : 'Key level: ERL (Swing Low)'
  }

  let htfState: Ila2StepState = 'active'
  let htfHeadline = 'Menunggu C2 closure dalam cycle'
  const htfExtra: string[] = []

  if (isInvalid) {
    htfState = 'invalid'
    htfHeadline = htfTime
      ? `C2 close candle ${htfTime} — C3 selesai tanpa setup (Invalid)`
      : 'C3 sudah close tanpa setup (Invalid)'
  } else if (isC2Valid) {
    htfState = 'done'
    htfHeadline = htfTime ? `C2 valid — close candle ${htfTime}` : 'C2 valid (closure terkonfirmasi)'
    if (keyLevelInfo) htfExtra.push(keyLevelInfo)
    if (isWaitLtf) {
      htfExtra.push(`C3 ${htfLabel} sedang berjalan di dalam cycle`)
    }
  } else if (isC2Running) {
    htfState = 'active'
    htfHeadline = 'Candle C2 berjalan menyentuh acuan (belum close)'
    if (keyLevelInfo) htfExtra.push(keyLevelInfo)
  } else {
    htfState = 'active'
    if (/konflik arah/i.test(summaryText)) {
      htfHeadline = 'Konflik arah pada C2 (bearish & bullish sekaligus) — dilewati'
    } else if (/di luar cycle/i.test(summaryText)) {
      htfHeadline = 'C2/C3 di luar time cycle — menunggu cycle baru'
    } else {
      htfHeadline = 'Menunggu C2 closure pada key level (ERL/IRL)'
    }
    if (keyLevelInfo) htfExtra.push(keyLevelInfo)
  }

  // --- 2. LTF (ERL LTF & CISD Kondisi 1 / 2 / 3) ---
  const cisdLabel = cisdName(signal.direction, cisdDetails)
  const keyLevelLtf = extractMatch(cisdDetails, /key level\s+((?:swing high|swing low)\s+[\d.,]+)/i)
  const breakTime =
    formatJamRingkas(signal.cisd?.break_time) ??
    extractMatch(cisdDetails, /pada\s+([^;]+?)(?:;|$)/i)

  let ltfState: Ila2StepState = 'pending'
  let erlStatus = 'Menunggu C2 HTF valid'
  let cisdStatus = 'Menunggu fase LTF aktif'

  if (isInvalid) {
    ltfState = 'invalid'
    erlStatus = 'Setup dibatalkan — C3 HTF sudah close'
    cisdStatus = 'Tidak ada konfirmasi CISD dalam masa C3'
  } else if (isEntryConfirmed) {
    ltfState = 'done'
    if (/KONDISI_1/i.test(cisdDetails)) {
      erlStatus = keyLevelLtf
        ? `Swept ERL LTF valid terkena wick (${keyLevelLtf})`
        : 'Swept ERL LTF valid terkena wick'
      cisdStatus = `${cisdLabel} Kondisi 1 (ERL valid + wick)${breakTime ? ` · close ${breakTime}` : ''}`
    } else if (/KONDISI_2/i.test(cisdDetails)) {
      erlStatus = keyLevelLtf
        ? `ERL LTF FTM terbentuk (${keyLevelLtf}) — terkonfirmasi iFVG`
        : 'ERL LTF FTM terbentuk — terkonfirmasi iFVG'
      cisdStatus = `${cisdLabel} Kondisi 2 (FTM + iFVG)${breakTime ? ` · close ${breakTime}` : ''}`
    } else if (/KONDISI_3/i.test(cisdDetails)) {
      erlStatus = keyLevelLtf
        ? `ERL LTF FTM terbentuk (${keyLevelLtf}) — terkonfirmasi SWL/SWH FTM`
        : 'ERL LTF FTM terbentuk — terkonfirmasi SWL/SWH FTM'
      cisdStatus = `${cisdLabel} Kondisi 3 (FTM + SWL/SWH FTM)${breakTime ? ` · close ${breakTime}` : ''}`
    } else {
      erlStatus = keyLevelLtf ? `ERL LTF terkonfirmasi (${keyLevelLtf})` : 'ERL LTF terkonfirmasi'
      cisdStatus = `${cisdLabel} terkonfirmasi${breakTime ? ` · close ${breakTime}` : ''}`
    }
  } else if (isWaitLtf) {
    ltfState = 'active'
    if (/valid tapi belum terkena wick/i.test(cisdDetails)) {
      erlStatus = 'ERL LTF valid — menunggu wick menyentuh level'
      cisdStatus = `Menunggu ${cisdLabel} Kondisi 1 (ERL valid + wick)`
    } else if (/belum iFVG/i.test(cisdDetails)) {
      erlStatus = 'FTM terbentuk — menunggu iFVG terkonfirmasi'
      cisdStatus = `Menunggu ${cisdLabel} Kondisi 2 (FTM + iFVG)`
    } else if (/menunggu SW[LH]\s*FTM|tanpa iFVG/i.test(cisdDetails)) {
      erlStatus = 'FTM terbentuk (tanpa iFVG) — menunggu SWL/SWH FTM'
      cisdStatus = `Menunggu ${cisdLabel} Kondisi 3 (FTM + SWL/SWH FTM)`
    } else if (/belum ada swing/i.test(cisdDetails)) {
      erlStatus = 'Sedang menunggu terbentuknya ERL LTF (Swing High/Low)'
      cisdStatus = `Menunggu ${cisdLabel} body close`
    } else {
      erlStatus = 'Sedang memantau sweep ERL LTF / FTM'
      cisdStatus =
        signal.cisd && signal.cisd.cisd_level > 0
          ? `Menunggu ${cisdLabel} body close menembus level CISD`
          : `Menunggu ${cisdLabel} terkonfirmasi`
    }
  }

  // --- 3. Entry ---
  let entryState: Ila2StepState = 'pending'
  let entryHeadline = 'Belum ada'
  let entryNote: string | undefined

  if (isInvalid) {
    entryState = 'invalid'
    entryHeadline = 'Belum ada (Setup invalid)'
    entryNote = `C3 ${htfLabel} sudah close sebelum CISD ${ltfLabel} terkonfirmasi.`
  } else if (isEntryConfirmed && signal.entry_price > 0) {
    entryState = 'done'
    entryHeadline = 'Entry terkonfirmasi'
  } else {
    entryState = 'pending'
    entryHeadline = 'Belum ada'
    entryNote = isWaitLtf
      ? `Menunggu konfirmasi ${cisdLabel} di ${ltfLabel}.`
      : `Menunggu C2 ${htfLabel} valid dan konfirmasi CISD di ${ltfLabel}.`
  }

  const completedSteps =
    (htfState === 'done' ? 1 : 0) +
    (ltfState === 'done' ? 1 : 0) +
    (entryState === 'done' ? 1 : 0)

  return {
    completedSteps,
    htf: {
      state: htfState,
      title: `HTF (${htfLabel})`,
      headline: htfHeadline,
      details: htfExtra,
    },
    ltf: {
      state: ltfState,
      title: `LTF (${ltfLabel})`,
      erlStatus,
      cisdStatus,
      cisdLevel: signal.cisd && signal.cisd.cisd_level > 0 ? signal.cisd.cisd_level : undefined,
    },
    entry: {
      state: entryState,
      confirmed: entryState === 'done',
      headline: entryHeadline,
      note: entryNote,
      direction: signal.direction,
      entryPrice: signal.entry_price > 0 ? signal.entry_price : undefined,
      stopLoss: signal.stop_loss > 0 ? signal.stop_loss : undefined,
      takeProfit: signal.take_profit > 0 ? signal.take_profit : undefined,
      riskReward: signal.risk_reward > 0 ? signal.risk_reward : undefined,
    },
  }
}
