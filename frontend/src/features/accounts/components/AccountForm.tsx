import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Alert, Button, Input, Label, Select } from '@/components/ui'
import { ApiError } from '@/lib/api-error'
import { useCreateAccount } from '@/lib/queries'
import type { Tier } from '@/types/api'

const schema = z.object({
  label: z.string().min(1, 'Nama akun wajib diisi').max(100, 'Maksimal 100 karakter'),
  token: z.string().min(20, 'Access token minimal 20 karakter'),
  mode: z.enum(['manual', 'auto']),
  risk_percent: z.coerce.number().min(0.1, 'Minimal 0,10%').max(1, 'Maksimal 1,00%'),
  is_dry_run: z.boolean(),
})

type FormValues = z.input<typeof schema>

/** Form penambahan akun Match-Trader. Token hanya dikirim sekali, tidak pernah ditampilkan lagi. */
export function AccountForm({ tier, onDone }: { tier: Tier; onDone?: () => void }) {
  const [submitError, setSubmitError] = useState<string | null>(null)
  const createAccount = useCreateAccount()

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      label: '',
      token: '',
      mode: 'manual',
      risk_percent: 0.5,
      is_dry_run: true,
    },
  })

  const onSubmit = handleSubmit(async (values) => {
    setSubmitError(null)
    try {
      await createAccount.mutateAsync({
        broker_type: 'matchtrader',
        label: values.label,
        mode: values.mode,
        risk_percent: Number(values.risk_percent),
        is_dry_run: Boolean(values.is_dry_run),
        token: values.token,
        base_url: '',
      })
      reset()
      onDone?.()
    } catch (error) {
      setSubmitError(error instanceof ApiError ? error.message : 'Gagal menyimpan akun.')
    }
  })

  return (
    <form onSubmit={onSubmit} className="flex flex-col gap-4" noValidate>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="label">Nama akun</Label>
        <Input id="label" placeholder="Akun Demo" {...register('label')} />
        {errors.label && <p className="text-[12px] text-destructive">{errors.label.message}</p>}
      </div>

      <div className="flex flex-col gap-1.5">
        <Label htmlFor="token">Access token Match-Trader</Label>
        <Input
          id="token"
          type="password"
          autoComplete="off"
          placeholder="Tempel token dari aplikasi Match-Trader"
          {...register('token')}
        />
        {errors.token && <p className="text-[12px] text-destructive">{errors.token.message}</p>}
        <p className="text-[11px] text-muted-foreground">
          Token disimpan terenkripsi dan tidak dapat ditampilkan kembali.
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="mode">Mode eksekusi</Label>
          <Select id="mode" {...register('mode')} disabled={tier !== 'plus'}>
            <option value="manual">Manual (konfirmasi dulu)</option>
            <option value="auto">Otomatis</option>
          </Select>
          {tier !== 'plus' && (
            <p className="text-[11px] text-muted-foreground">Otomatis hanya untuk tier Plus.</p>
          )}
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="risk_percent">Risiko per trade (%)</Label>
          <Input
            id="risk_percent"
            type="number"
            step="0.05"
            min="0.1"
            max="1"
            {...register('risk_percent')}
          />
          {errors.risk_percent && (
            <p className="text-[12px] text-destructive">{errors.risk_percent.message}</p>
          )}
        </div>
      </div>

      <label className="flex items-center gap-2 text-[13px]">
        <input type="checkbox" className="h-4 w-4 accent-primary" {...register('is_dry_run')} />
        Jalankan sebagai simulasi (tidak mengirim order nyata)
      </label>

      {submitError && <Alert>{submitError}</Alert>}

      <Button type="submit" isLoading={isSubmitting}>
        Hubungkan akun
      </Button>
    </form>
  )
}
