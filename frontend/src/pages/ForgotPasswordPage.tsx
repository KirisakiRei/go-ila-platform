import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { Alert, Button, Card, CardContent, CardDescription, CardHeader, CardTitle, Input, Label } from '@/components/ui'
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
    <div className="mx-auto flex min-h-screen max-w-md flex-col justify-center p-6">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Lupa kata sandi</CardTitle>
          <CardDescription className="text-xs">
            Masukkan alamat email akun Anda. Tautan berlaku satu jam dan hanya bisa dipakai sekali.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {sent ? (
            <Alert variant="success">
              Bila alamat itu terdaftar, tautan reset sudah dikirim. Periksa kotak masuk dan folder spam.
            </Alert>
          ) : (
            <>
              <div>
                <Label htmlFor="email">Email</Label>
                <Input
                  id="email"
                  type="email"
                  autoComplete="email"
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                  className="mt-1"
                />
              </div>
              <Button className="w-full" disabled={!email || submit.isPending} onClick={() => submit.mutate()}>
                Kirim tautan reset
              </Button>
            </>
          )}
          <p className="text-center text-[12px] text-muted-foreground">
            <Link to="/login" className="underline">
              Kembali ke halaman masuk
            </Link>
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
