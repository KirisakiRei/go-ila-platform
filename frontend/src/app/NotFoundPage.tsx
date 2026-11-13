import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'

/**
 * Halaman 404 di dalam panel.
 *
 * Sengaja lebih tenang daripada versi situs publik: pengunjung di sini sudah
 * punya sesi dan sedang bekerja, jadi yang dibutuhkan bukan ilustrasi besar
 * melainkan jalan kembali secepat mungkin.
 */
export function NotFoundPage() {
  return (
    <main className="flex min-h-full flex-col items-center justify-center gap-3 px-4 py-16 text-center">
      <h1 className="text-xl font-semibold tracking-tight">Halaman ini tidak ditemukan</h1>
      <p className="max-w-xs text-[13px] leading-relaxed text-muted-foreground">
        Alamat yang Anda buka tidak tersedia. Kembali ke dashboard untuk melanjutkan.
      </p>
      <Button asChild size="sm" className="mt-1">
        <Link to="/">Kembali ke dashboard</Link>
      </Button>
    </main>
  )
}
