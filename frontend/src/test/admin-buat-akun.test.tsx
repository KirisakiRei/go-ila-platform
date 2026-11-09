import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { AdminAuthProvider } from '@/features/admin/AdminAuthProvider'
import AdminRbacPage from '@/pages/admin/AdminRbacPage'
import { adminAuthApi, adminRoleApi, adminUserApi } from '@/lib/admin-api'

/**
 * Tab User pada halaman RBAC membuat akun pengelola.
 *
 * Akun pengelola sengaja tidak melewati alur pelanggan: cukup username, kata
 * sandi, dan role. Uji ini memastikan formulir mengirim ketiganya (tanpa
 * menunggu verifikasi email) dan menolak kata sandi yang terlalu pendek.
 */

const ROLE_ID = '11111111-1111-1111-1111-111111111111'


/** tungguOpsiRole memastikan daftar role sudah termuat sebelum dipilih. */
async function tungguOpsiRole() {
  await waitFor(
    () => {
      const opsi = Array.from(screen.getByLabelText('Role').querySelectorAll('option')).map(
        (o) => o.textContent,
      )
      expect(opsi.join('|')).toContain('Gudang')
    },
    { timeout: 5000 },
  )
}

function renderTabUser() {
  vi.spyOn(adminAuthApi, 'me').mockResolvedValue({
    user: { id: 'a1', name: 'Superadmin', email: 'sa@rekavia.com', role: 'admin' },
    role: { code: 'superadmin', name: 'Superadmin', is_superadmin: true },
    permissions: ['role.view', 'role.edit', 'role.create', 'user.view'],
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
      user_count: 0,
      created_at: '2026-11-09T00:00:00Z',
    },
  ])
  vi.spyOn(adminRoleApi, 'permissions').mockResolvedValue([])
  vi.spyOn(adminUserApi, 'list').mockResolvedValue({ data: [], meta: undefined })

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

describe('Buat akun admin dari tab User', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('mengirim username, kata sandi, dan role tanpa menunggu verifikasi email', async () => {
    const user = userEvent.setup()
    const create = vi.spyOn(adminRoleApi, 'createAdmin').mockResolvedValue({
      id: 'u9',
      name: 'budi.admin',
      email: 'budi.admin@admin.invalid',
      tier: 'free',
      status: 'active',
      role: 'admin',
      role_code: 'gudang',
      telegram_linked: false,
      account_count: 0,
      created_at: { utc: '', wib: '', ny: '' },
    })

    renderTabUser()

    await user.click(await screen.findByRole('button', { name: /Buat akun admin/ }))

    await user.type(screen.getByLabelText('Username'), 'Budi.Admin')
    await user.type(screen.getByLabelText('Kata sandi'), 'RahasiaPanjang1')
    await user.type(screen.getByLabelText('Ulangi kata sandi'), 'RahasiaPanjang1')

    await tungguOpsiRole()
    await user.selectOptions(screen.getByLabelText('Role'), ROLE_ID)

    await user.click(screen.getByRole('button', { name: 'Buat akun' }))

    // Tombol hanya aktif bila seluruh syarat terpenuhi; bila tetap mati, ada
    // isian yang belum sah dan pengiriman tidak boleh terjadi.

    await waitFor(() => expect(create).toHaveBeenCalledTimes(1))
    expect(create.mock.calls[0]?.[0]).toEqual({
      username: 'budi.admin',
      password: 'RahasiaPanjang1',
      role_id: ROLE_ID,
      name: undefined,
      email: undefined,
    })
  })

  it('menahan pengiriman saat kata sandi belum mencapai 10 karakter', async () => {
    const user = userEvent.setup()
    const create = vi.spyOn(adminRoleApi, 'createAdmin').mockResolvedValue({} as never)

    renderTabUser()

    await user.click(await screen.findByRole('button', { name: /Buat akun admin/ }))
    await user.type(screen.getByLabelText('Username'), 'budi.admin')
    await user.type(screen.getByLabelText('Kata sandi'), 'pendek')
    await user.type(screen.getByLabelText('Ulangi kata sandi'), 'pendek')
    await tungguOpsiRole()
    await user.selectOptions(screen.getByLabelText('Role'), ROLE_ID)

    const tombol = screen.getByRole('button', { name: 'Buat akun' })
    expect(tombol).toBeDisabled()
    await user.click(tombol)
    expect(create).not.toHaveBeenCalled()
  })

  it('menahan pengiriman saat konfirmasi kata sandi berbeda', async () => {
    const user = userEvent.setup()

    renderTabUser()

    await user.click(await screen.findByRole('button', { name: /Buat akun admin/ }))
    await user.type(screen.getByLabelText('Username'), 'budi.admin')
    await user.type(screen.getByLabelText('Kata sandi'), 'RahasiaPanjang1')
    await user.type(screen.getByLabelText('Ulangi kata sandi'), 'RahasiaPanjang2')
    await tungguOpsiRole()
    await user.selectOptions(screen.getByLabelText('Role'), ROLE_ID)

    expect(screen.getByRole('button', { name: 'Buat akun' })).toBeDisabled()
  })
})
