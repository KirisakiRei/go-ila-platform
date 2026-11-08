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
  Modal,
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

function FormTambahPair({ onDone }: { onDone?: () => void }) {
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
      if (onDone) onDone()
    } catch (err) {
      setGalat(err instanceof ApiError ? err.message : 'Gagal menambahkan pair pantauan.')
    }
  })

  return (
    <form onSubmit={submit} className="flex flex-col gap-4 py-1" noValidate>
      <div className="grid gap-3 sm:grid-cols-3">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="symbol">Simbol Pair</Label>
          <Input id="symbol" placeholder="XAUUSD, EURUSD" {...register('symbol')} />
          {errors.symbol && <p className="text-[11px] text-destructive">{errors.symbol.message}</p>}
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="strategy">Metode Analisis</Label>
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
          <Label htmlFor="broker_account_id">Akun Broker Terkait</Label>
          <Select id="broker_account_id" {...register('broker_account_id')}>
            <option value="">Pilih akun broker</option>
            {accounts?.map((a) => (
              <option key={a.id} value={a.id}>
                {a.label}
              </option>
            ))}
          </Select>
          {errors.broker_account_id && (
            <p className="text-[11px] text-destructive">{errors.broker_account_id.message}</p>
          )}
        </div>
      </div>

      <p className="text-[12px] text-muted-foreground leading-relaxed">
        Simbol yang Anda tambahkan di sini akan langsung dipantau oleh engine dan ditampilkan sebagai
        kartu sinyal di halaman Sinyal.
      </p>

      {galat && <Alert variant="destructive">{galat}</Alert>}

      <div className="flex justify-end gap-2 pt-2">
        {onDone && (
          <Button type="button" variant="outline" size="sm" onClick={onDone}>
            Batal
          </Button>
        )}
        <Button type="submit" size="sm" isLoading={isSubmitting}>
          Simpan &amp; Pantau Pair
        </Button>
      </div>
    </form>
  )
}

export default function StrategiesPage() {
  const { data: strategies, isLoading: loadingStrategies } = useStrategies()
  const { data: subs, isLoading: loadingSubs } = useSubscriptions()
  const remove = useDeleteSubscription()

  const [modalTambah, setModalTambah] = useState(false)
  const [deletingSub, setDeletingSub] = useState<{ id: string; symbol: string } | null>(null)

  const activeCount = (subs ?? []).filter((s) => s.is_active).length
  const canAdd = activeCount < 8

  return (
    <>
      <PageHeader
        title="Pair Pantauan & Strategi"
        description="Pilih metode analisis dan daftarkan instrumen pair yang dipantau engine untuk menghasilkan sinyal di dashboard."
        actions={
          canAdd ? (
            <Button size="sm" onClick={() => setModalTambah(true)}>
              + Tambah Pair Pantauan
            </Button>
          ) : null
        }
      />

      <Section title="Metode Analisis Tersedia" description="Versi engine yang aktif menjalankan pemindaian struktur pasar.">
        {loadingStrategies ? (
          <Skeleton className="h-24 w-full" />
        ) : (
          <div className="grid gap-3 sm:grid-cols-2">
            {strategies?.map((s) => (
              <article key={s.version} className="flex flex-col gap-2 rounded-xl border border-border bg-card p-4">
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
        title="Daftar Pair Pantauan"
        description={`Maksimal 8 simbol aktif (${activeCount}/8). Simbol yang ada di sini otomatis muncul di halaman Sinyal.`}
      >
        {loadingSubs && <Skeleton className="h-16 w-full" />}

        {subs?.length === 0 && (
          <EmptyState
            title="Belum ada pair pantauan"
            description="Tambahkan instrumen pair (misal XAUUSD atau EURUSD) untuk mulai menerima dan memantau sinyal."
            action={
              <Button size="sm" onClick={() => setModalTambah(true)}>
                + Tambah Pair Pantauan
              </Button>
            }
          />
        )}

        {subs && subs.length > 0 && (
          <div className="overflow-hidden rounded-xl border border-border bg-card">
            <ul className="divide-y divide-border">
              {subs.map((sub) => (
                <li
                  key={sub.id}
                  className="flex flex-wrap items-center justify-between gap-3 px-4 py-3 text-[13px] hover:bg-secondary/30 transition-colors"
                >
                  <div className="flex items-center gap-3">
                    <span className="font-bold text-foreground text-sm">{sub.symbol}</span>
                    <Badge variant="outline">{sub.strategy}</Badge>
                    {!sub.is_active && <Badge variant="muted">Nonaktif</Badge>}
                  </div>

                  <Button
                    size="sm"
                    variant="ghost"
                    className="text-xs text-destructive hover:bg-destructive/10"
                    onClick={() => setDeletingSub({ id: sub.id, symbol: sub.symbol })}
                  >
                    Berhenti Pantau
                  </Button>
                </li>
              ))}
            </ul>
          </div>
        )}
      </Section>

      {/* Modal Tambah Pair Pantauan */}
      <Modal
        open={modalTambah}
        onClose={() => setModalTambah(false)}
        size="lg"
        title="Tambah Pair Pantauan Baru"
        description="Pilih simbol instrumen dan metode analisis. Sinyal pair ini akan langsung dipantau oleh engine."
      >
        <FormTambahPair onDone={() => setModalTambah(false)} />
      </Modal>

      {/* Modal Konfirmasi Hapus Pantauan */}
      <Modal
        open={Boolean(deletingSub)}
        onClose={() => setDeletingSub(null)}
        title="Hentikan Pantauan Pair?"
        description={
          deletingSub
            ? `Engine tidak akan lagi memindai setup sinyal untuk pair ${deletingSub.symbol}.`
            : undefined
        }
        footer={
          <div className="flex justify-end gap-2">
            <Button variant="outline" size="sm" onClick={() => setDeletingSub(null)}>
              Batal
            </Button>
            <Button
              variant="destructive"
              size="sm"
              isLoading={remove.isPending}
              onClick={() => {
                if (!deletingSub) return
                remove.mutate(deletingSub.id, {
                  onSuccess: () => setDeletingSub(null),
                })
              }}
            >
              Hentikan Pantauan
            </Button>
          </div>
        }
      >
        <p className="py-2 text-xs text-muted-foreground leading-relaxed">
          Anda dapat menambahkan kembali simbol ini kapan saja selama kuota maksimal 8 pair masih tersedia.
        </p>
      </Modal>
    </>
  )
}
