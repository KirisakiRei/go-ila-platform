import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { Alert, Button, Input, Label } from '@/components/ui'
import { AuthLayout } from '@/features/auth/components/AuthLayout'
import { authApi } from '@/lib/auth-api'

/**
 * ForgotPasswordPage meminta tautan reset.
 *
 * Pesan sukses selalu sama, termasuk untuk alamat yang tidak terdaftar:
 * membedakan keduanya akan membocorkan daftar pelanggan (plans/17 §3.2).
 */
export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [sent, setSent] = useState(false)

  const submit = useMutation({
    mutationFn: () => authApi.forgotPassword(email),
    onSuccess: () => setSent(true),
    onError: () => setSent(true),
  })

  return (
    <AuthLayout
      footer={
        <>
          Ingat kata sandi?{' '}
          <Link to="/login" className="font-medium text-foreground underline underline-offset-4">
            Masuk
          </Link>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <header className="flex flex-col gap-1">
          <h1 className="text-xl font-semibold tracking-tight">Lupa kata sandi</h1>
          <p className="text-[13px] text-muted-foreground">
            Masukkan alamat email akun Anda. Tautan berlaku satu jam dan hanya bisa dipakai sekali.
          </p>
        </header>

        {sent ? (
          <div className="flex flex-col gap-4">
            <Alert variant="success">
              Bila alamat itu terdaftar, tautan reset sudah dikirim. Periksa kotak masuk dan folder spam.
            </Alert>
            <Button asChild variant="outline" className="w-full">
              <Link to="/login">Kembali ke halaman masuk</Link>
            </Button>
          </div>
        ) : (
          <form
            className="flex flex-col gap-4"
            onSubmit={(event) => {
              event.preventDefault()
              if (email && !submit.isPending) submit.mutate()
            }}
          >
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                autoComplete="email"
                placeholder="nama@email.com"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
              />
            </div>
            <Button type="submit" className="w-full" disabled={!email || submit.isPending}>
              {submit.isPending ? 'Mengirim tautan…' : 'Kirim tautan reset'}
            </Button>
          </form>
        )}
      </div>
    </AuthLayout>
  )
}
