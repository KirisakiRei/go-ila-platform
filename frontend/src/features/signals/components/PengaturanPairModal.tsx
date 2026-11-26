import { useState } from 'react'
import { Loader2 } from 'lucide-react'
import { Badge, Modal, Skeleton } from '@/components/ui'
import { getCycleBadge, getIla3Mtf } from '@/features/signals/lib/status'
import {
  useSetStrategySwitch,
  useSetTimeframeSwitch,
  useStrategySwitchMatrix,
  useTimeframeSwitchMatrix,
} from '@/lib/queries'
import { cn } from '@/lib/utils'
import type { StrategyVersion, TimeframeSwitchItem } from '@/types/api'

type Metode = 'ila-2.0' | 'ila-3.0'

const METODE_LABEL: Record<Metode, string> = {
  'ila-2.0': 'ILA 2.0',
  'ila-3.0': 'ILA 3.0',
}

/**
 * PengaturanPairModal menyatukan saklar aktif/nonaktif metode dan toggle
 * Pantau/Eksekusi tiap Time Cycle dalam satu modal bersih (bukan tabel lebar).
 */
export function PengaturanPairModal({
  open,
  onClose,
  symbol,
  initialStrategy,
  adaIla3,
}: {
  open: boolean
  onClose: () => void
  symbol: string
  initialStrategy: Metode
  adaIla3: boolean
}) {
  const [selectedTab, setSelectedTab] = useState<Metode | null>(null)
  const tab: Metode = !adaIla3 ? 'ila-2.0' : (selectedTab ?? initialStrategy)
  const handleClose = () => {
    setSelectedTab(null)
    onClose()
  }

  const {
    data: stratMatrix,
    isLoading: loadingStrat,
    isError: errorStrat,
  } = useStrategySwitchMatrix(open ? symbol : null)
  const {
    data: tfMatrix,
    isLoading: loadingTf,
    isError: errorTf,
  } = useTimeframeSwitchMatrix(open ? symbol : null)

  const ubahMetode = useSetStrategySwitch(symbol)
  const ubahTf = useSetTimeframeSwitch(symbol)

  const tabs: Metode[] = adaIla3 ? ['ila-2.0', 'ila-3.0'] : ['ila-2.0']
  const accounts = stratMatrix?.accounts ?? []
  const tfGroup = tfMatrix?.groups?.find((g) => g.strategy === tab)
  const autoAllowed = Boolean(tfMatrix?.auto_allowed)

  const toggleTf = (
    account: { account_id: string; pairs: TimeframeSwitchItem[] },
    strategy: StrategyVersion,
    target: TimeframeSwitchItem,
    perubahan: { watch: boolean; auto: boolean },
  ) => {
    const pairs = account.pairs.map((pair) =>
      pair.htf === target.htf && pair.ltf === target.ltf
        ? { ...pair, ...perubahan }
        : pair,
    )
    ubahTf.mutate({ accountID: account.account_id, strategy, pairs })
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      size="lg"
      title={`Pengaturan ${symbol}`}
      description="Aktifkan metode analisis dan atur toggle pantau serta eksekusi otomatis per time cycle untuk tiap akun."
    >
      {/* Tab Metode: ILA 2.0 vs ILA 3.0 */}
      <div
        role="tablist"
        aria-label={`Metode pengaturan ${symbol}`}
        className="flex items-center gap-1 border-b border-border"
      >
        {tabs.map((m) => {
          const aktif = tab === m
          return (
            <button
              key={m}
              type="button"
              role="tab"
              aria-selected={aktif}
              onClick={() => setSelectedTab(m)}
              className={cn(
                '-mb-px border-b-2 px-4 py-2 text-xs font-semibold transition-colors',
                aktif
                  ? 'border-foreground text-foreground'
                  : 'border-transparent text-muted-foreground hover:text-foreground',
              )}
            >
              {METODE_LABEL[m]}
            </button>
          )
        })}
      </div>

      {/* Konten per Akun */}
      {loadingStrat || loadingTf ? (
        <div className="space-y-3">
          <Skeleton className="h-36 w-full rounded-xl" />
        </div>
      ) : errorStrat || errorTf ? (
        <div className="rounded-xl border border-border bg-secondary/20 px-4 py-6 text-center text-xs text-muted-foreground">
          Gagal memuat pengaturan akun untuk {symbol}. Tutup lalu coba buka kembali.
        </div>
      ) : accounts.length === 0 ? (
        <div className="rounded-xl border border-border bg-secondary/20 px-4 py-6 text-center text-xs text-muted-foreground">
          Belum ada akun broker terhubung. Hubungkan akun terlebih dahulu untuk mengatur {symbol}.
        </div>
      ) : (
        <div className="space-y-3.5">
          {accounts.map((account) => {
            const metodeNyala = Boolean(account.strategies[tab])
            const sibukMetode =
              ubahMetode.isPending && ubahMetode.variables?.accountID === account.account_id
            const tfAccount = tfGroup?.accounts.find((a) => a.account_id === account.account_id)
            const pairDiizinkan = tfAccount?.pair_allowed !== false
            const sibukTf =
              ubahTf.isPending &&
              ubahTf.variables?.accountID === account.account_id &&
              ubahTf.variables?.strategy === tab
            const namaAkun = account.label || 'Akun Broker'

            return (
              <div
                key={account.account_id}
                className="overflow-hidden rounded-xl border border-border bg-card"
              >
                {/* Header Akun + Toggle Aktifkan Metode */}
                <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border bg-secondary/35 px-3.5 py-2.5">
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-1.5">
                      <span className="truncate text-xs font-bold text-foreground">
                        {namaAkun}
                      </span>
                      <span className="text-[10px] uppercase text-muted-foreground">
                        {account.broker_type}
                      </span>
                      {account.is_dry_run && (
                        <Badge variant="muted" className="text-[9.5px]">
                          Simulasi
                        </Badge>
                      )}
                      {!account.is_active && (
                        <Badge variant="outline" className="text-[9.5px]">
                          Nonaktif
                        </Badge>
                      )}
                    </div>
                  </div>

                  <div className="flex items-center gap-2">
                    <span className="text-[11px] font-medium text-muted-foreground">
                      {metodeNyala ? `${METODE_LABEL[tab]} Aktif` : `${METODE_LABEL[tab]} Nonaktif`}
                    </span>
                    <ToggleSaklar
                      label={`Aktifkan ${METODE_LABEL[tab]} untuk ${namaAkun}`}
                      menyala={metodeNyala}
                      sibuk={sibukMetode}
                      onToggle={(nilai) =>
                        ubahMetode.mutate({
                          accountID: account.account_id,
                          strategies: { ...account.strategies, [tab]: nilai },
                        })
                      }
                    />
                  </div>
                </div>

                {/* Daftar Time Cycle (Toggle Pantau & Eksekusi) */}
                {!metodeNyala ? (
                  <div className="px-3.5 py-4 text-xs text-muted-foreground">
                    Aktifkan saklar <span className="font-semibold text-foreground">{METODE_LABEL[tab]}</span> di atas untuk memilih time cycle yang dipantau dan dieksekusi pada akun ini.
                  </div>
                ) : !tfAccount || tfAccount.pairs.length === 0 ? (
                  <div className="px-3.5 py-4 text-xs text-muted-foreground">
                    Daftar time cycle {METODE_LABEL[tab]} sedang diselaraskan dengan engine. Coba muat ulang sesaat lagi.
                  </div>
                ) : (
                  <div className="divide-y divide-border/60">
                    <div className="flex items-center justify-between bg-secondary/15 px-3.5 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                      <span>{tab === 'ila-2.0' ? 'Time Cycle' : 'Siklus Timeframe'}</span>
                      <div className="flex items-center gap-5">
                        <span className="w-11 text-center">Pantau</span>
                        {autoAllowed && <span className="w-11 text-center">Eksekusi</span>}
                      </div>
                    </div>

                    {tfAccount.pairs.map((pair) => {
                      const cycle = getCycleBadge(tab, pair.htf, pair.ltf)
                      const tfLabel =
                        tab === 'ila-3.0'
                          ? `${pair.htf} · ${getIla3Mtf(pair.htf, pair.ltf)} · ${pair.ltf}`
                          : `${pair.htf} · ${pair.ltf}`

                      return (
                        <div
                          key={`${account.account_id}-${pair.htf}-${pair.ltf}`}
                          className="flex items-center justify-between gap-3 px-3.5 py-2.5"
                        >
                          <div className="min-w-0">
                            <p className="truncate text-xs font-semibold text-foreground">
                              {cycle ?? tfLabel}
                            </p>
                            {cycle && (
                              <p className="font-mono text-[10.5px] text-muted-foreground">
                                {tfLabel}
                              </p>
                            )}
                          </div>

                          <div className="flex shrink-0 items-center gap-5">
                            <ToggleSaklar
                              label={`Pantau ${cycle ?? tfLabel} untuk ${namaAkun}`}
                              menyala={pair.watch}
                              sibuk={sibukTf}
                              onToggle={(nilai) =>
                                toggleTf(tfAccount, tab, pair, {
                                  watch: nilai,
                                  auto: nilai ? pair.auto : false,
                                })
                              }
                            />
                            {autoAllowed && (
                              <ToggleSaklar
                                label={`Eksekusi ${cycle ?? tfLabel} untuk ${namaAkun}`}
                                menyala={pairDiizinkan ? pair.auto : false}
                                sibuk={sibukTf}
                                nonaktif={!pair.watch || !pairDiizinkan}
                                onToggle={(nilai) =>
                                  toggleTf(tfAccount, tab, pair, {
                                    watch: pair.watch,
                                    auto: nilai,
                                  })
                                }
                              />
                            )}
                          </div>
                        </div>
                      )
                    })}

                    {autoAllowed && !pairDiizinkan && (
                      <div className="bg-secondary/20 px-3.5 py-2 text-[11px] text-muted-foreground">
                        Eksekusi otomatis nonaktif karena {symbol} tidak dicentang di Filter Pair
                        Eksekusi akun ini. Setelan pantau tetap mengikuti pilihan Anda.
                      </div>
                    )}

                    {!autoAllowed && (
                      <div className="bg-secondary/10 px-3.5 py-2 text-[11px] text-muted-foreground">
                        Eksekusi otomatis tidak tersedia pada paket saat ini.
                      </div>
                    )}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}
    </Modal>
  )
}

function ToggleSaklar({
  label,
  menyala,
  sibuk,
  nonaktif,
  onToggle,
}: {
  label: string
  menyala: boolean
  sibuk: boolean
  nonaktif?: boolean
  onToggle: (nilai: boolean) => void
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-label={label}
      aria-checked={menyala}
      aria-disabled={nonaktif}
      disabled={sibuk || nonaktif}
      title={nonaktif ? 'Nyalakan Pantau lebih dulu' : undefined}
      onClick={() => onToggle(!menyala)}
      className={cn(
        'inline-flex h-6 w-11 shrink-0 cursor-pointer items-center rounded-full border p-0.5 transition-colors',
        menyala
          ? 'border-primary bg-primary'
          : 'border-border bg-secondary hover:bg-secondary/80',
        (sibuk || nonaktif) && 'cursor-not-allowed opacity-50',
        sibuk && 'cursor-wait',
      )}
    >
      <span
        className={cn(
          'pointer-events-none flex h-5 w-5 items-center justify-center rounded-full bg-background shadow-xs ring-0 transition-transform duration-200',
          menyala ? 'translate-x-5' : 'translate-x-0',
        )}
      >
        {sibuk && <Loader2 className="h-3 w-3 animate-spin text-muted-foreground" />}
      </span>
    </button>
  )
}
