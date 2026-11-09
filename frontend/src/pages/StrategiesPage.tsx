import { useMemo, useState } from 'react'
import { Bot, Check, Info, Search } from 'lucide-react'
import {
  Alert,
  Badge,
  Button,
  EmptyState,
  Input,
  Modal,
  PageHeader,
  Section,
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
import { cn } from '@/lib/utils'
import type { PairCatalogResponse, StrategyVersion } from '@/types/api'

/** Satu pilihan pair beserta kategori dan grupnya, siap difilter dan dipilih. */
type OpsiPair = {
  symbol: string
  name: string
  description?: string
  aliases?: string
  categoryKey: string
  categoryLabel: string
  groupKey: string
  groupLabel: string
}

/**
 * ratakanKatalog mengubah katalog bersarang (kategori → grup → pair) menjadi
 * satu daftar datar agar mudah dicari dan disaring di modal.
 */
function ratakanKatalog(catalog?: PairCatalogResponse): OpsiPair[] {
  if (!catalog) return []

  const out: OpsiPair[] = []
  for (const category of catalog.categories) {
    for (const group of category.groups) {
      for (const pair of group.pairs) {
        out.push({
          symbol: pair.symbol,
          name: pair.name,
          description: pair.description,
          aliases: pair.aliases,
          categoryKey: category.key,
          categoryLabel: category.label,
          groupKey: group.key,
          groupLabel: group.label,
        })
      }
    }
  }
  return out
}

/**
 * ModalPilihPair adalah pemilih pair: pencarian + saring per kategori dan
 * sub-kategori. Sengaja bukan isian teks bebas supaya simbol yang dipantau
 * selalu berasal dari kamus yang dikelola admin.
 */
function ModalPilihPair({
  onDone,
  onClose,
}: {
  onDone?: () => void
  onClose: () => void
}) {
  const { data: catalog, isLoading: loadingCatalog } = usePairCatalog()
  const { data: subs } = useSubscriptions()
  const create = useCreateSubscription()

  const [cari, setCari] = useState('')
  const [kategori, setKategori] = useState('all')
  const [grup, setGrup] = useState('all')
  const [dipilih, setDipilih] = useState('')
  const [galat, setGalat] = useState<string | null>(null)

  const semua = useMemo(() => ratakanKatalog(catalog), [catalog])

  // Daftar kategori mengikuti katalog aktif; urutannya sudah ditentukan admin.
  const daftarKategori = useMemo(() => {
    const map = new Map<string, string>()
    for (const item of semua) {
      if (!map.has(item.categoryKey)) map.set(item.categoryKey, item.categoryLabel)
    }
    return Array.from(map, ([key, label]) => ({ key, label }))
  }, [semua])

  // Sub-kategori hanya ditampilkan setelah kategori dipilih, agar daftarnya
  // tetap pendek dan tidak membingungkan.
  const daftarGrup = useMemo(() => {
    const map = new Map<string, string>()
    for (const item of semua) {
      if (kategori !== 'all' && item.categoryKey !== kategori) continue
      if (!map.has(item.groupKey)) map.set(item.groupKey, item.groupLabel)
    }
    return Array.from(map, ([key, label]) => ({ key, label }))
  }, [semua, kategori])

  const sudahDipantau = useMemo(
    () => new Set((subs ?? []).map((sub) => sub.symbol.toUpperCase())),
    [subs],
  )

  const hasil = useMemo(() => {
    const kata = cari.trim().toLowerCase()
    return semua.filter((item) => {
      if (kategori !== 'all' && item.categoryKey !== kategori) return false
      if (grup !== 'all' && item.groupKey !== grup) return false
      if (!kata) return true
      return (
        item.symbol.toLowerCase().includes(kata) ||
        item.name.toLowerCase().includes(kata) ||
        (item.aliases ?? '').toLowerCase().includes(kata)
      )
    })
  }, [semua, kategori, grup, cari])

  const ubahKategori = (key: string) => {
    setKategori(key)
    setGrup('all')
  }

  const simpan = async () => {
    if (!dipilih) return
    setGalat(null)
    try {
      // Pair pantauan dipantau oleh kedua metode; strategi ditentukan per akun.
      await create.mutateAsync({ symbol: dipilih })
      if (onDone) onDone()
    } catch (err) {
      setGalat(err instanceof ApiError ? err.message : 'Gagal menambahkan watchlist pair.')
    }
  }

  return (
    <div className="flex flex-col gap-4">
      {galat && <Alert variant="destructive">{galat}</Alert>}

      {/* Pencarian */}
      <div className="relative">
        <Search
          className="pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-muted-foreground"
          aria-hidden="true"
        />
        <Input
          value={cari}
          onChange={(event) => setCari(event.target.value)}
          placeholder="Cari simbol, nama, atau alias (mis. gold, EURUSD)"
          className="pl-9"
          autoFocus
          aria-label="Cari pair"
        />
      </div>

      {/* Saring kategori */}
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="mr-1 text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">
          Kategori
        </span>
        <Button
          type="button"
          size="sm"
          variant={kategori === 'all' ? 'default' : 'outline'}
          className="h-7 text-xs"
          onClick={() => ubahKategori('all')}
        >
          Semua
        </Button>
        {daftarKategori.map((entry) => (
          <Button
            key={entry.key}
            type="button"
            size="sm"
            variant={kategori === entry.key ? 'default' : 'outline'}
            className="h-7 text-xs"
            onClick={() => ubahKategori(entry.key)}
          >
            {entry.label}
          </Button>
        ))}
      </div>

      {/* Saring sub-kategori */}
      {kategori !== 'all' && daftarGrup.length > 0 && (
        <div className="flex flex-wrap items-center gap-1.5">
          <span className="mr-1 text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">
            Sub-kategori
          </span>
          <Button
            type="button"
            size="sm"
            variant={grup === 'all' ? 'default' : 'outline'}
            className="h-7 text-xs"
            onClick={() => setGrup('all')}
          >
            Semua
          </Button>
          {daftarGrup.map((entry) => (
            <Button
              key={entry.key}
              type="button"
              size="sm"
              variant={grup === entry.key ? 'default' : 'outline'}
              className="h-7 text-xs"
              onClick={() => setGrup(entry.key)}
            >
              {entry.label}
            </Button>
          ))}
        </div>
      )}

      {/* Daftar pair */}
      <div className="max-h-[22rem] overflow-y-auto rounded-xl border border-border">
        {loadingCatalog && (
          <div className="space-y-2 p-3">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        )}

        {!loadingCatalog && hasil.length === 0 && (
          <div className="px-4 py-10 text-center">
            <p className="text-sm font-medium text-foreground">Tidak ada pair yang cocok</p>
            <p className="mt-1 text-[12px] text-muted-foreground">
              Coba kata kunci lain atau reset penyaringan.
            </p>
            <Button
              type="button"
              size="sm"
              variant="outline"
              className="mt-3 text-xs"
              onClick={() => {
                setCari('')
                setKategori('all')
                setGrup('all')
              }}
            >
              Reset Penyaringan
            </Button>
          </div>
        )}

        {!loadingCatalog && hasil.length > 0 && (
          <ul className="divide-y divide-border">
            {hasil.map((item) => {
              const terpakai = sudahDipantau.has(item.symbol)
              const aktif = dipilih === item.symbol

              return (
                <li key={item.symbol}>
                  <button
                    type="button"
                    disabled={terpakai}
                    aria-pressed={aktif}
                    onClick={() => setDipilih(item.symbol)}
                    className={cn(
                      'flex w-full items-center gap-3 px-3.5 py-2.5 text-left transition-colors',
                      terpakai
                        ? 'cursor-not-allowed opacity-55'
                        : 'hover:bg-secondary/40 focus-visible:bg-secondary/40 focus-visible:outline-none',
                      aktif && !terpakai && 'bg-secondary/60',
                    )}
                  >
                    <span
                      className={cn(
                        'flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border text-[10px] font-bold',
                        aktif && !terpakai
                          ? 'border-foreground bg-foreground text-background'
                          : 'border-border bg-secondary/60 text-muted-foreground',
                      )}
                      aria-hidden="true"
                    >
                      {item.symbol.slice(0, 3)}
                    </span>

                    <span className="min-w-0 flex-1">
                      <span className="flex flex-wrap items-center gap-1.5">
                        <span className="text-[13px] font-bold text-foreground">{item.symbol}</span>
                        {item.name && (
                          <span className="truncate text-[12px] text-muted-foreground">
                            {item.name}
                          </span>
                        )}
                        {terpakai && (
                          <Badge variant="muted" className="text-[9.5px]">
                            Sudah dipantau
                          </Badge>
                        )}
                      </span>
                      <span className="mt-0.5 block text-[10.5px] text-muted-foreground">
                        {item.categoryLabel} · {item.groupLabel}
                        {item.aliases ? ` · ${item.aliases}` : ''}
                      </span>
                    </span>

                    {aktif && !terpakai && (
                      <Check className="h-4 w-4 shrink-0 text-foreground" aria-hidden="true" />
                    )}
                  </button>
                </li>
              )
            })}
          </ul>
        )}
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-border pt-3">
        <span className="text-[11.5px] text-muted-foreground">
          {loadingCatalog ? 'Memuat katalog…' : `${hasil.length} dari ${semua.length} pair`}
        </span>
        <div className="flex items-center gap-2">
          <Button type="button" variant="outline" size="sm" onClick={onClose}>
            Batal
          </Button>
          <Button
            type="button"
            size="sm"
            disabled={!dipilih || create.isPending}
            onClick={() => void simpan()}
          >
            {create.isPending ? 'Menambahkan…' : dipilih ? `Tambah ${dipilih}` : 'Tambah Pair'}
          </Button>
        </div>
      </div>
    </div>
  )
}

/** Ringkasan publik sebuah metode analisis. Tanpa detail cara kerja internal. */
const METODE_INFO: Record<StrategyVersion, {
  judul: string
  tagline: string
  poin: string[]
  cocok: string
}> = {
  'ila-2.0': {
    judul: 'ILA 2.0',
    tagline: 'Fokus eksternal & internal',
    poin: [
      'Membaca struktur harga eksternal pada chart.',
      'Mengonfirmasi dorongan momentum internal pasar.',
      'Sinyal terbit saat struktur eksternal dan konfirmasi internal selaras.',
    ],
    cocok: 'Pasar trending · Timeframe M15–H1 · Pemula sampai menengah',
  },
  'ila-3.0': {
    judul: 'ILA 3.0',
    tagline: 'Multi-konsep lanjutan',
    poin: [
      'Menggabungkan multi-frame confluence dalam satu penilaian.',
      'Mengintegrasikan kerangka Quotational Thinking (QT) untuk validasi setup.',
      'Filter ketat dengan akurasi terarah untuk setup berkualitas tinggi.',
    ],
    cocok: 'Trader berpengalaman · Timeframe H1–H4 · Tersedia untuk tier Plus',
  },
}

export default function StrategiesPage() {
  const { data: strategies, isLoading: loadingStrategies } = useStrategies()
  const { data: subs, isLoading: loadingSubs } = useSubscriptions()
  const { data: catalog } = usePairCatalog()
  const remove = useDeleteSubscription()

  const [modalTambah, setModalTambah] = useState(false)
  const [infoMetode, setInfoMetode] = useState<StrategyVersion | null>(null)
  const [deletingSub, setDeletingSub] = useState<{ id: string; symbol: string } | null>(null)

  const activeCount = (subs ?? []).filter((s) => s.is_active).length
  const canAdd = activeCount < 8

  // Metode yang boleh ditampilkan ke tier ini. ILA 3.0 disembunyikan total
  // bagi yang belum berhak agar tidak membocorkan fitur yang belum dibeli.
  const metodeTampil = useMemo(
    () => (strategies ?? []).filter((s) => s.allowed),
    [strategies],
  )

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
  const adaIla3 = metodeTampil.some((s) => s.version === 'ila-3.0')

  return (
    <>
      <PageHeader
        title="Watchlist Pair"
        description="Daftar instrumen yang dipantau sistem. Setiap pair dipantau oleh kedua metode analisis; pengaturan eksekusi ditentukan per akun trading."
        actions={
          canAdd ? (
            <Button size="sm" onClick={() => setModalTambah(true)}>
              + Tambah Watchlist Pair
            </Button>
          ) : null
        }
      />

      <Section
        title="Metode Analisis"
        description="Ringkasan metode yang tersedia. Pemilihan metode dilakukan di halaman detail masing-masing akun trading."
      >
        {loadingStrategies ? (
          <Skeleton className="h-24 w-full" />
        ) : metodeTampil.length === 0 ? (
          <EmptyState
            title="Belum ada metode analisis"
            description="Metode analisis akan tampil di sini setelah tersedia untuk akun Anda."
          />
        ) : (
          <div className="grid gap-3 sm:grid-cols-2">
            {metodeTampil.map((s) => {
              const info = METODE_INFO[s.version]
              return (
                <button
                  key={s.version}
                  type="button"
                  onClick={() => setInfoMetode(s.version)}
                  aria-haspopup="dialog"
                  className="flex flex-col gap-2 rounded-xl border border-border bg-card p-4 text-left transition-all hover:border-zinc-400 hover:shadow-sm"
                >
                  <div className="flex items-center justify-between gap-2">
                    <h3 className="text-[14px] font-semibold">{info?.judul ?? s.name}</h3>
                    <span className="inline-flex items-center gap-1 text-[11px] font-medium text-muted-foreground">
                      <Info className="h-3.5 w-3.5" aria-hidden="true" />
                      Ringkasan
                    </span>
                  </div>
                  <p className="text-[12px] font-medium text-foreground">{info?.tagline}</p>
                  <p className="text-[11.5px] leading-relaxed text-muted-foreground">
                    {info?.poin[0]}
                  </p>
                </button>
              )
            })}
          </div>
        )}
      </Section>

      <Section
        title="Daftar Watchlist Pair"
        description={`Maksimal 8 simbol aktif (${activeCount}/8). Simbol yang ada di sini otomatis muncul di halaman Signals.`}
      >
        {loadingSubs && <Skeleton className="h-16 w-full" />}

        {subs?.length === 0 && (
          <EmptyState
            title="Watchlist pair masih kosong"
            description="Pilih instrumen dari katalog pair resmi untuk mulai menerima dan memantau sinyal."
            action={
              <Button size="sm" onClick={() => setModalTambah(true)}>
                + Tambah Watchlist Pair
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
                        <span className="text-[11px] text-muted-foreground">
                          {nama ?? 'Nama belum tersedia'}
                        </span>
                      </div>
                      <Badge variant="outline">ILA 2.0</Badge>
                      {adaIla3 && <Badge variant="outline">ILA 3.0</Badge>}
                      {!sub.is_active && <Badge variant="muted">Nonaktif</Badge>}
                      {diLuarKatalog && <Badge variant="muted">Di luar katalog</Badge>}
                    </div>

                    <Button
                      size="sm"
                      variant="ghost"
                      className="text-xs text-destructive hover:bg-destructive/10"
                      onClick={() => setDeletingSub({ id: sub.id, symbol: sub.symbol })}
                    >
                      Hapus
                    </Button>
                  </li>
                )
              })}
            </ul>
          </div>
        )}
      </Section>

      {/* Modal Tambah Watchlist Pair */}
      <Modal
        open={modalTambah}
        onClose={() => setModalTambah(false)}
        size="xl"
        title="Tambah Watchlist Pair"
        description="Cari dan pilih instrumen dari katalog resmi. Pair yang ditambahkan akan dipantau oleh kedua metode analisis."
      >
        <ModalPilihPair
          onClose={() => setModalTambah(false)}
          onDone={() => setModalTambah(false)}
        />
      </Modal>

      {/* Modal Ringkasan Metode */}
      <Modal
        open={Boolean(infoMetode)}
        onClose={() => setInfoMetode(null)}
        size="lg"
        title={infoMetode ? METODE_INFO[infoMetode].judul : undefined}
        description={infoMetode ? METODE_INFO[infoMetode].tagline : undefined}
      >
        {infoMetode && (
          <div className="space-y-4">
            <ul className="space-y-2.5">
              {METODE_INFO[infoMetode].poin.map((poin) => (
                <li key={poin} className="flex gap-2.5 text-[13px] leading-relaxed text-foreground">
                  <span
                    className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-foreground"
                    aria-hidden="true"
                  />
                  {poin}
                </li>
              ))}
            </ul>

            <div className="rounded-lg border border-border bg-secondary/40 px-3.5 py-2.5">
              <p className="text-[10px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">
                Cocok untuk
              </p>
              <p className="mt-1 text-[12.5px] leading-relaxed text-foreground">
                {METODE_INFO[infoMetode].cocok}
              </p>
            </div>

            <div className="flex items-start gap-2 text-[11.5px] leading-relaxed text-muted-foreground">
              <Bot className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden="true" />
              <span>Detail cara kerja internal tidak dipublikasikan.</span>
            </div>

            <div className="flex justify-end border-t border-border pt-3">
              <Button size="sm" onClick={() => setInfoMetode(null)}>
                Mengerti
              </Button>
            </div>
          </div>
        )}
      </Modal>

      {/* Modal Konfirmasi Hapus Pantauan */}
      <Modal
        open={Boolean(deletingSub)}
        onClose={() => setDeletingSub(null)}
        title="Hapus dari Watchlist?"
        description={
          deletingSub
            ? `Sistem tidak lagi memindai setup sinyal untuk pair ${deletingSub.symbol}.`
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
