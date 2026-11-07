import { Link } from 'react-router-dom'
import { Button } from '@/components/ui'

/**
 * Aksi ringkas di kanan judul halaman Ringkasan.
 * Tujuannya mengarahkan ke pekerjaan yang paling sering dibutuhkan.
 */
export function PageHeaderActions() {
  return (
    <Button asChild variant="outline" size="sm">
      <Link to="/akun">Kelola akun broker</Link>
    </Button>
  )
}
