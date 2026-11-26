import { afterEach, describe, expect, it, vi } from 'vitest'
import type { ReactNode } from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AuthProvider } from '@/features/auth/AuthProvider'
import { authApi } from '@/lib/auth-api'
import LoginPage from '@/pages/LoginPage'
import { RegisterForm } from '@/features/auth/components/RegisterForm'

/** Provider minimal: router dan query client yang dibutuhkan form. */
function renderWithProviders(ui: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: 0 }, queries: { retry: 0 } },
  })

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <AuthProvider>{ui}</AuthProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('LoginPage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('menampilkan formulir masuk sesuai tata letak baru tanpa tombol Telegram', async () => {
    const user = userEvent.setup()
    renderWithProviders(<LoginPage />)

    expect(screen.getByRole('heading', { name: 'Masuk ke akun' })).toBeInTheDocument()
    expect(screen.getByLabelText('Username atau Email')).toBeInTheDocument()
    const passwordInput = screen.getByLabelText('Kata Sandi')
    expect(passwordInput).toHaveAttribute('type', 'password')
    const rememberBox = screen.getByRole('checkbox', { name: /Ingat perangkat ini/i })
    expect(rememberBox).toBeChecked()
    expect(screen.getByRole('button', { name: /Masuk ke Dashboard/i })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Telegram/i })).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Tampilkan kata sandi' }))
    expect(passwordInput).toHaveAttribute('type', 'text')
    await user.click(screen.getByRole('button', { name: 'Sembunyikan kata sandi' }))
    expect(passwordInput).toHaveAttribute('type', 'password')
  })

  it('mengirim remember_me sesuai centang Ingat perangkat ini', async () => {
    const user = userEvent.setup()
    const loginSpy = vi.spyOn(authApi, 'login').mockResolvedValue({
      step: 'complete',
      expires_at: '2026-12-27T00:00:00Z',
      user: {
        id: 'u1',
        name: 'Ryan',
        username: 'ryan',
        email: 'ryan@example.com',
        phone: '',
        role: 'user',
        tier: 'plus',
        idle_warning: false,
        email_verified_at: null,
        created_at: '2026-11-01T00:00:00Z',
        updated_at: '2026-11-01T00:00:00Z',
      },
    })

    renderWithProviders(<LoginPage />)
    await user.type(screen.getByLabelText('Username atau Email'), 'ryan@example.com')
    await user.type(screen.getByLabelText('Kata Sandi'), 'password-aman-123')
    await user.click(screen.getByRole('checkbox', { name: /Ingat perangkat ini/i }))
    await user.click(screen.getByRole('button', { name: /Masuk ke Dashboard/i }))

    await waitFor(() => {
      expect(loginSpy).toHaveBeenCalledWith({
        identifier: 'ryan@example.com',
        password: 'password-aman-123',
        remember_me: false,
      })
    })
  })
})

describe('RegisterForm', () => {
  // Pendaftaran dibagi dua langkah (Akun -> Keamanan).
  it('menampilkan pesan saat password terlalu pendek', async () => {
    const user = userEvent.setup()
    renderWithProviders(<RegisterForm />)

    await user.type(screen.getByLabelText('Nama lengkap'), 'Budi Santoso')
    await user.type(screen.getByLabelText('Username'), 'budi.santoso')
    await user.type(screen.getByLabelText('Email'), 'budi@example.com')
    await user.click(screen.getByRole('button', { name: 'Lanjut' }))

    await user.type(await screen.findByLabelText('Password'), 'pendek')
    await user.click(screen.getByRole('button', { name: 'Buat akun' }))

    expect(await screen.findByText(/minimal 10/i)).toBeInTheDocument()
  })

  it('menahan langkah pertama bila username tidak sah', async () => {
    const user = userEvent.setup()
    renderWithProviders(<RegisterForm />)

    await user.type(screen.getByLabelText('Nama lengkap'), 'Budi Santoso')
    await user.type(screen.getByLabelText('Username'), 'budi santoso')
    await user.type(screen.getByLabelText('Email'), 'budi@example.com')
    await user.click(screen.getByRole('button', { name: 'Lanjut' }))

    expect(await screen.findByText(/Hanya huruf, angka/i)).toBeInTheDocument()
    expect(screen.queryByLabelText('Password')).not.toBeInTheDocument()
  })
})
