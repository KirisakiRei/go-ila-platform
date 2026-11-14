/**
 * BacktestTradeChart menggambar satu entry backtest beserta tahapannya.
 *
 * Ditata seperti chart eksekusi live: dua panel bertumpuk, timeframe besar di
 * atas untuk konteks dan timeframe kecil di bawah untuk titik masuk. Keduanya
 * memakai level, penanda, dan zona yang SAMA, sehingga pembaca melihat satu
 * kejadian dari dua jarak pandang — bukan dua cerita berbeda.
 *
 * Semua angka diambil apa adanya dari mesin: candle yang benar-benar
 * dievaluasi, harga entry, batas rugi, target, serta koordinat FVG, sweep, dan
 * CISD. Tidak ada yang dihitung ulang di sini. Chart yang "kira-kira mirip"
 * justru membuat pengguna menyimpulkan hal yang salah.
 */
import { useMemo } from 'react'
import { Skeleton } from '@/components/ui'
import { useBacktestTrade } from '@/lib/queries'
import { priceDigits } from '@/lib/pricePrecision'
import {
  PriceChart,
  type ChartMarkerSpec,
  type PriceLineSpec,
  type SegmentSpec,
} from '@/features/charts/PriceChart'
import { chartTheme } from '@/features/charts/lib/theme'
import type { ZoneSpec } from '@/features/charts/lib/zonePrimitive'
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
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-[240px] w-full" />
        <Skeleton className="h-[240px] w-full" />
      </div>
    )
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

  return <IsiChart detail={detail} symbol={symbol} />
}

function IsiChart({ detail, symbol }: { detail: BacktestTradeDetail; symbol: string }) {
  const theme = chartTheme()
  const digits = priceDigits(symbol, detail.entry)

  // Level dan penanda sama untuk kedua panel: entry, batas rugi, target, sweep,
  // CISD, dan garis perjalanan posisi.
  const bersama = useMemo(() => {
    const lines: PriceLineSpec[] = [
      {
        price: detail.entry,
        color: theme.entry,
        title: `Entry ${detail.entry.toFixed(digits)}`,
        style: 'dashed',
        width: 1,
      },
      {
        price: detail.stop_loss,
        color: theme.stopLoss,
        title: `SL ${detail.stop_loss.toFixed(digits)}`,
        style: 'solid',
        width: 1,
      },
      {
        price: detail.take_profit,
        color: theme.takeProfit,
        title: `TP ${detail.take_profit.toFixed(digits)}`,
        style: 'solid',
        width: 1,
      },
    ]

    const isBuy = detail.arah === 'BUY'
    const marks: ChartMarkerSpec[] = [
      {
        time: detail.waktu_masuk,
        position: isBuy ? 'belowBar' : 'aboveBar',
        color: isBuy ? theme.up : theme.down,
        shape: isBuy ? 'arrowUp' : 'arrowDown',
        text: `${detail.arah} ${detail.entry.toFixed(digits)}`,
      },
    ]

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

    const segments: SegmentSpec[] = []
    if (detail.waktu_keluar > detail.waktu_masuk) {
      const hargaKeluar =
        detail.hasil === 'TP'
          ? detail.take_profit
          : detail.hasil === 'SL'
            ? detail.stop_loss
            : detail.entry
      segments.push({
        fromTime: detail.waktu_masuk,
        fromPrice: detail.entry,
        toTime: detail.waktu_keluar,
        toPrice: hargaKeluar,
        color: detail.hasil === 'TP' ? theme.profit : theme.loss,
      })
    }

    const zones = detail.zona_fvg
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

    return { lines, marks, segments, zones }
    // Nilai primitif sebagai dependensi: objek baru di tiap render akan
    // memicu penggambaran ulang chart tanpa perubahan apa pun.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    detail.seq,
    detail.entry,
    detail.stop_loss,
    detail.take_profit,
    detail.hasil,
    detail.r,
    detail.waktu_masuk,
    detail.waktu_keluar,
    theme,
    digits,
  ])

  const kosongLTF = detail.candle_ltf.length === 0
  const kosongHTF = detail.candle_htf.length === 0

  return (
    <div className="flex flex-col gap-4">
      <RingkasanEntry detail={detail} />

      <LembarChart
        judul={`${detail.htf} — timeframe acuan`}
        keterangan="Zona dan sapuan likuiditas terbaca dari sini."
        candles={detail.candle_htf}
        lines={bersama.lines}
        markers={bersama.marks}
        segments={bersama.segments}
        zones={bersama.zones}
        focusTime={detail.waktu_masuk}
        digits={digits}
        height={220}
        kosong={kosongHTF}
      />

      <LembarChart
        judul={`${detail.ltf} — timeframe eksekusi`}
        keterangan="Titik masuk, CISD, dan perjalanan posisi sampai ditutup."
        candles={detail.candle_ltf}
        lines={bersama.lines}
        markers={bersama.marks}
        segments={bersama.segments}
        zones={bersama.zones}
        focusTime={detail.waktu_masuk}
        digits={digits}
        height={280}
        kosong={kosongLTF}
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

/** RingkasanEntry menaruh angka posisi sebelum chart, seperti panel live. */
function RingkasanEntry({ detail }: { detail: BacktestTradeDetail }) {
  return (
    <div className="flex flex-wrap items-baseline gap-x-4 gap-y-1 border-b border-border pb-2 text-[12px]">
      <span className="text-[13px] font-semibold text-foreground">
        Entry {detail.seq} · {detail.arah}
      </span>
      <span className="text-muted-foreground">{formatWaktu(detail.waktu_masuk)}</span>
      <span className="text-muted-foreground">ditutup {formatWaktu(detail.waktu_keluar)}</span>
      <span className="text-muted-foreground">{formatDurasi(detail.durasi_detik)}</span>
      <span className="tabular text-muted-foreground">RR rencana {detail.rr_rencana.toFixed(2)}</span>
    </div>
  )
}

/** LembarChart adalah satu panel: judul, chart, dan keterangannya. */
function LembarChart({
  judul,
  keterangan,
  candles,
  lines,
  markers,
  segments,
  zones,
  focusTime,
  digits,
  height,
  kosong,
}: {
  judul: string
  keterangan: string
  candles: BacktestTradeDetail['candle_ltf']
  lines: PriceLineSpec[]
  markers: ChartMarkerSpec[]
  segments: SegmentSpec[]
  	zones: ZoneSpec[]
  focusTime: number
  digits: number
  height: number
  kosong: boolean
}) {
  return (
    <section className="flex min-w-0 flex-col gap-1.5">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 className="text-[12px] font-semibold">{judul}</h2>
        <p className="text-[11px] text-muted-foreground">{keterangan}</p>
      </div>
      {kosong ? (
        <div
          className="flex items-center justify-center border border-dashed border-border text-[12px] text-muted-foreground"
          style={{ height }}
        >
          Candle timeframe ini tidak tersedia untuk entry tersebut.
        </div>
      ) : (
        <PriceChart
          candles={candles}
          priceLines={lines}
          markers={markers}
          segments={segments}
          zones={zones}
          focusTime={focusTime}
          digits={digits}
          height={height}
          emptyMessage="Candle timeframe ini tidak tersedia."
        />
      )}
    </section>
  )
}
