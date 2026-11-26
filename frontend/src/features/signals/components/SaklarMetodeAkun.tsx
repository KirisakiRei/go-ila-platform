import { Badge, Card, CardContent, Skeleton, Switch } from '@/components/ui'
import { useSetStrategySwitch, useStrategySwitchMatrix } from '@/lib/queries'
import { useIsMobile } from '@/lib/use-mobile-viewport'
import type { StrategySwitchAccount } from '@/types/api'

/**
 * Metode yang boleh ditampilkan. ILA 3.0 hanya muncul bila server menyatakan
 * tier pengguna berhak, sehingga fitur berbayar tidak pernah bocor di UI.
 */
const METODE = ['ila-2.0', 'ila-3.0'] as const
type Metode = (typeof METODE)[number]

const LABEL: Record<Metode, string> = {
  'ila-2.0': 'ILA 2.0',
  'ila-3.0': 'ILA 3.0',
}

/**
 * SaklarMetodeAkun menampilkan matriks akun × metode untuk satu pair.
 *
 * Pair pantauan bersifat global (dipantau kedua metode), sedangkan saklar di
 * sini menentukan metode mana yang benar-benar dieksekusi tiap akun.
 */
export function SaklarMetodeAkun({ symbol }: { symbol: string }) {
  const { data, isLoading, isError } = useStrategySwitchMatrix(symbol)
  const ubah = useSetStrategySwitch(symbol)
  const isMobile = useIsMobile()

  if (isLoading) {
    return <Skeleton className="h-40 w-full rounded-xl" />
  }

  if (isError) {
    return (
      <Card>
        <CardContent className="py-6 text-center text-xs text-muted-foreground">
          Gagal memuat saklar metode untuk {symbol}. Coba muat ulang halaman.
        </CardContent>
      </Card>
    )
  }

  const accounts = data?.accounts ?? []
  if (accounts.length === 0) {
    return (
      <Card>
        <CardContent className="py-6 text-center text-xs text-muted-foreground">
          Belum ada akun broker. Hubungkan akun terlebih dahulu agar metode pada {symbol} dapat diatur.
        </CardContent>
      </Card>
    )
  }

  const metodeTampil = daftarMetode(accounts)

  if (isMobile) {
    return (
      <div className="space-y-2.5">
        {accounts.map((account) => {
          const sibuk = ubah.isPending && ubah.variables?.accountID === account.account_id
          return (
            <div
              key={account.account_id}
              className="overflow-hidden rounded-xl border border-border bg-card"
            >
              <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border bg-secondary/40 px-3 py-2">
                <div className="flex min-w-0 items-center gap-1.5">
                  <span className="truncate text-xs font-semibold text-foreground">
                    {account.label || 'Akun Broker'}
                  </span>
                  {account.is_dry_run && (
                    <Badge variant="muted" className="shrink-0 text-[9.5px]">
                      Simulasi
                    </Badge>
                  )}
                  {!account.is_active && (
                    <Badge variant="outline" className="shrink-0 text-[9.5px]">
                      Nonaktif
                    </Badge>
                  )}
                </div>
                <span className="text-[10px] uppercase text-muted-foreground">
                  {account.broker_type}
                </span>
              </div>
              <div className="divide-y divide-border/70">
                {metodeTampil.map((metode) => (
                  <div
                    key={metode}
                    className="flex items-center justify-between gap-2 px-3 py-2.5 text-xs"
                  >
                    <span className="font-medium text-foreground">Metode {LABEL[metode]}</span>
                    <Switch
                      aria-label={`Metode ${LABEL[metode]} untuk ${account.label || 'Akun Broker'}`}
                      checked={account.strategies[metode] ?? false}
                      loading={sibuk}
                      onCheckedChange={(nilai) =>
                        ubah.mutate({
                          accountID: account.account_id,
                          strategies: { ...account.strategies, [metode]: nilai },
                        })
                      }
                    />
                  </div>
                ))}
              </div>
            </div>
          )
        })}
      </div>
    )
  }

  return (
    <div className="overflow-x-auto rounded-xl border border-border bg-card">
      <table className="w-full text-left text-xs">
        <thead className="border-b border-border bg-secondary/50 text-muted-foreground">
          <tr>
            <th className="px-3.5 py-2.5 font-medium">Akun</th>
            {metodeTampil.map((metode) => (
              <th key={metode} className="px-3.5 py-2.5 text-center font-medium">
                {LABEL[metode]}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-border">
          {accounts.map((account) => (
            <tr key={account.account_id} className="transition-colors hover:bg-secondary/30">
              <td className="px-3.5 py-2.5">
                <div className="flex items-center gap-2">
                  <span className="font-semibold text-foreground">{account.label || 'Akun Broker'}</span>
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
                <p className="text-[10.5px] uppercase text-muted-foreground">{account.broker_type}</p>
              </td>
              {metodeTampil.map((metode) => (
                <td key={metode} className="px-3.5 py-2.5 text-center">
                  <Switch
                    aria-label={`Metode ${LABEL[metode]} untuk ${account.label || 'Akun Broker'}`}
                    checked={account.strategies[metode] ?? false}
                    loading={
                      ubah.isPending &&
                      ubah.variables?.accountID === account.account_id
                    }
                    onCheckedChange={(nilai) =>
                      ubah.mutate({
                        accountID: account.account_id,
                        strategies: { ...account.strategies, [metode]: nilai },
                      })
                    }
                  />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

/**
 * daftarMetode menyaring metode yang boleh tampil. Bila server tidak mengirim
 * peta hak akses, hanya ILA 2.0 yang ditampilkan agar aman.
 */
function daftarMetode(accounts: StrategySwitchAccount[]): Metode[] {
  const allowed = accounts[0]?.allowed ?? {}
  return METODE.filter((metode) => metode !== 'ila-3.0' || allowed[metode] === true)
}
