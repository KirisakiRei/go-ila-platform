import { describe, expect, it } from 'vitest'
import type { ReactNode } from 'react'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AuthProvider } from '@/features/auth/AuthProvider'
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
  it('menampilkan formulir masuk sesuai tata letak baru tanpa tombol Telegram', async () => {
    const user = userEvent.setup()
    renderWithProviders(<LoginPage />)

    expect(screen.getByRole('heading', { name: 'Masuk ke akun' })).toBeInTheDocument()
    expect(screen.getByLabelText('Username atau Email')).toBeInTheDocument()
    const passwordInput = screen.getByLabelText('Kata Sandi')
    expect(passwordInput).toHaveAttribute('type', 'password')
    expect(screen.getByRole('button', { name: /Masuk ke Dashboard/i })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Telegram/i })).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Tampilkan kata sandi' }))
    expect(passwordInput).toHaveAttribute('type', 'text')
    await user.click(screen.getByRole('button', { name: 'Sembunyikan kata sandi' }))
    expect(passwordInput).toHaveAttribute('type', 'password')
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
