import { Loader2 } from 'lucide-react'
import { Badge, Card, CardContent, Skeleton } from '@/components/ui'
import { useSetStrategySwitch, useStrategySwitchMatrix } from '@/lib/queries'
import type { StrategySwitchAccount } from '@/types/api'
import { cn } from '@/lib/utils'

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

  return (
    <div className="overflow-x-auto rounded-xl border border-border bg-card">
      <table className="w-full text-left text-xs">
        <thead className="border-b border-border bg-secondary/50 text-muted-foreground">
          <tr>
            <th className="px-3.5 py-2.5 font-medium">Akun</th>
            {daftarMetode(accounts).map((metode) => (
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
                    <Badge variant="muted" className="text-micro">
                      Simulasi
                    </Badge>
                  )}
                  {!account.is_active && (
                    <Badge variant="outline" className="text-micro">
                      Nonaktif
                    </Badge>
                  )}
                </div>
                <p className="text-micro uppercase text-muted-foreground">{account.broker_type}</p>
              </td>
              {daftarMetode(accounts).map((metode) => (
                <td key={metode} className="px-3.5 py-2.5 text-center">
                  <Saklar
                    menyala={account.strategies[metode] ?? false}
                    sibuk={
                      ubah.isPending &&
                      ubah.variables?.accountID === account.account_id
                    }
                    onToggle={(nilai) =>
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

function Saklar({
  menyala,
  sibuk,
  onToggle,
}: {
  menyala: boolean
  sibuk: boolean
  onToggle: (nilai: boolean) => void
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={menyala}
      disabled={sibuk}
      onClick={() => onToggle(!menyala)}
      className={cn(
        'inline-flex h-6 w-11 shrink-0 cursor-pointer items-center rounded-full p-0.5 border transition-colors',
        menyala
          ? 'border-primary bg-primary'
          : 'border-border bg-secondary hover:bg-secondary/80',
        sibuk && 'cursor-wait opacity-70',
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
