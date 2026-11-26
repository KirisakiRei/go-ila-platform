import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { ArrowRight, Eye, EyeOff } from 'lucide-react'
import { Alert, Button, Input, Label } from '@/components/ui'
import { ApiError } from '@/lib/api-error'
import { useInviteOnly } from '@/lib/use-invite-only'
import { LOGOUT_REASON_KEY } from '../AuthProvider'
import { useSignIn, useVerifyTwoFactor } from '../hooks/use-auth'
import { loginSchema, type LoginFormValues } from '../schemas'

const AUTH_INPUT_CLASS =
  'h-11 rounded-lg border-zinc-200 bg-zinc-50/80 px-3.5 text-base sm:text-sm placeholder:text-muted-foreground/60 dark:border-white/[0.1] dark:bg-[#141417]'

const AUTH_PRIMARY_BUTTON_CLASS =
  'press h-11 w-full rounded-lg bg-[#09090b] text-sm font-semibold text-white hover:bg-zinc-800 dark:bg-white dark:text-[#09090b] dark:hover:bg-zinc-200'

export function LoginForm() {
  const signIn = useSignIn()
  const verify = useVerifyTwoFactor()
  const inviteOnly = useInviteOnly()
  const [formError, setFormError] = useState('')
  const [logoutNotice] = useState(readLogoutNotice)
  const [showPassword, setShowPassword] = useState(false)
  // Tantangan 2FA dipegang di sini: selama belum diverifikasi, belum ada sesi.
  const [challenge, setChallenge] = useState<{
    id: string
    code: string
    rememberMe: boolean
  } | null>(null)
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { identifier: '', password: '', rememberMe: true },
  })

  const onSubmit = handleSubmit((values) => {
    setFormError('')
    signIn.mutate(
      {
        identifier: values.identifier,
        password: values.password,
        remember_me: values.rememberMe,
      },
      {
        onSuccess: (result) => {
          if (result.step === 'two_factor_required' && result.challenge_id) {
            setChallenge({
              id: result.challenge_id,
              code: '',
              rememberMe: values.rememberMe,
            })
          }
        },
        onError: (error) => setFormError(describeError(error)),
      },
    )
  })

  const onSubmitCode = (event: FormEvent) => {
    event.preventDefault()
    if (!challenge) return
    setFormError('')
    verify.mutate(
      { challengeId: challenge.id, code: challenge.code, rememberMe: challenge.rememberMe },
      { onError: (error) => setFormError(describeError(error)) },
    )
  }

  if (challenge) {
    return (
      <form className="flex flex-col gap-4 sm:gap-5" onSubmit={onSubmitCode} noValidate>
        <header className="flex flex-col gap-1 sm:gap-1.5">
          <h1 className="text-[24px] font-bold leading-tight tracking-tight text-foreground sm:text-[28px]">
            Verifikasi dua langkah
          </h1>
          <p className="text-[13px] leading-relaxed text-muted-foreground sm:text-[13.5px]">
            Masukkan kode 6 digit dari aplikasi authenticator, atau salah satu kode pemulihan Anda.
          </p>
        </header>

        <div className="flex flex-col gap-2">
          <Label htmlFor="otp" className="text-[13px] font-medium text-foreground">
            Kode verifikasi
          </Label>
          <Input
            id="otp"
            inputMode="numeric"
            autoComplete="one-time-code"
            autoFocus
            maxLength={40}
            placeholder="000000"
            className={AUTH_INPUT_CLASS}
            value={challenge.code}
            onChange={(event) => setChallenge({ ...challenge, code: event.target.value.trim() })}
          />
        </div>

        {formError && <Alert>{formError}</Alert>}

        <Button
          type="submit"
          className={AUTH_PRIMARY_BUTTON_CLASS}
          isLoading={verify.isPending}
          disabled={!challenge.code}
        >
          <span>Verifikasi</span>
          <ArrowRight className="h-4 w-4" aria-hidden="true" />
        </Button>

        <button
          type="button"
          className="self-center text-[12.5px] text-muted-foreground underline underline-offset-4 transition-colors hover:text-foreground"
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
    <form className="flex flex-col gap-4 sm:gap-5" onSubmit={onSubmit} noValidate>
      <header className="flex flex-col gap-1 sm:gap-1.5">
        <h1 className="text-[24px] font-bold leading-tight tracking-tight text-foreground sm:text-[28px]">
          Masuk ke akun
        </h1>
        <p className="text-[13px] leading-relaxed text-muted-foreground sm:text-[13.5px]">
          {inviteOnly
            ? 'Gunakan username atau email terdaftar dari undangan akses Anda.'
            : 'Masuk dengan username atau email akun Anda.'}
        </p>
      </header>

      {logoutNotice && <Alert>{logoutNotice}</Alert>}

      <div className="flex flex-col gap-2">
        <Label htmlFor="identifier" className="text-[13px] font-medium text-foreground">
          Username atau Email
        </Label>
        <Input
          id="identifier"
          type="text"
          autoComplete="username"
          autoCapitalize="none"
          autoCorrect="off"
          spellCheck={false}
          placeholder="username atau nama@email.com"
          className={AUTH_INPUT_CLASS}
          hasError={Boolean(errors.identifier)}
          aria-describedby={errors.identifier ? 'identifier-error' : undefined}
          {...register('identifier')}
        />
        {errors.identifier && (
          <p id="identifier-error" className="text-xs text-destructive">
            {errors.identifier.message}
          </p>
        )}
      </div>

      <div className="flex flex-col gap-2">
        <div className="flex items-center justify-between gap-2">
          <Label htmlFor="password" className="text-[13px] font-medium text-foreground">
            Kata Sandi
          </Label>
          <Link
            to="/lupa-sandi"
            className="text-[12.5px] text-muted-foreground transition-colors hover:text-foreground"
          >
            Lupa kata sandi?
          </Link>
        </div>
        <div className="relative">
          <Input
            id="password"
            type={showPassword ? 'text' : 'password'}
            autoComplete="current-password"
            placeholder="••••••••••••"
            className={`${AUTH_INPUT_CLASS} pr-10`}
            hasError={Boolean(errors.password)}
            aria-describedby={errors.password ? 'password-error' : undefined}
            {...register('password')}
          />
          <button
            type="button"
            onClick={() => setShowPassword((prev) => !prev)}
            aria-label={showPassword ? 'Sembunyikan kata sandi' : 'Tampilkan kata sandi'}
            className="absolute right-1.5 top-1/2 inline-flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-md text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            {showPassword ? (
              <EyeOff className="h-4 w-4" aria-hidden="true" />
            ) : (
              <Eye className="h-4 w-4" aria-hidden="true" />
            )}
          </button>
        </div>
        {errors.password && (
          <p id="password-error" className="text-xs text-destructive">
            {errors.password.message}
          </p>
        )}
      </div>

      <label className="flex cursor-pointer select-none items-center gap-2.5 text-[13px] text-muted-foreground transition-colors hover:text-foreground">
        <input
          type="checkbox"
          className="h-4 w-4 rounded border-border accent-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
          {...register('rememberMe')}
        />
        <span>Ingat perangkat ini (tetap masuk 30 hari)</span>
      </label>

      {formError && <Alert>{formError}</Alert>}

      <Button type="submit" className={`mt-1 ${AUTH_PRIMARY_BUTTON_CLASS}`} isLoading={signIn.isPending}>
        <span>Masuk ke Dashboard</span>
        <ArrowRight className="h-4 w-4" aria-hidden="true" />
      </Button>
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
