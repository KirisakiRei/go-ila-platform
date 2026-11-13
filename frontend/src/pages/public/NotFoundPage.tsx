import { Link } from 'react-router-dom'
import explorer from '@/assets/404-explorer.png'

/**
 * Halaman 404 situs publik.
 *
 * Ilustrasinya satu-satunya hal yang berwarna di halaman ini, dan itu memang
 * disengaja: di sistem monokrom proyek, satu gambar hitam-putih besar sudah
 * cukup membawa perhatian tanpa perlu aksen tambahan. Teks dibuat sekecil
 * mungkin perannya — satu kalimat besar, satu penjelasan, satu tindakan.
 *
 * Tombol kembali sengaja hanya satu. Halaman galat yang menawarkan lima jalan
 * keluar justru membuat pengunjung berhenti dan memilih; satu tombol membuat
 * keputusannya jelas: kembali ke beranda.
 */
export default function NotFoundPage() {
  return (
    <main className="mx-auto flex w-full max-w-6xl flex-col items-center px-4 pb-24 pt-20 text-center sm:px-6 sm:pt-28">
      <img
        src={explorer}
        alt="Jam pasir berkarakter detektif sedang memeriksa peta dengan kaca pembesar"
        width={760}
        height={695}
        /* Ukuran tampil dikunci supaya ruang gambar tidak bergeser saat
           berkasnya selesai dimuat. Di layar sempit gambar mengecil sendiri,
           bukan terpotong. */
        className="h-auto w-full max-w-[380px] select-none sm:max-w-[420px]"
        decoding="async"
        fetchPriority="high"
      />

      <h1 className="mt-8 text-2xl font-semibold leading-tight tracking-tight sm:text-3xl">
        Halaman ini tidak ditemukan
      </h1>

      <p className="mt-3 max-w-sm text-[14px] leading-relaxed text-muted-foreground">
        Alamat yang Anda buka mungkin sudah dipindahkan atau salah ketik. Selebihnya, semua
        halaman lain masih ada di tempatnya.
      </p>

      <Link
        to="/"
        className="press mt-7 inline-flex items-center justify-center rounded-lg bg-[#09090b] px-5 py-2.5 text-[13px] font-medium text-white hover:bg-[#27272a]"
      >
        Kembali ke beranda
      </Link>
    </main>
  )
}
