import { Link } from 'react-router-dom'
import { AuthLayout } from '@/features/auth/components/AuthLayout'
import { RegisterForm } from '@/features/auth/components/RegisterForm'

export default function RegisterPage() {
  return (
    <AuthLayout
      footer={
        <>
          Sudah punya akun?{' '}
          <Link to="/login" className="font-medium text-foreground underline underline-offset-4">
            Masuk
          </Link>
        </>
      }
    >
      <RegisterForm />
    </AuthLayout>
  )
}
