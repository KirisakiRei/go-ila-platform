import { afterEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import LaporkanBugPage from '@/pages/LaporkanBugPage'
import { bugApi, BUG_MAX_LAMPIRAN_BYTE } from '@/lib/bug-api'

function renderHalaman() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: 0 } } })
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={['/laporkan-bug']}>
        <LaporkanBugPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/** Zona tarik-lepas adalah elemen pertama yang berperan sebagai tombol. */
async function zona() {
  return screen.findByRole('button', { name: /tempel berkas/i })
}

function berkas(nama: string, tipe: string, ukuran = 8) {
  const f = new File(['x'], nama, { type: tipe })
  Object.defineProperty(f, 'size', { value: ukuran })
  return f
}

describe('Lampiran bug: pilih, tarik-lepas, dan tempel', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('menerima gambar yang ditarik lalu dilepas', async () => {
    vi.spyOn(bugApi, 'categories').mockResolvedValue(['tampilan'])
    renderHalaman()

    const drop = await zona()
    const gambar = berkas('layar.png', 'image/png', 2048)

    fireEvent.dragOver(drop, { dataTransfer: { files: [gambar], dropEffect: '' } })
    fireEvent.drop(drop, { dataTransfer: { files: [gambar] } })

    expect(await screen.findByText('layar.png')).toBeInTheDocument()
    expect(screen.getByText('2 KB')).toBeInTheDocument()
  })

  it('menolak berkas melebihi batas ukuran saat dilepas', async () => {
    vi.spyOn(bugApi, 'categories').mockResolvedValue(['tampilan'])
    renderHalaman()

    const drop = await zona()
    const besar = berkas('besar.png', 'image/png', BUG_MAX_LAMPIRAN_BYTE + 1)
    fireEvent.drop(drop, { dataTransfer: { files: [besar] } })

    expect(await screen.findByText(/Ukuran lampiran maksimal 1 MB/i)).toBeInTheDocument()
    expect(screen.queryByText('besar.png')).not.toBeInTheDocument()
  })

  it('menolak tipe berkas yang tidak didukung', async () => {
    vi.spyOn(bugApi, 'categories').mockResolvedValue(['tampilan'])
    renderHalaman()

    const drop = await zona()
    fireEvent.drop(drop, { dataTransfer: { files: [berkas('catatan.txt', 'text/plain')] } })

    expect(await screen.findByText(/harus berupa gambar/i)).toBeInTheDocument()
  })

  it('tempelan dari papan klip langsung menjadi lampiran', async () => {
    vi.spyOn(bugApi, 'categories').mockResolvedValue(['tampilan'])
    renderHalaman()

    const form = await screen.findByRole('button', { name: /kirim laporan/i }).then((b) => {
      const el = b.closest('form')
      if (!el) throw new Error('form tidak ditemukan')
      return el
    })

    const tempel = berkas('image.png', 'image/png', 1024)
    const event = new Event('paste', { bubbles: true, cancelable: true })
    Object.defineProperty(event, 'clipboardData', {
      value: { files: [tempel], items: [] },
    })
    act(() => {
      form.dispatchEvent(event)
    })

    expect(await screen.findByText('image.png')).toBeInTheDocument()
    // Berkas ditempel, jadi perilaku bawaan (menempelkan berkas ke halaman) dicegah.
    expect(event.defaultPrevented).toBe(true)
  })

  it('tempel teks biasa tidak dicegat', async () => {
    vi.spyOn(bugApi, 'categories').mockResolvedValue(['tampilan'])
    renderHalaman()

    const judul = await screen.findByLabelText(/Judul/i)
    const event = new Event('paste', { bubbles: true, cancelable: true })
    Object.defineProperty(event, 'clipboardData', {
      value: { files: [], items: [] },
    })
    act(() => {
      judul.dispatchEvent(event)
    })

    // Jika dicegat, pengguna tidak akan bisa menempelkan teks ke kolom mana pun.
    expect(event.defaultPrevented).toBe(false)
  })

  it('lampiran bisa dihapus kembali sebelum dikirim', async () => {
    vi.spyOn(bugApi, 'categories').mockResolvedValue(['tampilan'])
    renderHalaman()

    const drop = await zona()
    fireEvent.drop(drop, { dataTransfer: { files: [berkas('layar.png', 'image/png')] } })
    expect(await screen.findByText('layar.png')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: /hapus lampiran layar.png/i }))
    expect(screen.queryByText('layar.png')).not.toBeInTheDocument()
  })
})
