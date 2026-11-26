import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import type { Position } from '@/types/api'

const marketCandles = vi.hoisted(() => vi.fn())
vi.mock('@/lib/queries', () => ({ useMarketCandles: marketCandles }))
vi.mock('@/features/charts/PriceChart', () => ({
  PriceChart: (props: {
    priceLines: { price: number }[]
    segments: { fromPrice: number; toPrice: number }[]
    footer: React.ReactNode
    emptyMessage: string
  }) => (
    <div data-testid="chart">
      <span>{props.priceLines.map((line) => line.price).join('|')}</span>
      <span>
        {props.segments.map((segment) => `${segment.fromPrice}→${segment.toPrice}`).join('|')}
      </span>
      {props.footer}
      <span>{props.emptyMessage}</span>
    </div>
  ),
}))
import { PositionExecutionChart } from '@/features/charts/PositionExecutionChart'

const position = {
  id: 'p1',
  symbol: 'XAUUSD',
  source_signal_id: 's1',
  htf: 'H4',
  ltf: 'M5',
  feed_source: 'ICMARKETS',
  direction: 'BUY',
  lot_size: 0.1,
  open_price: 4001,
  stop_loss: 3991,
  take_profit: 4031,
  exit_price: 4030,
  opened_at: '2026-11-25T10:00:00Z',
  closed_at: '2026-11-25T11:00:00Z',
  status: 'CLOSED_TP',
  pnl_known: true,
  profit_loss: 29,
  outcome_r: 2.9,
} as Position

beforeEach(() => {
  marketCandles.mockReset()
  marketCandles.mockReturnValue({
    data: {
      feed: 'ICMARKETS',
      candles: [
        {
          time: Date.parse(position.opened_at) / 1000,
          open: 4000,
          high: 4040,
          low: 3980,
          close: 4020,
        },
      ],
    },
    isLoading: false,
    isError: false,
  })
})

describe('Position execution HTF/LTF', () => {
  it('renders two desktop panels with actual broker levels and static closed requests', () => {
    render(<PositionExecutionChart position={position} />)
    expect(screen.getAllByTestId('chart')).toHaveLength(2)
    expect(screen.getAllByText('4001|3991|4031')).toHaveLength(2)
    expect(screen.getAllByText('4001→4030')).toHaveLength(2)
    expect(marketCandles).toHaveBeenCalledWith('XAUUSD', 'H4', 500, true, { staticSnapshot: true })
    expect(marketCandles).toHaveBeenCalledWith('XAUUSD', 'M5', 500, true, { staticSnapshot: true })
  })

  it('loads one mobile panel and supports keyboard tab switching', () => {
    render(<PositionExecutionChart position={position} mobile enabled={false} />)
    expect(screen.getAllByTestId('chart')).toHaveLength(1)
    const htf = screen.getByRole('tab', { name: /HTF/ })
    const ltf = screen.getByRole('tab', { name: /LTF/ })
    fireEvent.keyDown(htf, { key: 'ArrowRight' })
    expect(ltf).toHaveAttribute('aria-selected', 'true')
    expect(ltf).toHaveFocus()
    expect(marketCandles).toHaveBeenLastCalledWith('XAUUSD', 'M5', 500, false, {
      staticSnapshot: true,
    })
    expect(screen.getByRole('tabpanel')).toHaveAttribute('aria-labelledby', ltf.id)
  })

  it('falls back to the existing auto timeframe when source metadata is absent', () => {
    render(
      <PositionExecutionChart
        position={{
          ...position,
          htf: undefined,
          ltf: undefined,
          opened_at: new Date().toISOString(),
        }}
      />,
    )
    expect(screen.getAllByTestId('chart')).toHaveLength(1)
    expect(screen.getByText(/memakai timeframe otomatis/)).toBeInTheDocument()
    expect(marketCandles).toHaveBeenLastCalledWith('XAUUSD', 'M15', 500, true, {
      staticSnapshot: true,
    })
  })

  it('labels missing historical candles and feed mismatch without inventing data', () => {
    marketCandles.mockReturnValue({
      data: {
        feed: 'FOREXCOM',
        candles: [{ time: Date.now() / 1000, open: 4100, high: 4110, low: 4090, close: 4105 }],
      },
    })
    render(<PositionExecutionChart position={position} mobile />)
    expect(screen.getByText(/Histori candle saat entry tidak tersedia/)).toBeInTheDocument()
    expect(screen.getByText(/Feed source ICMARKETS tidak tersedia/)).toBeInTheDocument()
  })

  it.each([
    ['DAILY', 'D1', '2026-11-25', '2026-11-26'],
    ['WEEKLY', 'W1', '2026-11-23', '2026-11-30'],
    ['MN1', 'MN', '2026-11-01', '2026-12-01'],
    ['Q', 'Q', '2026-10-01', '2027-01-01'],
  ])('requests canonical %s and uses the true last candle end', (alias, tf, start, end) => {
    marketCandles.mockReturnValue({
      data: {
        candles: [
          { time: Date.parse(start) / 1000, open: 4000, high: 4040, low: 3980, close: 4020 },
        ],
      },
    })
    const { rerender } = render(<PositionExecutionChart position={position} timeframe={alias} />)
    expect(marketCandles).toHaveBeenLastCalledWith('XAUUSD', tf, 500, true, {
      staticSnapshot: true,
    })
    expect(screen.queryByText(/Histori candle saat entry tidak tersedia/)).not.toBeInTheDocument()
    rerender(
      <PositionExecutionChart
        position={{ ...position, opened_at: `${end}T00:00:00Z` }}
        timeframe={alias}
      />,
    )
    expect(screen.getByText(/Histori candle saat entry tidak tersedia/)).toBeInTheDocument()
  })
  it('keeps loading, error and empty states explicit', () => {
    marketCandles.mockReturnValue({ isLoading: true })
    const { rerender } = render(<PositionExecutionChart position={position} mobile />)
    expect(screen.queryByTestId('chart')).not.toBeInTheDocument()
    marketCandles.mockReturnValue({ isError: true })
    rerender(<PositionExecutionChart position={position} mobile />)
    expect(screen.getByText(/Chart posisi belum bisa dimuat/)).toBeInTheDocument()
    marketCandles.mockReturnValue({ data: { candles: [] } })
    rerender(<PositionExecutionChart position={position} mobile />)
    expect(screen.getByText(/Candle untuk rentang posisi ini tidak tersedia/)).toBeInTheDocument()
  })
})
