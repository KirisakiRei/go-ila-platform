import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router-dom'
import { Alert, Button, Card, CardContent, CardDescription, CardHeader, CardTitle, Input, Label } from '@/components/ui'
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
    onError: (err) => setError(err instanceof Error ? err.message : 'Tautan tidak sah atau sudah kedaluwarsa.'),
  })

  const mismatch = confirm.length > 0 && password !== confirm
  const canSubmit = token !== '' && password.length >= 8 && password === confirm

  return (
    <div className="mx-auto flex min-h-screen max-w-md flex-col justify-center p-6">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Atur ulang kata sandi</CardTitle>
          <CardDescription className="text-xs">
            Minimal 8 karakter. Setelah tersimpan, semua perangkat akan diminta masuk kembali.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {!token && <Alert variant="destructive">Tautan tidak lengkap. Minta tautan baru dari halaman lupa kata sandi.</Alert>}
          {error && <Alert variant="destructive">{error}</Alert>}

          {done ? (
            <>
              <Alert variant="success">Kata sandi diperbarui.</Alert>
              <Button className="w-full" asChild={false} onClick={() => (window.location.href = '/login')}>
                Masuk dengan kata sandi baru
              </Button>
            </>
          ) : (
            <>
              <div>
                <Label htmlFor="password">Kata sandi baru</Label>
                <Input
                  id="password"
                  type="password"
                  autoComplete="new-password"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  className="mt-1"
                />
              </div>
              <div>
                <Label htmlFor="confirm">Ulangi kata sandi</Label>
                <Input
                  id="confirm"
                  type="password"
                  autoComplete="new-password"
                  value={confirm}
                  onChange={(event) => setConfirm(event.target.value)}
                  className="mt-1"
                />
                {mismatch && <p className="mt-1 text-[11px] text-muted-foreground">Kata sandi belum sama.</p>}
              </div>
              <Button className="w-full" disabled={!canSubmit || submit.isPending} onClick={() => submit.mutate()}>
                Simpan kata sandi
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
