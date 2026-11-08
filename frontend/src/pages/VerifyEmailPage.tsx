import { Link, useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Alert, Button, Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui'
import { authApi } from '@/lib/auth-api'

/**
 * VerifyEmailPage memproses tautan verifikasi sekali pakai.
 *
 * Verifikasi dijalankan lewat query dengan `retry: false`: tautan yang gagal
 * tidak dicoba ulang, karena tokennya sudah dianggap tidak sah.
 */
export default function VerifyEmailPage() {
  const [params] = useSearchParams()
  const token = params.get('token') ?? ''

  const verify = useQuery({
    queryKey: ['auth', 'verify-email', token],
    queryFn: () => authApi.verifyEmail(token),
    enabled: token !== '',
    retry: false,
    staleTime: Infinity,
  })

  return (
    <div className="mx-auto flex min-h-screen max-w-md flex-col justify-center p-6">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Verifikasi email</CardTitle>
          <CardDescription className="text-xs">
            Verifikasi membuka trial, koneksi broker, dan pembayaran.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {!token && (
            <Alert variant="destructive">
              Tautan tidak lengkap. Minta tautan baru dari halaman Keamanan di pengaturan.
            </Alert>
          )}
          {token && verify.isPending && <Alert variant="info">Memeriksa tautan…</Alert>}
          {verify.isSuccess && <Alert variant="success">Email terverifikasi. Anda bisa melanjutkan.</Alert>}
          {verify.isError && (
            <Alert variant="destructive">
              {verify.error instanceof Error
                ? verify.error.message
                : 'Tautan tidak sah atau sudah kedaluwarsa.'}
            </Alert>
          )}
          <Button className="w-full" onClick={() => (window.location.href = '/app/mulai')}>
            Lanjutkan ke langkah berikutnya
          </Button>
          <p className="text-center text-[12px] text-muted-foreground">
            <Link to="/pengaturan" className="underline">
              Buka pengaturan
            </Link>
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
