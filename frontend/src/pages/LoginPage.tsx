import { Link } from 'react-router-dom'
import { AuthLayout } from '@/features/auth/components/AuthLayout'
import { LoginForm } from '@/features/auth/components/LoginForm'

export default function LoginPage() {
  return (
    <AuthLayout
      footer={
        <>
          Belum punya akun?{' '}
          <Link to="/register" className="font-medium text-foreground underline underline-offset-4">
            Daftar
          </Link>
        </>
      }
    >
      <LoginForm />
    </AuthLayout>
  )
}
