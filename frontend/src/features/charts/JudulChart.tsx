import type { ReactNode } from 'react'

/**
 * JudulChart menaruh identitas satu panel chart tepat di atas bidang gambar.
 *
 * Bentuknya sengaja ringkas dan rata kiri: yang dibaca pertama adalah nama
 * timeframe, sisanya keterangan kecil. Label tidak ditaruh di dalam chart
 * supaya tidak menutupi candle.
 */
export function JudulChart({
  timeframe,
  peran,
  keterangan,
}: {
  /** Timeframe chart ini, mis. "D1". */
  timeframe: string
  /** Peran timeframe dalam pembacaan, mis. "acuan struktur". */
  peran: string
  /** Satu kalimat tentang apa yang dibaca dari panel ini. */
  keterangan?: string
}) {
  return (
    <div className="flex min-w-0 flex-wrap items-baseline gap-x-2 gap-y-0.5 px-1">
      <h3 className="text-[13px] font-semibold leading-none text-foreground">{timeframe}</h3>
      <span className="text-[12px] leading-none text-muted-foreground">{peran}</span>
      {keterangan && (
        <p className="w-full text-[11.5px] leading-snug text-muted-foreground sm:ml-auto sm:w-auto sm:text-right">
          {keterangan}
        </p>
      )}
    </div>
  )
}

/**
 * PanelChart membingkai satu chart: judul di atas, bidang gambar, lalu
 * keterangan di bawah.
 *
 * Dipisah dari PriceChart karena PriceChart juga dipakai radar dan posisi
 * live yang tidak butuh pembingkaian ini.
 */
export function PanelChart({
  judul,
  keterangan,
  children,
}: {
  judul: ReactNode
  keterangan?: ReactNode
  children: ReactNode
}) {
  return (
    <section className="flex min-w-0 flex-col gap-2">
      {judul}
      {children}
      {keterangan && (
        <div className="min-w-0 px-1 text-[11.5px] leading-relaxed text-muted-foreground">
          {keterangan}
        </div>
      )}
    </section>
  )
}
