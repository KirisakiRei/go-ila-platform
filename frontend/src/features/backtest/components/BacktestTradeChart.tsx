/**
 * BacktestTradeChart menggambar satu entry backtest beserta tahapannya.
 *
 * Ditata seperti chart eksekusi live: dua panel bertumpuk, timeframe besar di
 * atas untuk konteks dan timeframe kecil di bawah untuk titik masuk. Keduanya
 * memakai level yang SAMA, sehingga pembaca melihat satu kejadian dari dua
 * jarak pandang — bukan dua cerita berbeda.
 *
 * Semua angka diambil apa adanya dari mesin: candle yang benar-benar
 * dievaluasi, harga entry, batas rugi, target, serta koordinat FVG, sweep, dan
 * CISD. Tidak ada yang dihitung ulang di sini. Chart yang "kira-kira mirip"
 * justru membuat pengguna menyimpulkan hal yang salah.
 *
 * Level sengaja dipisah per panel: sweep, zona FVG, dan CISD hanya digambar di
 * timeframe asalnya. Sweep pada HTF yang dipaksakan ke panel LTF akan muncul
 * di lilin LTF yang salah dan membuat urutan kejadian terbaca terbalik.
 */
import { useMemo, useState, type ReactNode } from 'react'
import { MobileSegmentedTabs, Skeleton } from '@/components/ui'
import { useBacktestTrade } from '@/lib/queries'
import { priceDigits } from '@/lib/pricePrecision'
import { useIsMobile } from '@/lib/use-mobile-viewport'
import {
  PriceChart,
  type ChartMarkerSpec,
  type PriceLineSpec,
  type SegmentSpec,
} from '@/features/charts/PriceChart'
import { chartTheme } from '@/features/charts/lib/theme'
import type { ZoneSpec } from '@/features/charts/lib/zonePrimitive'
import { JudulChart, PanelChart } from '@/features/charts/JudulChart'
import type { BacktestTradeDetail } from '@/types/api'
import { formatDurasi, formatWaktu } from '../lib/tanggal'

/**
 * Tinggi kedua panel.
 *
 * Angka ini dipilih setelah panel lama (220 dan 280 piksel) dinilai terlalu
 * tipis: pada lebar konten penuh, candle menjadi pipih dan sweep sulit
 * dibedakan dari noise. Panel acuan sedikit lebih pendek karena tugasnya hanya
 * menunjukkan struktur, sedangkan panel eksekusi perlu ruang untuk seluruh
 * perjalanan posisi.
 */
const TINGGI_PANEL_ACUAN = 360
const TINGGI_PANEL_EKSEKUSI = 460

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
      <div className="flex w-full flex-col gap-4">
        <Skeleton className="w-full" style={{ height: TINGGI_PANEL_ACUAN }} />
        <Skeleton className="w-full" style={{ height: TINGGI_PANEL_EKSEKUSI }} />
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
  const isMobile = useIsMobile()
  const [mobilePanel, setMobilePanel] = useState<'htf' | 'ltf'>('ltf')
  const theme = chartTheme()
  const digits = priceDigits(symbol, detail.entry)
  const isBuy = detail.arah === 'BUY'

  // Level harga dan penanda hasil berlaku di kedua panel: entry, batas rugi,
  // target, arah masuk, dan cara posisi berakhir.
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

    const marks: ChartMarkerSpec[] = [
      {
        time: detail.waktu_masuk,
        position: isBuy ? 'belowBar' : 'aboveBar',
        color: isBuy ? theme.up : theme.down,
        shape: isBuy ? 'arrowUp' : 'arrowDown',
        text: `${detail.arah} ${detail.entry.toFixed(digits)}`,
      },
    ]

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

    return { lines, marks, segments }
    // Nilai primitif sebagai dependensi: objek baru di tiap render akan
    // memicu penggambaran ulang chart tanpa perubahan apa pun.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    detail.seq,
    detail.arah,
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

  // Sweep adalah kejadian timeframe acuan: yang menyapu likuiditas adalah
  // lilin besar, bukan lilin kecil di dalamnya.
  const penandaAcuan = useMemo<ChartMarkerSpec[]>(() => {
    if (!detail.sweep) return []
    return [
      {
        time: detail.sweep.waktu,
        position: isBuy ? 'aboveBar' : 'belowBar',
        color: theme.sweep,
        shape: 'square',
        text: detail.sweep.label || 'sweep',
      },
    ]
  }, [detail.sweep, isBuy, theme.sweep])

  // CISD adalah konfirmasi perubahan struktur pada timeframe eksekusi.
  const penandaEksekusi = useMemo<ChartMarkerSpec[]>(() => {
    if (!detail.cisd) return []
    return [
      {
        time: detail.cisd.waktu,
        position: isBuy ? 'belowBar' : 'aboveBar',
        color: theme.cisd,
        shape: 'square',
        text: detail.cisd.label || 'CISD',
      },
    ]
  }, [detail.cisd, isBuy, theme.cisd])

  // Zona FVG terbentuk di timeframe acuan, jadi hanya digambar di panel HTF.
  const zonaAcuan = useMemo<ZoneSpec[]>(() => {
    if (!detail.zona_fvg) return []
    return [
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
  }, [detail.zona_fvg, theme.fvg, theme.fvgBorder])

  const warnaHasil =
    detail.hasil === 'TP' ? theme.profit : detail.hasil === 'SL' ? theme.loss : theme.neutral

  return (
    <div className="flex w-full min-w-0 flex-col gap-5 sm:gap-6">
      <RingkasanEntry detail={detail} digits={digits} warnaHasil={warnaHasil} />

      {isMobile && (
        <MobileSegmentedTabs<'htf' | 'ltf'>
          ariaLabel="Pilih panel timeframe chart"
          value={mobilePanel}
          onChange={setMobilePanel}
          options={[
            { id: 'ltf', label: `Eksekusi (${detail.ltf})` },
            { id: 'htf', label: `Acuan (${detail.htf})` },
          ]}
        />
      )}

      {(!isMobile || mobilePanel === 'htf') && (
        <PanelChart
          judul={
            <JudulChart
              timeframe={detail.htf}
              peran={PERAN_TIMEFRAME[detail.htf] ?? 'timeframe acuan'}
              keterangan="Zona imbalance dan sapuan likuiditas terbaca dari sini."
            />
          }
          keterangan={
            <>
              {detail.candle_htf.length} candle tersimpan, dari sebelum masuk sampai sesudah
              ditutup. Garis entry, batas rugi, dan target di panel ini menandai harga yang sama
              dengan panel {detail.ltf}.
            </>
          }
        >
          <PanelIsi
            candle={detail.candle_htf}
            lines={bersama.lines}
            markers={[...bersama.marks, ...penandaAcuan]}
            segments={bersama.segments}
            zones={zonaAcuan}
            focusTime={detail.waktu_masuk}
            digits={digits}
            tinggi={isMobile ? 270 : TINGGI_PANEL_ACUAN}
            pesanKosong={`Candle ${detail.htf} tidak tersedia untuk entry ini.`}
          />
        </PanelChart>
      )}

      {(!isMobile || mobilePanel === 'ltf') && (
        <PanelChart
          judul={
            <JudulChart
              timeframe={detail.ltf}
              peran={PERAN_TIMEFRAME[detail.ltf] ?? 'timeframe eksekusi'}
              keterangan="Titik masuk, CISD, dan perjalanan posisi sampai ditutup."
            />
          }
          keterangan={
            <>
              {detail.candle_ltf.length} candle tersimpan. Garis putus-putus menandai harga entry,
              garis tebal adalah batas rugi dan target.
            </>
          }
        >
          <PanelIsi
            candle={detail.candle_ltf}
            lines={bersama.lines}
            markers={[...bersama.marks, ...penandaEksekusi]}
            segments={bersama.segments}
            zones={[]}
            focusTime={detail.waktu_masuk}
            digits={digits}
            tinggi={isMobile ? 300 : TINGGI_PANEL_EKSEKUSI}
            pesanKosong={`Candle ${detail.ltf} tidak tersedia untuk entry ini.`}
          />
        </PanelChart>
      )}

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

/** PERAN_TIMEFRAME menjelaskan tugas tiap timeframe dalam satu kalimat pendek. */
const PERAN_TIMEFRAME: Record<string, string> = {
  W: 'acuan mingguan',
  D1: 'acuan harian',
  H4: 'acuan empat jam',
  H1: 'acuan per jam',
  M15: 'eksekusi lima belas menit',
  M5: 'eksekusi lima menit',
}

/**
 * RingkasanEntry menaruh angka posisi sebelum chart, seperti panel live.
 *
 * Urutannya mengikuti cara trader membaca satu posisi: arah dan harga, lalu
 * waktu, lalu hasilnya.
 */
function RingkasanEntry({
  detail,
  digits,
  warnaHasil,
}: {
  detail: BacktestTradeDetail
  digits: number
  warnaHasil: string
}) {
  const harga = (angka: number) => angka.toFixed(digits)
  const teksHasil =
    detail.hasil === 'TP' ? 'target tercapai' : detail.hasil === 'SL' ? 'batas rugi' : 'belum tuntas'

  return (
    <dl className="grid grid-cols-2 gap-x-4 gap-y-3 border-y border-border py-3 sm:grid-cols-3 lg:grid-cols-6">
      <Kolom label="Arah" nilai={<span className="font-semibold">{detail.arah}</span>} />
      <Kolom label="Entry" nilai={harga(detail.entry)} />
      <Kolom label="Batas rugi" nilai={harga(detail.stop_loss)} />
      <Kolom label="Target" nilai={harga(detail.take_profit)} />
      <Kolom
        label="Hasil"
        nilai={
          <span className="font-semibold" style={{ color: warnaHasil }}>
            {teksHasil} ({detail.r > 0 ? '+' : ''}
            {detail.r.toFixed(2)}R)
          </span>
        }
      />
      <Kolom
        label="Terbuka"
        nilai={`${formatDurasi(detail.durasi_detik)} · RR ${detail.rr_rencana.toFixed(2)}`}
      />
      <Kolom label="Masuk" nilai={formatWaktu(detail.waktu_masuk)} lebarPenuh />
      <Kolom label="Ditutup" nilai={formatWaktu(detail.waktu_keluar)} lebarPenuh />
    </dl>
  )
}

function Kolom({
  label,
  nilai,
  lebarPenuh,
}: {
  label: string
  nilai: ReactNode
  lebarPenuh?: boolean
}) {
  return (
    <div className={lebarPenuh ? 'col-span-2 flex flex-col gap-0.5 sm:col-span-1' : 'flex flex-col gap-0.5'}>
      <dt className="text-[11px] text-muted-foreground">{label}</dt>
      <dd className="tabular truncate text-[13px] text-foreground">{nilai}</dd>
    </div>
  )
}

/**
 * PanelIsi menggambar satu bidang chart atau keterangan penggantinya.
 *
 * Tinggi maksimal tetapi lebar penuh: chart harga butuh ruang horizontal untuk
 * menyebar waktu, dan membatasi lebarnya justru membuat candle kembali pipih.
 */
function PanelIsi({
  candle,
  lines,
  markers,
  segments,
  zones,
  focusTime,
  digits,
  tinggi,
  pesanKosong,
}: {
  candle: BacktestTradeDetail['candle_ltf']
  lines: PriceLineSpec[]
  markers: ChartMarkerSpec[]
  segments: SegmentSpec[]
  zones: ZoneSpec[]
  focusTime: number
  digits: number
  tinggi: number
  pesanKosong: string
}) {
  if (candle.length === 0) {
    return (
      <div
        className="flex w-full items-center justify-center border border-dashed border-border text-[12px] text-muted-foreground"
        style={{ height: tinggi }}
      >
        {pesanKosong}
      </div>
    )
  }

  return (
    <PriceChart
      candles={candle}
      priceLines={lines}
      markers={markers}
      segments={segments}
      zones={zones}
      focusTime={focusTime}
      digits={digits}
      height={tinggi}
      className="w-full"
      emptyMessage={pesanKosong}
    />
  )
}
