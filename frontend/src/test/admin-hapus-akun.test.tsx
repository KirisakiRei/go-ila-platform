import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { AdminAuthProvider } from '@/features/admin/AdminAuthProvider'
import AdminRbacPage from '@/pages/admin/AdminRbacPage'
import { adminAuthApi, adminRoleApi, adminUserApi } from '@/lib/admin-api'
import type { AdminUserListItem } from '@/types/admin'

/**
 * Penghapusan akun pengelola dari tab User.
 *
 * Yang dijaga:
 *  - Akun superadmin utama (is_protected) tidak ditawari tombol hapus sama
 *    sekali, bukan sekadar digagalkan server setelah diklik.
 *  - Penghapusan hanya mengirim setelah alasan minimal 10 karakter terisi dan
 *    konfirmasi mengetik username akun.
 */

const ROLE_ID = '11111111-1111-1111-1111-111111111111'

function akun(overrides: Partial<AdminUserListItem> = {}): AdminUserListItem {
  return {
    id: 'u9',
    name: 'Budi Admin',
    email: 'budi.admin@admin.invalid',
    tier: 'free',
    status: 'active',
    role: 'admin',
    role_code: 'gudang',
    telegram_linked: false,
    account_count: 0,
    created_at: { utc: '', wib: '', ny: '' },
    ...overrides,
  }
}

function renderTabUser(rows: AdminUserListItem[], permissions: string[]) {
  vi.spyOn(adminAuthApi, 'me').mockResolvedValue({
    user: { id: 'a1', name: 'Superadmin', email: 'sa@rekavia.com', role: 'admin' },
    role: { code: 'superadmin', name: 'Superadmin', is_superadmin: true },
    permissions,
  })

  vi.spyOn(adminRoleApi, 'list').mockResolvedValue([
    {
      id: ROLE_ID,
      code: 'gudang',
      name: 'Gudang',
      description: '',
      is_system: false,
      editable: true,
      permissions: ['user.view'],
      user_count: rows.length,
      created_at: '2026-11-09T00:00:00Z',
    },
  ])
  vi.spyOn(adminRoleApi, 'permissions').mockResolvedValue([])
  vi.spyOn(adminUserApi, 'list').mockResolvedValue({ data: rows, meta: undefined })

  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: 0 } } })

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/admin/rbac?tab=user']}>
        <AdminAuthProvider>
          <AdminRbacPage />
        </AdminAuthProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('Hapus akun admin dari tab User', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('mengirim alasan setelah konfirmasi username diketik', async () => {
    const user = userEvent.setup()
    const hapus = vi.spyOn(adminRoleApi, 'deleteAdmin').mockResolvedValue(undefined)

    renderTabUser([akun()], ['role.view', 'role.edit', 'role.delete'])

    await user.click(await screen.findByRole('button', { name: 'Hapus' }))

    const tombol = screen.getByRole('button', { name: 'Hapus akun' })
    expect(tombol).toBeDisabled()

    await user.type(screen.getByLabelText('Alasan penghapusan'), 'Akun tidak lagi bertugas')
    await user.type(
      screen.getByLabelText(/Ketik/),
      'budi.admin',
    )

    await waitFor(() => expect(tombol).toBeEnabled())
    await user.click(tombol)

    await waitFor(() => expect(hapus).toHaveBeenCalledTimes(1))
    expect(hapus.mock.calls[0]).toEqual(['u9', 'Akun tidak lagi bertugas'])
  })

  it('akun terlindungi tidak menampilkan tombol hapus', async () => {
    renderTabUser(
      [akun({ id: 'u-utama', email: 'utama@rekavia.com', is_protected: true })],
      ['role.view', 'role.edit', 'role.delete'],
    )

    expect(await screen.findByText('Terlindungi')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Hapus' })).not.toBeInTheDocument()
  })

  it('tanpa permission role.delete tidak ada aksi hapus', async () => {
    renderTabUser([akun()], ['role.view', 'role.edit'])

    await screen.findByText('Budi Admin')
    expect(screen.queryByRole('button', { name: 'Hapus' })).not.toBeInTheDocument()
  })
})
