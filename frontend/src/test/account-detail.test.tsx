import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'

vi.mock('@/features/auth/hooks/use-auth', () => ({
  useAuthContext: () => ({
    user: { id: 'u1', name: 'Ryan', email: 'ryan@example.com', tier: 'plus', role: 'user' },
  }),
}))

import AccountDetailPage from '@/pages/AccountDetailPage'
import { accountApi, subscriptionApi } from '@/lib/api'

describe('AccountDetailPage — Simulasi Risiko modal & Filter Pair Watchlist', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  function renderDetail() {
    vi.spyOn(accountApi, 'list').mockResolvedValue([
      {
        id: 'acc-1',
        label: 'Akun Utama',
        broker_type: 'matchtrader',
        external_account_id: '12345',
        mode: 'auto',
        risk_mode: 'percent',
        risk_percent: 1,
        lot_size: 0.01,
        balance: 10000,
        equity: 10000,
        free_margin: 9500,
        is_dry_run: false,
        is_active: true,
        last_synced_at: '2026-11-21T00:00:00Z',
        last_error: null,
        created_at: '2026-11-01T00:00:00Z',
      } as never,
    ])
    vi.spyOn(accountApi, 'pairs').mockResolvedValue({
      account_id: 'acc-1',
      risk_percent: 1,
      selected: ['XAUUSD', 'GBPUSD'],
      catalog: {
        symbols: ['XAUUSD', 'EURUSD', 'GBPUSD'],
        categories: [
          {
            key: 'all',
            label: 'All',
            groups: [
              {
                key: 'fx',
                label: 'FX & Metal',
                pairs: [
                  { symbol: 'XAUUSD', name: 'Gold' },
                  { symbol: 'EURUSD', name: 'Euro / US Dollar' },
                  { symbol: 'GBPUSD', name: 'Pound / US Dollar' },
                ],
              },
            ],
          },
        ],
      },
    })
    vi.spyOn(subscriptionApi, 'list').mockResolvedValue([
      { id: 'sub-1', symbol: 'XAUUSD', broker_account_id: null, is_active: true },
      { id: 'sub-2', symbol: 'EURUSD', broker_account_id: null, is_active: true },
      { id: 'sub-3', symbol: 'GBPUSD', broker_account_id: null, is_active: false },
    ])

    const qc = new QueryClient({ defaultOptions: { queries: { retry: 0 } } })
    return render(
      <QueryClientProvider client={qc}>
        <MemoryRouter initialEntries={['/akun/acc-1']}>
          <Routes>
            <Route path="/akun/:id" element={<AccountDetailPage />} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    )
  }

  it('memindahkan Simulasi Risiko ke modal dan menyaring Filter Pair hanya dari Watchlist aktif', async () => {
    const updatePairsSpy = vi
      .spyOn(accountApi, 'updatePairs')
      .mockResolvedValue({ symbols: ['XAUUSD', 'EURUSD'] })

    renderDetail()

    // Simulator tidak tampil inline sebelum tombol diklik.
    const simButton = await screen.findByRole('button', { name: /Simulasi Risiko/i })
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()

    // Klik tombol Simulasi Risiko -> modal terbuka.
    fireEvent.click(simButton)
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByLabelText('Pair')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Tutup dialog' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()

    // Pindah ke tab Filter Pair Eksekusi: hanya pair aktif di watchlist (XAUUSD, EURUSD) yang tampil.
    fireEvent.click(screen.getByRole('button', { name: /2\. Filter Pair Eksekusi/i }))
    expect(await screen.findByText('XAUUSD')).toBeInTheDocument()
    expect(screen.getByText('EURUSD')).toBeInTheDocument()
    expect(screen.queryByText('GBPUSD')).not.toBeInTheDocument()

    // Centang EURUSD lalu simpan.
    fireEvent.click(screen.getByRole('button', { name: /EURUSD/i }))
    fireEvent.click(screen.getByRole('button', { name: /Simpan Pemilihan Pair/i }))
    await waitFor(() => {
      expect(updatePairsSpy).toHaveBeenCalledWith('acc-1', ['XAUUSD', 'EURUSD'])
    })
  })
})
