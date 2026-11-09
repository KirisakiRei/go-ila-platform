import { afterEach, describe, expect, it, vi } from 'vitest'
import type { ReactNode } from 'react'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { AdminAuthProvider } from '@/features/admin/AdminAuthProvider'
import { AdminShell } from '@/features/admin/components/AdminShell'
import { RequireAdminPermission } from '@/features/admin/components/RequireAdminPermission'
import { adminAuthApi } from '@/lib/admin-api'

/**
 * Panel admin menyembunyikan menu dan halaman yang tidak boleh dibuka.
 *
 * Ini hanya lapisan tampilan; server tetap menolak dengan 403. Yang diuji di
 * sini adalah admin tidak melihat menu yang pasti ditolak, kelompok menu kosong
 * ikut hilang, dan halaman tanpa izin menampilkan penjelasan — bukan pengalihan
 * diam-diam yang membuat menu seolah hilang.
 */
function renderSeperti(permissions: string[], ui: ReactNode, path = '/admin') {
  vi.spyOn(adminAuthApi, 'me').mockResolvedValue({
    user: { id: 'a1', name: 'Admin', email: 'admin@rekavia.com', role: 'admin' },
    role: { code: 'support', name: 'Support', is_superadmin: false },
    permissions,
  })

  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: 0 } } })

  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[path]}>
        <AdminAuthProvider>
          <Routes>
            <Route path={path} element={<AdminShell>{ui}</AdminShell>} />
          </Routes>
        </AdminAuthProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('Pembatas tampilan admin', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('hanya menampilkan menu yang diizinkan dan membuang kelompok kosong', async () => {
    renderSeperti(['user.view'], <p>isi</p>)

    expect(await screen.findByRole('link', { name: /Pengguna/ })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /Role & Permission/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /Flag Sistem/ })).not.toBeInTheDocument()
    // Ringkasan tidak butuh permission dan selalu tampil.
    expect(screen.getByRole('link', { name: /Ringkasan/ })).toBeInTheDocument()
    // Kelompok "Kepatuhan & Laporan" seluruhnya tersaring.
    expect(screen.queryByText('Kepatuhan & Laporan')).not.toBeInTheDocument()
  })

  it('menampilkan menu RBAC saat role.view dipegang', async () => {
    renderSeperti(['role.view'], <p>isi</p>)

    expect(await screen.findByRole('link', { name: /Role & Permission/ })).toBeInTheDocument()
  })

  it('menampilkan penjelasan saat halaman tidak berhak dibuka', async () => {
    renderSeperti(
      ['user.view'],
      <RequireAdminPermission permission="role.view">
        <p>isi rahasia</p>
      </RequireAdminPermission>,
    )

    expect(await screen.findByText('Anda tidak berhak membuka halaman ini')).toBeInTheDocument()
    expect(screen.queryByText('isi rahasia')).not.toBeInTheDocument()
  })

  it('menampilkan isi halaman saat permission dipegang', async () => {
    renderSeperti(
      ['role.view'],
      <RequireAdminPermission permission="role.view">
        <p>isi rahasia</p>
      </RequireAdminPermission>,
    )

    expect(await screen.findByText('isi rahasia')).toBeInTheDocument()
  })
})
