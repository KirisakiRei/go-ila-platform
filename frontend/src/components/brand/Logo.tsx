import { cn } from '@/lib/utils'

/**
 * Logo ILA (jam pasir) dengan dua varian tema.
 *
 * Latar masalah: berkas aslinya 711×971 piksel (≈660 KB) padahal paling besar
 * tampil 64px. Browser mengunduh seluruh berkas itu untuk setiap tata letak,
 * sehingga header dan halaman login terasa berkedip. Sekarang yang dikirim
 * adalah turunan 64/128px di `public/brand/`, dan `srcset` membiarkan browser
 * memilih satu varian saja: 64px untuk layar biasa, 128px untuk layar retina.
 *
 * Catatan penting: kedua tema sengaja TIDAK dipasang bersamaan dengan
 * `dark:hidden`/`dark:block`. `display:none` tidak menghentikan unduhan, jadi
 * memasang keduanya berarti tetap mengunduh dua logo di setiap halaman.
 * Yang diunduh hanya logo sesuai tema yang sedang aktif.
 *
 * Rasio asli 711:971 ≈ 0,732. Kelas ukuran cukup mengatur tinggi (`h-7`),
 * lebar mengikuti otomatis sehingga tidak ada ruang kosong yang menggeser
 * header saat gambar selesai dimuat.
 */

const SOURCES = {
  light: {
    src: '/brand/logo-light-64.png',
    srcSet: '/brand/logo-light-64.png 1x, /brand/logo-light-128.png 2x',
  },
  dark: {
    src: '/brand/logo-dark-64.png',
    srcSet: '/brand/logo-dark-64.png 1x, /brand/logo-dark-128.png 2x',
  },
} as const

type LogoProps = {
  /** Kelas pengatur ukuran, mis. `h-7` atau `h-8`. Lebar selalu otomatis. */
  className?: string
  /**
   * Teks alternatif. Kosongkan (default) saat nama merek sudah tertulis di
   * sebelahnya, supaya pembaca layar tidak membacanya dua kali.
   */
  alt?: string
  /** Dipaksa memakai satu tema; `auto` mengikuti kelas `dark` di <html>. */
  theme?: 'auto' | 'light' | 'dark'
  /** Atribut `sizes` untuk kasus ukuran tampil berbeda per layar. */
  sizes?: string
}

export function Logo({ className, alt = '', theme = 'auto', sizes }: LogoProps) {
  const variant = theme === 'auto' ? undefined : SOURCES[theme]

  if (variant) {
    return (
      <img
        src={variant.src}
        srcSet={variant.srcSet}
        sizes={sizes}
        alt={alt}
        aria-hidden={alt ? undefined : true}
        className={cn('object-contain', className)}
        decoding="async"
      />
    )
  }

  // Dua varian tema dipilih CSS lewat `dark:`. Keduanya hanya berbeda pada
  // kelas tampil, bukan pada `src`, sehingga tidak ada unduhan ganda.
  return (
    <>
      <img
        src={SOURCES.light.src}
        srcSet={SOURCES.light.srcSet}
        sizes={sizes}
        alt={alt}
        aria-hidden={alt ? undefined : true}
        className={cn('object-contain dark:hidden', className)}
        decoding="async"
      />
      <img
        src={SOURCES.dark.src}
        srcSet={SOURCES.dark.srcSet}
        sizes={sizes}
        alt=""
        aria-hidden="true"
        className={cn('hidden object-contain dark:block', className)}
        decoding="async"
      />
    </>
  )
}
