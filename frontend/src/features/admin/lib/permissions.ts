/**
 * Pemeriksaan permission panel admin.
 *
 * Server tetap penentu akhir: setiap route admin diperiksa ulang di backend dan
 * membalas 403 bila tidak berwenang. Fungsi di sini hanya mengatur tampilan agar
 * admin tidak melihat tombol yang pasti ditolak — bukan pengganti otorisasi.
 *
 * Aturan penting: daftar permission kosong berarti TIDAK berwenang (deny by
 * default). Ini mencegah panel terbuka lebar saat profil gagal memuat daftar
 * permission, misalnya karena respons server berubah bentuk.
 */

/** Daftar permission yang dipegang admin yang sedang masuk. */
export type PermissionList = readonly string[]

/**
 * hasPermission menandai apakah satu permission ada di daftar.
 *
 * Tanpa permission yang diminta (undefined) fungsi mengembalikan true:
 * halaman yang tidak dijaga permission tetap boleh dibuka.
 */
export function hasPermission(permissions: PermissionList | null | undefined, key?: string): boolean {
  if (!key) return true
  if (!permissions || permissions.length === 0) return false
  return permissions.includes(key)
}

/**
 * hasAnyPermission menandai apakah salah satu permission dimiliki.
 *
 * Dipakai halaman yang isinya bisa dibuka dengan beberapa kewenangan berbeda,
 * misalnya panel RBAC (role.view) atau tab detail pelanggan.
 */
export function hasAnyPermission(
  permissions: PermissionList | null | undefined,
  keys?: readonly string[],
): boolean {
  if (!keys || keys.length === 0) return true
  if (!permissions || permissions.length === 0) return false
  return keys.some((key) => permissions.includes(key))
}

/**
 * PermissionsInfo adalah ringkasan tetap dari daftar permission.
 *
 * Sengaja berupa fungsi murni (bukan hook) agar bisa dipakai di tempat yang
 * tidak boleh memakai hook, seperti filter menu di luar komponen kecil.
 */
export function permissionSet(permissions: PermissionList | null | undefined): Set<string> {
  return new Set(permissions ?? [])
}
