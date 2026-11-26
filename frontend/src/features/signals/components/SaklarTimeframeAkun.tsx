import { Badge, Card, CardContent, Skeleton, Switch } from '@/components/ui'
import { getCycleBadge, getIla3Mtf } from '@/features/signals/lib/status'
import { useSetTimeframeSwitch, useTimeframeSwitchMatrix } from '@/lib/queries'
import { useIsMobile } from '@/lib/use-mobile-viewport'
import type { StrategyVersion, TimeframeSwitchItem } from '@/types/api'

/**
 * SaklarTimeframeAkun menampilkan pasangan timeframe per akun dan per metode.
 *
 * Dua saklar terpisah:
 *  - Pantau     : sinyal pasangan ini diminta ke engine dan ditampilkan.
 *  - Eksekusi   : pasangan ini boleh dieksekusi otomatis.
 *
 * Pasangan timeframe berasal dari engine (aturan trading ILA). Bila engine belum
 * pernah mengirim pasangan untuk sebuah metode, blok itu menampilkan keterangan
 * "menunggu data engine" alih-alih daftar karangan.
 */
export function SaklarTimeframeAkun({
  symbol,
  strategy,
}: {
  symbol: string
  /** Bila diisi, hanya menampilkan grup timeframe untuk metode tersebut. */
  strategy?: StrategyVersion
}) {
  const { data, isLoading, isError } = useTimeframeSwitchMatrix(symbol)
  const ubah = useSetTimeframeSwitch(symbol)
  const isMobile = useIsMobile()

  if (isLoading) return <Skeleton className="h-40 w-full rounded-xl" />

  if (isError) {
    return (
      <Card>
        <CardContent className="py-6 text-center text-xs text-muted-foreground">
          Gagal memuat saklar timeframe untuk {symbol}. Coba muat ulang halaman.
        </CardContent>
      </Card>
    )
  }

  const allGroups = data?.groups ?? []
  const groups = strategy ? allGroups.filter((g) => g.strategy === strategy) : allGroups
  if (groups.length === 0) {
    return (
      <Card>
        <CardContent className="py-6 text-center text-xs text-muted-foreground">
          {strategy
            ? `Metode ${strategy.toUpperCase()} belum aktif pada akun mana pun. Nyalakan metode pada akun terlebih dahulu.`
            : 'Belum ada metode yang aktif. Nyalakan metode pada akun terlebih dahulu.'}
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-4">
      {groups.map((group) => {
        const daftar = data?.tersedia?.[group.strategy] ?? []

        if (daftar.length === 0) {
          return (
            <Card key={group.strategy}>
              <CardContent className="py-5 text-center text-xs text-muted-foreground">
                Pasangan timeframe {group.strategy} belum tersedia. Data timeframe sedang diselaraskan.
                Silakan muat ulang sesaat lagi.
              </CardContent>
            </Card>
          )
        }

        if (isMobile) {
          return (
            <div key={group.strategy} className="space-y-2">
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                  {group.strategy}
                </span>
                {!data?.auto_allowed && (
                  <Badge variant="muted" className="text-[9.5px]">
                    Eksekusi otomatis tidak tersedia
                  </Badge>
                )}
              </div>

              {group.accounts.map((account) => {
                const sibuk =
                  ubah.isPending &&
                  ubah.variables?.accountID === account.account_id &&
                  ubah.variables?.strategy === group.strategy

                return (
                  <div
                    key={account.account_id}
                    className="overflow-hidden rounded-xl border border-border bg-card"
                  >
                    <div className="border-b border-border bg-secondary/40 px-3 py-2 text-xs font-semibold text-foreground">
                      {account.label || 'Akun Broker'}
                    </div>
                    <div className="divide-y divide-border/70">
                      {account.pairs.map((pair) => {
                        const tfLabel =
                          group.strategy === 'ila-3.0'
                            ? `${pair.htf} · ${getIla3Mtf(pair.htf, pair.ltf)} · ${pair.ltf}`
                            : `${pair.htf} · ${pair.ltf}`
                        const cycle = getCycleBadge(group.strategy, pair.htf, pair.ltf)

                        return (
                          <div
                            key={`${account.account_id}-${pair.htf}-${pair.ltf}`}
                            className="flex flex-col gap-2 px-3 py-2.5 text-xs"
                          >
                            <div className="flex flex-wrap items-center gap-1.5">
                              <span className="font-mono text-[11.5px] font-semibold text-foreground">
                                {tfLabel}
                              </span>
                              {cycle && (
                                <span className="inline-flex items-center rounded border border-primary/20 bg-primary/10 px-1.5 py-0.5 text-[9.5px] font-medium text-primary">
                                  {cycle}
                                </span>
                              )}
                            </div>

                            <div className="divide-y divide-border/60 rounded-lg border border-border/70 bg-secondary/20">
                              <div className="flex items-center justify-between gap-3 px-2.5 py-2">
                                <span className="text-[11px] font-medium text-foreground">Pantau</span>
                                <Switch
                                  aria-label={`Pantau ${tfLabel} untuk ${account.label || 'Akun Broker'}`}
                                  checked={pair.watch}
                                  loading={sibuk}
                                  onCheckedChange={(nilai) =>
                                    kirim(ubah, account, group.strategy, pair, {
                                      watch: nilai,
                                      auto: nilai ? pair.auto : false,
                                    })
                                  }
                                />
                              </div>

                              {data?.auto_allowed && (
                                <div className="flex items-center justify-between gap-3 px-2.5 py-2">
                                  <span className="text-[11px] font-medium text-foreground">Eksekusi</span>
                                  <Switch
                                    aria-label={`Eksekusi ${tfLabel} untuk ${account.label || 'Akun Broker'}`}
                                    checked={pair.auto}
                                    loading={sibuk}
                                    disabled={!pair.watch}
                                    title={!pair.watch ? 'Nyalakan Pantau lebih dulu' : undefined}
                                    onCheckedChange={(nilai) =>
                                      kirim(ubah, account, group.strategy, pair, {
                                        watch: nilai ? true : pair.watch,
                                        auto: nilai,
                                      })
                                    }
                                  />
                                </div>
                              )}
                            </div>
                          </div>
                        )
                      })}
                    </div>
                  </div>
                )
              })}
            </div>
          )
        }

        return (
          <div key={group.strategy} className="space-y-2">
            <div className="flex items-center gap-2">
              <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                {group.strategy}
              </span>
              {!data?.auto_allowed && (
                <Badge variant="muted" className="text-[9.5px]">
                  Eksekusi otomatis tidak tersedia
                </Badge>
              )}
            </div>

            <div className="overflow-x-auto rounded-xl border border-border bg-card">
              <table className="w-full text-left text-xs">
                <thead className="border-b border-border bg-secondary/50 text-muted-foreground">
                  <tr>
                    <th className="px-3.5 py-2.5 font-medium">Akun</th>
                    <th className="px-3.5 py-2.5 font-medium">Timeframe</th>
                    <th className="w-24 px-3.5 py-2.5 text-center font-medium">Pantau</th>
                    {data?.auto_allowed && (
                      <th className="w-24 px-3.5 py-2.5 text-center font-medium">Eksekusi</th>
                    )}
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {group.accounts.map((account) =>
                    account.pairs.map((pair, index) => {
                      const sibuk =
                        ubah.isPending &&
                        ubah.variables?.accountID === account.account_id &&
                        ubah.variables?.strategy === group.strategy

                      return (
                        <tr key={`${account.account_id}-${pair.htf}-${pair.ltf}`} className="hover:bg-secondary/30">
                          {index === 0 && (
                            <td className="px-3.5 py-2.5 align-top" rowSpan={account.pairs.length}>
                              <span className="font-semibold text-foreground">
                                {account.label || 'Akun Broker'}
                              </span>
                            </td>
                          )}
                          <td className="px-3.5 py-2.5">
                            <div className="flex flex-wrap items-center gap-1.5">
                              <span className="font-mono text-[11.5px] font-medium text-foreground">
                                {(() => {
                                  if (group.strategy === 'ila-3.0') {
                                    const mtf = getIla3Mtf(pair.htf, pair.ltf)
                                    return `${pair.htf} · ${mtf} · ${pair.ltf}`
                                  }
                                  return `${pair.htf} · ${pair.ltf}`
                                })()}
                              </span>
                              {(() => {
                                const cycle = getCycleBadge(group.strategy, pair.htf, pair.ltf)
                                if (!cycle) return null
                                return (
                                  <span className="inline-flex items-center rounded border border-primary/20 bg-primary/10 px-1.5 py-0.5 text-[10px] font-medium text-primary">
                                    {cycle}
                                  </span>
                                )
                              })()}
                            </div>
                          </td>
                          <td className="px-3.5 py-2.5 text-center">
                            <Switch
                              aria-label={`Pantau ${pair.htf} ${pair.ltf} untuk ${account.label || 'Akun Broker'}`}
                              checked={pair.watch}
                              loading={sibuk}
                              onCheckedChange={(nilai) =>
                                kirim(ubah, account, group.strategy, pair, {
                                  watch: nilai,
                                  auto: nilai ? pair.auto : false,
                                })
                              }
                            />
                          </td>
                          {data?.auto_allowed && (
                            <td className="px-3.5 py-2.5 text-center">
                              <Switch
                                aria-label={`Eksekusi ${pair.htf} ${pair.ltf} untuk ${account.label || 'Akun Broker'}`}
                                checked={pair.auto}
                                loading={sibuk}
                                disabled={!pair.watch}
                                title={!pair.watch ? 'Nyalakan Pantau lebih dulu' : undefined}
                                onCheckedChange={(nilai) =>
                                  kirim(ubah, account, group.strategy, pair, {
                                    watch: nilai ? true : pair.watch,
                                    auto: nilai,
                                  })
                                }
                              />
                            </td>
                          )}
                        </tr>
                      )
                    }),
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )
      })}
    </div>
  )
}

function kirim(
  ubah: ReturnType<typeof useSetTimeframeSwitch>,
  account: { account_id: string; pairs: TimeframeSwitchItem[] },
  strategy: StrategyVersion,
  target: TimeframeSwitchItem,
  perubahan: { watch: boolean; auto: boolean },
) {
  const pairs = account.pairs.map((pair) =>
    pair.htf === target.htf && pair.ltf === target.ltf
      ? { ...pair, ...perubahan }
      : pair,
  )
  ubah.mutate({ accountID: account.account_id, strategy, pairs })
}
