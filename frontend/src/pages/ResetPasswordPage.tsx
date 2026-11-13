import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router-dom'
import { Alert, Button, Input, Label } from '@/components/ui'
import { AuthLayout } from '@/features/auth/components/AuthLayout'
import { authApi } from '@/lib/auth-api'

/**
 * ResetPasswordPage menukar token menjadi kata sandi baru.
 *
 * Sesudah berhasil, seluruh perangkat keluar otomatis: backend menaikkan
 * `session_version` sehingga token lama tidak berlaku (plans/17 §3.3).
 */
export default function ResetPasswordPage() {
  const [params] = useSearchParams()
  const token = params.get('token') ?? ''
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [done, setDone] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const submit = useMutation({
    mutationFn: () => authApi.resetPassword(token, password),
    onSuccess: () => setDone(true),
    onError: (err) =>
      setError(err instanceof Error ? err.message : 'Tautan tidak sah atau sudah kedaluwarsa.'),
  })

  const mismatch = confirm.length > 0 && password !== confirm
  const canSubmit = token !== '' && password.length >= 8 && password === confirm

  return (
    <AuthLayout
      footer={
        <>
          Sudah ingat kata sandi?{' '}
          <Link to="/login" className="font-medium text-foreground underline underline-offset-4">
            Masuk
          </Link>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <header className="flex flex-col gap-1">
          <h1 className="text-xl font-semibold tracking-tight">Atur ulang kata sandi</h1>
          <p className="text-[13px] text-muted-foreground">
            Minimal 8 karakter. Setelah tersimpan, semua perangkat akan diminta masuk kembali.
          </p>
        </header>

        {!token && (
          <Alert variant="destructive">
            Tautan tidak lengkap. Minta tautan baru dari halaman lupa kata sandi.
          </Alert>
        )}
        {error && <Alert variant="destructive">{error}</Alert>}

        {done ? (
          <div className="flex flex-col gap-4">
            <Alert variant="success">Kata sandi diperbarui.</Alert>
            <Button asChild className="w-full">
              <Link to="/login">Masuk dengan kata sandi baru</Link>
            </Button>
          </div>
        ) : (
          <form
            className="flex flex-col gap-4"
            onSubmit={(event) => {
              event.preventDefault()
              if (canSubmit && !submit.isPending) submit.mutate()
            }}
          >
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="password">Kata sandi baru</Label>
              <Input
                id="password"
                type="password"
                autoComplete="new-password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="confirm">Ulangi kata sandi</Label>
              <Input
                id="confirm"
                type="password"
                autoComplete="new-password"
                value={confirm}
                onChange={(event) => setConfirm(event.target.value)}
              />
              {mismatch && (
                <p className="text-micro text-muted-foreground">Kata sandi belum sama.</p>
              )}
            </div>
            <Button type="submit" className="w-full" disabled={!canSubmit || submit.isPending}>
              {submit.isPending ? 'Menyimpan…' : 'Simpan kata sandi'}
            </Button>
          </form>
        )}
      </div>
    </AuthLayout>
  )
}
