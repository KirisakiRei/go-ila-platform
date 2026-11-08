import { useMemo, useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import {
  Alert,
  Badge,
  Button,
  EmptyState,
  Label,
  Modal,
  PageHeader,
  Section,
  Select,
  Skeleton,
} from '@/components/ui'
import {
  useCreateSubscription,
  useDeleteSubscription,
  usePairCatalog,
  useStrategies,
  useSubscriptions,
} from '@/lib/queries'
import { ApiError } from '@/lib/api-error'
import type { PairCatalogResponse, StrategyVersion } from '@/types/api'

const skema = z.object({
  symbol: z.string().min(3, 'Pilih pair dari katalog'),
  strategy: z.enum(['ila-2.0', 'ila-3.0']),
})

type NilaiForm = z.input<typeof skema>

/** Satu pilihan pair beserta label kategorinya, siap ditampilkan di dropdown. */
type PairOption = {
  symbol: string
  name: string
  description?: string
}

/**
 * ratakanKatalog mengubah katalog bersarang (kategori → grup → pair) menjadi
 * kelompok datar untuk <optgroup>, karena HTML tidak mendukung optgroup
 * bertingkat.
 */
function ratakanKatalog(catalog?: PairCatalogResponse): { label: string; pairs: PairOption[] }[] {
  if (!catalog) return []

  const out: { label: string; pairs: PairOption[] }[] = []
  for (const category of catalog.categories) {
    for (const group of category.groups) {
      if (group.pairs.length === 0) continue
      out.push({
        label: `${category.label} — ${group.label}`,
        pairs: group.pairs.map((pair) => ({
          symbol: pair.symbol,
          name: pair.name,
          description: pair.description,
        })),
      })
    }
  }
  return out
}

function FormTambahPair({ onDone }: { onDone?: () => void }) {
  const [galat, setGalat] = useState<string | null>(null)
  const { data: strategies } = useStrategies()
  const { data: catalog, isLoading: loadingCatalog } = usePairCatalog()
  const create = useCreateSubscription()

  const kelompok = useMemo(() => ratakanKatalog(catalog), [catalog])

  const {
    register,
    control,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<NilaiForm>({
    resolver: zodResolver(skema),
    defaultValues: { symbol: '', strategy: 'ila-2.0' },
  })

  const simbolTerpilih = useWatch({ control, name: 'symbol' })
  const deskripsiTerpilih = useMemo(() => {
    for (const grup of kelompok) {
      const found = grup.pairs.find((pair) => pair.symbol === simbolTerpilih)
      if (found) return found.description
    }
    return undefined
  }, [kelompok, simbolTerpilih])

  // Hanya strategi yang sudah dirilis dan boleh untuk tier ini yang muncul.
  const tersedia = (strategies ?? []).filter((s) => s.available && s.allowed)

  const submit = handleSubmit(async (nilai) => {
    setGalat(null)
    try {
      // Pair pantauan bersifat global: tidak terikat akun broker mana pun.
      await create.mutateAsync({
        symbol: nilai.symbol,
        strategy: nilai.strategy as StrategyVersion,
      })
      reset({ symbol: '', strategy: nilai.strategy })
      if (onDone) onDone()
    } catch (err) {
      setGalat(err instanceof ApiError ? err.message : 'Gagal menambahkan pair pantauan.')
    }
  })

  return (
    <form onSubmit={submit} className="flex flex-col gap-4 py-1" noValidate>
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="symbol">Pair dari Katalog</Label>
          <Select id="symbol" {...register('symbol')} disabled={loadingCatalog}>
            <option value="">{loadingCatalog ? 'Memuat katalog…' : 'Pilih pair'}</option>
            {kelompok.map((grup) => (
              <optgroup key={grup.label} label={grup.label}>
                {grup.pairs.map((pair) => (
                  <option key={pair.symbol} value={pair.symbol}>
                    {pair.name ? `${pair.symbol} — ${pair.name}` : pair.symbol}
                  </option>
                ))}
              </optgroup>
            ))}
          </Select>
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
      </div>

      {!loadingCatalog && kelompok.length === 0 && (
        <Alert variant="destructive">
          Katalog pair belum tersedia. Hubungi dukungan agar admin mengaktifkan daftar pair terlebih
          dahulu.
        </Alert>
      )}

      {deskripsiTerpilih && (
        <p className="text-[12px] leading-relaxed text-muted-foreground">{deskripsiTerpilih}</p>
      )}

      <p className="text-[12px] leading-relaxed text-muted-foreground">
        Pair diambil dari katalog resmi yang dikelola admin, sehingga simbol yang dipantau selalu
        konsisten. Sinyal pair ini akan tampil di halaman Sinyal.
      </p>

      {galat && <Alert variant="destructive">{galat}</Alert>}

      <div className="flex justify-end gap-2 pt-2">
        {onDone && (
          <Button type="button" variant="outline" size="sm" onClick={onDone}>
            Batal
          </Button>
        )}
        <Button type="submit" size="sm" isLoading={isSubmitting} disabled={kelompok.length === 0}>
          Simpan &amp; Pantau Pair
        </Button>
      </div>
    </form>
  )
}

export default function StrategiesPage() {
  const { data: strategies, isLoading: loadingStrategies } = useStrategies()
  const { data: subs, isLoading: loadingSubs } = useSubscriptions()
  const { data: catalog } = usePairCatalog()
  const remove = useDeleteSubscription()

  const [modalTambah, setModalTambah] = useState(false)
  const [deletingSub, setDeletingSub] = useState<{ id: string; symbol: string } | null>(null)

  const activeCount = (subs ?? []).filter((s) => s.is_active).length
  const canAdd = activeCount < 8

  // Peta simbol → nama tampilan, dipakai agar daftar pantauan terbaca ramah.
  const namaBySymbol = useMemo(() => {
    const map = new Map<string, string>()
    for (const kategori of catalog?.categories ?? []) {
      for (const grup of kategori.groups) {
        for (const pair of grup.pairs) {
          map.set(pair.symbol, pair.name)
        }
      }
    }
    return map
  }, [catalog])

  const simbolKatalog = useMemo(() => new Set(catalog?.symbols ?? []), [catalog])

  return (
    <>
      <PageHeader
        title="Pair Pantauan & Strategi"
        description="Pilih metode analisis lalu pilih instrumen dari katalog pair resmi. Sinyal pair ini dipantau engine untuk seluruh akun Anda."
        actions={
          canAdd ? (
            <Button size="sm" onClick={() => setModalTambah(true)}>
              + Tambah Pair Pantauan
            </Button>
          ) : null
        }
      />

      <Section
        title="Metode Analisis Tersedia"
        description="Versi engine yang aktif menjalankan pemindaian struktur pasar."
      >
        {loadingStrategies ? (
          <Skeleton className="h-24 w-full" />
        ) : (
          <div className="grid gap-3 sm:grid-cols-2">
            {strategies?.map((s) => (
              <article
                key={s.version}
                className="flex flex-col gap-2 rounded-xl border border-border bg-card p-4"
              >
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
            description="Pilih instrumen dari katalog pair resmi untuk mulai menerima dan memantau sinyal."
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
              {subs.map((sub) => {
                const nama = namaBySymbol.get(sub.symbol)
                const diLuarKatalog = catalog ? !simbolKatalog.has(sub.symbol) : false
                return (
                  <li
                    key={sub.id}
                    className="flex flex-wrap items-center justify-between gap-3 px-4 py-3 text-[13px] hover:bg-secondary/30 transition-colors"
                  >
                    <div className="flex flex-wrap items-center gap-3">
                      <div className="flex flex-col">
                        <span className="font-bold text-foreground text-sm">{sub.symbol}</span>
                        {nama && <span className="text-[11px] text-muted-foreground">{nama}</span>}
                      </div>
                      <Badge variant="outline">{sub.strategy}</Badge>
                      {!sub.is_active && <Badge variant="muted">Nonaktif</Badge>}
                      {diLuarKatalog && <Badge variant="muted">Di luar katalog</Badge>}
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
                )
              })}
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
        description="Pilih pair dari katalog resmi dan metode analisis. Sinyal pair ini akan langsung dipantau oleh engine."
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
          Anda dapat menambahkan kembali simbol ini kapan saja selama kuota maksimal 8 pair masih
          tersedia.
        </p>
      </Modal>
    </>
  )
}
