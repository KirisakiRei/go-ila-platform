import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import LandingPage from '@/pages/public/LandingPage'
import { PublicLayout } from '@/pages/public/PublicLayout'
import { publicPlanApi } from '@/lib/admin-api'
import { publicContentApi } from '@/lib/system-api'
import type { PublicPlan } from '@/types/admin'
import type { PublicStatus } from '@/types/system'

function makePublicStatus(mode: string, message = ''): PublicStatus {
  return {
    mode,
    message,
    server_time: '2026-11-08T12:00:00Z',
    time_pair: { wib: '19:00 WIB', ny: '07:00 NY', utc: '12:00 UTC' },
    market: {
      open: false,
      label: 'Tutup',
      next_change: '2026-11-08T22:00:00Z',
      next_label: 'Buka',
      next_change_pair: { wib: '05:00 WIB', ny: '17:00 NY', utc: '22:00 UTC' },
    },
  }
}

function renderPublicLanding() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: 0 } },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/']}>
        <Routes>
          <Route element={<PublicLayout />}>
            <Route index element={<LandingPage />} />
          </Route>
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

const SAMPLE_PLANS: PublicPlan[] = [
  {
    code: 'free',
    name: 'Free',
    description: 'Simulasi dasar',
    price_idr: 0,
    interval: 'month',
    features: {
      account_limit: '1',
      auto_execute: 'false',
      max_symbols: '1',
      signal_delay_minutes: '15',
      strategies: 'ila-2.0',
    },
    ila3_coming_soon: false,
    active: true,
  },
  {
    code: 'plus',
    name: 'Plus',
    description: 'Otomatis penuh',
    price_idr: 299000,
    interval: 'month',
    features: {
      account_limit: '2',
      auto_execute: 'true',
      max_symbols: '8',
      reason_ai: 'true',
      signal_delay_minutes: '0',
      strategies: 'ila-2.0',
    },
    ila3_coming_soon: true,
    active: true,
  },
]

describe('LandingPage & PublicLayout', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('menampilkan satu h1, disclaimer risiko, paket DB, dan slug legal baku', async () => {
    vi.spyOn(publicContentApi, 'status').mockResolvedValue(makePublicStatus('normal'))
    vi.spyOn(publicPlanApi, 'list').mockResolvedValue(SAMPLE_PLANS)

    const { container } = renderPublicLanding()

    const h1List = container.querySelectorAll('h1')
    expect(h1List).toHaveLength(1)
    expect(h1List[0]).toHaveTextContent(/Eksekusi Trading/i)

    expect(
      screen.getByText(/Perdagangan mata uang dan derivatif memiliki risiko kerugian tinggi/i),
    ).toBeInTheDocument()
    expect(screen.queryByText(/profit pasti/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/win rate/i)).not.toBeInTheDocument()

    // Paket dimuat dari API, harga diformat rupiah, dan benefit tidak mentah.
    expect(await screen.findByText('Plus')).toBeInTheDocument()
    expect(screen.getByText(/Rp\s?299\.000/)).toBeInTheDocument()
    expect(screen.getByText(/Eksekusi otomatis & konfirmasi manual/i)).toBeInTheDocument()
    expect(screen.getByText(/Sinyal real-time tanpa jeda/i)).toBeInTheDocument()
    expect(screen.queryByText(/auto_execute/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/signal_delay_minutes/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/AES-256/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/mTLS/i)).not.toBeInTheDocument()

    // Footer memakai slug legal resmi backend.
    const hrefs = Array.from(container.querySelectorAll('a')).map((a) => a.getAttribute('href'))
    expect(hrefs).toContain('/legal/tos')
    expect(hrefs).toContain('/legal/privacy')
    expect(hrefs).toContain('/legal/risk')
    expect(hrefs).toContain('/legal/refund')
    expect(hrefs).toContain('/legal/fair_use')
    expect(hrefs).toContain('/app/register')
    expect(hrefs).toContain('/app/login')

    // Header pada posisi atas landing page menyatu transparan tanpa border bawah,
    // lalu berubah menjadi latar putih saat di-scroll ke bagian terang.
    const header = container.querySelector('header')
    expect(header?.className).toContain('border-transparent')
    expect(header?.className).toContain('bg-transparent')

    Object.defineProperty(window, 'scrollY', { value: 640, configurable: true })
    fireEvent.scroll(window)
    expect(header?.className).toContain('bg-white/95')
    Object.defineProperty(window, 'scrollY', { value: 0, configurable: true })
  })

  it('menampilkan banner pemeliharaan saat mode sistem bukan normal', async () => {
    vi.spyOn(publicContentApi, 'status').mockResolvedValue(
      makePublicStatus(
        'maintenance',
        'Sedang pemeliharaan terjadwal; entry baru ditahan sementara.',
      ),
    )
    vi.spyOn(publicPlanApi, 'list').mockResolvedValue([])

    renderPublicLanding()

    expect(
      await screen.findByText(/Sedang pemeliharaan terjadwal; entry baru ditahan sementara\./i),
    ).toBeInTheDocument()
  })
})
