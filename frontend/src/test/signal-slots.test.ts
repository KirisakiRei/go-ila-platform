import { describe, expect, it } from 'vitest'

import { mapSignalsToSlots } from '@/features/signals/lib/signal-slots'
import type { Signal } from '@/types/api'

/**
 * sinyal membuat objek Signal minimal untuk pengujian pemetaan slot.
 * Hanya kolom yang dibaca mapSignalsToSlots yang diisi.
 */
function sinyal(over: Partial<Signal> & Pick<Signal, 'signal_id'>): Signal {
  return {
    symbol: 'XAUUSD',
    strategy_version: 'ila-2.0',
    engine_version: '2.0.0',
    htf: 'H4',
    ltf: 'M15',
    direction: 'SELL',
    entry_price: 0,
    stop_loss: 0,
    take_profit: 0,
    risk_reward: 0,
    status: 'WAIT_HTF_SWEEP',
    summary: '',
    next_action: '',
    reject_reason: '',
    published_at: '2026-11-09T14:59:00Z',
    updated_at: '2026-11-09T14:59:00Z',
    expires_at: '2026-11-09T15:14:00Z',
    can_confirm: false,
    delayed: false,
    ...over,
  }
}

describe('mapSignalsToSlots', () => {
  it('ILA 2.0 menghasilkan 4 slot pasangan timeframe', () => {
    const slots = mapSignalsToSlots([], 'ila-2.0')
    expect(slots).toHaveLength(4)
    expect(slots.every((s) => s.mtf === undefined)).toBe(true)
  })

  it('ILA 3.0 menghasilkan 4 siklus dengan MTF terisi', () => {
    const slots = mapSignalsToSlots([], 'ila-3.0')
    expect(slots).toHaveLength(4)
    expect(slots.every((s) => Boolean(s.mtf))).toBe(true)
    expect(slots.every((s) => Boolean(s.cycle))).toBe(true)
  })

  it('tidak mencampur sinyal ILA 3.0 ke dalam slot ILA 2.0', () => {
    const slots = mapSignalsToSlots(
      [
        sinyal({
          signal_id: 'a',
          strategy_version: 'ila-3.0',
          htf: 'D',
          ltf: '15M',
          status: 'READY_TO_ENTRY',
        }),
      ],
      'ila-2.0',
    )
    expect(slots.some((s) => s.latestSignal)).toBe(false)
  })

  it('memilih baris radar yang masih aktif meski published_at-nya lebih lama', () => {
    // Radar memakai identitas tetap: published_at lama, updated_at baru.
    // Sinyal EXPIRED terbit lebih baru tetapi sudah tidak bergerak.
    const radar = sinyal({
      signal_id: 'status-XAUUSD-H4-M15',
      status: 'WAIT_HTF_SWEEP',
      published_at: '2026-11-09T14:59:00Z',
      updated_at: '2026-11-12T07:36:00Z',
    })
    const expired = sinyal({
      signal_id: 'sig-XAUUSD-H4-M15-1',
      status: 'EXPIRED',
      published_at: '2026-11-11T14:00:00Z',
      updated_at: '2026-11-11T14:15:00Z',
    })

    // API mengirim urutan published_at DESC, jadi sinyal EXPIRED di depan.
    const slots = mapSignalsToSlots([expired, radar], 'ila-2.0')
    const slot = slots.find((s) => s.key === 'H4-M15')

    expect(slot?.latestSignal?.signal_id).toBe('status-XAUUSD-H4-M15')
    expect(slot?.status).toBe('WAIT_SWEEP')
  })
})
