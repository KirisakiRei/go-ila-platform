import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { AppShell } from '@/components/layout/AppShell'
import { AdminShell } from '@/features/admin/components/AdminShell'
import { AuthProvider } from '@/features/auth/AuthProvider'
import { AdminAuthProvider } from '@/features/admin/AdminAuthProvider'
import { authApi } from '@/lib/auth-api'
import { adminAuthApi } from '@/lib/admin-api'
import type { User } from '@/types/api'

const mockUser: User = {
  id: 'u1',
  name: 'John Doe',
  username: 'johndoe',
  phone: '+628123456789',
  email: 'john@example.com',
  role: 'user',
  tier: 'basic',
  idle_warning: false,
  email_verified_at: '2026-01-01T00:00:00Z',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}

function renderCustomerShell(user: User = mockUser) {
  vi.spyOn(authApi, 'me').mockResolvedValue(user)
  vi.spyOn(authApi, 'csrf').mockResolvedValue({ csrf_token: 'test' })

  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: 0 } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/dashboard']}>
        <AuthProvider>
          <AppShell user={user}>
            <div data-testid="dashboard-content">Isi Dashboard</div>
          </AppShell>
        </AuthProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function renderAdminShell() {
  vi.spyOn(adminAuthApi, 'me').mockResolvedValue({
    user: { id: 'a1', name: 'Admin', email: 'admin@rekavia.com', role: 'admin' },
    role: { code: 'superadmin', name: 'Superadmin', is_superadmin: true },
    permissions: ['*'],
  })

  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: 0 } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/admin']}>
        <AdminAuthProvider>
          <Routes>
            <Route
              path="/admin"
              element={
                <AdminShell>
                  <div data-testid="admin-content">Isi Admin</div>
                </AdminShell>
              }
            />
          </Routes>
        </AdminAuthProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('Sidebar Fixed & Collapse/Expand (User & Admin)', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  afterEach(() => {
    vi.restoreAllMocks()
    localStorage.clear()
  })

  it('AppShell: merender sidebar desktop fixed dengan tombol collapse & expand', async () => {
    renderCustomerShell()

    // Cek bahwa sidebar desktop ada dan memiliki class fixed
    const aside = screen.getByRole('complementary')
    expect(aside).toHaveClass('lg:fixed')
    expect(aside).toHaveClass('lg:inset-y-0')
    expect(aside).toHaveClass('lg:left-0')
    expect(aside).toHaveClass('lg:w-64')

    // Teks menu terlihat saat expanded
    expect(screen.getByText('Signals & Radar')).toBeInTheDocument()

    // Cari tombol ciutkan (PanelLeftClose)
    const collapseBtn = screen.getByRole('button', { name: /ciutkan bilah navigasi/i })
    expect(collapseBtn).toBeInTheDocument()

    // Klik tombol collapse
    fireEvent.click(collapseBtn)

    // Sidebar mengecil ke w-[68px]
    expect(aside).toHaveClass('lg:w-[68px]')
    expect(localStorage.getItem('ila_user_sidebar_collapsed')).toBe('true')

    // Cari tombol perluas (PanelLeftOpen)
    const expandBtn = screen.getByRole('button', { name: /perluas bilah navigasi/i })
    expect(expandBtn).toBeInTheDocument()

    // Klik expand kembali
    fireEvent.click(expandBtn)
    expect(aside).toHaveClass('lg:w-64')
    expect(localStorage.getItem('ila_user_sidebar_collapsed')).toBe('false')
  })

  it('AppShell: memuat status collapsed dari localStorage saat inisialisasi', () => {
    localStorage.setItem('ila_user_sidebar_collapsed', 'true')
    renderCustomerShell()

    const aside = screen.getByRole('complementary')
    expect(aside).toHaveClass('lg:w-[68px]')
    expect(screen.getByRole('button', { name: /perluas bilah navigasi/i })).toBeInTheDocument()
  })

  it('AdminShell: merender sidebar desktop fixed dan mendukung toggle collapse/expand', async () => {
    renderAdminShell()

    const aside = screen.getByRole('complementary')
    expect(aside).toHaveClass('lg:fixed')
    expect(aside).toHaveClass('lg:inset-y-0')
    expect(aside).toHaveClass('lg:left-0')

    // Tombol collapse ada saat expanded
    const collapseBtn = screen.getByRole('button', { name: /ciutkan bilah navigasi/i })
    expect(collapseBtn).toBeInTheDocument()

    // Klik collapse
    fireEvent.click(collapseBtn)
    expect(aside).toHaveClass('lg:w-[68px]')
    expect(localStorage.getItem('ila_admin_sidebar_collapsed')).toBe('true')

    // Tombol expand ada saat collapsed
    const expandBtn = screen.getByRole('button', { name: /perluas bilah navigasi/i })
    expect(expandBtn).toBeInTheDocument()

    // Klik expand kembali
    fireEvent.click(expandBtn)
    expect(aside).toHaveClass('lg:w-64')
    expect(localStorage.getItem('ila_admin_sidebar_collapsed')).toBe('false')
  })
})
