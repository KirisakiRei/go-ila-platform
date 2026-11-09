import { describe, expect, it } from 'vitest'
import { hasAnyPermission, hasPermission, permissionSet } from '@/features/admin/lib/permissions'
import { mapProfile } from '@/features/admin/lib/profile'

/**
 * Pemeriksaan permission panel admin.
 *
 * Uji ini menjaga dua hal yang pernah salah: provider membuang daftar
 * permission dari server, dan panel tidak pernah memeriksanya sama sekali.
 * Aturan penting: daftar kosong berarti TIDAK berwenang.
 */
describe('hasPermission', () => {
  it('menolak saat daftar kosong (deny by default)', () => {
    expect(hasPermission([], 'user.view')).toBe(false)
    expect(hasPermission(null, 'user.view')).toBe(false)
    expect(hasPermission(undefined, 'user.view')).toBe(false)
  })

  it('hanya mengizinkan permission yang benar-benar dipegang', () => {
    const owned = ['user.view', 'signal.view']
    expect(hasPermission(owned, 'user.view')).toBe(true)
    expect(hasPermission(owned, 'user.delete')).toBe(false)
  })

  it('membiarkan halaman tanpa permission tetap terbuka', () => {
    expect(hasPermission([], undefined)).toBe(true)
    expect(hasPermission(null, '')).toBe(true)
  })
})

describe('hasAnyPermission', () => {
  it('cukup salah satu dimiliki', () => {
    expect(hasAnyPermission(['audit.view'], ['role.view', 'audit.view'])).toBe(true)
    expect(hasAnyPermission(['cms.view'], ['role.view', 'audit.view'])).toBe(false)
  })

  it('daftar yang diminta kosong berarti tidak menyaring', () => {
    expect(hasAnyPermission([], [])).toBe(true)
  })

  it('daftar milik kosong tetap menolak', () => {
    expect(hasAnyPermission([], ['role.view'])).toBe(false)
  })
})

describe('permissionSet', () => {
  it('membuat himpunan dari daftar, termasuk saat kosong', () => {
    expect(permissionSet(['a', 'b']).has('b')).toBe(true)
    expect(permissionSet(undefined).size).toBe(0)
  })
})

describe('mapProfile', () => {
  it('mempertahankan permission dan status superadmin dari server', () => {
    const admin = mapProfile({
      user: { id: 'u1', name: 'Ryan', email: 'ryan@rekavia.com', role: 'admin' },
      role: { code: 'superadmin', name: 'Superadmin', is_superadmin: true },
      permissions: ['role.view', 'role.edit'],
    })

    expect(admin.role).toBe('superadmin')
    expect(admin.roleName).toBe('Superadmin')
    expect(admin.isSuperadmin).toBe(true)
    expect(admin.permissions).toEqual(['role.view', 'role.edit'])
  })

  it('memakai role akun bila role RBAC kosong dan tidak mengarang permission', () => {
    const admin = mapProfile({
      user: { id: 'u2', name: 'Budi', email: 'budi@rekavia.com', role: 'admin' },
      role: { code: '', name: '', is_superadmin: false },
      permissions: [],
    })

    expect(admin.role).toBe('admin')
    expect(admin.permissions).toEqual([])
  })
})
