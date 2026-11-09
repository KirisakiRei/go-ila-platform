import { useState } from 'react'
import { Calculator, Info } from 'lucide-react'
import { Alert, Button, Input, Label, Select } from '@/components/ui'
import { usePairCatalog, useRiskSimulation } from '@/lib/queries'
import { ApiError } from '@/lib/api-error'
import { cn } from '@/lib/utils'
import { formatMoney, formatNumber } from '@/lib/format'
import type { RiskMode } from '@/types/api'

/**
 * RiskSimulatorPanel menghitung pratinjau risiko di server.
 *
 * Angka yang tampil berasal dari endpoint yang sama dengan tempat rumus lot
 * ditegakkan, sehingga tidak ada dua versi hitungan yang bisa berbeda diam-diam.
 */
export function RiskSimulatorPanel({
  accountId,
  defaultEquity,
  riskMode,
  riskPercent,
  lotSize,
}: {
  accountId: string
  defaultEquity: number
  riskMode: RiskMode
  riskPercent: number
  lotSize: number
}) {
  const { data: catalog } = usePairCatalog()
  const simulasi = useRiskSimulation()

  const [symbol, setSymbol] = useState('')
  const [entry, setEntry] = useState('')
  const [stop, setStop] = useState('')
  const [equity, setEquity] = useState(defaultEquity > 0 ? String(defaultEquity) : '')
  const [galat, setGalat] = useState<string | null>(null)

  const opsiPair = (catalog?.categories ?? []).flatMap((kategori) =>
    kategori.groups.flatMap((grup) =>
      grup.pairs.map((pair) => ({
        symbol: pair.symbol,
        label: pair.name ? `${pair.symbol} — ${pair.name}` : pair.symbol,
      })),
    ),
  )

  const hitung = async () => {
    setGalat(null)
    try {
      await simulasi.mutateAsync({
        account_id: accountId,
        symbol,
        entry_price: Number(entry),
        stop_loss: Number(stop),
        equity: Number(equity) || undefined,
        mode: riskMode,
        risk_percent: riskPercent,
        lot_size: lotSize,
      })
    } catch (err) {
      setGalat(err instanceof ApiError ? err.message : 'Gagal menghitung simulasi.')
    }
  }

  const hasil = simulasi.data
  const siap = Boolean(symbol) && Number(entry) > 0 && Number(stop) > 0

  return (
    <div className="space-y-3 rounded-xl border border-border bg-secondary/20 p-3.5">
      <div className="flex items-center gap-2">
        <Calculator className="h-4 w-4 text-muted-foreground" aria-hidden="true" />
        <p className="text-[12.5px] font-semibold text-foreground">Simulasi Risiko</p>
      </div>

      <p className="text-[11.5px] leading-relaxed text-muted-foreground">
        Masukkan rencana entry dan stop loss. Hasilnya menunjukkan lot yang dipakai, nominal yang
        dipertaruhkan, serta padanan risiko 1% sampai 5%.
      </p>

      <div className="grid gap-3 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sim-symbol">Pair</Label>
          <Select
            id="sim-symbol"
            value={symbol}
            onChange={(e) => setSymbol(e.target.value)}
            className="text-xs"
          >
            <option value="">Pilih pair</option>
            {opsiPair.map((item) => (
              <option key={item.symbol} value={item.symbol}>
                {item.label}
              </option>
            ))}
          </Select>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sim-equity">Ekuitas</Label>
          <Input
            id="sim-equity"
            type="number"
            step="0.01"
            min="0"
            value={equity}
            onChange={(e) => setEquity(e.target.value)}
            className="text-xs"
            placeholder="10000"
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sim-entry">Harga Entry</Label>
          <Input
            id="sim-entry"
            type="number"
            step="0.00001"
            min="0"
            value={entry}
            onChange={(e) => setEntry(e.target.value)}
            className="text-xs"
            placeholder="4120.50"
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sim-stop">Stop Loss</Label>
          <Input
            id="sim-stop"
            type="number"
            step="0.00001"
            min="0"
            value={stop}
            onChange={(e) => setStop(e.target.value)}
            className="text-xs"
            placeholder="4112.30"
          />
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Button
          type="button"
          size="sm"
          variant="outline"
          className="gap-1.5 text-xs"
          disabled={!siap || simulasi.isPending}
          onClick={() => void hitung()}
        >
          <Calculator className="h-3.5 w-3.5" />
          {simulasi.isPending ? 'Menghitung…' : 'Hitung Simulasi'}
        </Button>
        {!siap && (
          <span className="text-[11px] text-muted-foreground">
            Lengkapi pair, entry, dan stop loss.
          </span>
        )}
      </div>

      {galat && <Alert variant="destructive">{galat}</Alert>}

      {hasil && (
        <div className="space-y-3" aria-live="polite">
          <div className="grid gap-2 sm:grid-cols-3">
            <div className="rounded-lg border border-border bg-card px-3 py-2">
              <p className="text-[10px] uppercase tracking-[0.12em] text-muted-foreground">Lot</p>
              <p className="text-sm font-bold tabular-nums text-foreground">{hasil.hasil.lots}</p>
            </div>
            <div className="rounded-lg border border-border bg-card px-3 py-2">
              <p className="text-[10px] uppercase tracking-[0.12em] text-muted-foreground">
                Nominal Risiko
              </p>
              <p className="text-sm font-bold tabular-nums text-foreground">
                {formatMoney(hasil.hasil.risk_amount)}
              </p>
            </div>
            <div className="rounded-lg border border-border bg-card px-3 py-2">
              <p className="text-[10px] uppercase tracking-[0.12em] text-muted-foreground">
                % Ekuitas
              </p>
              <p
                className={cn(
                  'text-sm font-bold tabular-nums',
                  hasil.hasil.warning ? 'text-amber-600' : 'text-foreground',
                )}
              >
                {formatNumber(hasil.hasil.risk_percent, 2)}%
              </p>
            </div>
          </div>

          <p className="flex items-start gap-1.5 text-[11px] leading-relaxed text-muted-foreground">
            <Info className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden="true" />
            <span>
              Nilai per lot {formatNumber(hasil.value_per_lot, 2)} (
              {hasil.sumber_nilai === 'broker'
                ? 'spesifikasi broker'
                : hasil.sumber_nilai === 'katalog'
                  ? 'katalog pair'
                  : 'bawaan kategori'}
              ) · jarak stop {formatNumber(hasil.stop_distance, 5)}
            </span>
          </p>

          <div className="overflow-x-auto rounded-lg border border-border bg-card">
            <table className="w-full text-left text-[11.5px]">
              <thead className="border-b border-border bg-secondary/40 text-muted-foreground">
                <tr>
                  <th className="px-3 py-2 font-medium">Risiko</th>
                  <th className="px-3 py-2 font-medium text-right">Lot</th>
                  <th className="px-3 py-2 font-medium text-right">Nominal</th>
                  <th className="px-3 py-2 font-medium">Catatan</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {hasil.tabel.map((baris) => (
                  <tr key={baris.risk_percent}>
                    <td className="px-3 py-1.5 font-semibold tabular-nums">
                      {formatNumber(baris.risk_percent, 0)}%
                    </td>
                    <td className="px-3 py-1.5 text-right tabular-nums">{baris.lots}</td>
                    <td className="px-3 py-1.5 text-right tabular-nums">
                      {formatMoney(baris.risk_amount)}
                    </td>
                    <td className="px-3 py-1.5">
                      {baris.warning ? (
                        <span className="text-[11px] font-medium text-amber-600">Agresif</span>
                      ) : (
                        <span className="text-[11px] text-muted-foreground">Aman</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="overflow-x-auto rounded-lg border border-border bg-card">
            <table className="w-full text-left text-[11.5px]">
              <thead className="border-b border-border bg-secondary/40 text-muted-foreground">
                <tr>
                  <th className="px-3 py-2 font-medium">Lot</th>
                  <th className="px-3 py-2 font-medium text-right">Nominal</th>
                  <th className="px-3 py-2 font-medium text-right">% Ekuitas</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {hasil.tabel_lot.map((baris) => (
                  <tr key={baris.lots}>
                    <td className="px-3 py-1.5 font-semibold tabular-nums">{baris.lots}</td>
                    <td className="px-3 py-1.5 text-right tabular-nums">
                      {formatMoney(baris.risk_amount)}
                    </td>
                    <td
                      className={cn(
                        'px-3 py-1.5 text-right tabular-nums',
                        baris.warning ? 'font-medium text-amber-600' : '',
                      )}
                    >
                      {formatNumber(baris.risk_percent, 2)}%
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {hasil.peringatan.length > 0 && (
            <Alert variant="warning" className="text-[11.5px]">
              <div className="space-y-1">
                {hasil.peringatan.map((pesan) => (
                  <p key={pesan}>{pesan}</p>
                ))}
              </div>
            </Alert>
          )}
        </div>
      )}
    </div>
  )
}
