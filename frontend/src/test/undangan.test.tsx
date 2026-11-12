import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import RegisterPage from '@/pages/RegisterPage'
import { publicContentApi } from '@/lib/system-api'
import { inviteApi } from '@/lib/invite-api'
import { authApi } from '@/lib/auth-api'
import type { PublicStatus } from '@/types/system'

function statusInviteOnly(inviteOnly: boolean): PublicStatus {
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

function renderDaftar(path: string) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: 0 } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/daftar" element={<RegisterPage />} />
          <Route path="/login" element={<p>halaman masuk</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('Pendaftaran lewat tautan undangan', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('tautan sah menampilkan formulir daftar, bukan pesan tertutup', async () => {
    vi.spyOn(publicContentApi, 'status').mockResolvedValue(statusInviteOnly(true))
    vi.spyOn(inviteApi, 'periksa').mockResolvedValue({
      valid: true,
      tier: 'plus',
      expires_at: '2026-11-20T12:00:00Z',
      sisa_kuota: 3,
      email_terkunci: false,
    })

    renderDaftar('/daftar?invite=ABC123')

    // Formulir muncul: judul "Daftar" dan isian nama tersedia.
    expect(await screen.findByRole('heading', { name: 'Daftar' })).toBeInTheDocument()
    expect(screen.getByLabelText('Nama lengkap')).toBeInTheDocument()
    expect(screen.queryByText(/Akses lewat undangan/i)).not.toBeInTheDocument()
  })

  it('tautan tidak sah menampilkan penjelasan, bukan formulir', async () => {
    vi.spyOn(publicContentApi, 'status').mockResolvedValue(statusInviteOnly(true))
    vi.spyOn(inviteApi, 'periksa').mockRejectedValue(new Error('403'))

    renderDaftar('/daftar?invite=KADALUARSA')

    expect(await screen.findByText(/Undangan tidak berlaku/i)).toBeInTheDocument()
    expect(screen.queryByLabelText('Nama lengkap')).not.toBeInTheDocument()
  })

  it('tanpa tautan saat invite only: hanya ajakan masuk', async () => {
    vi.spyOn(publicContentApi, 'status').mockResolvedValue(statusInviteOnly(true))
    const periksa = vi.spyOn(inviteApi, 'periksa')

    renderDaftar('/daftar')

    expect(await screen.findByText(/Akses lewat undangan/i)).toBeInTheDocument()
    // Tanpa kode, pemeriksaan undangan tidak boleh dipanggil sama sekali.
    expect(periksa).not.toHaveBeenCalled()
  })

  it('kode undangan ikut terkirim saat mendaftar', async () => {
    vi.spyOn(publicContentApi, 'status').mockResolvedValue(statusInviteOnly(true))
    vi.spyOn(inviteApi, 'periksa').mockResolvedValue({
      valid: true,
      tier: 'plus',
      expires_at: '2026-11-20T12:00:00Z',
      sisa_kuota: 1,
      email_terkunci: false,
    })

    // Payload yang benar-benar dikirim ke backend ditangkap di sini.
    const terkirim: { current?: Record<string, unknown> } = {}
    vi.spyOn(authApi, 'register').mockImplementation((payload) => {
      terkirim.current = payload as unknown as Record<string, unknown>
      return Promise.resolve({ step: 'authenticated', user: { id: 'u1' } } as never)
    })

    renderDaftar('/daftar?invite=KODE-UJI-123')

    fireEvent.change(await screen.findByLabelText('Nama lengkap'), {
      target: { value: 'Budi Santoso' },
    })
    fireEvent.change(screen.getByLabelText('Username'), { target: { value: 'budisantoso' } })
    fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'budi@contoh.id' } })
    fireEvent.click(screen.getByRole('button', { name: /lanjut/i }))

    fireEvent.change(await screen.findByLabelText('Password'), {
      target: { value: 'rahasia-panjang-123' },
    })
    fireEvent.click(screen.getByRole('checkbox'))
    fireEvent.click(screen.getByRole('button', { name: /buat akun/i }))

    await waitFor(() => expect(terkirim.current).toBeDefined())
    // Tanpa invite_code, backend menolak karena platform sedang invite only.
    expect(terkirim.current?.invite_code).toBe('KODE-UJI-123')
  })

  it('tanpa tautan undangan, invite_code tidak dikirim', async () => {
    vi.spyOn(publicContentApi, 'status').mockResolvedValue(statusInviteOnly(false))

    const terkirim: { current?: Record<string, unknown> } = {}
    vi.spyOn(authApi, 'register').mockImplementation((payload) => {
      terkirim.current = payload as unknown as Record<string, unknown>
      return Promise.resolve({ step: 'authenticated', user: { id: 'u1' } } as never)
    })

    renderDaftar('/daftar')

    fireEvent.change(await screen.findByLabelText('Nama lengkap'), {
      target: { value: 'Budi Santoso' },
    })
    fireEvent.change(screen.getByLabelText('Username'), { target: { value: 'budisantoso' } })
    fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'budi@contoh.id' } })
    fireEvent.click(screen.getByRole('button', { name: /lanjut/i }))

    fireEvent.change(await screen.findByLabelText('Password'), {
      target: { value: 'rahasia-panjang-123' },
    })
    fireEvent.click(screen.getByRole('checkbox'))
    fireEvent.click(screen.getByRole('button', { name: /buat akun/i }))

    await waitFor(() => expect(terkirim.current).toBeDefined())
    expect(terkirim.current?.invite_code).toBeUndefined()
  })
})
