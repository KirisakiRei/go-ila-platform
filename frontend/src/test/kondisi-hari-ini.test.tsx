import { describe, expect, it, vi, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { KondisiHariIni } from '@/pages/components/KondisiHariIni'
import * as queries from '@/lib/queries'
import { publicContentApi } from '@/lib/system-api'
import type { Position, SystemStatus } from '@/types/api'
import type { PublicStatus } from '@/types/system'

function status(over: Partial<SystemStatus> = {}): SystemStatus {
  return {
    market: { state: 'UNKNOWN', label: 'Tidak diketahui', updated_at: '2026-11-13T09:00:00Z' },
    engines: [
      {
        strategy: 'ila-2.0',
        connected: true,
        engine_version: '2.0.0',
        last_seen_at: '2026-11-13T09:00:00Z',
      },
    ],
    brokers: { total: 1, connected: 1, any_error: false },
    daily: {
      day: '2026-11-13',
      win_count: 1,
      loss_count: 0,
      total_trades: 0,
      max_trades: 2,
      open_count: 0,
      locked: false,
      lock_reason: '',
    },
    server_time: '2026-11-13T09:00:00Z',
    daily_timezone: 'Asia/Jakarta',
    ...over,
  }
}

const publik: PublicStatus = {
  mode: 'open',
  message: '',
  server_time: '2026-11-13T09:00:00Z',
  time_pair: { wib: '16:00 · 13 Nov', ny: '04:00 · 13 Nov', utc: '' },
  market: {
    open: true,
    label: 'Pasar buka',
    next_change: '2026-11-14T05:00:00+07:00',
    next_label: 'Pasar tutup',
    next_change_pair: { wib: '14 Nov 05:00', ny: '13 Nov 17:00', utc: '' },
  },
  invite_only: false,
}

function posisi(over: Partial<Position> = {}): Position {
  return {
    id: 'p1',
    symbol: 'XAUUSD',
    direction: 'BUY',
    volume: 0.66,
    entry_price: 4000,
    exit_price: 4010,
    stop_loss: 3990,
    take_profit: 4020,
    profit_loss: 207.9,
    outcome_r: 1.92,
    status: 'closed',
    closed_at: '2026-11-13T03:00:00Z',
    opened_at: '2026-11-13T01:00:00Z',
    ...over,
  } as Position
}

function tampilkan(statusData: SystemStatus, posisiData: Position[] = []) {
  vi.spyOn(queries, 'useSystemStatus').mockReturnValue({
    data: statusData,
    isLoading: false,
    isError: false,
    error: null,
  } as unknown as ReturnType<typeof queries.useSystemStatus>)
  vi.spyOn(queries, 'usePositions').mockReturnValue({
    data: posisiData,
  } as unknown as ReturnType<typeof queries.usePositions>)
  vi.spyOn(publicContentApi, 'status').mockResolvedValue(publik)

  const qc = new QueryClient({ defaultOptions: { queries: { retry: 0 } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter>
        <KondisiHariIni />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('KondisiHariIni', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('menampilkan Pasar Buka, bukan "Tidak diketahui", saat engine belum tahu', async () => {
    tampilkan(status())
    // Label muncul pada baris status; teks boleh berulang di tempat lain.
    expect((await screen.findAllByText('Pasar Buka')).length).toBeGreaterThan(0)
    expect(screen.queryByText(/belum diketahui/i)).not.toBeInTheDocument()
  })

  it('menghitung P/L hanya dari posisi yang ditutup hari ini', async () => {
    tampilkan(status(), [
      posisi({ id: 'hari-ini', profit_loss: 207.9, closed_at: '2026-11-13T03:00:00Z' }),
      // Kemarin: tidak boleh ikut dihitung.
      posisi({ id: 'kemarin', profit_loss: -50, closed_at: '2026-11-12T03:00:00Z' }),
    ])
    // Format id-ID memakai koma desimal.
    expect(await screen.findByText(/207,90/)).toBeInTheDocument()
  })

  it('menampilkan tanda hubung saat belum ada posisi ditutup hari ini', async () => {
    tampilkan(status(), [posisi({ profit_loss: 999, closed_at: '2026-11-01T03:00:00Z' })])
    expect(await screen.findByText('P/L Hari Ini')).toBeInTheDocument()
    // 999 dari tanggal lain tidak boleh muncul sebagai P/L hari ini.
    expect(screen.queryByText(/999/)).not.toBeInTheDocument()
  })

  it('menjelaskan alasan saat batas harian tercapai', async () => {
    tampilkan(
      status({
        daily: {
          day: '2026-11-13',
          win_count: 1,
          loss_count: 0,
          total_trades: 1,
          max_trades: 1,
          open_count: 0,
          locked: true,
          lock_reason: 'Target harian tercapai (1 win).',
        },
      }),
    )
    expect(await screen.findByText(/Target harian tercapai/)).toBeInTheDocument()
  })

  it('menautkan ke halaman akun yang benar', async () => {
    tampilkan(status())
    const tautan = await screen.findByRole('link', { name: /akun/i })
    // /akun-broker tidak ada di router; tautan mati akan membuka 404.
    expect(tautan).toHaveAttribute('href', '/akun')
  })

  it('memakai limit posisi yang sama dengan daftar posisi terkini', async () => {
    // Kunci cache menyertakan limit. Bila berbeda, dashboard mengirim dua
    // permintaan /positions setiap 30 detik untuk data yang sama.
    const posisiSpy = vi.spyOn(queries, 'usePositions')
    tampilkan(status())
    expect(posisiSpy).toHaveBeenCalledWith(100)
  })
})
