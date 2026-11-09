import { Loader2 } from 'lucide-react'
import { Badge, Card, CardContent, Skeleton } from '@/components/ui'
import { useSetTimeframeSwitch, useTimeframeSwitchMatrix } from '@/lib/queries'
import type { StrategyVersion, TimeframeSwitchItem } from '@/types/api'
import { cn } from '@/lib/utils'

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
export function SaklarTimeframeAkun({ symbol }: { symbol: string }) {
  const { data, isLoading, isError } = useTimeframeSwitchMatrix(symbol)
  const ubah = useSetTimeframeSwitch(symbol)

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

  const groups = data?.groups ?? []
  if (groups.length === 0) {
    return (
      <Card>
        <CardContent className="py-6 text-center text-xs text-muted-foreground">
          Belum ada metode yang aktif. Nyalakan metode pada akun terlebih dahulu.
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
                    <th className="px-3.5 py-2.5 text-center font-medium">Pantau</th>
                    {data?.auto_allowed && (
                      <th className="px-3.5 py-2.5 text-center font-medium">Eksekusi</th>
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
                          <td className="px-3.5 py-2.5 font-mono text-[11.5px]">
                            {pair.htf} · {pair.ltf}
                          </td>
                          <td className="px-3.5 py-2.5 text-center">
                            <Saklar
                              menyala={pair.watch}
                              sibuk={sibuk}
                              onToggle={(nilai) =>
                                kirim(ubah, account, group.strategy, pair, {
                                  watch: nilai,
                                  // Mematikan pantau harus ikut mematikan auto:
                                  // sinyal yang tidak diminta tak bisa dieksekusi.
                                  auto: nilai ? pair.auto : false,
                                })
                              }
                            />
                          </td>
                          {data?.auto_allowed && (
                            <td className="px-3.5 py-2.5 text-center">
                              <Saklar
                                menyala={pair.auto}
                                sibuk={sibuk}
                                nonaktif={!pair.watch}
                                onToggle={(nilai) =>
                                  kirim(ubah, account, group.strategy, pair, {
                                    // Menyalakan auto otomatis menyalakan pantau.
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

/**
 * kirim menyusun ulang seluruh pasangan akun dengan satu nilai diubah.
 *
 * Server menimpa seluruh saklar akun untuk metode ini, jadi baris lain wajib
 * ikut dikirim agar tidak hilang.
 */
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

function Saklar({
  menyala,
  sibuk,
  nonaktif,
  onToggle,
}: {
  menyala: boolean
  sibuk: boolean
  nonaktif?: boolean
  onToggle: (nilai: boolean) => void
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={menyala}
      aria-disabled={nonaktif}
      disabled={sibuk || nonaktif}
      title={nonaktif ? 'Nyatakan Pantau lebih dulu' : undefined}
      onClick={() => onToggle(!menyala)}
      className={cn(
        'inline-flex h-6 w-11 items-center rounded-full border transition-colors',
        menyala ? 'border-foreground bg-foreground' : 'border-border bg-secondary',
        (sibuk || nonaktif) && 'cursor-not-allowed opacity-60',
        sibuk && 'cursor-wait',
      )}
    >
      <span
        className={cn(
          'flex h-4.5 w-4.5 items-center justify-center rounded-full bg-background transition-transform',
          menyala ? 'translate-x-[22px]' : 'translate-x-1',
        )}
      >
        {sibuk && <Loader2 className="h-3 w-3 animate-spin text-muted-foreground" />}
      </span>
    </button>
  )
}
