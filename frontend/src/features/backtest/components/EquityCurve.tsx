/**
 * EquityCurve menggambar perjalanan ekuitas beserta pita penurunan.
 *
 * Digambar sebagai SVG biasa, bukan chart interaktif: yang dibaca di sini
 * adalah bentuk keseluruhan (berapa dalam turun, apakah pulih), bukan angka per
 * titik. Ringkasan satu run cukup empat puluh titik, jadi gambar vektor sederhana
 * lebih ringan dan lebih jernih daripada pustaka chart penuh.
 */
import { useMemo } from 'react'
import type { BacktestEquityPoint } from '@/types/api'
import { formatWaktu } from '../lib/tanggal'

const TINGGI = 180
const LEBAR_SVD = 600

export function EquityCurve({
  titik,
  ekuitasAwal,
}: {
  titik: BacktestEquityPoint[]
  ekuitasAwal: number
}) {
  const geometri = useMemo(() => hitungGeometri(titik, ekuitasAwal), [titik, ekuitasAwal])

  if (titik.length < 2) {
    return (
      <div className="flex h-[180px] items-center justify-center border border-border text-[13px] text-muted-foreground">
        Kurva ekuitas muncul setelah ada minimal dua trade.
      </div>
    )
  }

  const { garis, bayangan, tandon, nilaiMin, nilaiMaks, titikDrawdownMaks } = geometri

  return (
    <figure className="flex flex-col gap-3">
      <svg
        viewBox={`0 0 ${LEBAR_SVD} ${TINGGI}`}
        className="h-[180px] w-full"
        role="img"
        aria-label={`Kurva ekuitas dari ${formatUang(nilaiMin)} ke ${formatUang(nilaiMaks)}`}
        preserveAspectRatio="none"
      >
        {/* Garis modal awal: patokan tanpa itu, kurva tidak bisa dinilai. */}
        <line
          x1={0}
          x2={LEBAR_SVD}
          y1={tandon(ekuitasAwal)}
          y2={tandon(ekuitasAwal)}
          stroke="currentColor"
          strokeOpacity={0.2}
          strokeDasharray="4 4"
          strokeWidth={1}
        />
        {/* Pita penurunan antara ekuitas berjalan dan puncaknya. */}
        <path d={bayangan} fill="currentColor" fillOpacity={0.08} />
        <path
          d={garis}
          fill="none"
          stroke="currentColor"
          strokeWidth={2}
          strokeLinejoin="round"
          strokeLinecap="round"
          vectorEffect="non-scaling-stroke"
        />
        {titikDrawdownMaks && (
          <circle
            cx={titikDrawdownMaks.x}
            cy={titikDrawdownMaks.y}
            r={3}
            fill="currentColor"
            vectorEffect="non-scaling-stroke"
          />
        )}
      </svg>

      <figcaption className="flex flex-wrap items-baseline justify-between gap-2 text-[12px] text-muted-foreground">
        <span className="tabular">
          {formatUang(nilaiMin)} sampai {formatUang(nilaiMaks)}
        </span>
        <span className="tabular">
          puncak terendah {formatWaktu(titikDrawdownMaks?.waktu ?? 0)}
        </span>
      </figcaption>
    </figure>
  )
}

function hitungGeometri(titik: BacktestEquityPoint[], ekuitasAwal: number) {
  const nilai = titik.map((satu) => satu.ekuitas)
  let nilaiMin = Math.min(...nilai, ekuitasAwal)
  let nilaiMaks = Math.max(...nilai, ekuitasAwal)
  if (nilaiMin === nilaiMaks) {
    nilaiMin -= 1
    nilaiMaks += 1
  }
  const jarak = nilaiMaks - nilaiMin

  const posisiX = (indeks: number) =>
    titik.length === 1 ? 0 : (indeks / (titik.length - 1)) * LEBAR_SVD
  const posisiY = (angka: number) => TINGGI - ((angka - nilaiMin) / jarak) * TINGGI

  const koordinat = titik.map((satu, indeks) => ({
    x: posisiX(indeks),
    y: posisiY(satu.ekuitas),
    waktu: satu.waktu,
    ekuitas: satu.ekuitas,
  }))

  const garis = koordinat
    .map((satu, indeks) => `${indeks === 0 ? 'M' : 'L'}${satu.x.toFixed(1)},${satu.y.toFixed(1)}`)
    .join(' ')

  // Pita penurunan: sisi atas mengikuti ekuitas, sisi bawah mengikuti puncak
  // tertinggi sejauh itu. Selisihnya adalah drawdown pada tiap titik.
  let puncak = ekuitasAwal
  const atas = koordinat.map((satu, indeks) => {
    puncak = Math.max(puncak, titik[indeks]?.ekuitas ?? puncak)
    return { x: satu.x, y: posisiY(puncak) }
  })
  const bawah = koordinat.map((satu) => ({ x: satu.x, y: satu.y }))

  const bayangan =
    atas.map((satu, indeks) => `${indeks === 0 ? 'M' : 'L'}${satu.x.toFixed(1)},${satu.y.toFixed(1)}`).join(' ') +
    ' ' +
    [...bawah]
      .reverse()
      .map((satu) => `L${satu.x.toFixed(1)},${satu.y.toFixed(1)}`)
      .join(' ') +
    ' Z'

  let titikDrawdownMaks: { x: number; y: number; waktu: number } | null = null
  let drawdownTerburuk = 0
  for (const satu of titik) {
    if (satu.drawdown > drawdownTerburuk) {
      drawdownTerburuk = satu.drawdown
      const indeks = titik.indexOf(satu)
      titikDrawdownMaks = { x: posisiX(indeks), y: posisiY(satu.ekuitas), waktu: satu.waktu }
    }
  }

  return { garis, bayangan, tandon: posisiY, nilaiMin, nilaiMaks, titikDrawdownMaks }
}

function formatUang(nilai: number): string {
  return new Intl.NumberFormat('id-ID', { maximumFractionDigits: 0 }).format(nilai)
}
