import { Link } from 'react-router-dom'
import { AuthLayout } from '@/features/auth/components/AuthLayout'
import { RegisterForm } from '@/features/auth/components/RegisterForm'
import { useInviteOnly } from '@/lib/use-invite-only'

export default function RegisterPage() {
  const inviteOnly = useInviteOnly()

  if (inviteOnly) {
    // Pendaftaran mandiri ditutup di server; formulir yang tetap tampil hanya
    // akan berakhir dengan 403. Halaman menampilkan jalur masuk yang benar.
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
        <div className="flex flex-col gap-3">
          <h1 className="text-lg font-bold tracking-tight text-foreground">Akses lewat undangan</h1>
          <p className="text-sm leading-relaxed text-muted-foreground">
            Pendaftaran mandiri sedang ditutup. Akun dibuat oleh admin, lalu Anda menerima
            undangan untuk masuk. Sudah punya akun? Silakan masuk seperti biasa.
          </p>
          <Link
            to="/login"
            className="mt-1 inline-flex h-10 items-center justify-center rounded-lg bg-foreground px-4 text-sm font-semibold text-background"
          >
            Masuk ke akun
          </Link>
        </div>
      </AuthLayout>
    )
  }

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
