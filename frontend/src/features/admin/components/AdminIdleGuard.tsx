import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAdminAuth } from '@/features/admin/admin-auth-context'
import { IdleWarningDialog } from '@/features/auth/components/IdleWarningDialog'
import { useAdminIdleTimer } from '../hooks/use-admin-idle-timer'

/**
 * AdminIdleGuard menutup panel admin setelah pengguna diam terlalu lama.
 *
 * Batasnya lebih ketat daripada sesi pelanggan karena panel admin bisa
 * mengubah tier, memblokir akun, dan melihat data pelanggan (plans/23 §2).
 */
export function AdminIdleGuard() {
  const { signOut } = useAdminAuth()
  const navigate = useNavigate()

  const idle = useAdminIdleTimer({
    enabled: true,
    onTimeout: () => {
      signOut()
      navigate('/admin/login', { replace: true })
    },
  })

  // Sesi admin bisa habis lebih dulu di server; jangan biarkan modal
  // peringatan menutupi halaman yang sudah tidak sah.
  useEffect(() => {
    if (!idle.warningVisible) return
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') idle.leaveNow()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [idle])

  if (!idle.warningVisible) return null

  return (
    <IdleWarningDialog
      remainingSeconds={idle.remainingSeconds}
      onStay={idle.stay}
      onLeaveNow={idle.leaveNow}
    />
  )
}
