import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Alert, Button, Input, Label } from '@/components/ui'
import { ApiError } from '@/lib/api-error'
import { useSignUp } from '../hooks/use-auth'
import { registerSchema, type RegisterFormValues } from '../schemas'

export function RegisterForm() {
  const signUp = useSignUp()
  const [formError, setFormError] = useState('')
  const [isDone, setIsDone] = useState(false)
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<RegisterFormValues>({
    resolver: zodResolver(registerSchema),
    defaultValues: { name: '', email: '', password: '', acceptTerms: false },
  })

  const onSubmit = handleSubmit((values) => {
    setFormError('')
    signUp.mutate(
      {
        name: values.name,
        email: values.email,
        password: values.password,
        accept_terms: values.acceptTerms,
      },
      {
        onSuccess: () => setIsDone(true),
        onError: (error) => setFormError(describeError(error)),
      },
    )
  })

  if (isDone) {
    return (
      <div className="flex flex-col gap-4">
        <header className="flex flex-col gap-1">
          <h1 className="text-xl font-semibold tracking-tight">Akun dibuat</h1>
          <p className="text-[13px] text-muted-foreground">
            Sekarang Anda bisa masuk dengan email tersebut.
          </p>
        </header>
        <Button asChild className="w-full">
          <Link to="/login">Ke halaman masuk</Link>
        </Button>
      </div>
    )
  }

  return (
    <form className="flex flex-col gap-4" onSubmit={onSubmit} noValidate>
      <header className="flex flex-col gap-1">
        <h1 className="text-xl font-semibold tracking-tight">Daftar</h1>
        <p className="text-[13px] text-muted-foreground">Akun baru dimulai pada tier Free.</p>
      </header>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="name">Nama</Label>
        <Input
          id="name"
          type="text"
          autoComplete="name"
          hasError={Boolean(errors.name)}
          aria-describedby={errors.name ? 'name-error' : undefined}
          {...register('name')}
        />
        {errors.name && (
          <p id="name-error" className="text-xs text-destructive">
            {errors.name.message}
          </p>
        )}
      </div>

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
          autoComplete="new-password"
          hasError={Boolean(errors.password)}
          aria-describedby={errors.password ? 'password-error' : 'password-hint'}
          {...register('password')}
        />
        {errors.password ? (
          <p id="password-error" className="text-xs text-destructive">
            {errors.password.message}
          </p>
        ) : (
          <p id="password-hint" className="text-xs text-muted-foreground">
            Minimal 10 karakter.
          </p>
        )}
      </div>

      <div className="flex flex-col gap-1">
        <label className="flex items-start gap-2 text-[13px]">
          <input type="checkbox" className="mt-0.5" {...register('acceptTerms')} />
          <span>
            Saya menyetujui{' '}
            <Link to="/legal/tos" className="underline underline-offset-2">
              Syarat Layanan
            </Link>
            ,{' '}
            <Link to="/legal/privacy" className="underline underline-offset-2">
              Kebijakan Privasi
            </Link>
            , dan{' '}
            <Link to="/legal/fair_use" className="underline underline-offset-2">
              Kebijakan Penggunaan Wajar
            </Link>
            .
          </span>
        </label>
        {errors.acceptTerms && (
          <p className="text-xs text-destructive">{errors.acceptTerms.message}</p>
        )}
      </div>

      {formError && <Alert>{formError}</Alert>}

      <Button type="submit" className="w-full" isLoading={signUp.isPending}>
        Daftar
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
