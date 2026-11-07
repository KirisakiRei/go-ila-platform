import { Link } from 'react-router-dom'

export function NotFoundPage() {
  return (
    <main className="flex min-h-full flex-col items-center justify-center gap-3 px-4 text-center">
      <h1 className="text-xl font-semibold tracking-tight">Halaman tidak ditemukan</h1>
      <p className="text-[13px] text-muted-foreground">Tautan yang Anda buka tidak tersedia.</p>
      <Link to="/" className="text-[13px] font-medium text-primary hover:underline">
        Kembali ke dashboard
      </Link>
    </main>
  )
}
