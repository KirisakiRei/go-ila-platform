import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render } from '@testing-library/react'
import type { MarketCandle } from '@/types/api'

/**
 * Tes ini memastikan presisi harga benar-benar sampai ke lightweight-charts,
 * bukan hanya benar di modul bantu. Kesalahan yang ingin dicegah: angka sudah
 * dihitung benar tetapi chart tetap menggambar 2 desimal karena `precision`
 * tidak diteruskan.
 */

const applyOptions = vi.fn()
const addSeries = vi.fn(
  (
    _tipe: unknown,
    _opsi?: { priceFormat?: { precision?: number; minMove?: number } },
  ) => ({
    applyOptions,
    setData: vi.fn(),
    update: vi.fn(),
    createPriceLine: vi.fn(() => ({})),
    removePriceLine: vi.fn(),
    attachPrimitive: vi.fn(),
  }),
)

vi.mock('lightweight-charts', () => ({
  createChart: vi.fn(() => ({
    addSeries,
    applyOptions: vi.fn(),
    removeSeries: vi.fn(),
    timeScale: () => ({
      fitContent: vi.fn(),
      getVisibleLogicalRange: () => null,
      setVisibleLogicalRange: vi.fn(),
    }),
    remove: vi.fn(),
  })),
  createSeriesMarkers: vi.fn(() => ({ setMarkers: vi.fn() })),
  CandlestickSeries: 'Candlestick',
  LineSeries: 'Line',
  LineStyle: { Solid: 0, Dashed: 1, Dotted: 2 },
}))

const candles: MarketCandle[] = [
  { time: 1_700_000_000, open: 1.0846, high: 1.0848, low: 1.0845, close: 1.0847, volume: 0 },
]

describe('PriceChart presisi harga', () => {
  beforeEach(() => {
    applyOptions.mockClear()
    addSeries.mockClear()
  })

  it('memakai jumlah desimal pair saat chart dibuat', async () => {
    const { PriceChart } = await import('@/features/charts/PriceChart')
    render(<PriceChart candles={candles} digits={5} />)

    const opsi = addSeries.mock.calls[0]?.[1]
    expect(opsi?.priceFormat?.precision).toBe(5)
    // minMove harus 10^-5 agar sumbu digambar sehalus presisi datanya.
    expect(opsi?.priceFormat?.minMove).toBeCloseTo(0.00001, 10)
  })

  it('memakai 2 desimal untuk emas', async () => {
    const { PriceChart } = await import('@/features/charts/PriceChart')
    render(<PriceChart candles={candles} digits={2} />)

    const opsi = addSeries.mock.calls[0]?.[1]
    expect(opsi?.priceFormat?.precision).toBe(2)
  })

  it('memperbarui presisi saat pair berganti tanpa membuat ulang chart', async () => {
    const { PriceChart } = await import('@/features/charts/PriceChart')
    const { rerender } = render(<PriceChart candles={candles} digits={5} />)

    applyOptions.mockClear()
    rerender(<PriceChart candles={candles} digits={3} />)

    const dipanggil = applyOptions.mock.calls.some(
      (call) => (call[0] as { priceFormat?: { precision?: number } })?.priceFormat?.precision === 3,
    )
    expect(dipanggil).toBe(true)
  })
})
