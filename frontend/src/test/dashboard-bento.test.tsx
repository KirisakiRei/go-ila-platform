import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import {
  buildEquityCurve,
  buildRadarHighlights,
  buildSmoothCurvePath,
  findActionableSignals,
  labelTahapDashboard,
} from '@/features/dashboard/lib/dashboard-metrics'
import { PerformanceCurveCard } from '@/features/dashboard/components/PerformanceCurveCard'
import { RadarPriorityCard } from '@/features/dashboard/components/RadarPriorityCard'
import * as queries from '@/lib/queries'
import type { Position, Signal, Subscription } from '@/types/api'

function makePosition(over: Partial<Position> = {}): Position {
  return {
    id: 'pos-1',
    symbol: 'XAUUSD',
    strategy_version: 'ila-2.0',
    direction: 'SELL',
    ticket_number: '1001',
    lot_size: 0.1,
    open_price: 2650,
    stop_loss: 2660,
    take_profit: 2630,
    exit_price: 2630,
    profit_loss: 200,
    outcome_r: 2,
    risk_percent: 1,
    status: 'CLOSED_TP',
    is_dry_run: false,
    opened_at: '2026-11-18T10:00:00Z',
    opened_at_pair: { utc: '', wib: '', ny: '' },
    closed_at: '2026-11-19T12:00:00Z',
    closed_at_pair: { utc: '', wib: '', ny: '' },
    pnl_known: true,
    ...over,
  }
}

function makeSignal(over: Partial<Signal> = {}): Signal {
  return {
    signal_id: 'sig-1',
    symbol: 'XAUUSD',
    strategy_version: 'ila-2.0',
    engine_version: '2.0.0',
    htf: 'H4',
    ltf: 'M15',
    direction: 'SELL',
    status: 'WAIT_LTF_CISD',
    entry_price: 0,
    stop_loss: 0,
    take_profit: 0,
    risk_reward: 2,
    summary: '',
    next_action: '',
    published_at: '2026-11-21T10:00:00Z',
    updated_at: '2026-11-21T11:00:00Z',
    expires_at: '2026-11-21T16:00:00Z',
    ...over,
  } as Signal
}

describe('dashboard-metrics', () => {
  it('menghitung kurva kumulatif R, P/L, win rate, dan max drawdown sesuai periode', () => {
    const now = new Date('2026-11-21T12:00:00Z')
    const positions = [
      makePosition({ id: 'old', closed_at: '2026-10-01T12:00:00Z', profit_loss: 500, outcome_r: 2 }),
      makePosition({ id: 'p1', closed_at: '2026-11-16T10:00:00Z', profit_loss: 200, outcome_r: 2 }),
      makePosition({
        id: 'p2',
        closed_at: '2026-11-18T10:00:00Z',
        profit_loss: -100,
        outcome_r: -1,
        status: 'CLOSED_SL',
      }),
      makePosition({ id: 'open', status: 'OPEN', closed_at: null, pnl_known: false }),
    ]

    const d7 = buildEquityCurve(positions, '7d', now)
    expect(d7.count).toBe(2)
    expect(d7.totalR).toBe(1)
    expect(d7.totalPnL).toBe(100)
    expect(d7.winRate).toBe(50)
    expect(d7.maxDrawdownR).toBe(1)

    const all = buildEquityCurve(positions, 'all', now)
    expect(all.count).toBe(3)
    expect(all.totalR).toBe(3)
  })

  it('mengurutkan Radar Terpanas dari terdekat entry dengan istilah New ILA 2.0', () => {
    expect(labelTahapDashboard('ila-2.0', 'WAIT_HTF_CANDLE_CLOSE').label).toBe('HTF C2 Closure')
    expect(labelTahapDashboard('ila-2.0', 'WAIT_LTF_CISD').label).toBe('LTF CISD (C3)')
    expect(labelTahapDashboard('ila-2.0', 'READY_TO_ENTRY').label).toBe('Siap Entry')

    const signals = [
      makeSignal({ signal_id: 's1', symbol: 'EURUSD', status: 'WAIT_HTF_SWEEP', htf: 'H4', ltf: 'M15' }),
      makeSignal({ signal_id: 's2', symbol: 'XAUUSD', status: 'WAIT_LTF_CISD', htf: 'H4', ltf: 'M15' }),
      makeSignal({ signal_id: 's3', symbol: 'USTEC', status: 'READY_TO_ENTRY', htf: 'H1', ltf: 'M5', direction: 'BUY' }),
    ]

    const highlights = buildRadarHighlights(signals, ['EURUSD', 'USTEC', 'XAUUSD'])
    expect(highlights.map((h) => h.symbol)).toEqual(['USTEC', 'XAUUSD', 'EURUSD'])
    expect(highlights[0]?.stageLabel).toBe('Siap Entry')
    expect(highlights[0]?.cycleName).toBe('90M Cycle')
    expect(highlights[1]?.stageLabel).toBe('LTF CISD (C3)')
    expect(highlights[1]?.cycleName).toBe('Daily Cycle')

    expect(findActionableSignals(signals).map((s) => s.symbol)).toEqual(['USTEC'])
  })

  it('buildSmoothCurvePath menghasilkan kurva cubic Bezier halus untuk >= 3 titik', () => {
    const path = buildSmoothCurvePath([
      { x: 0, y: 100 },
      { x: 50, y: 40 },
      { x: 100, y: 130 },
    ])
    expect(path.startsWith('M 0.0 100.0 C ')).toBe(true)
    expect(path).toContain('100.0 130.0')
  })
})

describe('Komponen Dashboard Bento', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('PerformanceCurveCard merender kurva SVG dan berpindah toggle R <-> P/L', () => {
    vi.spyOn(queries, 'usePositions').mockReturnValue({
      data: [
        makePosition({ id: 'p1', closed_at: new Date().toISOString(), profit_loss: 200, outcome_r: 2 }),
      ],
      isLoading: false,
    } as unknown as ReturnType<typeof queries.usePositions>)

    render(
      <MemoryRouter>
        <PerformanceCurveCard />
      </MemoryRouter>,
    )

    expect(screen.getByRole('img', { name: /Kurva performa kumulatif 1 trade/i })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'P/L' }))
    expect(screen.getByText(/1 trade/i)).toBeInTheDocument()
  })

  it('RadarPriorityCard menampilkan Time Cycle & tahap New ILA 2.0', () => {
    vi.spyOn(queries, 'useSubscriptions').mockReturnValue({
      data: [{ id: 'sub-1', symbol: 'XAUUSD', is_active: true } as Subscription],
      isLoading: false,
    } as unknown as ReturnType<typeof queries.useSubscriptions>)

    vi.spyOn(queries, 'useSignals').mockReturnValue({
      data: [makeSignal({ symbol: 'XAUUSD', status: 'WAIT_LTF_CISD', htf: 'H4', ltf: 'M15', direction: 'SELL' })],
      isLoading: false,
    } as unknown as ReturnType<typeof queries.useSignals>)

    render(
      <MemoryRouter>
        <RadarPriorityCard />
      </MemoryRouter>,
    )

    expect(screen.getByText('XAUUSD')).toBeInTheDocument()
    expect(screen.getByText('LTF CISD (C3)')).toBeInTheDocument()
    expect(screen.getByText(/Daily Cycle/i)).toBeInTheDocument()
    expect(screen.getByText('-CISD')).toBeInTheDocument()
  })
})
