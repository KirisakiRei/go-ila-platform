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
  it('menampilkan formulir masuk', () => {
    renderWithProviders(<LoginPage />)

    expect(screen.getByRole('heading', { name: 'Masuk' })).toBeInTheDocument()
    expect(screen.getByLabelText('Email')).toBeInTheDocument()
    expect(screen.getByLabelText('Password')).toBeInTheDocument()
  })
})

describe('RegisterForm', () => {
  it('menampilkan pesan saat password terlalu pendek', async () => {
    const user = userEvent.setup()
    renderWithProviders(<RegisterForm />)

    await user.type(screen.getByLabelText('Nama'), 'Budi')
    await user.type(screen.getByLabelText('Email'), 'budi@example.com')
    await user.type(screen.getByLabelText('Password'), 'pendek')
    await user.click(screen.getByRole('button', { name: 'Daftar' }))

    expect(await screen.findByText(/minimal 10/i)).toBeInTheDocument()
  })
})
