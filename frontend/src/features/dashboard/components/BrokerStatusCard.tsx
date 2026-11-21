import { Link } from 'react-router-dom'
import { ArrowRight, ShieldAlert, Wallet } from 'lucide-react'
import { Card, CardContent, Skeleton, StatusDot } from '@/components/ui'
import { useAccounts, useSystemStatus } from '@/lib/queries'
import { formatMoney } from '@/lib/format'
import { cn } from '@/lib/utils'

export function BrokerStatusCard() {
  const { data: accounts, isLoading } = useAccounts()
  const { data: status } = useSystemStatus()

  const list = accounts ?? []

  return (
    <Card className="flex h-full flex-col border-border/80 bg-card/95">
      <CardContent className="flex flex-1 flex-col gap-3.5 p-5">
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <h2 className="text-sm font-bold tracking-tight text-foreground">Akun Broker</h2>
            {list.length > 0 && (
              <span className="rounded-md bg-secondary px-1.5 py-0.5 font-mono text-[10.5px] font-semibold text-muted-foreground">
                {list.filter((a) => a.is_active).length}/{list.length} aktif
              </span>
            )}
          </div>
          <Link
            to="/akun"
            className="inline-flex items-center gap-1 text-xs font-semibold text-foreground transition-colors hover:text-muted-foreground"
          >
            Kelola
            <ArrowRight className="h-3.5 w-3.5" />
          </Link>
        </div>

        {isLoading ? (
          <div className="space-y-2.5">
            <Skeleton className="h-16 w-full rounded-lg" />
            <Skeleton className="h-16 w-full rounded-lg" />
          </div>
        ) : list.length === 0 ? (
          <div className="flex flex-1 flex-col items-center justify-center rounded-xl border border-dashed border-border bg-secondary/20 px-4 py-7 text-center">
            <Wallet className="h-6 w-6 text-muted-foreground" />
            <p className="mt-2.5 text-xs font-semibold text-foreground">
              Belum ada akun broker terhubung
            </p>
            <p className="mt-1 text-[11.5px] text-muted-foreground">
              Hubungkan Match-Trader atau cTrader agar sinyal bisa dieksekusi.
            </p>
            <Link
              to="/akun"
              className="mt-3 inline-flex items-center gap-1 text-xs font-semibold text-foreground hover:underline"
            >
              Hubungkan Akun
              <ArrowRight className="h-3.5 w-3.5" />
            </Link>
          </div>
        ) : (
          <div className="flex flex-1 flex-col gap-2">
            {list.slice(0, 4).map((acc) => {
              const healthy = acc.is_active && !acc.is_frozen && !acc.last_error
              const riskLabel =
                acc.risk_mode === 'lot'
                  ? `${acc.lot_size} lot`
                  : `${acc.risk_percent}% risiko`

              return (
                <Link
                  key={acc.id}
                  to={`/akun/${acc.id}`}
                  className="flex items-center justify-between gap-3 rounded-xl border border-border/80 bg-secondary/20 p-3 transition-colors duration-150 hover:border-foreground/30 hover:bg-secondary/40 active:scale-[0.98]"
                >
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-1.5">
                      <StatusDot
                        state={
                          !acc.is_active
                            ? 'idle'
                            : healthy
                              ? 'online'
                              : acc.is_frozen
                                ? 'pending'
                                : 'offline'
                        }
                        label={acc.label || acc.broker_type}
                      />
                      <span className="truncate text-xs font-semibold text-foreground">
                        {acc.label || acc.external_account_id || acc.broker_type}
                      </span>
                      {acc.is_dry_run && (
                        <span className="rounded bg-secondary px-1.5 py-0.5 text-[9.5px] font-bold uppercase text-muted-foreground">
                          Demo
                        </span>
                      )}
                    </div>
                    <div className="mt-1 flex flex-wrap items-center gap-1.5 text-[11px] text-muted-foreground">
                      <span className="capitalize">{acc.broker_type}</span>
                      <span>·</span>
                      <span
                        className={cn(
                          'font-mono text-[10.5px] font-semibold uppercase',
                          acc.mode === 'auto'
                            ? 'text-emerald-700 dark:text-emerald-400'
                            : 'text-foreground',
                        )}
                      >
                        {acc.mode}
                      </span>
                      <span>·</span>
                      <span className="font-mono">{riskLabel}</span>
                    </div>
                    {acc.last_error && (
                      <p className="mt-1 flex items-center gap-1 truncate text-[10.5px] text-rose-600 dark:text-rose-400">
                        <ShieldAlert className="h-3 w-3 shrink-0" />
                        <span className="truncate">{acc.last_error}</span>
                      </p>
                    )}
                  </div>

                  <div className="shrink-0 text-right">
                    <p className="font-mono text-xs font-bold tabular-nums text-foreground">
                      {acc.equity > 0
                        ? formatMoney(acc.equity)
                        : acc.balance > 0
                          ? formatMoney(acc.balance)
                          : '—'}
                    </p>
                    <p className="text-[10px] text-muted-foreground">
                      {acc.equity > 0 ? 'Ekuitas' : 'Saldo'}
                    </p>
                  </div>
                </Link>
              )
            })}
          </div>
        )}

        {/* Footer koneksi Engine */}
        {status && status.engines.length > 0 && (
          <div className="flex flex-wrap items-center justify-between gap-2 border-t border-border/70 pt-3 text-[11px] text-muted-foreground">
            <span>Engine</span>
            <div className="flex flex-wrap items-center gap-3">
              {status.engines.map((e) => (
                <span key={e.strategy} className="inline-flex items-center gap-1.5 font-mono">
                  <StatusDot
                    state={e.connected ? 'online' : 'offline'}
                    label={`${e.strategy} ${e.connected ? 'terhubung' : 'terputus'}`}
                  />
                  <span>{e.strategy === 'ila-2.0' ? 'ILA 2.0' : 'ILA 3.0'}</span>
                </span>
              ))}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
