import { Link } from 'react-router-dom'
import explorer from '@/assets/404-explorer.png'

/**
 * Halaman 404 situs publik.
 *
 * Ilustrasinya satu-satunya hal yang berwarna di halaman ini, dan itu memang
 * disengaja: di sistem monokrom proyek, satu gambar hitam-putih besar sudah
 * cukup membawa perhatian tanpa perlu aksen tambahan.
 *
 * Tidak ada paragraf penjelasan. Pengunjung yang sampai di sini sudah tahu
 * apa yang terjadi begitu membaca judulnya; kalimat tambahan hanya menunda
 * satu-satunya tindakan yang berguna. Judul dan tombol karena itu diletakkan
 * sedekat mungkin dengan gambar, dalam satu layar tanpa perlu menggulir.
 *
 * Tombol kembali sengaja hanya satu. Halaman galat yang menawarkan lima jalan
 * keluar justru membuat pengunjung berhenti dan memilih.
 */
export default function NotFoundPage() {
  return (
    <main className="mx-auto flex w-full max-w-6xl flex-col items-center px-4 pb-16 pt-6 text-center sm:px-6 sm:pt-8">
      <img
        src={explorer}
        alt="Jam pasir berkarakter detektif sedang memeriksa peta dengan kaca pembesar"
        width={760}
        height={695}
        /* Ukuran tampil dikunci supaya ruang gambar tidak bergeser saat
           berkasnya selesai dimuat. Tinggi dibatasi dengan vh agar halaman
           tetap muat satu layar di jendela yang pendek, bukan hanya sempit. */
        className="h-auto w-full max-w-[300px] select-none sm:max-w-[340px] lg:max-h-[46vh] lg:w-auto"
        decoding="async"
        fetchPriority="high"
      />

      <h1 className="mt-5 text-2xl font-semibold leading-tight tracking-tight sm:text-3xl">
        Halaman ini tidak ditemukan
      </h1>

      <Link
        to="/"
        className="press mt-5 inline-flex items-center justify-center rounded-lg bg-[#09090b] px-5 py-2.5 text-[13px] font-medium text-white hover:bg-[#27272a]"
      >
        Kembali ke beranda
      </Link>
    </main>
  )
}
