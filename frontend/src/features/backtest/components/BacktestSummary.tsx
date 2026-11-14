/**
 * BacktestSummary menampilkan angka ringkas satu run.
 *
 * Urutannya sengaja: hasil (trade, menang, R) lebih dulu, lalu mutu (drawdown,
 * profit factor), lalu catatan kejujuran data (sinyal ditolak, candle ambigu).
 * Catatan itu bukan hiasan: tanpa keduanya, pembaca tidak tahu bahwa sebagian
 * setup memang dibuang dan sebagian hasil bergantung pada urutan yang tidak
 * bisa dipastikan.
 */
import { Info } from 'lucide-react'
import { Section, Stat, StatGrid } from '@/components/ui'
import type { BacktestRunDetail } from '@/types/api'
import { EquityCurve } from './EquityCurve'

export function BacktestSummary({ run }: { run: BacktestRunDetail }) {
  const statistik = run.statistik

  return (
    <div className="flex flex-col gap-5">
      <StatGrid>
        <Stat label="Total trade" value={statistik.total_trade} hint={`${statistik.kedaluwarsa} belum tuntas`} />
        <Stat
          label="Win rate"
          value={`${(statistik.win_rate * 100).toFixed(1)}%`}
          hint={`${statistik.menang} menang, ${statistik.kalah} kalah`}
        />
        <Stat
          label="Total R"
          value={`${statistik.total_r > 0 ? '+' : ''}${statistik.total_r.toFixed(2)}`}
          hint="jumlah risiko menang dikurangi kalah"
        />
        <Stat
          label="Drawdown"
          value={`${(statistik.maks_drawdown * 100).toFixed(1)}%`}
          hint="penurunan terdalam dari puncak"
        />
        <Stat
          label="Profit factor"
          value={statistik.profit_factor > 0 ? statistik.profit_factor.toFixed(2) : '—'}
          hint="laba kotor dibagi rugi kotor"
        />
        <Stat
          label="Ekuitas akhir"
          value={formatUang(statistik.ekuitas_akhir || run.equity)}
          hint={`mulai dari ${formatUang(run.equity)}`}
        />
        <Stat label="Sinyal ditolak" value={statistik.sinyal_ditolak} hint="rasio imbal risiko di bawah batas" />
        <Stat label="Candle ambigu" value={statistik.candle_ambigu} hint="TP dan SL tersentuh bersamaan" />
      </StatGrid>

      <Section title="Kurva ekuitas" description="Garis putus-putus adalah modal awal.">
        <EquityCurve titik={run.kurva_ekuitas ?? []} ekuitasAwal={run.equity} />
      </Section>

      <div className="flex items-start gap-2 border border-border bg-secondary/30 p-3 text-[12px] leading-relaxed text-muted-foreground">
        <Info className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
        <div className="flex flex-col gap-1">
          <span>
            Hasil simulasi atas data historis, bukan performa live. Aturan yang dipakai sama dengan
            mesin live ILA {run.strategi.replace('ila-', '')} versi {run.versi}.
          </span>
          {statistik.candle_ambigu > 0 && (
            <span>
              {statistik.candle_ambigu} candle menyentuh target dan batas rugi sekaligus. Tanpa data
              tick, urutannya tidak bisa dipastikan; semuanya dihitung sebagai kalah.
            </span>
          )}
        </div>
      </div>
    </div>
  )
}

function formatUang(nilai: number): string {
  return new Intl.NumberFormat('id-ID', { maximumFractionDigits: 0 }).format(nilai)
}
