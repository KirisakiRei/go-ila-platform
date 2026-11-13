import { useState, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { useForm, useWatch } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Alert, Button, Input, Label } from '@/components/ui'
import { ApiError } from '@/lib/api-error'
import { cn } from '@/lib/utils'
import { useSignUp } from '../hooks/use-auth'
import {
  REGISTER_STEPS,
  REGISTER_STEP_FIELDS,
  registerSchema,
  type RegisterFormValues,
  type RegisterStep,
} from '../schemas'

/**
 * RegisterForm membagi pendaftaran menjadi tiga langkah.
 *
 * Satu layar berisi lima isian membuat pengguna melewatkan bagian yang salah
 * tanpa sadar. Dengan langkah, kesalahan tertangkap per bagian dan pengguna
 * tahu seberapa jauh prosesnya.
 */
export function RegisterForm({ inviteCode }: { inviteCode?: string } = {}) {
  const signUp = useSignUp()
  const [formError, setFormError] = useState('')
  const [isDone, setIsDone] = useState(false)
  const [stepIndex, setStepIndex] = useState(0)

  const {
    register,
    handleSubmit,
    trigger,
    control,
    formState: { errors },
  } = useForm<RegisterFormValues>({
    resolver: zodResolver(registerSchema),
    mode: 'onTouched',
    defaultValues: { name: '', username: '', email: '', phone: '', password: '', acceptTerms: false },
  })

  const step = REGISTER_STEPS[stepIndex] ?? REGISTER_STEPS[0]
  const isLastStep = stepIndex === REGISTER_STEPS.length - 1
  const username = useWatch({ control, name: 'username' })

  /** Lanjut hanya bila bagian ini sah; validasi akhir tetap saat kirim. */
  async function next() {
    setFormError('')
    const valid = await trigger(REGISTER_STEP_FIELDS[step])
    if (valid) setStepIndex((index) => Math.min(index + 1, REGISTER_STEPS.length - 1))
  }

  function back() {
    setFormError('')
    setStepIndex((index) => Math.max(index - 1, 0))
  }

  const onSubmit = handleSubmit((values) => {
    setFormError('')
    signUp.mutate(
      {
        name: values.name,
        username: values.username,
        email: values.email,
        phone: values.phone,
        password: values.password,
        accept_terms: values.acceptTerms,
        // Kode undangan diteruskan apa adanya; backend memvalidasi ulang dan
        // memotong kuota di sana. Tanpa ini, pendaftaran saat invite only ditolak.
        ...(inviteCode ? { invite_code: inviteCode } : {}),
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
            Masuk dengan email Anda, atau dengan username{' '}
            <span className="font-medium text-foreground">{username}</span>.
          </p>
        </header>
        <Button asChild className="w-full">
          <Link to="/login">Ke halaman masuk</Link>
        </Button>
      </div>
    )
  }

  return (
    <form className="flex flex-col gap-5" onSubmit={onSubmit} noValidate>
      <header className="flex flex-col gap-2">
        <h1 className="text-xl font-semibold tracking-tight">Daftar</h1>
        <StepIndicator steps={REGISTER_STEPS} current={stepIndex} />
      </header>

      {step === 'Akun' && (
        <>
          <Field
            id="name"
            label="Nama lengkap"
            error={errors.name?.message}
            input={
              <Input
                id="name"
                type="text"
                autoComplete="name"
                placeholder="Nama yang ditampilkan"
                hasError={Boolean(errors.name)}
                {...register('name')}
              />
            }
          />

          <Field
            id="username"
            label="Username"
            hint="Dipakai untuk masuk. Huruf, angka, titik, garis bawah, dan strip."
            error={errors.username?.message}
            input={
              <Input
                id="username"
                type="text"
                autoComplete="username"
                autoCapitalize="none"
                autoCorrect="off"
                spellCheck={false}
                placeholder="mis. budi.santoso"
                hasError={Boolean(errors.username)}
                {...register('username')}
              />
            }
          />

          <Field
            id="email"
            label="Email"
            hint="Dipakai untuk pemulihan akun dan pemberitahuan penting."
            error={errors.email?.message}
            input={
              <Input
                id="email"
                type="email"
                autoComplete="email"
                placeholder="nama@email.com"
                hasError={Boolean(errors.email)}
                {...register('email')}
              />
            }
          />

          <Field
            id="phone"
            label="Nomor telepon"
            hint="Opsional. Hanya dipakai bila kami perlu menghubungi Anda."
            error={errors.phone?.message}
            input={
              <Input
                id="phone"
                type="tel"
                inputMode="tel"
                autoComplete="tel"
                placeholder="+62 812 3456 7890"
                hasError={Boolean(errors.phone)}
                {...register('phone')}
              />
            }
          />
        </>
      )}

      {step === 'Keamanan' && (
        <>
          <Field
            id="password"
            label="Password"
            hint="Minimal 10 karakter. Gunakan kombinasi yang tidak dipakai di tempat lain."
            error={errors.password?.message}
            input={
              <Input
                id="password"
                type="password"
                autoComplete="new-password"
                hasError={Boolean(errors.password)}
                {...register('password')}
              />
            }
          />

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
        </>
      )}

      <p className="text-[12px] text-muted-foreground">Akun baru dimulai pada tier Free.</p>

      {formError && <Alert>{formError}</Alert>}

      <div className="flex gap-2">
        {stepIndex > 0 && (
          <Button type="button" variant="outline" className="flex-1" onClick={back}>
            Kembali
          </Button>
        )}
        {isLastStep ? (
          <Button type="submit" className="flex-1" isLoading={signUp.isPending}>
            Buat akun
          </Button>
        ) : (
          <Button type="button" className="flex-1" onClick={() => void next()}>
            Lanjut
          </Button>
        )}
      </div>
    </form>
  )
}

/** StepIndicator menunjukkan posisi dan panjang proses pendaftaran. */
function StepIndicator({ steps, current }: { steps: readonly string[]; current: number }) {
  return (
    <ol className="flex items-center gap-1.5" aria-label="Langkah pendaftaran">
      {steps.map((label, index) => {
        const state = index < current ? 'done' : index === current ? 'active' : 'next'
        return (
          <li key={label} className="flex flex-1 flex-col gap-1.5">
            <span
              className={cn(
                'h-1 w-full rounded-full',
                state === 'next' ? 'bg-border' : 'bg-foreground',
              )}
            />
            <span
              className={cn(
                'text-micro uppercase tracking-[0.15em]',
                state === 'active' ? 'text-foreground' : 'text-muted-foreground',
              )}
            >
              {label}
            </span>
          </li>
        )
      })}
    </ol>
  )
}

/** Field membungkus label, isian, bantuan, dan galat dengan susunan seragam. */
function Field({
  id,
  label,
  hint,
  error,
  input,
}: {
  id: string
  label: string
  hint?: string
  error?: string
  input: ReactNode
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={id}>{label}</Label>
      {input}
      {error ? (
        <p id={`${id}-error`} className="text-xs text-destructive">
          {error}
        </p>
      ) : (
        hint && (
          <p id={`${id}-hint`} className="text-xs text-muted-foreground">
            {hint}
          </p>
        )
      )}
    </div>
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

export type { RegisterStep }
