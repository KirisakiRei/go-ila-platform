import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { buildIla2Timeline } from '@/features/signals/lib/ila2-progress'
import { SignalDetail } from '@/features/signals/components/SignalDetail'
import { TimeframeRadarMatrix } from '@/features/signals/components/TimeframeRadarMatrix'
import type { Signal } from '@/types/api'

function makeIla2Signal(over: Partial<Signal> = {}): Signal {
  return {
    signal_id: 'sig-xau-h4-m15',
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
    published_at: '2026-11-18T12:00:00Z',
    updated_at: '2026-11-18T12:00:00Z',
    expires_at: '2026-11-18T16:00:00Z',
    can_confirm: false,
    delayed: false,
    ...over,
  }
}

describe('buildIla2Timeline', () => {
  it('menangani WAIT_HTF_SWEEP dan WAIT_HTF_CANDLE_CLOSE', () => {
    const waitC2 = buildIla2Timeline(makeIla2Signal({ status: 'WAIT_HTF_SWEEP' }))
    expect(waitC2.completedSteps).toBe(0)
    expect(waitC2.htf.state).toBe('active')
    expect(waitC2.ltf.state).toBe('pending')
    expect(waitC2.entry.confirmed).toBe(false)
    expect(waitC2.entry.headline).toBe('Belum ada')

    const running = buildIla2Timeline(
      makeIla2Signal({
        status: 'WAIT_HTF_CANDLE_CLOSE',
        htf_sweep: {
          detected: false,
          sweep_in_progress: true,
          direction: 'SELL',
          stop_loss: 0,
          target_dol: 0,
          details: 'SIBI H4: Candle berjalan menyentuh acuan 2650.00 pada SIBI, status C2 belum final',
        },
      }),
    )
    expect(running.htf.state).toBe('active')
    expect(running.htf.headline).toMatch(/belum close/i)
    expect(running.htf.details[0]).toMatch(/IRL \(SIBI/i)
  })

  it('membedakan status LTF pending (wick, FTM + iFVG, FTM + SWL)', () => {
    const baseHtf = {
      detected: true,
      sweep_in_progress: false,
      direction: 'SELL',
      stop_loss: 0,
      target_dol: 0,
      sweep_time: '2026-11-18T13:00:00Z',
      details: 'Swing High H4: C2 closure aktif pada Swing High di candle 18 Nov 08:00 NY (acuan 2655.40)',
    }

    const waitWick = buildIla2Timeline(
      makeIla2Signal({
        status: 'WAIT_LTF_CISD',
        htf_sweep: baseHtf,
        cisd: {
          detected: false,
          direction: 'SELL',
          cisd_level: 2651.2,
          extreme_sl: 2658.0,
          entry_price: 0,
          details:
            '-CISD belum terkonfirmasi: level CISD 2651.20 (open candle bullish pertama 18 Nov 09:15 NY), SL 2658.00. swing high 2656.00 valid tapi belum terkena wick',
        },
      }),
    )
    expect(waitWick.completedSteps).toBe(1)
    expect(waitWick.htf.state).toBe('done')
    expect(waitWick.htf.headline).toMatch(/C2 valid — close candle/i)
    expect(waitWick.ltf.erlStatus).toMatch(/menunggu wick/i)
    expect(waitWick.ltf.cisdStatus).toMatch(/Kondisi 1/i)

    const waitIfvg = buildIla2Timeline(
      makeIla2Signal({
        status: 'WAIT_LTF_CISD',
        htf_sweep: baseHtf,
        cisd: {
          detected: false,
          direction: 'SELL',
          cisd_level: 2651.2,
          extreme_sl: 2658.0,
          entry_price: 0,
          details:
            '-CISD belum terkonfirmasi: level CISD 2651.20 (...), SL 2658.00. swing high FTM, BISI terbaru di rangkaian belum iFVG',
        },
      }),
    )
    expect(waitIfvg.ltf.erlStatus).toMatch(/FTM terbentuk — menunggu iFVG/i)
    expect(waitIfvg.ltf.cisdStatus).toMatch(/Kondisi 2/i)

    const waitSwl = buildIla2Timeline(
      makeIla2Signal({
        status: 'WAIT_LTF_CISD',
        htf_sweep: baseHtf,
        cisd: {
          detected: false,
          direction: 'SELL',
          cisd_level: 2651.2,
          extreme_sl: 2658.0,
          entry_price: 0,
          details:
            '-CISD belum terkonfirmasi: level CISD 2651.20 (...), SL 2658.00. swing high FTM tanpa iFVG, menunggu SWL FTM',
        },
      }),
    )
    expect(waitSwl.ltf.erlStatus).toMatch(/menunggu SWL\/SWH FTM/i)
    expect(waitSwl.ltf.cisdStatus).toMatch(/Kondisi 3/i)
  })

  it('menangani READY_TO_ENTRY (Kondisi 1/2/3) dan INVALIDATED', () => {
    const ready = buildIla2Timeline(
      makeIla2Signal({
        status: 'READY_TO_ENTRY',
        direction: 'SELL',
        entry_price: 2648.5,
        stop_loss: 2658.5,
        take_profit: 2628.5,
        risk_reward: 2,
        htf_sweep: {
          detected: true,
          sweep_in_progress: false,
          direction: 'SELL',
          stop_loss: 0,
          target_dol: 0,
          sweep_time: '2026-11-18T13:00:00Z',
          details: 'SIBI H4: C2 closure aktif pada SIBI di candle 18 Nov 08:00 NY (acuan 2654.00)',
        },
        cisd: {
          detected: true,
          direction: 'SELL',
          cisd_level: 2650.0,
          extreme_sl: 2658.5,
          entry_price: 2648.5,
          break_time: '2026-11-18T14:30:00Z',
          details:
            '-CISD terkonfirmasi (KONDISI_2): close 2648.50 menembus level CISD 2650.00 pada 18 Nov 09:30 NY; key level swing high 2655.00; SL 2658.50',
        },
      }),
    )
    expect(ready.completedSteps).toBe(3)
    expect(ready.ltf.erlStatus).toMatch(/FTM terbentuk.*iFVG/i)
    expect(ready.ltf.cisdStatus).toMatch(/-CISD Kondisi 2/i)
    expect(ready.entry.confirmed).toBe(true)
    expect(ready.entry.headline).toBe('Entry terkonfirmasi')

    const inv = buildIla2Timeline(makeIla2Signal({ status: 'INVALIDATED' }))
    expect(inv.htf.state).toBe('invalid')
    expect(inv.ltf.state).toBe('invalid')
    expect(inv.entry.state).toBe('invalid')
    expect(inv.entry.confirmed).toBe(false)
  })
})

describe('SignalDetail & TimeframeRadarMatrix — New ILA 2.0 Time Cycle', () => {
  it('merender vertical stepper HTF -> LTF -> Entry dengan parameter posisi saat terkonfirmasi', () => {
    const sig = makeIla2Signal({
      status: 'READY_TO_ENTRY',
      direction: 'SELL',
      entry_price: 2648.5,
      stop_loss: 2658.5,
      take_profit: 2628.5,
      risk_reward: 2,
      htf_sweep: {
        detected: true,
        sweep_in_progress: false,
        direction: 'SELL',
        stop_loss: 0,
        target_dol: 0,
        sweep_time: '2026-11-18T13:00:00Z',
        details: 'SIBI H4: C2 closure aktif pada SIBI di candle 18 Nov 08:00 NY (acuan 2654.00)',
      },
      cisd: {
        detected: true,
        direction: 'SELL',
        cisd_level: 2650.0,
        extreme_sl: 2658.5,
        entry_price: 2648.5,
        details:
          '-CISD terkonfirmasi (KONDISI_1): close 2648.50 menembus level CISD 2650.00 pada 18 Nov 09:30 NY; key level swing high 2655.00; SL 2658.50',
      },
    })

    render(<SignalDetail signal={sig} />)

    expect(screen.getByText('Daily Cycle')).toBeInTheDocument()
    expect(screen.getByText('HTF (H4)')).toBeInTheDocument()
    expect(screen.getByText('LTF (M15)')).toBeInTheDocument()
    expect(screen.getByText(/Swept ERL LTF valid terkena wick/i)).toBeInTheDocument()
    expect(screen.getByText(/-CISD Kondisi 1/i)).toBeInTheDocument()
    expect(screen.getByText('Entry terkonfirmasi')).toBeInTheDocument()
    expect(screen.getByText('2648,50')).toBeInTheDocument()
    expect(screen.getByText('2658,50')).toBeInTheDocument()
    expect(screen.getByText('2628,50')).toBeInTheDocument()
  })

  it('menampilkan 5 tab Time Cycle baku pada TimeframeRadarMatrix ILA 2.0', () => {
    render(<TimeframeRadarMatrix signals={[]} strategy="ila-2.0" symbol="XAUUSD" />)

    expect(screen.getByRole('button', { name: /Monthly Cycle/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Weekly Cycle/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Daily Cycle/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /90M Cycle/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Micro Cycle/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Semua Cycle/i })).toBeInTheDocument()
  })
})
