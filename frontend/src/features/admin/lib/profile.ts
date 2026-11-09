import type { AdminProfile } from '@/lib/admin-api'
import type { AdminUser } from '../admin-auth-context'

/**
 * mapProfile menyusun identitas admin dari balasan server.
 *
 * Ditaruh terpisah dari provider agar pengujian bisa memakai pemetaan yang
 * sama: permission dan status superadmin wajib ikut, karena panel memakainya
 * untuk menyembunyikan menu — sebelumnya keduanya dibuang diam-diam.
 */
export function mapProfile(profile: AdminProfile): AdminUser {
  return {
    id: profile.user.id,
    name: profile.user.name,
    email: profile.user.email,
    role: profile.role.code || profile.user.role,
    roleName: profile.role.name,
    isSuperadmin: profile.role.is_superadmin,
    permissions: profile.permissions ?? [],
  }
}
