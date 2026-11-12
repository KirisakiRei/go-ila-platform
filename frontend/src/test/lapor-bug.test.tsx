import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import LaporkanBugPage from '@/pages/LaporkanBugPage'
import RiwayatLaporanBugPage from '@/pages/RiwayatLaporanBugPage'
import { bugApi, BUG_MAX_LAMPIRAN_BYTE } from '@/lib/bug-api'

function renderHalaman(ui: React.ReactNode, path = '/laporkan-bug') {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: 0 } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          {/* Urutan penting: rute yang lebih spesifik didahulukan. */}
          <Route path="/laporkan-bug" element={ui} />
          <Route path="/laporkan-bug/riwayat" element={null} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('Lapor bug', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('menolak lampiran melebihi 1 MB sebelum dikirim', async () => {
    vi.spyOn(bugApi, 'categories').mockResolvedValue(['tampilan', 'sinyal'])
    const kirim = vi.spyOn(bugApi, 'create')

    renderHalaman(<LaporkanBugPage />)

    const besar = new File(['x'], 'besar.png', { type: 'image/png' })
    Object.defineProperty(besar, 'size', { value: BUG_MAX_LAMPIRAN_BYTE + 1 })

    const input = await screen.findByLabelText(/Lampiran/i)
    fireEvent.change(input, { target: { files: [besar] } })

    // Pesan jelas, dan laporan tidak terkirim.
    expect(await screen.findByText(/Ukuran lampiran maksimal 1 MB/i)).toBeInTheDocument()
    expect(kirim).not.toHaveBeenCalled()
  })

  it('menampilkan galat bila kategori belum dipilih', async () => {
    vi.spyOn(bugApi, 'categories').mockResolvedValue(['tampilan'])
    const kirim = vi.spyOn(bugApi, 'create')

    renderHalaman(<LaporkanBugPage />)
    fireEvent.change(await screen.findByLabelText(/Judul/i), { target: { value: 'Tombol hilang' } })
    fireEvent.change(screen.getByLabelText(/Apa yang terjadi/i), {
      target: { value: 'Tombol simpan tidak muncul sama sekali di halaman ini.' },
    })
    fireEvent.click(screen.getByRole('button', { name: /kirim laporan/i }))

    expect(await screen.findByText(/Pilih kategori laporan lebih dulu/i)).toBeInTheDocument()
    expect(kirim).not.toHaveBeenCalled()
  })

  it('mengirim laporan lengkap dengan page_url terisi otomatis', async () => {
    vi.spyOn(bugApi, 'categories').mockResolvedValue(['tampilan'])
    const kirim = vi.spyOn(bugApi, 'create').mockResolvedValue({
      id: 'rep-1',
      category: 'tampilan',
      title: 'Tombol hilang',
      status: 'baru',
      admin_note: '',
      created_at: '2026-11-13T10:00:00Z',
      updated_at: '2026-11-13T10:00:00Z',
      body: 'Tombol simpan tidak muncul.',
      page_url: '',
      user_agent: '',
      user_id: 'u1',
      attachments: [],
    })

    renderHalaman(<LaporkanBugPage />)

    // Tunggu kategorinya benar-benar ter-render sebelum memilih.
    const pilih = await screen.findByLabelText(/Kategori/i)
    await waitFor(() => expect(pilih.querySelector('option[value="tampilan"]')).not.toBeNull())
    fireEvent.change(pilih, { target: { value: 'tampilan' } })
    fireEvent.change(screen.getByLabelText(/Judul/i), { target: { value: 'Tombol hilang' } })
    fireEvent.change(screen.getByLabelText(/Apa yang terjadi/i), {
      target: { value: 'Tombol simpan tidak muncul sama sekali di halaman ini.' },
    })
    fireEvent.click(screen.getByRole('button', { name: /kirim laporan/i }))

    await waitFor(() => expect(kirim).toHaveBeenCalled())
    const arg = kirim.mock.calls[0]?.[0]
    if (!arg) throw new Error('laporan tidak terkirim')
    expect(arg.category).toBe('tampilan')
    // page_url diisi otomatis supaya admin tahu halaman mana yang dimaksud.
    expect(arg.pageUrl).toContain('/laporkan-bug')
    expect(await screen.findByText(/Laporan terkirim/i)).toBeInTheDocument()
  })

  it('riwayat kosong memberi penjelasan, bukan layar kosong', async () => {
    vi.spyOn(bugApi, 'listMine').mockResolvedValue([])

    renderHalaman(<RiwayatLaporanBugPage />, '/laporkan-bug')

    expect(await screen.findByText(/Belum ada laporan/i)).toBeInTheDocument()
  })
})
