import { useCallback, useEffect } from 'react'
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

  // Callback dijaga stabil: identitas baru pada setiap render akan memicu
  // pemasangan ulang listener di hook idle.
  const handleTimeout = useCallback(() => {
    signOut()
    navigate('/admin/login', { replace: true })
  }, [signOut, navigate])

  const idle = useAdminIdleTimer({ enabled: true, onTimeout: handleTimeout })
  const { warningVisible, leaveNow } = idle

  // Sesi admin bisa habis lebih dulu di server; jangan biarkan modal
  // peringatan menutupi halaman yang sudah tidak sah.
  useEffect(() => {
    if (!warningVisible) return
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') leaveNow()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [warningVisible, leaveNow])

  if (!warningVisible) return null

  return (
    <IdleWarningDialog
      remainingSeconds={idle.remainingSeconds}
      onStay={idle.stay}
      onLeaveNow={idle.leaveNow}
    />
  )
}
