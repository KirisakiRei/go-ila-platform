import { Stat, StatGrid } from '@/components/ui'
import { formatMoney, formatNumber, formatR } from '@/lib/format'
import type { JournalStats } from '@/types/api'

/**
 * JournalSummary merangkum performa eksekusi.
 *
 * Angka dipisah menjadi dua kelompok yang sengaja tidak dicampur:
 *  - Hasil (winrate, R, profit) hanya dari posisi yang sudah ditutup.
 *  - Aktivitas (total, berjalan) menghitung seluruh posisi.
 *
 * Pencampuran itu akan membuat winrate turun setiap kali posisi baru dibuka,
 * padahal hasilnya belum diketahui — angka yang berubah tanpa sebab nyata
 * tidak bisa dipakai memutuskan apa pun.
 */
export function JournalSummary({ stats }: { stats: JournalStats }) {
  const dievaluasi = stats.wins + stats.losses
  const belumSelesai = dievaluasi === 0

  return (
    <div className="space-y-3">
      <StatGrid>
        <Stat
          label="Winrate"
          value={belumSelesai ? '—' : `${formatNumber(stats.win_rate, 1)}%`}
          hint={
            belumSelesai
              ? 'Belum ada posisi yang ditutup'
              : `${stats.wins} menang dari ${dievaluasi} ditutup`
          }
        />
        <Stat
          label="Total R"
          value={belumSelesai ? '—' : formatR(stats.total_r)}
          hint={belumSelesai ? undefined : `Rata-rata ${formatR(stats.average_r)} per trade`}
        />
        <Stat
          label="Total P/L"
          value={belumSelesai ? '—' : formatMoney(stats.total_profit)}
          hint={belumSelesai ? undefined : 'Dari posisi yang sudah ditutup'}
        />
        <Stat
          label="Posisi Berjalan"
          value={String(stats.open_positions)}
          hint={`${stats.total_positions} posisi pada periode ini`}
        />
      </StatGrid>

      <dl className="flex flex-wrap items-center gap-x-4 gap-y-1 text-[11px] text-muted-foreground">
        <div className="flex items-center gap-1.5">
          <dt>Take Profit</dt>
          <dd className="tabular font-medium text-foreground">{stats.closed_tp}</dd>
        </div>
        <div className="flex items-center gap-1.5">
          <dt>Stop Loss</dt>
          <dd className="tabular font-medium text-foreground">{stats.closed_sl}</dd>
        </div>
        <div className="flex items-center gap-1.5">
          <dt>Tutup Manual</dt>
          <dd className="tabular font-medium text-foreground">{stats.closed_manual}</dd>
        </div>
        {stats.untracked > 0 && (
          <div className="flex items-center gap-1.5">
            <dt>Tak Terlacak</dt>
            <dd className="tabular font-medium text-foreground">{stats.untracked}</dd>
          </div>
        )}
        {stats.dry_run_positions > 0 && (
          <div className="flex items-center gap-1.5">
            <dt>Termasuk simulasi</dt>
            <dd className="tabular font-medium text-foreground">{stats.dry_run_positions}</dd>
          </div>
        )}
      </dl>
    </div>
  )
}
