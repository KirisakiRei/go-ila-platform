import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import {
  useCreateSubscription,
  useDeleteSubscription,
  useAccounts,
  useStrategies,
  useSubscriptions,
} from '@/lib/queries'
import { ApiError } from '@/lib/api-error'
import {
  Alert,
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Input,
  Label,
  Select,
  Skeleton,
} from '@/components/ui'
import type { StrategyVersion } from '@/types/api'

const schema = z.object({
  symbol: z
    .string()
    .min(3, 'Simbol minimal 3 karakter')
    .max(12, 'Simbol maksimal 12 karakter')
    .transform((value) => value.toUpperCase().replace(/\//g, '')),
  strategy: z.enum(['ila-2.0', 'ila-3.0']),
  broker_account_id: z.string(),
})

type FormValues = z.input<typeof schema>

function SubscriptionForm() {
  const [submitError, setSubmitError] = useState<string | null>(null)
  const { data: accounts } = useAccounts()
  const { data: strategies } = useStrategies()
  const create = useCreateSubscription()

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { symbol: '', strategy: 'ila-2.0', broker_account_id: '' },
  })

  // Strategi yang belum dirilis atau tier-nya tidak berhak tidak boleh dipilih.
  const selectable = (strategies ?? []).filter((s) => s.available && s.allowed)

  const onSubmit = handleSubmit(async (values) => {
    setSubmitError(null)
    try {
      await create.mutateAsync({
        symbol: values.symbol,
        strategy: values.strategy as StrategyVersion,
        broker_account_id: values.broker_account_id,
      })
      reset({ symbol: '', strategy: values.strategy, broker_account_id: values.broker_account_id })
    } catch (error) {
      setSubmitError(error instanceof ApiError ? error.message : 'Gagal berlangganan simbol.')
    }
  })

  return (
    <form onSubmit={onSubmit} className="flex flex-col gap-4" noValidate>
      <div className="grid gap-4 sm:grid-cols-3">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="symbol">Simbol</Label>
          <Input id="symbol" placeholder="XAUUSD" {...register('symbol')} />
          {errors.symbol && <p className="text-[12px] text-destructive">{errors.symbol.message}</p>}
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="strategy">Strategi</Label>
          <Select id="strategy" {...register('strategy')}>
            {selectable.length === 0 && <option value="ila-2.0">ILA 2.0</option>}
            {selectable.map((s) => (
              <option key={s.version} value={s.version}>
                {s.name}
              </option>
            ))}
          </Select>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="broker_account_id">Akun broker</Label>
          <Select id="broker_account_id" {...register('broker_account_id')}>
            <option value="">Pilih akun</option>
            {accounts?.map((a) => (
              <option key={a.id} value={a.id}>
                {a.label}
              </option>
            ))}
          </Select>
        </div>
      </div>

      {submitError && <Alert>{submitError}</Alert>}

      <Button type="submit" isLoading={isSubmitting} className="self-start">
        Berlangganan
      </Button>
    </form>
  )
}

export default function StrategiesPage() {
  const { data: strategies, isLoading: loadingStrategies } = useStrategies()
  const { data: subscriptions, isLoading: loadingSubs } = useSubscriptions()
  const remove = useDeleteSubscription()

  return (
    <div className="flex flex-col gap-4">
      <header className="flex flex-col gap-1">
        <h1 className="text-xl font-semibold tracking-tight">Strategi & Langganan</h1>
        <p className="text-[13px] text-muted-foreground">
          Pilih strategi dan pasangan yang ingin dipantau.
        </p>
      </header>

      <div className="grid gap-3 sm:grid-cols-2">
        {loadingStrategies && <Skeleton className="h-28 w-full" />}
        {strategies?.map((strategy) => (
          <Card key={strategy.version}>
            <CardHeader>
              <div className="flex items-center justify-between gap-2">
                <CardTitle>{strategy.name}</CardTitle>
                <Badge
                  variant={
                    strategy.status === 'active'
                      ? 'success'
                      : strategy.status === 'beta'
                        ? 'warning'
                        : 'default'
                  }
                >
                  {strategy.status === 'coming_soon'
                    ? 'Segera hadir'
                    : strategy.status === 'beta'
                      ? 'Beta'
                      : 'Aktif'}
                </Badge>
              </div>
              <CardDescription>{strategy.description}</CardDescription>
            </CardHeader>
            <CardContent className="flex items-center gap-2 text-[12px] text-muted-foreground">
              {strategy.selected && <Badge variant="primary">Sedang dipakai</Badge>}
              {!strategy.allowed && <span>Khusus tier Plus</span>}
              {strategy.allowed && !strategy.available && <span>Belum dirilis</span>}
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Langganan pasangan</CardTitle>
          <CardDescription>Maksimal 8 simbol aktif per akun.</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <SubscriptionForm />

          {loadingSubs && <Skeleton className="h-16 w-full" />}

          {subscriptions?.length === 0 && (
            <p className="text-[13px] text-muted-foreground">Belum ada langganan simbol.</p>
          )}

          <ul className="flex flex-col divide-y divide-border">
            {subscriptions?.map((sub) => (
              <li key={sub.id} className="flex items-center gap-3 py-2.5">
                <span className="font-medium">{sub.symbol}</span>
                <Badge variant="primary">{sub.strategy}</Badge>
                {!sub.is_active && <Badge variant="warning">Nonaktif</Badge>}
                <Button
                  size="sm"
                  variant="ghost"
                  className="ml-auto"
                  isLoading={remove.isPending && remove.variables === sub.id}
                  onClick={() => remove.mutate(sub.id)}
                >
                  Berhenti
                </Button>
              </li>
            ))}
          </ul>
        </CardContent>
      </Card>
    </div>
  )
}
