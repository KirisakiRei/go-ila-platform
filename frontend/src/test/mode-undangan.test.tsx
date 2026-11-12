import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import RegisterPage from '@/pages/RegisterPage'
import { MobileBottomNav } from '@/components/layout/MobileBottomNav'
import { publicContentApi } from '@/lib/system-api'
import type { PublicStatus } from '@/types/system'

function statusUndangan(inviteOnly: boolean): PublicStatus {
  return {
    mode: 'normal',
    message: '',
    server_time: '2026-11-08T12:00:00Z',
    time_pair: { wib: '19:00 WIB', ny: '07:00 NY', utc: '12:00 UTC' },
    market: {
      open: true,
      label: 'Buka',
      next_change: '2026-11-08T22:00:00Z',
      next_label: 'Tutup',
      next_change_pair: { wib: '05:00 WIB', ny: '17:00 NY', utc: '22:00 UTC' },
    },
    invite_only: inviteOnly,
  }
}

function bungkus(ui: React.ReactNode) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: 0 } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/daftar']}>
        <Routes>
          <Route path="/daftar" element={ui} />
          <Route path="/dashboard" element={<p>halaman dashboard</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('Mode undangan di UI', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('halaman daftar menampilkan ajakan undangan, bukan formulir', async () => {
    vi.spyOn(publicContentApi, 'status').mockResolvedValue(statusUndangan(true))

    bungkus(<RegisterPage />)

    expect(await screen.findByText(/Akses lewat undangan/i)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Masuk ke akun/i })).toHaveAttribute('href', '/login')
    // Formulir pendaftaran tidak boleh ada sama sekali.
    expect(screen.queryByLabelText('Nama lengkap')).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: 'Daftar' })).not.toBeInTheDocument()
  })

  it('halaman daftar kembali menampilkan formulir saat pendaftaran dibuka', async () => {
    vi.spyOn(publicContentApi, 'status').mockResolvedValue(statusUndangan(false))

    bungkus(<RegisterPage />)

    // Formulir pendaftaran bertahap: judul "Daftar" dan langkah pertama tampil.
    expect(await screen.findByRole('heading', { name: 'Daftar' })).toBeInTheDocument()
    expect(screen.getByLabelText('Nama lengkap')).toBeInTheDocument()
    expect(screen.queryByText(/Akses lewat undangan/i)).not.toBeInTheDocument()
  })

  it('navigasi bawah menyembunyikan tab Billing saat invite only', () => {
    bungkus(<MobileBottomNav inviteOnly />)
    expect(screen.queryByRole('link', { name: /billing/i })).not.toBeInTheDocument()
  })

  it('navigasi bawah tetap menampilkan tab Billing saat pendaftaran dibuka', () => {
    bungkus(<MobileBottomNav inviteOnly={false} />)
    expect(screen.getByRole('link', { name: /billing/i })).toBeInTheDocument()
  })

  it('status yang gagal dibaca diperlakukan sebagai tertutup', async () => {
    // Halaman daftar tidak boleh menawarkan formulir bila status tidak diketahui:
    // backend juga menutup pintu itu saat flag gagal dibaca.
    vi.spyOn(publicContentApi, 'status').mockRejectedValue(new Error('jaringan mati'))

    bungkus(<RegisterPage />)

    expect(await screen.findByText(/Akses lewat undangan/i)).toBeInTheDocument()
  })
})
