import { describe, expect, it } from 'vitest'
import type { ReactNode } from 'react'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AuthProvider } from '@/features/auth/AuthProvider'
import { SignalRow } from '@/features/signals/components/SignalRow'
import { SignalDetail } from '@/features/signals/components/SignalDetail'
import { TimeframeBadge } from '@/features/signals/components/TimeframeBadge'
import { TombolEksekusi } from '@/features/signals/components/TombolEksekusi'
import { formatDateTime, formatNumber, formatR } from '@/lib/format'
import { SIGNAL_LABEL, SIGNAL_TONE, isFinal } from '@/features/signals/lib/status'
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
  htf: 'H4',
  ltf: 'M15',
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

  // Engine ILA 3.0 mengirim tahapan sendiri. Sebelum labelnya didaftarkan,
  // badge status tampil kosong sehingga radar terlihat rusak.
  it('punya label dan tone untuk seluruh tahapan ILA 3.0', () => {
    const tahapan = [
      'WAIT_ILA3_HTF_KEY_LEVEL',
      'WAIT_ILA3_MTF_REVERSAL',
      'WAIT_ILA3_LTF_CISD_1',
      'ILA3_EXPANSION_1_ACTIVE',
      'WAIT_ILA3_MTF_FVG_RETR',
      'WAIT_ILA3_LTF_CISD_2',
      'ILA3_EXPANSION_2_ACTIVE',
      'ILA3_TARGET_REACHED',
    ] as const

    for (const status of tahapan) {
      expect(SIGNAL_LABEL[status]).toBeTruthy()
      expect(SIGNAL_TONE[status]).toBeTruthy()
    }
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

describe('TimeframeBadge', () => {
  it('menampilkan pasangan timeframe dari sinyal', () => {
    renderWithProviders(<TimeframeBadge htf="H4" ltf="M15" />)
    expect(screen.getByText('H4 · M15')).toBeDefined()
  })

  it('tidak menampilkan apa pun bila engine belum mengirim timeframe', () => {
    const { container } = renderWithProviders(<TimeframeBadge htf="" ltf="" />)
    expect(container.textContent).toBe('')
  })

  it('tetap tampil bila hanya satu sisi terisi', () => {
    renderWithProviders(<TimeframeBadge htf="H4" ltf="" />)
    expect(screen.getByText('H4')).toBeDefined()
  })
})

describe('TombolEksekusi', () => {
  const dasar: Signal = { ...signal, can_confirm: true, status: 'OPPORTUNITY' }

  it('menampilkan hitung mundur selama sinyal masih berlaku', () => {
    const berlaku = new Date(Date.now() + 5 * 60 * 1000).toISOString()
    renderWithProviders(<TombolEksekusi signal={{ ...dasar, expires_at: berlaku }} />)
    expect(screen.getByText(/Berlaku \d+m \d+s lagi/)).toBeDefined()
  })

  it('mematikan tombol setelah batas waktu lewat', () => {
    const lewat = new Date(Date.now() - 60 * 1000).toISOString()
    renderWithProviders(<TombolEksekusi signal={{ ...dasar, expires_at: lewat }} />)

    expect(screen.getByText('Batas waktu sudah lewat')).toBeDefined()
    const tombol = screen.getByRole('button', { name: /Eksekusi/ }) as HTMLButtonElement
    expect(tombol.disabled).toBe(true)
  })

  it('mematikan tombol bila server menyatakan belum bisa dikonfirmasi', () => {
    const berlaku = new Date(Date.now() + 5 * 60 * 1000).toISOString()
    renderWithProviders(
      <TombolEksekusi signal={{ ...dasar, can_confirm: false, expires_at: berlaku }} />,
    )
    const tombol = screen.getByRole('button', { name: /Eksekusi/ }) as HTMLButtonElement
    expect(tombol.disabled).toBe(true)
  })

  it('menampilkan pilihan untuk sinyal peluang', () => {
    const berlaku = new Date(Date.now() + 5 * 60 * 1000).toISOString()
    renderWithProviders(<TombolEksekusi signal={{ ...dasar, expires_at: berlaku }} />)

    expect(screen.getByRole('button', { name: /Eksekusi/ })).toBeDefined()
    expect(screen.getByRole('button', { name: /Lewati/ })).toBeDefined()
  })
})
