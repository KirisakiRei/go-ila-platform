import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import {
  Alert,
  Badge,
  Button,
  EmptyState,
  Input,
  Label,
  PageHeader,
  Section,
  Select,
  Skeleton,
} from '@/components/ui'
import {
  useAccounts,
  useCreateSubscription,
  useDeleteSubscription,
  useStrategies,
  useSubscriptions,
} from '@/lib/queries'
import { ApiError } from '@/lib/api-error'
import type { StrategyVersion } from '@/types/api'

const skema = z.object({
  symbol: z
    .string()
    .min(3, 'Simbol minimal 3 karakter')
    .max(12, 'Simbol maksimal 12 karakter')
    .transform((v) => v.toUpperCase().replace(/\//g, '')),
  strategy: z.enum(['ila-2.0', 'ila-3.0']),
  broker_account_id: z.string().min(1, 'Pilih akun broker'),
})

type NilaiForm = z.input<typeof skema>

function FormLangganan() {
  const [galat, setGalat] = useState<string | null>(null)
  const { data: accounts } = useAccounts()
  const { data: strategies } = useStrategies()
  const create = useCreateSubscription()

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<NilaiForm>({
    resolver: zodResolver(skema),
    defaultValues: { symbol: '', strategy: 'ila-2.0', broker_account_id: '' },
  })

  // Hanya strategi yang sudah dirilis dan boleh untuk tier ini yang muncul.
  const tersedia = (strategies ?? []).filter((s) => s.available && s.allowed)

  const submit = handleSubmit(async (nilai) => {
    setGalat(null)
    try {
      await create.mutateAsync({
        symbol: nilai.symbol,
        strategy: nilai.strategy as StrategyVersion,
        broker_account_id: nilai.broker_account_id,
      })
      reset({ symbol: '', strategy: nilai.strategy, broker_account_id: nilai.broker_account_id })
    } catch (err) {
      setGalat(err instanceof ApiError ? err.message : 'Gagal berlangganan simbol.')
    }
  })

  return (
    <form onSubmit={submit} className="flex flex-col gap-3" noValidate>
      <div className="grid gap-3 sm:grid-cols-3">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="symbol">Simbol</Label>
          <Input id="symbol" placeholder="XAUUSD" {...register('symbol')} />
          {errors.symbol && <p className="text-[11px] text-foreground">{errors.symbol.message}</p>}
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="strategy">Strategi</Label>
          <Select id="strategy" {...register('strategy')}>
            {tersedia.length === 0 && <option value="ila-2.0">ILA 2.0</option>}
            {tersedia.map((s) => (
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
          {errors.broker_account_id && (
            <p className="text-[11px] text-foreground">{errors.broker_account_id.message}</p>
          )}
        </div>
      </div>

      {galat && <Alert variant="destructive">{galat}</Alert>}

      <Button type="submit" size="sm" isLoading={isSubmitting} className="self-start">
        Berlangganan
      </Button>
    </form>
  )
}

export default function StrategiesPage() {
  const { data: strategies, isLoading: loadingStrategies } = useStrategies()
  const { data: subs, isLoading: loadingSubs } = useSubscriptions()
  const remove = useDeleteSubscription()

  return (
    <>
      <PageHeader
        title="Strategi & Langganan"
        description="Pilih strategi dan pasangan yang ingin dipantau."
      />

      <Section title="Strategi tersedia">
        {loadingStrategies ? (
          <Skeleton className="h-24 w-full" />
        ) : (
          <div className="grid gap-3 sm:grid-cols-2">
            {strategies?.map((s) => (
              <article key={s.version} className="flex flex-col gap-2 border border-border p-4">
                <div className="flex items-center justify-between gap-2">
                  <h3 className="text-[14px] font-semibold">{s.name}</h3>
                  <Badge variant={s.status === 'active' ? 'solid' : 'muted'}>
                    {s.status === 'coming_soon'
                      ? 'Segera hadir'
                      : s.status === 'beta'
                        ? 'Beta'
                        : 'Aktif'}
                  </Badge>
                </div>
                <p className="text-[12px] leading-relaxed text-muted-foreground">{s.description}</p>
                <div className="flex flex-wrap gap-1.5 pt-1">
                  {s.selected && <Badge variant="outline">Sedang dipakai</Badge>}
                  {!s.allowed && <Badge variant="muted">Khusus tier Plus</Badge>}
                  {s.allowed && !s.available && <Badge variant="muted">Belum dirilis</Badge>}
                </div>
              </article>
            ))}
          </div>
        )}
      </Section>

      <Section
        title="Langganan pasangan"
        description="Maksimal 8 simbol aktif. Simbol yang sama tidak digandakan."
      >
        <div className="border border-border p-4">
          <FormLangganan />
        </div>

        {loadingSubs && <Skeleton className="h-16 w-full" />}

        {subs?.length === 0 && (
          <EmptyState
            title="Belum ada langganan"
            description="Tambahkan pasangan di atas untuk mulai menerima sinyal."
          />
        )}

        {subs && subs.length > 0 && (
          <ul className="divide-y divide-border border border-border">
            {subs.map((sub) => (
              <li
                key={sub.id}
                className="flex flex-wrap items-center gap-2 px-4 py-2.5 text-[13px]"
              >
                <span className="w-24 font-semibold">{sub.symbol}</span>
                <Badge variant="outline">{sub.strategy}</Badge>
                {!sub.is_active && <Badge variant="muted">Nonaktif</Badge>}
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
        )}
      </Section>
    </>
  )
}
