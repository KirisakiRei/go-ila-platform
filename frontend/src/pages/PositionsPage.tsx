import { Fragment, useMemo, useState } from 'react'
import { ChevronDown, ChevronRight, Download, LineChart, SlidersHorizontal } from 'lucide-react'
import {
  Alert,
  Badge,
  Button,
  Card,
  EmptyState,
  Input,
  Label,
  MobileSegmentedTabs,
  PageHeader,
  Pagination,
  Select,
  Skeleton,
  StatusPill,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  usePaginasiLokal,
} from '@/components/ui'
import { useJournal } from '@/lib/queries'
import { downloadFile } from '@/lib/http'
import { formatDateTime, formatMoney, formatNumber, formatPrice, formatR } from '@/lib/format'
import { priceDigits } from '@/lib/pricePrecision'
import { useIsMobile, useIsTabletOrMobile } from '@/lib/use-mobile-viewport'
import { cn } from '@/lib/utils'
import type { JournalEvalStatus, JournalResponse, Position, PositionStatus } from '@/types/api'
import { PositionExecutionChart } from '@/features/charts/PositionExecutionChart'
import { chartTheme } from '@/features/charts/lib/theme'
import { JournalSummary } from '@/features/journal/components/JournalSummary'
import { JournalSkippedList } from '@/features/journal/components/JournalSkippedList'
import { SimulasiBadge } from '@/features/journal/components/SimulasiBadge'

const LABEL: Record<PositionStatus, string> = {
  OPEN: 'Open',
  CLOSED_TP: 'Take Profit',
  CLOSED_SL: 'Stop Loss',
  CLOSED_MANUAL: 'Closed Manual',
  UNTRACKED: 'Untracked',
}

const TONE: Record<PositionStatus, 'solid' | 'outline' | 'neutral' | 'muted'> = {
  OPEN: 'solid',
  CLOSED_TP: 'outline',
  CLOSED_SL: 'outline',
  CLOSED_MANUAL: 'muted',
  UNTRACKED: 'muted',
}

/** Tab memisahkan posisi berjalan, riwayat, dan peluang yang terlewat. */
type Tab = 'aktif' | 'riwayat' | 'terlewat'

const TABS: { id: Tab; label: string; keterangan: string }[] = [
  { id: 'aktif', label: 'Posisi Aktif', keterangan: 'Order yang masih hidup di broker.' },
  { id: 'riwayat', label: 'Riwayat', keterangan: 'Posisi yang sudah ditutup beserta hasilnya.' },
  {
    id: 'terlewat',
    label: 'Tidak Tereksekusi',
    keterangan: 'Simulasi hasil sinyal yang tidak menjadi transaksi akun. Bukan PnL akun.',
  },
]

/** Periode jurnal yang bisa dipilih pengguna. */
const PERIODE = [
  { hari: 7, label: '7 hari terakhir' },
  { hari: 30, label: '30 hari terakhir' },
  { hari: 90, label: '90 hari terakhir' },
  { hari: 365, label: '1 tahun terakhir' },
]

/**
 * PositionsPage adalah Jurnal Eksekusi.
 *
 * Halaman ini menjawab tiga pertanyaan yang berbeda, karena itu dipecah per
 * tab alih-alih dicampur dalam satu daftar:
 *  1. Apa yang sedang berjalan?          -> Posisi Aktif
 *  2. Bagaimana hasil yang sudah selesai? -> Riwayat
 *  3. Peluang mana yang terlewat?         -> Tidak Tereksekusi
 *
 * Ringkasan performa selalu terlihat di atas agar pengguna membaca angka
 * sebelum detailnya. Angka ringkasan hanya memakai posisi yang sudah ditutup;
 * posisi berjalan dipisahkan supaya winrate tidak berubah tanpa sebab.
 *
 * Setiap posisi dapat dibuka sebagai accordion berisi chart eksekusi: titik
 * entry, level SL/TP broker, dan garis penghubung ke titik exit.
 */
export default function PositionsPage() {
  const [tab, setTab] = useState<Tab>('aktif')
  const [hari, setHari] = useState(30)
  const [strategi, setStrategi] = useState('')
  const [symbol, setSymbol] = useState('')
  const [evalStatus, setEvalStatus] = useState<JournalEvalStatus | ''>('')
  // Rentang tanggal khusus. Dideklarasikan sebelum useJournal karena dipakai
  // sebagai bagian kunci permintaan.
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  /** Halaman dan ukuran daftar peluang terlewat; ikut dikirim ke server. */
  const [peluang, setPeluang] = useState({ page: 1, limit: 20 })

  const { data, isLoading, isError, error, isFetching } = useJournal({
    // Rentang tanggal khusus menang atas periode hari: bila pengguna memilih
    // tanggal, itulah yang diminta. Mengirim keduanya membuat server harus
    // memilih, dan hasilnya tidak lagi jelas bagi pengguna.
    ...(from || to ? { from, to } : { days: hari }),
    strategy: strategi,
    symbol: symbol,
    skippedPage: peluang.page,
    skippedLimit: peluang.limit,
    evalStatus,
  })

  const [downloading, setDownloading] = useState(false)
  const [downloadNotice, setDownloadNotice] = useState<string | null>(null)
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [mobileFilterOpen, setMobileFilterOpen] = useState(false)
  const isMobile = useIsMobile()
  const isTabletOrMobile = useIsTabletOrMobile()
  const theme = chartTheme()

  // positions di-memo agar identitasnya stabil: tanpa itu, setiap render
  // menghasilkan array baru dan seluruh useMemo di bawahnya dihitung ulang.
  const positions = useMemo(() => data?.positions ?? [], [data?.positions])
  const skipped = useMemo(() => data?.skipped ?? [], [data?.skipped])
  const skipped_meta = data?.skipped_meta
  const aktif = useMemo(() => positions.filter((pos) => pos.status === 'OPEN'), [positions])
  const riwayat = useMemo(() => positions.filter((pos) => pos.status !== 'OPEN'), [positions])

  /** daftarSimbol menyusun pilihan pair dari posisi dan peluang yang ada. */
  const daftarSimbol = useMemo(() => {
    const unik = new Set<string>()
    for (const pos of positions) unik.add(pos.symbol)
    for (const item of skipped) if (item.symbol) unik.add(item.symbol)
    return [...unik].sort()
  }, [positions, skipped])

  const download = async () => {
    setDownloading(true)
    setDownloadNotice(null)
    try {
      // Ekspor mengikuti filter yang sedang terlihat, supaya berkas yang diunduh
      // sama dengan yang ada di layar. Bila pengguna memakai periode hari,
      // rentang tanggalnya diturunkan dari periode itu.
      const rentang = rentangEkspor()
      const name = await downloadFile('/positions/export', rentang)
      setDownloadNotice(`Berkas ${name} disiapkan.`)
    } catch (cause) {
      setDownloadNotice(cause instanceof Error ? cause.message : 'Gagal mengunduh riwayat.')
    } finally {
      setDownloading(false)
    }
  }

  /**
   * rentangEkspor menentukan tanggal yang dikirim ke endpoint ekspor.
   * Tanggal yang dipilih pengguna diutamakan; bila kosong, periode hari
   * diterjemahkan menjadi tanggal agar isi CSV sama dengan yang tampil.
   */
  const rentangEkspor = () => {
    if (from || to) return { from, to }
    const akhir = new Date()
    const awal = new Date(akhir.getTime() - hari * 24 * 60 * 60 * 1000)
    const keTanggal = (d: Date) => d.toISOString().slice(0, 10)
    return { from: keTanggal(awal), to: keTanggal(akhir) }
  }

  /** jumlahFilter menghitung filter aktif selain periode bawaan. */
  const filterAktif = [
    strategi !== '',
    symbol !== '',
    from !== '',
    to !== '',
    evalStatus !== '',
  ].filter(Boolean).length

  /** resetFilter mengembalikan filter ke keadaan awal tanpa menyentuh tab. */
  const resetFilter = () => {
    setStrategi('')
    setSymbol('')
    setEvalStatus('')
    setFrom('')
    setTo('')
    setHari(30)
    setPeluang({ page: 1, limit: peluang.limit })
  }

  const toggle = (id: string) => setExpandedId((current) => (current === id ? null : id))

  const pnlColor = (position: Position) => (position.profit_loss >= 0 ? theme.profit : theme.loss)

  const tampil = tab === 'aktif' ? aktif : riwayat
  const paginasiMobile = usePaginasiLokal(tampil, 10)
  const kosong = data && positions.length === 0

  return (
    <>
      <PageHeader
        title="Jurnal Eksekusi"
        description="Rekam jejak trading dari sinyal ILA: posisi berjalan, hasil yang sudah ditutup, dan peluang yang terlewat."
      />

      {/* --- Ringkasan performa ------------------------------------------- */}
      {data ? (
        <JournalSummary stats={data.stats} />
      ) : isLoading ? (
        <Skeleton className="h-24 w-full" />
      ) : null}

      {/* --- Filter: periode, tanggal, metode, pair, dan ekspor -------------
          Semua kontrol penyaring dikumpulkan di satu tempat. Sebelumnya
          tanggal hanya ada di bagian bawah sebagai pilihan ekspor, sehingga
          pengguna mengira daftar sudah tersaring padahal hanya berkas CSV
          yang berubah. */}
      <Card className="p-3">
        {isMobile ? (
          <div className="space-y-2.5">
            <div className="flex items-end gap-2">
              <div className="min-w-0 flex-1">
                <Label htmlFor="jurnal-periode" className="text-[11px]">
                  Periode
                </Label>
                <Select
                  id="jurnal-periode"
                  className="mt-1 h-9 text-xs"
                  value={from || to ? 'custom' : String(hari)}
                  onChange={(event) => {
                    const nilai = event.target.value
                    if (nilai === 'custom') return
                    setFrom('')
                    setTo('')
                    setHari(Number(nilai))
                    setPeluang({ page: 1, limit: peluang.limit })
                    paginasiMobile.setPage(1)
                  }}
                >
                  {PERIODE.map((item) => (
                    <option key={item.hari} value={item.hari}>
                      {item.label}
                    </option>
                  ))}
                  {(from || to) && <option value="custom">Rentang tanggal</option>}
                </Select>
              </div>

              <Button
                type="button"
                variant={mobileFilterOpen || filterAktif > 0 ? 'default' : 'outline'}
                size="sm"
                className="h-9 shrink-0 gap-1.5 px-2.5 text-xs"
                onClick={() => setMobileFilterOpen((v) => !v)}
              >
                <SlidersHorizontal className="h-3.5 w-3.5" />
                <span>Filter{filterAktif > 0 ? ` (${filterAktif})` : ''}</span>
              </Button>

              <Button
                type="button"
                variant="outline"
                size="sm"
                className="h-9 shrink-0 gap-1 px-2.5 text-xs"
                disabled={downloading}
                onClick={() => void download()}
                aria-label="Export CSV"
              >
                <Download className="h-3.5 w-3.5" />
                <span>{downloading ? '…' : 'CSV'}</span>
              </Button>
            </div>

            {mobileFilterOpen && (
              <div className="space-y-2.5 border-t border-border pt-2.5">
                <div className="grid grid-cols-2 gap-2">
                  <div className="flex flex-col gap-1">
                    <Label htmlFor="jurnal-metode" className="text-[11px]">
                      Metode
                    </Label>
                    <Select
                      id="jurnal-metode"
                      className="h-9 text-xs"
                      value={strategi}
                      onChange={(event) => {
                        setStrategi(event.target.value)
                        setPeluang({ page: 1, limit: peluang.limit })
                        paginasiMobile.setPage(1)
                      }}
                    >
                      <option value="">Semua metode</option>
                      <option value="ila-2.0">ILA 2.0</option>
                      <option value="ila-3.0">ILA 3.0</option>
                    </Select>
                  </div>
                  <div className="flex flex-col gap-1">
                    <Label htmlFor="jurnal-pair" className="text-[11px]">
                      Pair
                    </Label>
                    <Select
                      id="jurnal-pair"
                      className="h-9 text-xs"
                      value={symbol}
                      onChange={(event) => {
                        setSymbol(event.target.value)
                        setPeluang({ page: 1, limit: peluang.limit })
                        paginasiMobile.setPage(1)
                      }}
                    >
                      <option value="">Semua pair</option>
                      {daftarSimbol.map((item) => (
                        <option key={item} value={item}>
                          {item}
                        </option>
                      ))}
                    </Select>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-2">
                  <div className="flex flex-col gap-1">
                    <Label htmlFor="jurnal-dari" className="text-[11px]">
                      Dari tanggal
                    </Label>
                    <Input
                      id="jurnal-dari"
                      type="date"
                      className="h-9 text-xs"
                      value={from}
                      max={to || undefined}
                      onChange={(event) => {
                        setFrom(event.target.value)
                        setPeluang({ page: 1, limit: peluang.limit })
                        paginasiMobile.setPage(1)
                      }}
                    />
                  </div>
                  <div className="flex flex-col gap-1">
                    <Label htmlFor="jurnal-sampai" className="text-[11px]">
                      Sampai tanggal
                    </Label>
                    <Input
                      id="jurnal-sampai"
                      type="date"
                      className="h-9 text-xs"
                      value={to}
                      min={from || undefined}
                      onChange={(event) => {
                        setTo(event.target.value)
                        setPeluang({ page: 1, limit: peluang.limit })
                        paginasiMobile.setPage(1)
                      }}
                    />
                  </div>
                </div>

                {filterAktif > 0 && (
                  <Button
                    variant="outline"
                    size="sm"
                    className="w-full text-xs"
                    onClick={() => {
                      resetFilter()
                      paginasiMobile.setPage(1)
                    }}
                  >
                    Reset filter ({filterAktif})
                  </Button>
                )}
              </div>
            )}

            {downloadNotice && <Alert className="mt-1">{downloadNotice}</Alert>}
          </div>
        ) : (
          <>
            <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
              <div className="flex flex-col gap-1">
                <Label htmlFor="jurnal-periode">Periode</Label>
                <Select
                  id="jurnal-periode"
                  value={from || to ? 'custom' : String(hari)}
                  onChange={(event) => {
                    const nilai = event.target.value
                    if (nilai === 'custom') return
                    setFrom('')
                    setTo('')
                    setHari(Number(nilai))
                    setPeluang({ page: 1, limit: peluang.limit })
                  }}
                >
                  {PERIODE.map((item) => (
                    <option key={item.hari} value={item.hari}>
                      {item.label}
                    </option>
                  ))}
                  {(from || to) && <option value="custom">Rentang tanggal</option>}
                </Select>
              </div>
              <div className="flex flex-col gap-1">
                <Label htmlFor="jurnal-metode">Metode</Label>
                <Select
                  id="jurnal-metode"
                  value={strategi}
                  onChange={(event) => {
                    setStrategi(event.target.value)
                    setPeluang({ page: 1, limit: peluang.limit })
                  }}
                >
                  <option value="">Semua metode</option>
                  <option value="ila-2.0">ILA 2.0</option>
                  <option value="ila-3.0">ILA 3.0</option>
                </Select>
              </div>
              <div className="flex flex-col gap-1">
                <Label htmlFor="jurnal-pair">Pair</Label>
                <Select
                  id="jurnal-pair"
                  value={symbol}
                  onChange={(event) => {
                    setSymbol(event.target.value)
                    setPeluang({ page: 1, limit: peluang.limit })
                  }}
                >
                  <option value="">Semua pair</option>
                  {daftarSimbol.map((item) => (
                    <option key={item} value={item}>
                      {item}
                    </option>
                  ))}
                </Select>
              </div>
              <div className="flex flex-col gap-1">
                <Label htmlFor="jurnal-dari">Dari tanggal</Label>
                <Input
                  id="jurnal-dari"
                  type="date"
                  value={from}
                  max={to || undefined}
                  onChange={(event) => {
                    setFrom(event.target.value)
                    setPeluang({ page: 1, limit: peluang.limit })
                  }}
                />
              </div>
            </div>

            <div className="mt-2 grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
              <div className="flex flex-col gap-1">
                <Label htmlFor="jurnal-sampai">Sampai tanggal</Label>
                <Input
                  id="jurnal-sampai"
                  type="date"
                  value={to}
                  min={from || undefined}
                  onChange={(event) => {
                    setTo(event.target.value)
                    setPeluang({ page: 1, limit: peluang.limit })
                  }}
                />
              </div>
              <div className="flex items-end gap-2 lg:col-span-2">
                <Button variant="outline" size="sm" disabled={filterAktif === 0} onClick={resetFilter}>
                  Reset filter{filterAktif > 0 ? ` (${filterAktif})` : ''}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={downloading}
                  onClick={() => void download()}
                >
                  {downloading ? 'Menyiapkan…' : 'Export CSV'}
                </Button>
              </div>
            </div>

            <p className="mt-2 text-[11px] text-muted-foreground">
              {from || to
                ? 'Daftar dan ekspor mengikuti rentang tanggal yang dipilih.'
                : 'Daftar dan ekspor mengikuti periode di atas. Isi tanggal untuk memilih rentang sendiri.'}
            </p>
            {downloadNotice && <Alert className="mt-2">{downloadNotice}</Alert>}
          </>
        )}
      </Card>

      {/* --- Tab ----------------------------------------------------------- */}
      {isMobile ? (
        <MobileSegmentedTabs<Tab>
          ariaLabel="Kategori jurnal eksekusi"
          value={tab}
          onChange={(next) => {
            setTab(next)
            paginasiMobile.setPage(1)
          }}
          options={[
            { id: 'aktif', label: 'Aktif', count: aktif.length },
            { id: 'riwayat', label: 'Riwayat', count: riwayat.length },
            { id: 'terlewat', label: 'Terlewat', count: skipped_meta?.total ?? skipped.length },
          ]}
        />
      ) : (
        <div className="flex flex-wrap gap-1 border-b border-border">
          {TABS.map((item) => {
            const jumlah =
              item.id === 'aktif'
                ? aktif.length
                : item.id === 'riwayat'
                  ? riwayat.length
                  : (skipped_meta?.total ?? skipped.length)

            return (
              <button
                key={item.id}
                type="button"
                onClick={() => setTab(item.id)}
                aria-selected={tab === item.id}
                role="tab"
                className={cn(
                  '-mb-px border-b-2 px-3 py-2 text-[12.5px] font-medium transition-colors',
                  tab === item.id
                    ? 'border-foreground text-foreground'
                    : 'border-transparent text-muted-foreground hover:text-foreground',
                )}
              >
                {item.label}
                <span className="ml-1.5 tabular text-[11px] text-muted-foreground">{jumlah}</span>
              </button>
            )
          })}
        </div>
      )}

      <p className="text-[11px] text-muted-foreground">
        {TABS.find((item) => item.id === tab)?.keterangan}
      </p>

      {isLoading && <Skeleton className="h-56 w-full" />}
      {isError && <Alert variant="destructive">{(error as Error).message}</Alert>}

      {/* --- Tab: peluang terlewat ----------------------------------------- */}
      {tab === 'terlewat' && (
        <div className="space-y-3">
          <div className="flex flex-wrap items-center gap-2">
            <label htmlFor="journal-eval-status" className="text-xs font-medium">
              Status simulasi
            </label>
            <select
              id="journal-eval-status"
              value={evalStatus}
              onChange={(event) => {
                setEvalStatus(event.target.value as JournalEvalStatus | '')
                setPeluang((current) => ({ ...current, page: 1 }))
              }}
              className="h-9 rounded-md border border-border bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              <option value="">Semua status</option>
              <option value="PENDING">Menunggu penilaian</option>
              <option value="RUNNING">Sedang berjalan</option>
              <option value="TP">TP</option>
              <option value="SL">SL</option>
              <option value="NO_ENTRY">Tidak masuk</option>
              <option value="UNKNOWN">Tidak dapat dipastikan</option>
            </select>
            {isFetching && (
              <span role="status" className="text-xs text-muted-foreground">
                Memuat hasil…
              </span>
            )}
          </div>
          {data && (
            <div aria-busy={isFetching}>
              <JournalSkippedTab data={data} isFetching={isFetching} onUbah={setPeluang} />
            </div>
          )}
        </div>
      )}

      {/* --- Tab: posisi --------------------------------------------------- */}
      {tab !== 'terlewat' && kosong && (
        <EmptyState
          title={tab === 'aktif' ? 'Belum ada posisi berjalan' : 'Belum ada riwayat posisi'}
          description={
            tab === 'aktif'
              ? 'Posisi muncul setelah sinyal dieksekusi pada akun broker Anda.'
              : 'Riwayat terisi begitu posisi ditutup oleh broker.'
          }
        />
      )}

      {tab !== 'terlewat' && !kosong && tampil.length === 0 && (
        <EmptyState
          title={tab === 'aktif' ? 'Tidak ada posisi berjalan' : 'Tidak ada posisi tertutup'}
          description="Coba ubah periode atau filter di atas."
        />
      )}

      {/* --- Desktop: tabel dengan accordion chart ------------------------- */}
      {tab !== 'terlewat' && tampil.length > 0 && !isTabletOrMobile && (
        <div className="hidden lg:block">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-8" />
                <TableHead>Simbol</TableHead>
                <TableHead>Arah</TableHead>
                <TableHead className="text-right">Lot</TableHead>
                <TableHead className="text-right">Open Price</TableHead>
                <TableHead className="text-right">SL</TableHead>
                <TableHead className="text-right">TP</TableHead>
                <TableHead className="text-right">PnL</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Waktu Buka</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {tampil.map((position) => {
                const expanded = expandedId === position.id
                return (
                  <Fragment key={position.id}>
                    <TableRow
                      className={cn('cursor-pointer', expanded && 'bg-secondary/40')}
                      onClick={() => toggle(position.id)}
                    >
                      <TableCell className="text-muted-foreground">
                        <button
                          type="button"
                          aria-expanded={expanded}
                          aria-label={expanded ? 'Tutup chart posisi' : 'Buka chart posisi'}
                          className="flex h-6 w-6 items-center justify-center rounded hover:bg-secondary"
                          onClick={(event) => {
                            event.stopPropagation()
                            toggle(position.id)
                          }}
                        >
                          {expanded ? (
                            <ChevronDown className="h-4 w-4" aria-hidden="true" />
                          ) : (
                            <ChevronRight className="h-4 w-4" aria-hidden="true" />
                          )}
                        </button>
                      </TableCell>
                      <TableCell className="whitespace-nowrap font-semibold">
                        <span className="flex items-center gap-1.5">
                          {position.symbol}
                          {position.is_dry_run && <SimulasiBadge />}
                        </span>
                      </TableCell>
                      <TableCell className="whitespace-nowrap text-muted-foreground">
                        {position.direction}
                      </TableCell>
                      <TableCell className="tabular whitespace-nowrap text-right">
                        {formatNumber(position.lot_size, 2)}
                      </TableCell>
                      <TableCell className="tabular whitespace-nowrap text-right">
                        {formatPrice(
                          position.open_price,
                          priceDigits(position.symbol, position.open_price),
                        )}
                      </TableCell>
                      <TableCell className="tabular whitespace-nowrap text-right text-muted-foreground">
                        {formatPrice(
                          position.stop_loss,
                          priceDigits(position.symbol, position.open_price),
                        )}
                      </TableCell>
                      <TableCell className="tabular whitespace-nowrap text-right text-muted-foreground">
                        {formatPrice(
                          position.take_profit,
                          priceDigits(position.symbol, position.open_price),
                        )}
                      </TableCell>
                      <TableCell className="whitespace-nowrap text-right font-medium">
                        {position.pnl_known ? (
                          <>
                            <span className="tabular">{formatMoney(position.profit_loss)}</span>
                            <span className="ml-1 tabular text-muted-foreground">
                              ({formatR(position.outcome_r)})
                            </span>
                          </>
                        ) : (
                          <span
                            className="text-muted-foreground"
                            title="Hasil belum dilaporkan broker"
                          >
                            —
                          </span>
                        )}
                      </TableCell>
                      <TableCell>
                        <StatusPill tone={TONE[position.status]}>
                          {LABEL[position.status]}
                        </StatusPill>
                      </TableCell>
                      <TableCell className="tabular whitespace-nowrap text-right text-muted-foreground">
                        {formatDateTime(position.opened_at)}
                      </TableCell>
                    </TableRow>

                    {expanded && (
                      <TableRow className="bg-secondary/20 hover:bg-secondary/20">
                        <TableCell colSpan={10} className="p-3">
                          <div className="flex items-center gap-1.5 pb-2 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                            <LineChart className="h-3.5 w-3.5" aria-hidden="true" />
                            Jejak eksekusi
                          </div>
                          <PositionExecutionChart position={position} enabled={expanded} />
                        </TableCell>
                      </TableRow>
                    )}
                  </Fragment>
                )
              })}
            </TableBody>
          </Table>
        </div>
      )}

      {/* --- Mobile: kartu dengan accordion chart -------------------------- */}
      {tab !== 'terlewat' && tampil.length > 0 && (
        <div className="space-y-2.5 lg:hidden">
          {paginasiMobile.baris.map((position) => {
            const expanded = expandedId === position.id
            const digits = priceDigits(position.symbol, position.open_price)
            return (
              <div
                key={position.id}
                className="overflow-hidden rounded-2xl border border-border bg-card shadow-xs"
              >
                <button
                  type="button"
                  onClick={() => toggle(position.id)}
                  aria-expanded={expanded}
                  className="flex w-full flex-col gap-2.5 p-3.5 text-left"
                >
                  <div className="flex w-full items-center justify-between gap-2">
                    <div className="flex min-w-0 flex-wrap items-center gap-1.5">
                      <span className="font-semibold text-foreground">{position.symbol}</span>
                      <Badge variant="outline" className="text-[10px]">
                        {position.direction}
                      </Badge>
                      <span className="font-mono text-[11px] text-muted-foreground">
                        {formatNumber(position.lot_size, 2)} lot
                      </span>
                      {position.is_dry_run && <SimulasiBadge />}
                    </div>
                    <div className="flex shrink-0 items-center gap-1.5">
                      <StatusPill tone={TONE[position.status]}>{LABEL[position.status]}</StatusPill>
                      {expanded ? (
                        <ChevronDown className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
                      ) : (
                        <ChevronRight
                          className="h-4 w-4 text-muted-foreground"
                          aria-hidden="true"
                        />
                      )}
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-2 rounded-xl border border-border/70 bg-secondary/25 p-2.5 text-[11px]">
                    <div className="min-w-0">
                      <p className="text-[10px] uppercase text-muted-foreground">Entry</p>
                      <p className="truncate font-mono font-semibold tabular">
                        {formatPrice(position.open_price, digits)}
                      </p>
                    </div>
                    <div className="min-w-0 text-right">
                      <p className="text-[10px] uppercase text-muted-foreground">Hasil</p>
                      {position.pnl_known ? (
                        <p className="truncate font-mono font-semibold tabular" style={{ color: pnlColor(position) }}>
                          {formatMoney(position.profit_loss)} ({formatR(position.outcome_r)})
                        </p>
                      ) : (
                        <p className="text-muted-foreground">—</p>
                      )}
                    </div>
                    <div className="min-w-0">
                      <p className="text-[10px] uppercase text-muted-foreground">Stop Loss</p>
                      <p className="truncate font-mono tabular text-muted-foreground">
                        {formatPrice(position.stop_loss, digits)}
                      </p>
                    </div>
                    <div className="min-w-0 text-right">
                      <p className="text-[10px] uppercase text-muted-foreground">Take Profit</p>
                      <p className="truncate font-mono tabular text-muted-foreground">
                        {formatPrice(position.take_profit, digits)}
                      </p>
                    </div>
                  </div>

                  <p className="text-[10.5px] text-muted-foreground">
                    Buka {formatDateTime(position.opened_at)}
                    {position.closed_at ? ` · Tutup ${formatDateTime(position.closed_at)}` : ''}
                  </p>
                </button>

                {expanded && (
                  <div className="border-t border-border p-3">
                    <PositionExecutionChart position={position} enabled={expanded} height={260} />
                  </div>
                )}
              </div>
            )
          })}

          {paginasiMobile.totalPages > 1 && (
            <Pagination
              page={paginasiMobile.page}
              totalPages={paginasiMobile.totalPages}
              shown={paginasiMobile.baris.length}
              total={paginasiMobile.total}
              unit="posisi"
              onPageChange={paginasiMobile.setPage}
            />
          )}
        </div>
      )}

      {/* Ekspor CSV dipindahkan ke bagian filter di atas agar tanggal yang
          menyaring daftar dan tanggal yang diekspor adalah hal yang sama. */}
    </>
  )
}

/**
 * JournalSkippedTab menampilkan peluang yang tidak tereksekusi beserta
 * paginasinya.
 *
 * Dipisah menjadi komponen sendiri karena memegang state halaman dan ukuran
 * halaman. Penyaringan tab yang lain (aktif/riwayat) tetap memakai state
 * halaman di induk agar mengganti tab tidak mereset pilihan pengguna.
 */
function JournalSkippedTab({
  data,
  isFetching,
  onUbah,
}: {
  data: JournalResponse
  isFetching: boolean
  /** Meneruskan pilihan halaman ke induk, yang memegang permintaan server. */
  onUbah: (next: { page: number; limit: number }) => void
}) {
  // Sumber kebenaran halaman adalah `meta` dari server: setelah permintaan
  // selesai, server dapat menjepit nomor halaman (mis. melewati batas), dan
  // tampilan harus mengikuti angka itu, bukan tebakan lokal.
  const [ukuran, setUkuran] = useState(data.skipped_meta?.per_page ?? 20)

  // Meta bisa tidak ada bila server yang menjawab masih versi lama (mis. saat
  // deploy bertahap atau respons dari cache). Tanpa penjagaan ini halaman
  // gagal render total; dengan penjagaan, daftar tetap tampil tanpa pager.
  const meta = data.skipped_meta
  if (!meta) {
    return <JournalSkippedList items={data.skipped} />
  }

  return (
    <>
      <JournalSkippedList items={data.skipped} />
      <Pagination
        page={meta.page}
        totalPages={meta.total_pages}
        shown={data.skipped.length}
        total={meta.total}
        unit="peluang"
        disabled={isFetching}
        onPageChange={(nilai) => {
          onUbah({ page: nilai, limit: ukuran })
        }}
        pageSize={ukuran}
        onPageSizeChange={(nilai) => {
          setUkuran(nilai)
          // Ukuran berubah berarti pembagian halaman berubah; kembali ke
          // halaman pertama agar tidak mendarat di halaman kosong.
          onUbah({ page: 1, limit: nilai })
        }}
      />
      {/* Halaman bisa kosong bila jumlah peluang menyusut (mis. filter
          dipersempit dari perangkat lain) sementara nomor halaman tetap.
          Kembalikan pengguna ke halaman terakhir yang benar-benar ada. */}
      {data.skipped.length === 0 && meta.total > 0 && meta.page > 1 && (
        <Alert>
          Halaman ini sudah tidak berisi data.{' '}
          <button
            type="button"
            className="font-semibold underline underline-offset-2"
            onClick={() => {
              const akhir = Math.max(1, meta.total_pages)
              onUbah({ page: akhir, limit: ukuran })
            }}
          >
            Buka halaman terakhir
          </button>
        </Alert>
      )}
    </>
  )
}
