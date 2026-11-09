import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { SignalOverviewGrid } from '@/pages/public/components/SignalOverviewGrid'
import { SignalOverviewCard } from '@/pages/public/components/SignalOverviewCards'
import { publicContentApi } from '@/lib/system-api'
import type { PublicSignalPreview } from '@/types/public-signal'

/**
 * Kartu ringkasan pantauan pasar di halaman depan.
 *
 * Yang dijaga:
 *  - Tanpa sinyal, kartu tetap tampil dan terbaca sebagai "sedang dipantau" —
 *    bukan galat dan bukan ruang kosong. Ini keadaan yang paling sering terjadi
 *    selama engine belum menerbitkan sinyal.
 *  - Tidak ada harga masuk, batas kerugian, maupun target di layar.
 *  - Nama instrumen yang dikenal pembaca (NAS100) tetap dipakai walau katalog
 *    menyimpan kunci kanonik yang berbeda.
 *  - Galat jaringan tidak menampilkan pesan teknis ke pengunjung.
 */

const ACUAN = new Date('2026-11-09T10:00:00Z')

function renderGrid() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: 0 } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <SignalOverviewGrid />
    </QueryClientProvider>,
  )
}

const previewDenganSinyal: PublicSignalPreview = {
  generated_at: '2026-11-09T09:58:00Z',
  market: { open: true, label: 'Pasar buka' },
  items: [
    {
      symbol: 'XAUUSD',
      display_name: 'Gold vs US Dollar',
      category: 'commodity',
      has_signal: true,
      signal: {
        direction: 'BULLISH',
        htf: 'H4',
        ltf: 'M15',
        status: 'ACTIVE',
        published_at: '2026-11-09T08:00:00Z',
      },
    },
    {
      symbol: 'EURUSD',
      display_name: 'Euro vs US Dollar',
      category: 'forex',
      has_signal: false,
    },
    {
      symbol: 'USTEC',
      display_name: 'US Tech 100',
      category: 'indices',
      has_signal: false,
    },
  ],
}

describe('Kartu ringkasan pantauan pasar', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('menampilkan tiga instrumen dengan arah pada kartu yang punya sinyal', async () => {
    vi.spyOn(publicContentApi, 'signalPreview').mockResolvedValue(previewDenganSinyal)
    renderGrid()

    expect(await screen.findByText('XAUUSD')).toBeInTheDocument()
    expect(screen.getByText('EURUSD')).toBeInTheDocument()
    expect(screen.getByText('USTEC')).toBeInTheDocument()

    // Arah ditulis dalam bahasa pembaca, bukan kode BULLISH.
    expect(screen.getByText('Kecenderungan naik')).toBeInTheDocument()
    expect(screen.getByText('Timeframe H4 / M15')).toBeInTheDocument()
    expect(screen.getByText('Sinyal Aktif')).toBeInTheDocument()
  })

  it('tidak menampilkan harga masuk, batas kerugian, atau target', async () => {
    vi.spyOn(publicContentApi, 'signalPreview').mockResolvedValue(previewDenganSinyal)
    const { container } = renderGrid()

    await screen.findByText('XAUUSD')

    // Diperiksa dari teks yang terlihat pengunjung, bukan dari data mentah.
    const teks = (container.textContent ?? '').toLowerCase()
    for (const terlarang of ['entry', 'stop loss', 'stop_loss', 'take profit', 'target harga']) {
      expect(teks).not.toContain(terlarang)
    }
    // Angka harga tidak boleh muncul sama sekali.
    expect(teks).not.toMatch(/\b\d{4,}\.\d{2}\b/)
  })

  it('memakai nama instrumen yang dikenal pembaca untuk kunci kanonik', async () => {
    vi.spyOn(publicContentApi, 'signalPreview').mockResolvedValue(previewDenganSinyal)
    renderGrid()

    await screen.findByText('USTEC')
    // Katalog menyimpan USTEC, pembaca mengenalnya sebagai Indeks Teknologi AS.
    expect(screen.getByText(/Indeks Teknologi AS/)).toBeInTheDocument()
    expect(screen.getByText(/Logam & Komoditas/)).toBeInTheDocument()
  })

  it('menampilkan keadaan dipantau saat belum ada sinyal', async () => {
    vi.spyOn(publicContentApi, 'signalPreview').mockResolvedValue({
      generated_at: '2026-11-09T09:58:00Z',
      market: { open: true, label: 'Pasar buka' },
      items: [
        { symbol: 'XAUUSD', display_name: '', category: 'commodity', has_signal: false },
        { symbol: 'EURUSD', display_name: '', category: 'forex', has_signal: false },
        { symbol: 'NAS100', display_name: '', category: 'indices', has_signal: false },
      ],
    })
    renderGrid()

    await waitFor(() => {
      expect(screen.getAllByText('Sedang dipantau')).toHaveLength(3)
    })
    // Tiga kartu tetap tampil — halaman tidak boleh terlihat rusak.
    expect(screen.getByText('XAUUSD')).toBeInTheDocument()
    expect(screen.getByText('NAS100')).toBeInTheDocument()
    expect(screen.getAllByText('Dipantau')).toHaveLength(3)
  })

  it('tidak menampilkan pesan teknis saat permintaan gagal', async () => {
    vi.spyOn(publicContentApi, 'signalPreview').mockRejectedValue(
      new Error('Network Error 500 internal server'),
    )
    renderGrid()

    // Instrumen tetap tampil; tidak ada pesan galat mentah.
    expect(await screen.findByText('XAUUSD')).toBeInTheDocument()
    expect(screen.queryByText(/Network Error/)).not.toBeInTheDocument()
    expect(screen.queryByText(/500/)).not.toBeInTheDocument()
    expect(screen.getAllByText('Sedang dipantau')).toHaveLength(3)
  })

  it('menandai pasar tutup tanpa menyembunyikan kartu', () => {
    render(
      <SignalOverviewCard
        item={{ symbol: 'EURUSD', display_name: '', category: 'forex', has_signal: false }}
        marketOpen={false}
        marketLabel="Pasar tutup"
        now={ACUAN}
      />,
    )

    // Tertulis dua kali: sebagai penanda keadaan di kepala kartu dan sebagai
    // keterangan jadwal di kaki kartu. Keduanya memang disengaja.
    expect(screen.getAllByText('Pasar tutup').length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText('Sedang dipantau')).toBeInTheDocument()
  })

  it('menulis arah turun untuk sinyal BEARISH', () => {
    render(
      <SignalOverviewCard
        item={{
          symbol: 'EURUSD',
          display_name: '',
          category: 'forex',
          has_signal: true,
          signal: {
            direction: 'BEARISH',
            htf: 'H1',
            ltf: 'M5',
            status: 'ACTIVE',
            published_at: '2026-11-09T08:00:00Z',
          },
        }}
        marketOpen
        marketLabel="Pasar buka"
        now={ACUAN}
      />,
    )

    expect(screen.getByText('Kecenderungan turun')).toBeInTheDocument()
    expect(screen.getByText('Diperbarui')).toBeInTheDocument()
  })
})
