import { formatAmount } from './model'
import type { JournalAccount } from './types'

export function AccountBalance({ account }: { account: JournalAccount }) {
  const complete = account.balance_complete && account.current_realized_balance !== null
  return (
    <div className="space-y-1" aria-label={`Saldo terealisasi ${account.name}`}>
      <p className="text-sm">
        Saldo terealisasi jurnal:{' '}
        <span className="font-semibold tabular-nums">
          {complete
            ? formatAmount(account.current_realized_balance!, account.currency)
            : 'Belum dapat ditentukan'}
        </span>
      </p>
      {!complete && (
        <p className="text-xs text-muted-foreground">
          {account.unknown_closed_pnl > 0
            ? `${account.unknown_closed_pnl} trade closed belum memiliki net P&L; saldo tidak lengkap.`
            : 'Data saldo belum lengkap; nilai kosong bukan nol.'}
        </p>
      )}
      <p className="max-w-prose text-xs text-muted-foreground">
        Modal awal + deposit − withdrawal + net P&L closed yang tercatat. Seluruh riwayat akun,
        bukan filter tabel; bukan equity broker dan tidak termasuk floating P&L.
      </p>
    </div>
  )
}
