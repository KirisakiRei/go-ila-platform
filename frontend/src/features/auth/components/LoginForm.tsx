import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Alert, Button, Input, Label } from '@/components/ui'
import { ApiError } from '@/lib/api-error'
import { LOGOUT_REASON_KEY } from '../AuthProvider'
import { useSignIn, useVerifyTwoFactor } from '../hooks/use-auth'
import { loginSchema, type LoginFormValues } from '../schemas'

export function LoginForm() {
  const signIn = useSignIn()
  const verify = useVerifyTwoFactor()
  const [formError, setFormError] = useState('')
  const [logoutNotice] = useState(readLogoutNotice)
  // Tantangan 2FA dipegang di sini: selama belum diverifikasi, belum ada sesi.
  const [challenge, setChallenge] = useState<{ id: string; code: string } | null>(null)
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: '', password: '' },
  })

  const onSubmit = handleSubmit((values) => {
    setFormError('')
    signIn.mutate(values, {
      onSuccess: (result) => {
        if (result.step === 'two_factor_required' && result.challenge_id) {
          setChallenge({ id: result.challenge_id, code: '' })
        }
      },
      onError: (error) => setFormError(describeError(error)),
    })
  })

  const onSubmitCode = (event: FormEvent) => {
    event.preventDefault()
    if (!challenge) return
    setFormError('')
    verify.mutate(
      { challengeId: challenge.id, code: challenge.code },
      { onError: (error) => setFormError(describeError(error)) },
    )
  }

  if (challenge) {
    return (
      <form className="flex flex-col gap-4" onSubmit={onSubmitCode} noValidate>
        <header className="flex flex-col gap-1">
          <h1 className="text-xl font-semibold tracking-tight">Verifikasi dua langkah</h1>
          <p className="text-[13px] text-muted-foreground">
            Masukkan kode 6 digit dari aplikasi authenticator, atau salah satu kode pemulihan Anda.
          </p>
        </header>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="otp">Kode</Label>
          <Input
            id="otp"
            inputMode="numeric"
            autoComplete="one-time-code"
            autoFocus
            maxLength={40}
            value={challenge.code}
            onChange={(event) => setChallenge({ ...challenge, code: event.target.value.trim() })}
          />
        </div>

        {formError && <Alert>{formError}</Alert>}

        <Button type="submit" className="w-full" isLoading={verify.isPending} disabled={!challenge.code}>
          Verifikasi
        </Button>
        <button
          type="button"
          className="text-[12px] text-muted-foreground underline"
          onClick={() => {
            setChallenge(null)
            setFormError('')
          }}
        >
          Kembali ke halaman masuk
        </button>
      </form>
    )
  }

  return (
    <form className="flex flex-col gap-4" onSubmit={onSubmit} noValidate>
      <header className="flex flex-col gap-1">
        <h1 className="text-xl font-semibold tracking-tight">Masuk</h1>
        <p className="text-[13px] text-muted-foreground">Gunakan akun ILA Bot Anda.</p>
      </header>

      {logoutNotice && <Alert>{logoutNotice}</Alert>}

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="email">Email</Label>
        <Input
          id="email"
          type="email"
          autoComplete="email"
          placeholder="nama@email.com"
          hasError={Boolean(errors.email)}
          aria-describedby={errors.email ? 'email-error' : undefined}
          {...register('email')}
        />
        {errors.email && (
          <p id="email-error" className="text-xs text-destructive">
            {errors.email.message}
          </p>
        )}
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="password">Password</Label>
        <Input
          id="password"
          type="password"
          autoComplete="current-password"
          hasError={Boolean(errors.password)}
          aria-describedby={errors.password ? 'password-error' : undefined}
          {...register('password')}
        />
        {errors.password && (
          <p id="password-error" className="text-xs text-destructive">
            {errors.password.message}
          </p>
        )}
      </div>

      {formError && <Alert>{formError}</Alert>}

      <Button type="submit" className="w-full" isLoading={signIn.isPending}>
        Masuk
      </Button>

      <Link to="/lupa-sandi" className="self-center text-[12px] text-muted-foreground underline">
        Lupa kata sandi?
      </Link>
    </form>
  )
}

/** Menerjemahkan kegagalan API menjadi pesan yang bisa dibaca pengguna. */
function describeError(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.isNetworkProblem) {
      return 'Server tidak dapat dihubungi. Coba beberapa saat lagi.'
    }
    return error.message
  }
  return 'Terjadi kesalahan tak terduga.'
}

/**
 * readLogoutNotice menjelaskan kenapa pengguna harus masuk lagi. Alasan yang
 * berbeda mencegah kebingungan "kok tiba-tiba keluar?" (plans/23 §6).
 */
function readLogoutNotice(): string {
  const reason = sessionStorage.getItem(LOGOUT_REASON_KEY)
  if (!reason) return ''

  sessionStorage.removeItem(LOGOUT_REASON_KEY)

  switch (reason) {
    case 'idle':
      return 'Anda keluar karena tidak ada aktivitas. Silakan masuk lagi.'
    case 'expired':
      return 'Sesi Anda berakhir. Silakan masuk lagi.'
    default:
      return 'Anda telah keluar. Silakan masuk lagi.'
  }
}
