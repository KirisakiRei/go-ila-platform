/**
 * BacktestTradeChart menggambar satu entry backtest beserta tahapannya.
 *
 * Yang digambar diambil apa adanya dari mesin: candle LTF di sekitar entry,
 * zona FVG, candle sweep, candle CISD, serta garis entry, batas rugi, dan
 * target. Tidak ada harga yang dihitung ulang di sini — chart harus sama
 * persis dengan yang dievaluasi saat simulasi, karena chart yang "kira-kira
 * mirip" justru membuat pengguna salah menyimpulkan.
 */
import { useMemo } from 'react'
import { Skeleton } from '@/components/ui'
import { useBacktestTrade } from '@/lib/queries'
import { priceDigits } from '@/lib/pricePrecision'
import { PriceChart, type ChartMarkerSpec, type PriceLineSpec, type SegmentSpec } from '@/features/charts/PriceChart'
import { chartTheme } from '@/features/charts/lib/theme'
import type { BacktestTradeDetail } from '@/types/api'
import { formatDurasi, formatWaktu } from '../lib/tanggal'

export function BacktestTradeChart({
  runID,
  symbol,
  seq,
}: {
  runID: string
  symbol: string
  seq: number
}) {
  const query = useBacktestTrade(runID, seq)
  const detail = query.data

  if (query.isLoading) {
    return <Skeleton className="h-[340px] w-full" />
  }

  if (!detail) {
    return (
      <div className="flex h-[240px] items-center justify-center border border-border text-[13px] text-muted-foreground">
        Detail entry tidak bisa dimuat.
      </div>
    )
  }

  if (detail.dibersihkan_pada) {
    return (
      <div className="flex h-[240px] flex-col items-center justify-center gap-1 border border-border px-4 text-center text-[13px] text-muted-foreground">
        <span>Potongan candle entry ini sudah dibersihkan.</span>
        <span className="text-[12px]">
          Data disimpan {formatWaktu(detail.dibersihkan_pada)} saat pembersihan berkala; angka
          entry, target, dan hasilnya tetap utuh.
        </span>
      </div>
    )
  }

  return <ChartIsi detail={detail} symbol={symbol} />
}

function ChartIsi({ detail, symbol }: { detail: BacktestTradeDetail; symbol: string }) {
  const theme = chartTheme()
  const digits = priceDigits(symbol, detail.entry)

  const { priceLines, markers, segments, zones, focusTime } = useMemo(() => {
    const lines: PriceLineSpec[] = [
      { price: detail.entry, color: theme.entry, title: `Entry ${detail.entry.toFixed(digits)}`, style: 'dashed', width: 1 },
      { price: detail.stop_loss, color: theme.stopLoss, title: `SL ${detail.stop_loss.toFixed(digits)}`, style: 'solid', width: 1 },
      { price: detail.take_profit, color: theme.takeProfit, title: `TP ${detail.take_profit.toFixed(digits)}`, style: 'solid', width: 1 },
    ]

    const marks: ChartMarkerSpec[] = []
    const isBuy = detail.arah === 'BUY'
    marks.push({
      time: detail.waktu_masuk,
      position: isBuy ? 'belowBar' : 'aboveBar',
      color: isBuy ? theme.up : theme.down,
      shape: isBuy ? 'arrowUp' : 'arrowDown',
      text: `${detail.arah} ${detail.entry.toFixed(digits)}`,
    })

    if (detail.waktu_keluar > detail.waktu_masuk) {
      const menang = detail.hasil === 'TP'
      marks.push({
        time: detail.waktu_keluar,
        position: menang ? 'aboveBar' : 'belowBar',
        color: menang ? theme.profit : theme.loss,
        shape: 'circle',
        text: `${detail.hasil} ${detail.r > 0 ? '+' : ''}${detail.r.toFixed(2)}R`,
      })
    }

    if (detail.sweep) {
      marks.push({
        time: detail.sweep.waktu,
        position: 'aboveBar',
        color: theme.sweep,
        shape: 'square',
        text: detail.sweep.label || 'sweep',
      })
    }

    if (detail.cisd) {
      marks.push({
        time: detail.cisd.waktu,
        position: 'belowBar',
        color: theme.cisd,
        shape: 'square',
        text: detail.cisd.label || 'CISD',
      })
    }

    const links: SegmentSpec[] = []
    if (detail.waktu_keluar > detail.waktu_masuk) {
      links.push({
        fromTime: detail.waktu_masuk,
        fromPrice: detail.entry,
        toTime: detail.waktu_keluar,
        toPrice: detail.hasil === 'TP' ? detail.take_profit : detail.hasil === 'SL' ? detail.stop_loss : detail.entry,
        color: detail.hasil === 'TP' ? theme.profit : theme.loss,
      })
    }

    const zona = detail.zona_fvg
      ? [
          {
            from: detail.zona_fvg.bottom,
            to: detail.zona_fvg.top,
            color: theme.fvg,
            borderColor: theme.fvgBorder,
            label: detail.zona_fvg.label || 'FVG',
            timeFrom: detail.zona_fvg.dari || undefined,
            timeTo: detail.zona_fvg.sampai || undefined,
          },
        ]
      : []

    return {
      priceLines: lines,
      markers: marks,
      segments: links,
      zones: zona,
      focusTime: detail.waktu_masuk,
    }
    // Nilai primitif dipakai sebagai dependensi agar objek baru di tiap render
    // tidak memicu penggambaran ulang chart.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [detail.seq, detail.entry, detail.stop_loss, detail.take_profit, detail.hasil, detail.r, theme, digits])

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-baseline gap-x-4 gap-y-1 text-[12px] text-muted-foreground">
        <span className="font-semibold text-foreground">
          Entry {detail.seq} · {detail.arah}
        </span>
        <span>{formatWaktu(detail.waktu_masuk)}</span>
        <span>ditutup {formatWaktu(detail.waktu_keluar)}</span>
        <span>{formatDurasi(detail.durasi_detik)}</span>
        <span className="tabular">RR rencana {detail.rr_rencana.toFixed(2)}</span>
      </div>

      <PriceChart
        candles={detail.candle_ltf}
        priceLines={priceLines}
        markers={markers}
        segments={segments}
        zones={zones}
        focusTime={focusTime}
        digits={digits}
        height={360}
        emptyMessage="Potongan candle entry ini tidak tersedia."
        footer={
          <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-[11px] text-muted-foreground">
            <span>Seri {detail.ltf}</span>
            {detail.zona_fvg && <span>Zona FVG dari {detail.ltf}</span>}
            {detail.sweep && <span>Sweep di {detail.htf}</span>}
            {detail.cisd && <span>CISD di {detail.ltf}</span>}
          </div>
        }
      />

      {(detail.ringkasan || detail.langkah_berikut) && (
        <div className="flex flex-col gap-2 border border-border bg-secondary/30 p-3 text-[12px] leading-relaxed">
          {detail.ringkasan && (
            <p>
              <span className="text-muted-foreground">Tahapan: </span>
              {detail.ringkasan}
            </p>
          )}
          {detail.langkah_berikut && (
            <p>
              <span className="text-muted-foreground">Yang dinantikan mesin: </span>
              {detail.langkah_berikut}
            </p>
          )}
        </div>
      )}
    </div>
  )
}
