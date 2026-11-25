import { Link } from 'react-router-dom'
import { AuthLayout } from '@/features/auth/components/AuthLayout'
import { LoginForm } from '@/features/auth/components/LoginForm'
import { useInviteOnly } from '@/lib/use-invite-only'

export default function LoginPage() {
  const inviteOnly = useInviteOnly()

  return (
    <AuthLayout
      footer={
        inviteOnly ? (
          <>
            Belum punya akses undangan?{' '}
            <a
              href="/#ajukan-akses"
              className="font-semibold text-foreground underline underline-offset-4 transition-opacity hover:opacity-80"
            >
              Ajukan Akses
            </a>
          </>
        ) : (
          <>
            Belum punya akun?{' '}
            <Link
              to="/register"
              className="font-semibold text-foreground underline underline-offset-4 transition-opacity hover:opacity-80"
            >
              Daftar
            </Link>
          </>
        )
      }
    >
      <LoginForm />
    </AuthLayout>
  )
}
