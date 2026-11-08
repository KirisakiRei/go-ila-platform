import { useState, useMemo, useEffect } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import {
  ArrowLeft,
  Check,
  Laptop,
  RotateCcw,
  Save,
  Search,
  Settings,
  Sliders,
  Trash2,
} from 'lucide-react'
import {
  Alert,
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  EmptyState,
  Input,
  Label,
  Modal,
  Select,
  Skeleton,
  StatusDot,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui'
import {
  useAccounts,
  useAccountPairs,
  useDeleteAccount,
  usePositions,
  useUpdateAccount,
  useUpdateAccountPairs,
} from '@/lib/queries'
import { formatDateTime, formatMoney, formatNumber } from '@/lib/format'
import { useAuthContext } from '@/features/auth/hooks/use-auth'
import { ApiError } from '@/lib/api-error'
import type { BrokerAccount, PairItem } from '@/types/api'

export default function AccountDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { user } = useAuthContext()

  const { data: accounts, isLoading: accountsLoading } = useAccounts()
  const account = accounts?.find((a) => a.id === id)

  const [activeTab, setActiveTab] = useState<'risk' | 'pairs' | 'history'>('risk')
  const [konfirmasiHapus, setKonfirmasiHapus] = useState(false)

  const remove = useDeleteAccount()

  if (accountsLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  if (!account) {
    return (
      <div className="space-y-4 py-8">
        <EmptyState
          title="Akun broker tidak ditemukan"
          description="Akun mungkin sudah dihapus atau tautan tidak valid."
          action={
            <Link
              to="/akun"
              className="inline-flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-semibold"
            >
              <ArrowLeft className="h-3.5 w-3.5" />
              Kembali ke Daftar Akun
            </Link>
          }
        />
      </div>
    )
  }

  const status = account.connected ? 'online' : account.last_error ? 'offline' : 'idle'
  const statusText = account.connected
    ? 'Sesi Aktif / Terhubung'
    : account.last_error
      ? 'Bermasalah'
      : 'Belum Tersambung'

  return (
    <div className="space-y-6">
      {/* Navigasi Kembali */}
      <div className="flex items-center justify-between">
        <Link
          to="/akun"
          className="inline-flex items-center gap-1.5 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          <span>Kembali ke Daftar Akun</span>
        </Link>

        <div className="flex items-center gap-2">
          <Button
            size="sm"
            variant="outline"
            className="text-destructive hover:bg-destructive/10"
            onClick={() => setKonfirmasiHapus(true)}
          >
            <Trash2 className="mr-1.5 h-3.5 w-3.5" />
            Hapus Akun
          </Button>
        </div>
      </div>

      {/* Header Info Akun */}
      <Card>
        <CardContent className="p-5">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="space-y-1">
              <div className="flex items-center gap-2.5">
                <StatusDot state={status} label={statusText} />
                <h1 className="text-lg font-bold text-foreground">{account.label}</h1>
                <Badge variant={account.broker_type === 'ctrader' ? 'solid' : 'outline'}>
                  {account.broker_type === 'ctrader' ? 'cTrader OAuth' : 'Match-Trader'}
                </Badge>
              </div>
              <p className="text-xs text-muted-foreground">
                Nomor Login: <strong>{account.external_account_id || 'Belum terbaca'}</strong> · Status: {statusText} · Sinkron terakhir: {formatDateTime(account.last_synced_at)}
              </p>
            </div>

            <div className="flex flex-wrap items-center gap-2">
              <Badge variant={account.is_dry_run ? 'muted' : 'solid'}>
                {account.is_dry_run ? 'Simulasi (Dry Run)' : 'Live Trading'}
              </Badge>
              <Badge variant="outline">{account.mode === 'auto' ? 'Mode Otomatis' : 'Mode Manual'}</Badge>
              {!account.is_active && (
                <Badge variant="outline" className="border-red-500/30 bg-red-500/10 text-red-600">
                  Nonaktif
                </Badge>
              )}
            </div>
          </div>

          {account.last_error && (
            <Alert variant="destructive" className="mt-4 text-xs">
              {account.last_error}
            </Alert>
          )}

          {/* 4 Kolom Metrik Finansial */}
          <div className="mt-5 grid grid-cols-2 gap-3 border-t border-border pt-4 sm:grid-cols-4">
            <div className="rounded-lg border border-border bg-secondary/20 p-3">
              <p className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                Saldo (Balance)
              </p>
              <p className="mt-1 text-base font-bold tabular-nums text-foreground">
                {formatMoney(account.balance)}
              </p>
            </div>

            <div className="rounded-lg border border-border bg-secondary/20 p-3">
              <p className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                Ekuitas (Equity)
              </p>
              <p className="mt-1 text-base font-bold tabular-nums text-foreground">
                {formatMoney(account.equity)}
              </p>
            </div>

            <div className="rounded-lg border border-border bg-secondary/20 p-3">
              <p className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                Margin Bebas
              </p>
              <p className="mt-1 text-base font-bold tabular-nums text-foreground">
                {formatMoney(account.free_margin)}
              </p>
            </div>

            <div className="rounded-lg border border-border bg-secondary/20 p-3">
              <p className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                Risiko per Trade
              </p>
              <p className="mt-1 text-base font-bold tabular-nums text-foreground">
                {formatNumber(account.risk_percent, 2)}%
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Tab Navigasi Halaman Detail */}
      <div className="flex border-b border-border">
        <button
          type="button"
          onClick={() => setActiveTab('risk')}
          className={`flex items-center gap-2 border-b-2 px-4 py-2.5 text-xs font-semibold transition-all ${
            activeTab === 'risk'
              ? 'border-foreground text-foreground'
              : 'border-transparent text-muted-foreground hover:text-foreground'
          }`}
        >
          <Sliders className="h-3.5 w-3.5" />
          <span>1. Manajemen Risiko & Pengaturan</span>
        </button>

        <button
          type="button"
          onClick={() => setActiveTab('pairs')}
          className={`flex items-center gap-2 border-b-2 px-4 py-2.5 text-xs font-semibold transition-all ${
            activeTab === 'pairs'
              ? 'border-foreground text-foreground'
              : 'border-transparent text-muted-foreground hover:text-foreground'
          }`}
        >
          <Settings className="h-3.5 w-3.5" />
          <span>2. Filter Pair Eksekusi Akun</span>
        </button>

        <button
          type="button"
          onClick={() => setActiveTab('history')}
          className={`flex items-center gap-2 border-b-2 px-4 py-2.5 text-xs font-semibold transition-all ${
            activeTab === 'history'
              ? 'border-foreground text-foreground'
              : 'border-transparent text-muted-foreground hover:text-foreground'
          }`}
        >
          <Laptop className="h-3.5 w-3.5" />
          <span>3. Riwayat Order Akun</span>
        </button>
      </div>

      {/* Konten Tab Aktif */}
      {activeTab === 'risk' && <TabRiskSettings account={account} tier={user?.tier ?? 'free'} />}
      {activeTab === 'pairs' && <TabPairFilter accountId={account.id} />}
      {activeTab === 'history' && <TabOrderHistory accountId={account.id} />}

      {/* Modal Konfirmasi Hapus Akun */}
      <Modal
        open={konfirmasiHapus}
        onClose={() => setKonfirmasiHapus(false)}
        title="Hapus Akun Broker?"
        description={`Anda akan menghapus akun ${account.label} (${account.external_account_id || account.broker_type}) secara permanen.`}
      >
        <div className="space-y-4 pt-1">
          <p className="text-xs text-muted-foreground leading-relaxed">
            Seluruh kredensial token broker yang tersimpan di sistem akan langsung dihapus. Sinyal trading
            tidak lagi dapat diteruskan ke akun ini.
          </p>
          <div className="flex justify-end gap-2">
            <Button variant="outline" size="sm" onClick={() => setKonfirmasiHapus(false)}>
              Batal
            </Button>
            <Button
              variant="destructive"
              size="sm"
              isLoading={remove.isPending}
              onClick={() => {
                remove.mutate(account.id, {
                  onSuccess: () => {
                    setKonfirmasiHapus(false)
                    navigate('/akun')
                  },
                })
              }}
            >
              Hapus Permanen
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  )
}

/** TAB 1: MANAJEMEN RISIKO & PENGATURAN */
function TabRiskSettings({ account, tier }: { account: BrokerAccount; tier: string }) {
  const update = useUpdateAccount()
  const [label, setLabel] = useState(account.label)
  const [mode, setMode] = useState(account.mode)
  const [riskPercent, setRiskPercent] = useState(account.risk_percent)
  const [isDryRun, setIsDryRun] = useState(account.is_dry_run)
  const [isActive, setIsActive] = useState(account.is_active)
  const [notice, setNotice] = useState<string | null>(null)

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    setNotice(null)
    try {
      await update.mutateAsync({
        id: account.id,
        payload: {
          label: label.trim(),
          mode,
          risk_percent: Number(riskPercent),
          is_dry_run: Boolean(isDryRun),
          is_active: Boolean(isActive),
        },
      })
      setNotice('Pengaturan akun berhasil disimpan.')
    } catch (err) {
      setNotice(err instanceof ApiError ? err.message : 'Gagal menyimpan pengaturan akun.')
    }
  }

  // Tombol Terapkan Default Template
  const handleResetToDefault = () => {
    setMode('manual')
    setRiskPercent(0.5)
    setIsDryRun(true)
    setIsActive(true)
    setNotice('Pengaturan disetel ke template default sistem (Manual, Risk 0,5%, Simulasi). Jangan lupa klik Simpan.')
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <CardTitle className="text-sm font-semibold">Pengaturan Risiko & Eksekusi</CardTitle>
            <CardDescription className="text-xs">
              Aturan ini hanya berlaku untuk eksekusi order pada akun broker ini.
            </CardDescription>
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleResetToDefault}
            className="gap-1.5 text-xs self-start sm:self-auto"
          >
            <RotateCcw className="h-3 w-3" />
            Terapkan Default Template
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSave} className="space-y-4">
          {notice && (
            <Alert variant={notice.includes('Gagal') ? 'destructive' : 'success'} className="text-xs">
              {notice}
            </Alert>
          )}

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="account-label">Nama Label Akun</Label>
              <Input
                id="account-label"
                value={label}
                onChange={(e) => setLabel(e.target.value)}
                className="text-xs"
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="account-mode">Mode Eksekusi Order</Label>
              <Select
                id="account-mode"
                value={mode}
                onChange={(e) => setMode(e.target.value as 'manual' | 'auto')}
                disabled={tier !== 'plus'}
              >
                <option value="manual">Manual (Perlu Konfirmasi)</option>
                <option value="auto">Otomatis (Langsung Ditembak)</option>
              </Select>
              {tier !== 'plus' && (
                <p className="text-[11px] text-muted-foreground">Otomatis hanya untuk tier Plus.</p>
              )}
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="risk-percent">Batas Risiko per Transaksi (%)</Label>
              <Input
                id="risk-percent"
                type="number"
                step="0.05"
                min="0.1"
                max="1"
                value={riskPercent}
                onChange={(e) => setRiskPercent(Number(e.target.value))}
                className="text-xs"
              />
              <p className="text-[11px] text-muted-foreground">
                Batas proteksi aman: 0,10% sampai 1,00% dari total ekuitas per trade.
              </p>
            </div>

            <div className="flex flex-col gap-3 justify-center pt-2">
              <label className="flex items-center gap-2 text-xs font-medium cursor-pointer">
                <input
                  type="checkbox"
                  checked={isDryRun}
                  onChange={(e) => setIsDryRun(e.target.checked)}
                  className="h-4 w-4 rounded accent-foreground"
                />
                <span>Jalankan sebagai Simulasi (Dry Run / Tidak mengirim uang nyata)</span>
              </label>

              <label className="flex items-center gap-2 text-xs font-medium cursor-pointer">
                <input
                  type="checkbox"
                  checked={isActive}
                  onChange={(e) => setIsActive(e.target.checked)}
                  className="h-4 w-4 rounded accent-foreground"
                />
                <span>Akun Aktif (Dapat menerima dan memproses sinyal)</span>
              </label>
            </div>
          </div>

          <div className="pt-2">
            <Button type="submit" size="sm" isLoading={update.isPending} className="gap-1.5">
              <Save className="h-3.5 w-3.5" />
              Simpan Perubahan Risiko
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  )
}

/** TAB 2: FILTER PAIR EKSEKUSI */
function TabPairFilter({ accountId }: { accountId: string }) {
  const { data, isLoading, isError, error } = useAccountPairs(accountId)
  const updatePairs = useUpdateAccountPairs()

  const [selectedSymbols, setSelectedSymbols] = useState<string[]>([])
  const [search, setSearch] = useState('')
  const [notice, setNotice] = useState<string | null>(null)

  // Inisialisasi daftar terpilih dari server
  useEffect(() => {
    if (data?.selected) {
      setSelectedSymbols(data.selected)
    }
  }, [data?.selected])

  // Kumpulkan seluruh item pair dari katalog
  const allPairs = useMemo(() => {
    if (!data?.catalog?.categories) return []
    const pairs: PairItem[] = []
    for (const cat of data.catalog.categories) {
      for (const grp of cat.groups) {
        for (const p of grp.pairs) {
          pairs.push(p)
        }
      }
    }
    return pairs
  }, [data?.catalog])

  const filteredPairs = useMemo(() => {
    if (!search.trim()) return allPairs
    const q = search.toLowerCase()
    return allPairs.filter(
      (p) =>
        p.symbol.toLowerCase().includes(q) ||
        p.name.toLowerCase().includes(q) ||
        (p.aliases && p.aliases.toLowerCase().includes(q)),
    )
  }, [allPairs, search])

  const toggleSymbol = (symbol: string) => {
    setSelectedSymbols((prev) =>
      prev.includes(symbol) ? prev.filter((s) => s !== symbol) : [...prev, symbol],
    )
  }

  const selectAll = () => {
    setSelectedSymbols(allPairs.map((p) => p.symbol))
  }

  const deselectAll = () => {
    setSelectedSymbols([])
  }

  const handleSavePairs = async () => {
    setNotice(null)
    try {
      await updatePairs.mutateAsync({
        id: accountId,
        symbols: selectedSymbols,
      })
      setNotice('Pilihan pair untuk akun ini berhasil diperbarui.')
    } catch (err) {
      setNotice(err instanceof ApiError ? err.message : 'Gagal menyimpan pilihan pair.')
    }
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <CardTitle className="text-sm font-semibold">Filter Pair Eksekusi untuk Akun Ini</CardTitle>
            <CardDescription className="text-xs">
              Hanya simbol pair yang Anda centang yang akan dieksekusi oleh akun ini.
            </CardDescription>
          </div>
          <div className="flex items-center gap-2">
            <Button type="button" variant="outline" size="sm" onClick={selectAll} className="text-xs">
              Pilih Semua
            </Button>
            <Button type="button" variant="outline" size="sm" onClick={deselectAll} className="text-xs">
              Batal Pilih
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {notice && (
          <Alert variant={notice.includes('Gagal') ? 'destructive' : 'success'} className="text-xs">
            {notice}
          </Alert>
        )}

        {isLoading && <Skeleton className="h-48 w-full" />}
        {isError && <Alert variant="destructive">{(error as Error).message}</Alert>}

        {!isLoading && (
          <>
            <div className="relative">
              <Search className="absolute left-3 top-2.5 h-3.5 w-3.5 text-muted-foreground" />
              <Input
                placeholder="Cari simbol pair (misal: XAUUSD, EURUSD)..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-9 text-xs"
              />
            </div>

            <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2 md:grid-cols-3">
              {filteredPairs.map((pair) => {
                const isSelected = selectedSymbols.includes(pair.symbol)
                return (
                  <button
                    key={pair.symbol}
                    type="button"
                    onClick={() => toggleSymbol(pair.symbol)}
                    className={`flex items-center justify-between rounded-xl border p-3 text-left transition-all ${
                      isSelected
                        ? 'border-foreground bg-secondary/60 shadow-sm'
                        : 'border-border bg-card hover:border-zinc-300'
                    }`}
                  >
                    <div>
                      <p className="text-xs font-bold text-foreground">{pair.symbol}</p>
                      <p className="text-[11px] text-muted-foreground">{pair.name}</p>
                    </div>

                    <div
                      className={`flex h-4 w-4 items-center justify-center rounded border ${
                        isSelected
                          ? 'border-foreground bg-foreground text-background'
                          : 'border-zinc-300'
                      }`}
                    >
                      {isSelected && <Check className="h-3 w-3 stroke-[3]" />}
                    </div>
                  </button>
                )
              })}
            </div>

            {filteredPairs.length === 0 && (
              <p className="py-6 text-center text-xs text-muted-foreground">
                Tidak ada simbol pair yang sesuai pencarian.
              </p>
            )}

            <div className="flex items-center justify-between border-t border-border pt-4">
              <span className="text-xs text-muted-foreground">
                Terpilih: <strong>{selectedSymbols.length}</strong> dari {allPairs.length} pair
              </span>
              <Button
                size="sm"
                isLoading={updatePairs.isPending}
                onClick={handleSavePairs}
                className="gap-1.5"
              >
                <Save className="h-3.5 w-3.5" />
                Simpan Pemilihan Pair
              </Button>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}

/** TAB 3: RIWAYAT ORDER AKUN */
function TabOrderHistory({ accountId: _accountId }: { accountId: string }) {
  const { data: positions, isLoading } = usePositions(100)

  const rows = positions ?? []

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-semibold">Riwayat Eksekusi Order</CardTitle>
        <CardDescription className="text-xs">
          Daftar tiket transaksi dan eksekusi sinyal trading terbaru.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading && <Skeleton className="h-32 w-full" />}

        {!isLoading && rows.length === 0 && (
          <p className="py-8 text-center text-xs text-muted-foreground">
            Belum ada posisi atau tiket eksekusi yang tercatat.
          </p>
        )}

        {!isLoading && rows.length > 0 && (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="text-xs">Waktu</TableHead>
                  <TableHead className="text-xs">Simbol</TableHead>
                  <TableHead className="text-xs">Arah</TableHead>
                  <TableHead className="text-xs">Lot</TableHead>
                  <TableHead className="text-xs">Open</TableHead>
                  <TableHead className="text-xs">TP / SL</TableHead>
                  <TableHead className="text-xs">PnL</TableHead>
                  <TableHead className="text-xs">Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rows.map((pos) => {
                  const isProfit = pos.pnl_known && pos.profit_loss > 0
                  const isLoss = pos.pnl_known && pos.profit_loss < 0

                  return (
                    <TableRow key={pos.id} className="text-xs">
                      <TableCell className="text-muted-foreground whitespace-nowrap">
                        {formatDateTime(pos.opened_at)}
                      </TableCell>
                      <TableCell className="font-bold">{pos.symbol}</TableCell>
                      <TableCell>
                        <Badge variant={pos.direction === 'BUY' ? 'solid' : 'outline'}>
                          {pos.direction}
                        </Badge>
                      </TableCell>
                      <TableCell className="tabular-nums font-mono">{pos.lot_size}</TableCell>
                      <TableCell className="tabular-nums font-mono">{pos.open_price}</TableCell>
                      <TableCell className="tabular-nums font-mono text-[11px] text-muted-foreground">
                        {pos.take_profit ?? '—'} / {pos.stop_loss ?? '—'}
                      </TableCell>
                      <TableCell
                        className={`tabular-nums font-bold ${
                          isProfit ? 'text-emerald-600' : isLoss ? 'text-red-500' : ''
                        }`}
                      >
                        {pos.pnl_known
                          ? `${pos.profit_loss >= 0 ? '+' : ''}${formatMoney(pos.profit_loss)}`
                          : '—'}
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className="text-[10px]">
                          {pos.status}
                        </Badge>
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
