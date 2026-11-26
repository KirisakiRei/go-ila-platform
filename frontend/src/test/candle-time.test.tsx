import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { candleEnd, normalizeTimeframe, snapKeBar } from '@/features/charts/lib/candleTime'
import { PriceChart } from '@/features/charts/PriceChart'

const { setMarkers } = vi.hoisted(() => ({ setMarkers: vi.fn() }))
vi.mock('lightweight-charts', () => ({
  createChart: vi.fn(() => ({
    addSeries: vi.fn(() => ({
      applyOptions: vi.fn(),
      setData: vi.fn(),
      update: vi.fn(),
      createPriceLine: vi.fn(),
      removePriceLine: vi.fn(),
      attachPrimitive: vi.fn(),
    })),
    applyOptions: vi.fn(),
    removeSeries: vi.fn(),
    remove: vi.fn(),
    timeScale: () => ({
      fitContent: vi.fn(),
      getVisibleLogicalRange: () => null,
      setVisibleLogicalRange: vi.fn(),
    }),
  })),
  createSeriesMarkers: vi.fn(() => ({ setMarkers })),
  CandlestickSeries: 'Candlestick',
  LineSeries: 'Line',
  LineStyle: { Solid: 0, Dashed: 1, Dotted: 2 },
}))
const unix = (iso: string) => Date.parse(iso) / 1000

const intervals = [
  [' daily ', 'D1', '2026-11-25', '2026-11-26'],
  ['D', 'D1', '2026-11-25', '2026-11-26'],
  ['WEEKLY', 'W1', '2026-11-23', '2026-11-30'],
  ['W', 'W1', '2026-11-23', '2026-11-30'],
  ['MN1', 'MN', '2026-02-01', '2026-03-01'],
  ['MONTHLY', 'MN', '2024-02-01', '2024-03-01'],
  ['MN', 'MN', '2026-07-01', '2026-08-01'],
  ['Q1', 'Q', '2026-10-01', '2027-01-01'],
  ['QUARTERLY', 'Q', '2026-01-01', '2026-04-01'],
  ['1M', 'M1', '2026-11-25T10:00:00Z', '2026-11-25T10:01:00Z'],
  ['4H', 'H4', '2026-11-25T08:00:00Z', '2026-11-25T12:00:00Z'],
]

describe('shared candle time bounds', () => {
  it.each(intervals)(
    '%s normalizes to %s with an exclusive calendar end',
    (alias, tf, start, end) => {
      const open = unix(start!)
      const close = unix(end!)
      expect(normalizeTimeframe(alias!)).toBe(tf)
      expect(candleEnd(open, alias)).toBe(close)
      expect(snapKeBar(open, [open], close)).toBe(open)
      expect(snapKeBar(close - 1, [open], close)).toBe(open)
      expect(snapKeBar(open - 1, [open], close)).toBeNull()
      expect(snapKeBar(close, [open], close)).toBeNull()
      expect(snapKeBar(close + 1, [open], close)).toBeNull()
    },
  )
  it('does not infer a last candle duration from sparse bars or unknown timeframes', () => {
    const times = [unix('2026-01-01'), unix('2026-03-01')]
    expect(snapKeBar(times[1]! + 1, times)).toBeNull()
    expect(candleEnd(times[1]!, 'X9')).toBeUndefined()
    expect(normalizeTimeframe('toString')).toBeUndefined()
    expect(snapKeBar(NaN, times)).toBeNull()
    expect(snapKeBar(1, [])).toBeNull()
  })
})

describe('actual PriceChart marker alignment', () => {
  it.each([
    ['DAILY', '2026-11-25', '2026-11-26'],
    ['MONTHLY', '2026-02-01', '2026-03-01'],
    ['MN', '2024-02-01', '2024-03-01'],
  ])('retains the last %s candle marker in embedded and fullscreen charts', (tf, start, end) => {
    const open = unix(start)
    const close = unix(end)
    const marker = (time: number, text: string) => ({
      time,
      text,
      position: 'belowBar' as const,
      shape: 'arrowUp' as const,
      color: '#fff',
    })
    setMarkers.mockClear()
    render(
      <PriceChart
        timeframe={tf}
        candles={[
          { time: unix('2023-01-01'), open: 1, high: 2, low: 1, close: 2, volume: 0 },
          { time: open, open: 1, high: 2, low: 1, close: 2, volume: 0 },
        ]}
        markers={[
          marker(close - 1, 'entry'),
          marker(close, 'boundary'),
          marker(close + 1, 'outside'),
        ]}
      />,
    )
    const expected = [expect.objectContaining({ time: open, text: 'entry' })]
    expect(setMarkers).toHaveBeenLastCalledWith(expected)
    setMarkers.mockClear()
    fireEvent.click(screen.getByRole('button', { name: 'Perbesar chart ke layar penuh' }))
    expect(setMarkers).toHaveBeenLastCalledWith(expected)
  })
})
