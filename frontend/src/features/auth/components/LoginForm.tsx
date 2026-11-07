import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Alert, Button, Input, Label } from '@/components/ui'
import { ApiError } from '@/lib/api-error'
import { useSignIn } from '../hooks/use-auth'
import { loginSchema, type LoginFormValues } from '../schemas'

export function LoginForm() {
  const signIn = useSignIn()
  const [formError, setFormError] = useState('')
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
      onError: (error) => setFormError(describeError(error)),
    })
  })

  return (
    <form className="flex flex-col gap-4" onSubmit={onSubmit} noValidate>
      <header className="flex flex-col gap-1">
        <h1 className="text-xl font-semibold tracking-tight">Masuk</h1>
        <p className="text-[13px] text-muted-foreground">Gunakan akun ILA Bot Anda.</p>
      </header>

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

      <p className="text-center text-[13px] text-muted-foreground">
        Belum punya akun?{' '}
        <Link to="/register" className="font-medium text-primary hover:underline">
          Daftar
        </Link>
      </p>
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
