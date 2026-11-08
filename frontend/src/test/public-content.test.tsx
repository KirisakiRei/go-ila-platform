import { afterEach, describe, expect, it, vi } from 'vitest'
import type { ReactNode } from 'react'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, useNavigate } from 'react-router-dom'
import { ScrollToTop } from '@/app/ScrollToTop'
import FeaturesPage from '@/pages/public/FeaturesPage'
import { publicContentApi } from '@/lib/system-api'
import type { PageContent } from '@/types/system'

/** Halaman contoh: dua bagian berbeda jenis agar bentuk render ikut teruji. */
function makePage(): PageContent {
  return {
    slug: 'fitur',
    eyebrow: 'Fitur',
    title: 'Yang dikerjakan sistem, dan yang tidak.',
    lead: 'Penjelasan pembuka halaman fitur.',
    meta_title: 'Fitur',
    meta_description: 'Deskripsi',
    is_published: true,
    updated_at: '2026-11-08T00:00:00Z',
    sections: [
      {
        key: 'kemampuan',
        kind: 'features',
        eyebrow: '',
        title: 'Kemampuan utama',
        lead: 'Batasan ditulis apa adanya.',
        body: '',
        layout: '',
        sort_order: 1,
        is_published: true,
        items: [
          {
            label: 'Pemetaan likuiditas',
            value: 'Menandai area acuan rencana.',
            note: 'Data lampau tidak menjamin pembalikan harga.',
            icon: 'layers',
            href: '',
            sort_order: 1,
          },
        ],
      },
      {
        key: 'batasan',
        kind: 'callout',
        eyebrow: '',
        title: 'Yang tidak dilakukan sistem',
        lead: '',
        body: 'Tidak menjamin hasil dan tidak mengelola dana Anda.',
        layout: '',
        sort_order: 2,
        is_published: true,
        items: [],
      },
    ],
  }
}

function renderWithProviders(ui: ReactNode, initial = '/fitur') {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: 0 } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initial]}>{ui}</MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('Halaman publik berbasis konten server', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('merender judul dan bagian dari server, bukan dari teks tetap', async () => {
    const page = makePage()
    vi.spyOn(publicContentApi, 'page').mockResolvedValue({
      ...page,
      title: 'Judul dari panel admin',
    })

    const { container } = renderWithProviders(<FeaturesPage />)

    expect(await screen.findByRole('heading', { level: 1 })).toHaveTextContent(
      'Judul dari panel admin',
    )
    // Judul bagian tampil sebagai heading; daftar isi hero memuat teks yang sama.
    expect(screen.getAllByText('Kemampuan utama').length).toBeGreaterThan(0)
    // Batasan tiap fitur wajib ikut tampil (plans/10 §4).
    expect(screen.getByText('Data lampau tidak menjamin pembalikan harga.')).toBeInTheDocument()
    // Di dalam bagian callout, judul hanya tampil sekali (bukan diulang di kotaknya).
    const callout = container.querySelector('#batasan')
    expect(callout?.querySelectorAll('h2')).toHaveLength(1)
  })

  it('menampilkan keadaan kosong bila halaman belum disiapkan', async () => {
    vi.spyOn(publicContentApi, 'page').mockRejectedValue(new Error('404'))

    renderWithProviders(<FeaturesPage />)

    expect(await screen.findByText(/Isi halaman ini sedang disiapkan/i)).toBeInTheDocument()
  })

  it('tidak memuat kalimat jaminan hasil', async () => {
    vi.spyOn(publicContentApi, 'page').mockResolvedValue(makePage())

    renderWithProviders(<FeaturesPage />)
    await screen.findByRole('heading', { level: 1 })

    expect(screen.queryByText(/profit pasti/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/dijamin untung/i)).not.toBeInTheDocument()
  })
})

describe('ScrollToTop', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  function Harness() {
    const navigate = useNavigate()
    return (
      <>
        <ScrollToTop />
        <button type="button" onClick={() => navigate('/harga')}>
          ke harga
        </button>
      </>
    )
  }

  it('mengembalikan gulir ke puncak saat alamat berubah tanpa anchor', async () => {
    const scrollTo = vi.fn()
    Object.defineProperty(window, 'scrollTo', { value: scrollTo, configurable: true })

    renderWithProviders(<Harness />, '/fitur')

    // Panggilan pertama terjadi saat halaman dibuka.
    await waitFor(() => expect(scrollTo).toHaveBeenCalled())
    scrollTo.mockClear()

    fireEvent.click(screen.getByRole('button', { name: 'ke harga' }))

    await waitFor(() => expect(scrollTo).toHaveBeenCalledTimes(1))
    const arg = scrollTo.mock.calls[0]?.[0] as { top?: number; behavior?: string }
    expect(arg.top).toBe(0)
    expect(arg.behavior).toBe('instant')
  })

  it('tidak menggulir ke puncak bila tujuan berupa anchor', async () => {
    const scrollTo = vi.fn()
    Object.defineProperty(window, 'scrollTo', { value: scrollTo, configurable: true })

    renderWithProviders(<ScrollToTop />, '/legal/tos#pasal-1')

    await waitFor(() => expect(scrollTo).not.toHaveBeenCalled())
  })
})
