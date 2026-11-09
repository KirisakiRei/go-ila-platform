import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'

import { Alert, Button, Input, Label } from '@/components/ui'
import { adminAuthApi, type AdminLoginResult } from '@/lib/admin-api'
import { useAdminAuth } from '@/features/admin/admin-auth-context'
import { Logo } from '@/components/brand/Logo'

/**
 * AdminLoginPage adalah pintu masuk panel admin.
 *
 * Halaman ini sengaja terpisah dari login pelanggan: server memakai cookie
 * sesi admin sendiri, dan hanya menerima akun ber-role admin. Di sini tidak ada
 * tautan pendaftaran — akun admin hanya dibuat dari dalam panel admin oleh
 * admin lain, bukan oleh siapa pun yang menemukan halamannya.
 */
export default function AdminLoginPage() {
  const navigate = useNavigate()
  const { signIn } = useAdminAuth()

  const [identifier, setIdentifier] = useState('')
  const [password, setPassword] = useState('')
  const [code, setCode] = useState('')
  const [challengeId, setChallengeId] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  const finish = (result: AdminLoginResult) => {
    // Permission ikut disimpan: tanpa ini panel tidak tahu menu mana yang boleh
    // ditampilkan dan memaksa setiap halaman menerima 403 dari server.
    signIn({
      id: result.user.id,
      name: result.user.name,
      email: result.user.email,
      role: result.role_code || result.user.role,
      isSuperadmin: result.role_code === 'superadmin',
      permissions: result.permissions ?? [],
    })
    navigate('/admin', { replace: true })
  }

  const submit = async (event: FormEvent) => {
    event.preventDefault()
    setError(null)
    setPending(true)

    try {
      const result = await adminAuthApi.login({ identifier: identifier.trim(), password })

      if (result.step === 'two_factor_required') {
        setChallengeId(result.challenge_id ?? '')
        return
      }

      finish(result)
    } catch (cause) {
      // Pesan server dipakai apa adanya, tetapi akun yang bukan admin
      // ditampilkan sebagai kredensial salah agar keberadaan akun tidak bocor.
      setError(cause instanceof Error ? cause.message : 'Gagal masuk.')
    } finally {
      setPending(false)
    }
  }

  const submitCode = async (event: FormEvent) => {
    event.preventDefault()
    setError(null)
    setPending(true)

    try {
      const result = await adminAuthApi.verify2fa({ challenge_id: challengeId, code: code.trim() })
      finish(result)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Kode tidak diterima.')
    } finally {
      setPending(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-[#f5f5f7] px-4 py-10">
      <div className="w-full max-w-sm">
        <div className="flex items-center gap-2.5">
          <Logo alt="ILA Admin" theme="light" className="h-8 w-auto" />
          <span className="text-sm font-bold tracking-tight text-[#09090b]">ILA Admin</span>
        </div>

        <h1 className="mt-6 text-[24px] font-semibold tracking-tight text-[#09090b]">
          Masuk ke panel admin
        </h1>
        <p className="mt-2 text-[13px] leading-relaxed text-[#52525b]">
          Halaman ini hanya untuk akun pengelola. Akun pelanggan tidak dapat masuk di sini.
        </p>

        {error && (
          <div className="mt-5">
            <Alert>{error}</Alert>
          </div>
        )}

        {challengeId ? (
          <form onSubmit={submitCode} className="mt-6 space-y-4">
            <div>
              <Label htmlFor="admin-code">Kode autentikasi dua faktor</Label>
              <Input
                id="admin-code"
                inputMode="numeric"
                autoComplete="one-time-code"
                value={code}
                onChange={(event) => setCode(event.target.value)}
                className="mt-1"
                placeholder="000000"
                required
              />
              <p className="mt-1 text-[11px] text-muted-foreground">
                Masukkan kode 6 digit dari aplikasi authenticator, atau salah satu kode pemulihan.
              </p>
            </div>
            <Button type="submit" className="w-full" disabled={pending || code.trim().length < 6}>
              {pending ? 'Memeriksa…' : 'Masuk'}
            </Button>
          </form>
        ) : (
          <form onSubmit={submit} className="mt-6 space-y-4">
            <div>
              <Label htmlFor="admin-identifier">Email atau username</Label>
              <Input
                id="admin-identifier"
                autoComplete="username"
                value={identifier}
                onChange={(event) => setIdentifier(event.target.value)}
                className="mt-1"
                required
              />
            </div>
            <div>
              <Label htmlFor="admin-password">Kata sandi</Label>
              <Input
                id="admin-password"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                className="mt-1"
                required
              />
            </div>
            <Button
              type="submit"
              className="w-full"
              disabled={pending || !identifier.trim() || !password}
            >
              {pending ? 'Memeriksa…' : 'Masuk'}
            </Button>
          </form>
        )}

        <p className="mt-6 text-[11px] leading-relaxed text-[#a1a1aa]">
          Akun admin tidak dibuat lewat pendaftaran. Tambahkan akun pengelola dari menu Pengguna di
          dalam panel admin.
        </p>
      </div>
    </div>
  )
}
