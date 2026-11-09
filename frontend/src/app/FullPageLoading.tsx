import { Logo } from '@/components/brand/Logo'

/**
 * FullPageLoading adalah layar tunggu saat sesi dipulihkan.
 *
 * Yang tampil adalah logo ILA, bukan spinner generik, supaya halaman pertama
 * langsung memperlihatkan identitas produk. Logo memakai turunan kecil
 * (`/brand/…`) dan `srcset`, jadi tidak ada gambar 660 KB yang diunduh hanya
 * untuk ditampilkan seukuran 32px.
 *
 * `role="status"` dengan `aria-live="polite"` menjaga pembaca layar tetap
 * diberi tahu bahwa sistem sedang memuat, karena animasinya murni visual.
 */
export function FullPageLoading({ label }: { label: string }) {
  return (
    <div
      className="flex min-h-full flex-col items-center justify-center gap-3 text-muted-foreground"
      role="status"
      aria-live="polite"
    >
      <Logo className="h-8 w-auto animate-pulse" />
      <span className="text-[13px]">{label}</span>
    </div>
  )
}
