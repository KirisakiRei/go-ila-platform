import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { ShieldOff } from 'lucide-react'
import { Button, EmptyState, PageHeader } from '@/components/ui'
import { usePermissions } from '../hooks/use-permissions'

/** ForbiddenPage menjelaskan penolakan akses tanpa mengeluarkan admin dari sesi. */
export function AdminForbiddenPage({ permission }: { permission?: string }) {
  return (
    <>
      <PageHeader
        title="Akses dibatasi"
        description="Halaman ini di luar kewenangan role Anda. Hubungi pemilik workspace bila memang perlu."
      />
      <EmptyState
        icon={<ShieldOff className="h-8 w-8" aria-hidden="true" />}
        title="Anda tidak berhak membuka halaman ini"
        description={
          permission
            ? `Role Anda tidak memegang permission "${permission}".`
            : 'Role Anda tidak memegang permission yang dibutuhkan halaman ini.'
        }
        action={
          <Button asChild variant="outline" size="sm">
            <Link to="/admin">Kembali ke ringkasan</Link>
          </Button>
        }
      />
    </>
  )
}

type RequireAdminPermissionProps = {
  /** Satu permission yang dibutuhkan untuk membuka halaman. */
  permission?: string
  /** Beberapa permission; cukup salah satu dimiliki. */
  anyOf?: readonly string[]
  children: ReactNode
}

/**
 * RequireAdminPermission menjaga isi halaman admin.
 *
 * Bila admin tidak berwenang, yang tampil adalah penjelasan "tidak berhak" —
 * bukan pengalihan diam-diam. Pengalihan menyembunyikan sebab dan membuat
 * admin mengira menunya hilang.
 */
export function RequireAdminPermission({ permission, anyOf, children }: RequireAdminPermissionProps) {
  const { can, canAny } = usePermissions()

  const allowed = permission ? can(permission) : canAny(anyOf)
  if (!allowed) return <AdminForbiddenPage permission={permission} />

  return <>{children}</>
}
