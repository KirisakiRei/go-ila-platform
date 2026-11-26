import { formatAmount } from './model'
import type { JournalAccount } from './types'
import { InfoTooltip } from './ui'

export function AccountBalance({
  account,
  compact = false,
}: {
  account: JournalAccount
  compact?: boolean
}) {
  const complete = account.balance_complete && account.current_realized_balance !== null
  const valueText = complete
    ? formatAmount(account.current_realized_balance!, account.currency)
    : 'Belum lengkap'

  const tooltipText = complete
    ? 'Modal awal + deposit − withdrawal + net P&L closed tercatat (seluruh riwayat akun, bukan equity broker).'
    : account.unknown_closed_pnl > 0
      ? `${account.unknown_closed_pnl} trade closed belum punya net P&L sehingga saldo belum lengkap. Nilai kosong tidak dianggap nol.`
      : 'Data saldo belum lengkap; nilai kosong tidak dianggap nol.'

  if (compact) {
    return (
      <div
        className="inline-flex items-center gap-1.5 rounded-lg border border-border/80 bg-secondary/40 px-2.5 py-1.5 text-xs"
        aria-label={`Saldo terealisasi ${account.name}`}
      >
        <span className="text-muted-foreground">Saldo</span>
        <span className="font-semibold tabular-nums text-foreground">{valueText}</span>
        <InfoTooltip label={`Info saldo ${account.name}`} content={tooltipText} align="end" />
      </div>
    )
  }

  return (
    <div
      className="flex items-center gap-1.5 text-xs text-muted-foreground"
      aria-label={`Saldo terealisasi ${account.name}`}
    >
      <span>Saldo terealisasi</span>
      <span className="font-semibold tabular-nums text-foreground">{valueText}</span>
      <InfoTooltip label={`Info saldo ${account.name}`} content={tooltipText} align="start" />
    </div>
  )
}
