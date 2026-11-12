import { describe, expect, it } from 'vitest'

import {
  slotPalingDekat,
  stageILA2,
  stageILA3,
  stageOf,
} from '@/features/signals/lib/slot-stage'

/**
 * Kartu overview menampilkan tahap tiap slot dalam satu kata. Uji ini mengunci
 * pemetaan status engine -> label, karena label yang salah membuat pengguna
 * mengira setup lebih dekat (atau lebih jauh) dari kenyataan.
 */
describe('stageILA2', () => {
  it('memetakan tiap status ke label pendek', () => {
    expect(stageILA2('WAIT_HTF_SWEEP').label).toBe('Sweep')
    expect(stageILA2('WAIT_HTF_CANDLE_CLOSE').label).toBe('Candle')
    expect(stageILA2('WAIT_LTF_CISD').label).toBe('CISD')
    expect(stageILA2('READY_TO_ENTRY').label).toBe('Ready')
    expect(stageILA2('INVALIDATED').label).toBe('Batal')
  })

  it('hanya READY_TO_ENTRY yang dianggap siap dieksekusi', () => {
    expect(stageILA2('READY_TO_ENTRY').ready).toBe(true)
    for (const status of ['WAIT_HTF_SWEEP', 'WAIT_HTF_CANDLE_CLOSE', 'WAIT_LTF_CISD', 'INVALIDATED']) {
      expect(stageILA2(status).ready).toBe(false)
    }
  })

  it('mengurutkan kedekatan ke entry: Sweep > Candle > CISD > Ready', () => {
    const sweep = stageILA2('WAIT_HTF_SWEEP').rank
    const candle = stageILA2('WAIT_HTF_CANDLE_CLOSE').rank
    const cisd = stageILA2('WAIT_LTF_CISD').rank
    const ready = stageILA2('READY_TO_ENTRY').rank

    expect(candle).toBeLessThan(sweep)
    expect(cisd).toBeLessThan(candle)
    expect(ready).toBeLessThan(cisd)
  })

  it('status tak dikenal tidak menghasilkan undefined', () => {
    const hasil = stageILA2('ENTAH_APA')
    expect(hasil.label).toBe('Scan')
    expect(hasil.ready).toBe(false)
  })

  it('status kosong juga aman', () => {
    expect(stageILA2('').label).toBe('Scan')
  })

  it('tidak peka huruf besar/kecil dan spasi', () => {
    expect(stageILA2(' wait_ltf_cisd ').label).toBe('CISD')
  })
})

describe('stageILA3', () => {
  it('memetakan tiap status ke label pendek', () => {
    expect(stageILA3('WAIT_ILA3_HTF_KEY_LEVEL').label).toBe('Key Lvl')
    expect(stageILA3('WAIT_ILA3_MTF_REVERSAL').label).toBe('Reversal')
    expect(stageILA3('WAIT_ILA3_LTF_CISD_1').label).toBe('CISD 1')
    expect(stageILA3('WAIT_ILA3_MTF_FVG_RETR').label).toBe('Retrace')
    expect(stageILA3('WAIT_ILA3_LTF_CISD_2').label).toBe('CISD 2')
    expect(stageILA3('ILA3_TARGET_REACHED').label).toBe('Target')
  })

  it('dua titik entry sama-sama siap dieksekusi', () => {
    // Backend menormalkan READY_ENTRY_* ke READY_TO_ENTRY, tetapi keduanya
    // harus tetap benar bila engine mengirim nilai aslinya.
    expect(stageILA3('READY_TO_ENTRY').ready).toBe(true)
    expect(stageILA3('READY_ENTRY_1').ready).toBe(true)
    expect(stageILA3('READY_ENTRY_2').ready).toBe(true)
  })

  it('mengurutkan fase sesuai alur trading', () => {
    const keyLevel = stageILA3('WAIT_ILA3_HTF_KEY_LEVEL').rank
    const reversal = stageILA3('WAIT_ILA3_MTF_REVERSAL').rank
    const cisd1 = stageILA3('WAIT_ILA3_LTF_CISD_1').rank
    const ready = stageILA3('READY_TO_ENTRY').rank

    expect(reversal).toBeLessThan(keyLevel)
    expect(cisd1).toBeLessThan(reversal)
    expect(ready).toBeLessThan(cisd1)
  })

  it('status ILA 2.0 tidak dikenali oleh tabel ILA 3.0', () => {
    expect(stageILA3('WAIT_HTF_SWEEP').label).toBe('Scan')
  })
})

describe('stageOf', () => {
  it('memilih tabel sesuai metode', () => {
    expect(stageOf('ila-2.0', 'WAIT_HTF_SWEEP').label).toBe('Sweep')
    expect(stageOf('ila-3.0', 'WAIT_ILA3_HTF_KEY_LEVEL').label).toBe('Key Lvl')
  })
})

describe('slotPalingDekat', () => {
  const slots = [
    { key: 'W-H4', latestSignal: { status: 'WAIT_HTF_SWEEP' } },
    { key: 'H4-M15', latestSignal: { status: 'WAIT_LTF_CISD' } },
    { key: 'H1-M5', latestSignal: { status: 'WAIT_HTF_SWEEP' } },
  ]

  it('memilih slot yang tahapnya paling dekat entry', () => {
    // CISD lebih dekat daripada Sweep, jadi H4-M15 yang diberi aksen.
    expect(slotPalingDekat(slots, 'ila-2.0')).toBe('H4-M15')
  })

  it('mengabaikan slot tanpa sinyal', () => {
    const tanpaSinyal = [
      { key: 'W-H4' },
      { key: 'H1-M5', latestSignal: { status: 'WAIT_LTF_CISD' } },
    ]
    expect(slotPalingDekat(tanpaSinyal, 'ila-2.0')).toBe('H1-M5')
  })

  it('null bila tidak ada slot yang berjalan', () => {
    expect(slotPalingDekat([{ key: 'W-H4' }], 'ila-2.0')).toBeNull()
    expect(slotPalingDekat([], 'ila-2.0')).toBeNull()
  })

  it('READY dipilih di atas semua tahap lain', () => {
    const campur = [
      { key: 'A', latestSignal: { status: 'WAIT_LTF_CISD' } },
      { key: 'B', latestSignal: { status: 'READY_TO_ENTRY' } },
    ]
    expect(slotPalingDekat(campur, 'ila-2.0')).toBe('B')
  })

  it('null bila beberapa slot seri di tahap yang sama', () => {
    // Menyorot satu dari beberapa slot yang sama-sama di tahap Sweep akan
    // menyesatkan: tidak ada yang benar-benar lebih dekat entry.
    const seri = [
      { key: 'A', latestSignal: { status: 'WAIT_HTF_SWEEP' } },
      { key: 'B', latestSignal: { status: 'WAIT_HTF_SWEEP' } },
      { key: 'C', latestSignal: { status: 'WAIT_HTF_SWEEP' } },
    ]
    expect(slotPalingDekat(seri, 'ila-2.0')).toBeNull()
  })

  it('satu slot paling maju tetap disorot walau sisanya seri', () => {
    const campur = [
      { key: 'A', latestSignal: { status: 'WAIT_HTF_SWEEP' } },
      { key: 'B', latestSignal: { status: 'WAIT_HTF_SWEEP' } },
      { key: 'C', latestSignal: { status: 'WAIT_LTF_CISD' } },
    ]
    expect(slotPalingDekat(campur, 'ila-2.0')).toBe('C')
  })
})
