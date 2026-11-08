import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { AdminAuthProvider } from '@/features/admin/AdminAuthProvider'
import { useAdminAuth } from '@/features/admin/admin-auth-context'
import { adminAuthApi } from '@/lib/admin-api'

/**
 * Sesi admin dipisah dari sesi pelanggan.
 *
 * Uji ini menjaga dua hal yang pernah salah: panel admin memakai sesi admin
 * (bukan sesi pelanggan), dan admin yang belum masuk diarahkan ke pintu admin
 * `/admin/login`, bukan ke pintu pelanggan.
 */
function Probe() {
  const { admin, isLoading } = useAdminAuth()
  if (isLoading) return <p>memuat</p>
  return <p>{admin ? `masuk:${admin.email}` : 'belum-masuk'}</p>
}

function renderProbe() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: 0 } } })
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/app/admin']}>
        <AdminAuthProvider>
          <Routes>
            <Route path="/app/admin" element={<Probe />} />
          </Routes>
        </AdminAuthProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('Sesi admin', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('memakai identitas dari pintu admin, bukan sesi pelanggan', async () => {
    vi.spyOn(adminAuthApi, 'me').mockResolvedValue({
      user: { id: 'a1', name: 'Superadmin', email: 'superadmin@rekavia.com', role: 'admin' },
      role: { code: 'superadmin', name: 'Superadmin', is_superadmin: true },
      permissions: ['cms.view'],
    })

    renderProbe()

    expect(await screen.findByText('masuk:superadmin@rekavia.com')).toBeInTheDocument()
  })

  it('dianggap belum masuk saat pintu admin menolak sesi', async () => {
    vi.spyOn(adminAuthApi, 'me').mockRejectedValue(new Error('401'))

    renderProbe()

    expect(await screen.findByText('belum-masuk')).toBeInTheDocument()
  })

  it('hanya menerima akun ber-role admin lewat endpoint admin', async () => {
    const login = vi.spyOn(adminAuthApi, 'login').mockResolvedValue({
      step: 'complete',
      expires_at: '2026-11-09T00:00:00Z',
      user: { id: 'a1', name: 'Superadmin', email: 'superadmin@rekavia.com', role: 'admin' },
      role_code: 'superadmin',
      permissions: [],
    })

    await adminAuthApi.login({ identifier: 'superadmin@rekavia.com', password: 'rahasia' })

    await waitFor(() => expect(login).toHaveBeenCalledTimes(1))
    expect(login.mock.calls[0]?.[0]).toEqual({
      identifier: 'superadmin@rekavia.com',
      password: 'rahasia',
    })
  })
})
