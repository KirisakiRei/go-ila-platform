import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import LandingPage from '@/pages/public/LandingPage'
import { PublicLayout } from '@/pages/public/PublicLayout'
import { inviteRequestApi } from '@/lib/invite-api'
import { publicContentApi } from '@/lib/system-api'
import type { PublicStatus } from '@/types/system'

function makePublicStatus(
  mode: string,
  message = '',
  overrides: Partial<PublicStatus> = {},
): PublicStatus {
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
    invite_only: true,
    social_links: {
      discord: 'https://discord.gg/imaginary',
      x: 'https://x.com/imaginaryliq',
      telegram: 'https://t.me/imaginaryliq',
    },
    ...overrides,
  }
}

const EMPTY_SIGNAL_PREVIEW = {
  generated_at: '2026-11-08T12:00:00Z',
  market: {
    open: false,
    label: 'Tutup',
    next_label: 'Buka',
    next_change_pair: { wib: '05:00 WIB', ny: '17:00 NY', utc: '22:00 UTC' },
  },
  items: [],
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

describe('LandingPage & PublicLayout', () => {
  afterEach(() => {
    window.localStorage.clear()
    vi.restoreAllMocks()
  })

  it('menampilkan susunan beranda baru, storyline ILA, form undangan, sosmed, dan menyembunyikan harga/trial', async () => {
    vi.spyOn(publicContentApi, 'status').mockResolvedValue(makePublicStatus('normal'))
    vi.spyOn(publicContentApi, 'signalPreview').mockResolvedValue(EMPTY_SIGNAL_PREVIEW)

    const { container } = renderPublicLanding()

    const h1List = container.querySelectorAll('h1')
    expect(h1List).toHaveLength(1)
    expect(h1List[0]).toHaveTextContent(/Imaginary Liquidity Algorithm/i)

    // Storyline ILA 1.0 -> 3.0 tampil di beranda.
    expect(screen.getAllByText('ILA 1.0').length).toBeGreaterThan(0)
    expect(screen.getAllByText('ILA 2.0').length).toBeGreaterThan(0)
    expect(screen.getAllByText('ILA 2.1').length).toBeGreaterThan(0)
    expect(screen.getAllByText('ILA 3.0').length).toBeGreaterThan(0)

    // Bagian lama yang diminta dihapus tidak muncul lagi.
    expect(screen.queryByText(/Pantau setup, eksekusi/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/Dua mesin analisis/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/Pilih paket sesuai/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/Cakupan pantauan dan/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/Mulai Trial 7 Hari/i)).not.toBeInTheDocument()

    // Disclaimer risiko tetap hadir dan tidak ada janji keuntungan.
    expect(
      screen.getByText(/Perdagangan mata uang dan derivatif memiliki risiko kerugian tinggi/i),
    ).toBeInTheDocument()
    expect(screen.queryByText(/profit pasti/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/win rate/i)).not.toBeInTheDocument()

    // Tautan sosmed dari status publik tampil.
    const discordLinks = await screen.findAllByRole('link', { name: 'Discord' })
    expect(discordLinks[0]).toHaveAttribute('href', 'https://discord.gg/imaginary')
    expect(screen.getAllByRole('link', { name: 'X' })[0]).toHaveAttribute(
      'href',
      'https://x.com/imaginaryliq',
    )
    expect(screen.getAllByRole('link', { name: 'Telegram' })[0]).toHaveAttribute(
      'href',
      'https://t.me/imaginaryliq',
    )

    // Saat invite_only aktif, menu Harga disembunyikan dari navigasi & footer.
    const hrefs = Array.from(container.querySelectorAll('a')).map((a) => a.getAttribute('href'))
    expect(hrefs).not.toContain('/harga')
    expect(hrefs).toContain('/legal/tos')
    expect(hrefs).toContain('/legal/privacy')
    expect(hrefs).toContain('/legal/risk')
    expect(hrefs).toContain('/legal/refund')
    expect(hrefs).toContain('/legal/fair_use')
    expect(hrefs).toContain('/app/login')

    // Urutan section utama: Storyline -> Pantauan Pasar -> Eksekusi cepat -> Ajukan Akses.
    const html = container.innerHTML
    const posStoryline = html.indexOf('id="evolusi-ila"')
    const posPantauan = html.indexOf('Pantauan Pasar')
    const posEksekusi = html.indexOf('Eksekusi cepat,')
    const posAjukan = html.indexOf('id="ajukan-akses"')
    expect(posStoryline).toBeGreaterThan(-1)
    expect(posPantauan).toBeGreaterThan(posStoryline)
    expect(posEksekusi).toBeGreaterThan(posPantauan)
    expect(posAjukan).toBeGreaterThan(posEksekusi)

    // Header pada posisi atas landing page menyatu transparan tanpa border bawah
    // dengan logo mode gelap tampil, lalu cross-fade ke logo terang saat di-scroll.
    const header = container.querySelector('header')
    expect(header?.className).toContain('border-transparent')
    expect(header?.className).toContain('bg-transparent')

    const headerLogos = Array.from(header?.querySelectorAll('img') ?? [])
    const lightLogo = headerLogos.find((img) => img.getAttribute('src')?.includes('logo-light'))
    const darkLogo = headerLogos.find((img) => img.getAttribute('src')?.includes('logo-dark'))
    expect(darkLogo?.className).toContain('opacity-100')
    expect(lightLogo?.className).toContain('opacity-0')

    Object.defineProperty(window, 'scrollY', { value: 640, configurable: true })
    fireEvent.scroll(window)
    expect(header?.className).toContain('bg-white/95')
    expect(lightLogo?.className).toContain('opacity-100')
    expect(darkLogo?.className).toContain('opacity-0')

    // Saat memasuki zona gelap Hero 5 & 6 (#dark-cta-band), navbar beralih gelap kembali.
    const darkBand = container.querySelector('#dark-cta-band')
    expect(darkBand).not.toBeNull()
    vi.spyOn(darkBand!, 'getBoundingClientRect').mockReturnValue({
      top: 40,
      bottom: 620,
      height: 580,
      left: 0,
      right: 1200,
      width: 1200,
      x: 0,
      y: 40,
      toJSON: () => ({}),
    })
    fireEvent.scroll(window)
    expect(header?.className).toContain('bg-[#050507]/80')
    expect(darkLogo?.className).toContain('opacity-100')
    expect(lightLogo?.className).toContain('opacity-0')

    // Setelah melewati zona gelap menuju footer terang, navbar kembali putih.
    vi.spyOn(darkBand!, 'getBoundingClientRect').mockReturnValue({
      top: -820,
      bottom: -240,
      height: 580,
      left: 0,
      right: 1200,
      width: 1200,
      x: 0,
      y: -820,
      toJSON: () => ({}),
    })
    fireEvent.scroll(window)
    expect(header?.className).toContain('bg-white/95')
    expect(lightLogo?.className).toContain('opacity-100')
    Object.defineProperty(window, 'scrollY', { value: 0, configurable: true })
  })

  it('mengirim form pengajuan akses tanpa kredensial dan menampilkan konfirmasi', async () => {
    vi.spyOn(publicContentApi, 'status').mockResolvedValue(makePublicStatus('normal'))
    vi.spyOn(publicContentApi, 'signalPreview').mockResolvedValue(EMPTY_SIGNAL_PREVIEW)
    const submitSpy = vi.spyOn(inviteRequestApi, 'submit').mockResolvedValue(undefined)

    renderPublicLanding()

    fireEvent.change(screen.getByLabelText(/Nama \/ Panggilan/i), {
      target: { value: 'Ryan Trader' },
    })
    fireEvent.change(screen.getByLabelText(/^Email/i), {
      target: { value: 'ryan@example.com' },
    })
    fireEvent.change(screen.getByLabelText(/Telegram \/ Discord/i), {
      target: { value: '@ryantrader' },
    })
    fireEvent.click(screen.getByRole('button', { name: /Ajukan Bergabung/i }))

    expect(await screen.findByText(/Permohonan akses diterima/i)).toBeInTheDocument()
    const lockedButton = screen.getByRole('button', { name: /Sudah Diajukan/i })
    expect(lockedButton).toBeDisabled()
    expect(submitSpy).toHaveBeenCalledWith({
      name: 'Ryan Trader',
      email: 'ryan@example.com',
      contact: '@ryantrader',
      note: undefined,
      website: '',
    })

    // Muat ulang komponen: status "Sudah Diajukan" tetap terkunci dari localStorage.
    const { unmount } = renderPublicLanding()
    expect(screen.getAllByRole('button', { name: /Sudah Diajukan/i })[0]).toBeDisabled()
    unmount()
  })

  it('menampilkan banner pemeliharaan saat mode sistem bukan normal', async () => {
    vi.spyOn(publicContentApi, 'status').mockResolvedValue(
      makePublicStatus(
        'maintenance',
        'Sedang pemeliharaan terjadwal; entry baru ditahan sementara.',
      ),
    )
    vi.spyOn(publicContentApi, 'signalPreview').mockResolvedValue(EMPTY_SIGNAL_PREVIEW)

    renderPublicLanding()

    expect(
      await screen.findByText(/Sedang pemeliharaan terjadwal; entry baru ditahan sementara\./i),
    ).toBeInTheDocument()
  })
})
