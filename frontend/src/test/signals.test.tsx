import { describe, expect, it } from 'vitest'
import type { ReactNode } from 'react'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AuthProvider } from '@/features/auth/AuthProvider'
import { SignalRow } from '@/features/signals/components/SignalRow'
import { SignalDetail } from '@/features/signals/components/SignalDetail'
import { formatDateTime, formatNumber, formatR } from '@/lib/format'
import { SIGNAL_LABEL, isFinal } from '@/features/signals/lib/status'
import type { Signal } from '@/types/api'

function renderWithProviders(ui: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: 0 }, queries: { retry: 0 } },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <AuthProvider>{ui}</AuthProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

const signal: Signal = {
  signal_id: 'sig-1',
  symbol: 'XAUUSD',
  strategy_version: 'ila-2.0',
  engine_version: 'v1.4.2',
  direction: 'BUY',
  entry_price: 4120.5,
  stop_loss: 4112.3,
  take_profit: 4149.2,
  risk_reward: 2.3,
  status: 'REJECTED',
  summary: 'Sweep lalu FVG terakhir.',
  next_action: 'Tunggu retest.',
  reject_reason: 'Batas harian tercapai.',
  published_at: '2026-11-07T10:00:00Z',
  expires_at: '2026-11-07T11:00:00Z',
  can_confirm: false,
  delayed: false,
}

describe('SignalRow', () => {
  it('menampilkan simbol, arah, dan label status', () => {
    renderWithProviders(<SignalRow signal={signal} selected={false} onSelect={() => {}} />)

    expect(screen.getByText('XAUUSD')).toBeInTheDocument()
    expect(screen.getByText('BUY')).toBeInTheDocument()
    expect(screen.getByText('Tertahan')).toBeInTheDocument()
  })

  it('menandai baris terpilih lewat aria-pressed', () => {
    renderWithProviders(<SignalRow signal={signal} selected onSelect={() => {}} />)

    expect(screen.getByRole('button')).toHaveAttribute('aria-pressed', 'true')
  })
})

describe('SignalDetail', () => {
  it('menampilkan alasan penolakan dan langkah berikutnya', () => {
    renderWithProviders(<SignalDetail signal={signal} />)

    expect(screen.getByText(/Batas harian tercapai/)).toBeInTheDocument()
    expect(screen.getByText(/Tunggu retest/)).toBeInTheDocument()
  })

  it('tidak menyediakan tombol eksekusi', () => {
    renderWithProviders(<SignalDetail signal={signal} />)

    expect(screen.queryByRole('button')).toBeNull()
  })
})

describe('status sinyal', () => {
  it('mengenali status final', () => {
    expect(isFinal('EXPIRED')).toBe(true)
    expect(isFinal('CONSUMED')).toBe(true)
    expect(isFinal('READY_TO_ENTRY')).toBe(false)
  })

  it('punya label untuk setiap status', () => {
    expect(SIGNAL_LABEL.READY_TO_ENTRY).toBe('Siap entry')
    expect(SIGNAL_LABEL.CONSUMED).toBe('Sudah dieksekusi')
  })
})

describe('format', () => {
  it('membulatkan angka sesuai digit yang diminta', () => {
    expect(formatNumber(4120.5, 2)).toBe('4.120,50')
  })

  it('memberi tanda plus hanya untuk R positif', () => {
    expect(formatR(2)).toBe('+2.00R')
    expect(formatR(-1)).toBe('-1.00R')
  })

  it('mengembalikan tanda pisah untuk tanggal kosong', () => {
    expect(formatDateTime(null)).toBe('—')
  })
})
