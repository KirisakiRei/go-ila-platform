import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'

vi.mock('@/features/auth/hooks/use-auth', () => ({
  useSignOut: () => vi.fn(),
  useAuthContext: () => ({
    profile: { name: 'Uji', email: 'uji@contoh.id', tier: 'basic' },
    isVerified: true,
  }),
}))

import { SettingsMobileView } from '@/pages/components/SettingsMobileView'

// Regresi: seluruh baris menu pernah menuju /pengaturan/keamanan, sehingga
// menekan "Profil & Akun" atau kartu profil mendarat di halaman yang salah.
describe('Menu pengaturan mobile', () => {
  it('mengarahkan tiap bagian ke halamannya sendiri', () => {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: 0 } } })
    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter>
          <SettingsMobileView />
        </MemoryRouter>
      </QueryClientProvider>,
    )

    expect(screen.getByText('Profil & Akun').closest('a')).toHaveAttribute(
      'href',
      '/pengaturan/profil',
    )
    expect(screen.getByText('Kata Sandi & 2FA').closest('a')).toHaveAttribute(
      'href',
      '/pengaturan/keamanan',
    )
    expect(screen.getByText('Perangkat & Sesi').closest('a')).toHaveAttribute(
      'href',
      '/pengaturan/perangkat',
    )
  })
})
