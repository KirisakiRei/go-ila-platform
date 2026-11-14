import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import type { ReactNode } from 'react'

// Riwayat harus muncul saat halaman baru dibuka, tanpa perlu menjalankan
// backtest lebih dulu. Uji ini menjaga janji itu: run yang sudah ada di server
// wajib tampil sejak render pertama.
//
// Sejak daftar riwayat punya halamannya sendiri, riwayat tidak lagi berada di
// halaman kerja backtest. Karena itu uji diarahkan ke halaman riwayat.

const historyResponse = vi.fn()

vi.mock('@/lib/api', async () => {
  const asli = await vi.importActual<typeof import('@/lib/api')>('@/lib/api')
  return {
    ...asli,
    backtestApi: {
      ...asli.backtestApi,
      history: () => historyResponse(),
      quota: () =>
        Promise.resolve({
          tier: 'plus',
          batas_harian: 20,
          terpakai: 0,
          sisa: 20,
          maks_hari: 190,
          rentang_maks_bulan: 6,
          bisa_jalankan: true,
        }),
      coverage: () => new Promise(() => {}),
    },
  }
})

vi.mock('@/features/backtest/components/BacktestForm', () => ({
  BacktestForm: () => <div data-testid="form" />,
}))

import BacktestPage from '@/pages/BacktestPage'
import BacktestHistoryPage from '@/pages/BacktestHistoryPage'

function bungkus(isi: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/backtest/riwayat']}>
        <Routes>
          <Route path="/backtest" element={isi} />
          <Route path="/backtest/riwayat" element={isi} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  )
}

function runLama() {
  return {
    id: 'run-1',
    symbol: 'XAUUSD',
    htf: 'D1',
    ltf: 'H1',
    status: 'done' as const,
    dari: 1_750_000_000,
    sampai: 1_790_000_000,
    equity: 10000,
    risk: 0.5,
    dibuat_pada: 1_790_000_000,
    strategi: 'ila-2.0',
    versi: 'ila-2.0-v1.0.0',
    statistik: {
      total_trade: 7,
      menang: 3,
      kalah: 4,
      kedaluwarsa: 0,
      candle_ambigu: 0,
      sinyal_ditolak: 0,
      win_rate: 0.4286,
      total_r: 1.5,
      maks_drawdown: 0.02,
      profit_factor: 1.4,
      ekuitas_akhir: 10150,
      ekuitas_mulai: 10000,
    },
  }
}

describe('Halaman riwayat backtest', () => {
  beforeEach(() => {
    historyResponse.mockReset()
  })

  it('menampilkan run yang sudah selesai tanpa menjalankan backtest baru', async () => {
    historyResponse.mockResolvedValue({ data: [runLama()], meta: { total: 1 } })

    render(bungkus(<BacktestHistoryPage />))

    await waitFor(() => {
      expect(screen.getByText('XAUUSD')).toBeInTheDocument()
    })
    // 7 trade dan 43% adalah angka run lama, bukan hasil run baru.
    expect(screen.getByText('7')).toBeInTheDocument()
    expect(screen.getByText('43%')).toBeInTheDocument()
    // Kartu membuka halaman detail run, bukan memuat ulang halaman kerja.
    expect(screen.getByRole('link', { name: 'XAUUSD' })).toHaveAttribute(
      'href',
      '/backtest/run-1',
    )
  })

  it('menampilkan keterangan kosong bila memang belum ada riwayat', async () => {
    historyResponse.mockResolvedValue({ data: [], meta: { total: 0 } })

    render(bungkus(<BacktestHistoryPage />))

    await waitFor(() => {
      expect(screen.getByText('Belum ada backtest')).toBeInTheDocument()
    })
  })

  it('halaman kerja tidak memuat riwayat, tetapi menautkannya', async () => {
    historyResponse.mockResolvedValue({ data: [runLama()], meta: { total: 1 } })

    render(bungkus(<BacktestPage />))

    await waitFor(() => expect(screen.getByTestId('form')).toBeInTheDocument())
    // Daftar run hanya ada di halaman riwayat.
    expect(screen.queryByText('XAUUSD')).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Riwayat/ })).toHaveAttribute(
      'href',
      '/backtest/riwayat',
    )
  })
})
