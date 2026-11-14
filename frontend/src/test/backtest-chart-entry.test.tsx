import { describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'

// Chart entry backtest harus menampilkan dua panel: timeframe acuan (HTF) untuk
// konteks dan timeframe eksekusi (LTF) untuk titik masuk. Keduanya memakai
// level yang sama, sehingga satu kejadian terbaca dari dua jarak pandang.

const detailResponse = vi.fn()

vi.mock('@/lib/api', async () => {
  const asli = await vi.importActual<typeof import('@/lib/api')>('@/lib/api')
  return {
    ...asli,
    backtestApi: {
      ...asli.backtestApi,
      trade: () => detailResponse(),
    },
  }
})

// PriceChart diganti penangkap agar uji tidak perlu canvas WebGL.
vi.mock('@/features/charts/PriceChart', () => ({
  PriceChart: (props: {
    candles: unknown[]
    priceLines?: { title: string }[]
    markers?: { text?: string }[]
    zones?: { label?: string }[]
  }) => (
    <div data-testid="price-chart">
      <span data-testid="jumlah-candle">{props.candles.length}</span>
      <span data-testid="garis">{(props.priceLines ?? []).map((l) => l.title).join('|')}</span>
      <span data-testid="penanda">{(props.markers ?? []).map((m) => m.text).join('|')}</span>
      <span data-testid="zona">{(props.zones ?? []).map((z) => z.label).join('|')}</span>
    </div>
  ),
}))

import { BacktestTradeChart } from '@/features/backtest/components/BacktestTradeChart'

function bungkus(isi: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return <QueryClientProvider client={client}>{isi}</QueryClientProvider>
}

function candle(jam: number, harga: number) {
  return { time: 1_790_000_000 + jam * 3600, open: harga, high: harga + 2, low: harga - 2, close: harga, volume: 0 }
}

function detail() {
  const masuk = 1_790_000_000 + 10 * 3600
  const keluar = 1_790_000_000 + 30 * 3600
  return {
    seq: 3,
    arah: 'SELL' as const,
    waktu_masuk: masuk,
    waktu_keluar: keluar,
    entry: 4400,
    stop_loss: 4420,
    take_profit: 4360,
    rr_rencana: 2,
    hasil: 'TP' as const,
    r: 2,
    durasi_detik: 20 * 3600,
    ekuitas_akhir: 10200,
    ambigu: false,
    ringkasan: 'Setup lengkap',
    langkah_berikut: 'Masuk di candle pembuka',
    chart_tersedia: true,
    htf: 'D1',
    ltf: 'H1',
    candle_htf: [candle(0, 4401), candle(24, 4399)],
    candle_ltf: Array.from({ length: 40 }, (_, i) => candle(i, 4400)),
    zona_fvg: { top: 4424, bottom: 4353, dari: masuk, sampai: keluar, label: 'BEARISH' },
    sweep: { waktu: masuk - 7200, harga: 4382, label: 'sweep' },
    cisd: { waktu: masuk - 3600, harga: 4256, label: 'CISD' },
  }
}

describe('Chart entry backtest', () => {
  it('menggambar panel HTF dan LTF dengan level yang sama', async () => {
    detailResponse.mockResolvedValue(detail())

    render(
      bungkus(<BacktestTradeChart runID="run-1" symbol="XAUUSD" seq={3} />),
    )

    // Dua panel: acuan dan eksekusi.
    await waitFor(() => {
      expect(screen.getByText(/D1 — timeframe acuan/)).toBeInTheDocument()
    })
    expect(screen.getByText(/H1 — timeframe eksekusi/)).toBeInTheDocument()

    const panel = screen.getAllByTestId('price-chart')
    expect(panel).toHaveLength(2)

    // Kedua panel memuat entry, batas rugi, dan target yang sama.
    for (const satu of panel) {
      const garis = satu.querySelector('[data-testid="garis"]')?.textContent ?? ''
      expect(garis).toContain('Entry 4400.00')
      expect(garis).toContain('SL 4420.00')
      expect(garis).toContain('TP 4360.00')
    }
  })

  it('menandai sweep, CISD, entry, dan exit pada kedua panel', async () => {
    detailResponse.mockResolvedValue(detail())

    render(bungkus(<BacktestTradeChart runID="run-1" symbol="XAUUSD" seq={3} />))

    await waitFor(() => expect(screen.getAllByTestId('price-chart')).toHaveLength(2))

    for (const satu of screen.getAllByTestId('price-chart')) {
      const penanda = satu.querySelector('[data-testid="penanda"]')?.textContent ?? ''
      expect(penanda).toContain('SELL 4400.00')
      expect(penanda).toContain('sweep')
      expect(penanda).toContain('CISD')
      expect(penanda).toContain('TP +2.00R')
      expect(satu.querySelector('[data-testid="zona"]')?.textContent).toBe('BEARISH')
    }
  })

  it('memakai candle yang tersimpan, bukan menghitung ulang', async () => {
    detailResponse.mockResolvedValue(detail())

    render(bungkus(<BacktestTradeChart runID="run-1" symbol="XAUUSD" seq={3} />))

    await waitFor(() => expect(screen.getAllByTestId('price-chart')).toHaveLength(2))

    const jumlah = screen.getAllByTestId('jumlah-candle').map((satu) => satu.textContent)
    // HTF 2 candle, LTF 40 candle: persis yang dikirim server.
    expect(jumlah).toEqual(['2', '40'])
  })
})
