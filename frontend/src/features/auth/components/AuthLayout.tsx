import type { ReactNode } from 'react'
import { ArrowLeft } from 'lucide-react'

/**
 * AuthLayout memusatkan formulir masuk dan daftar.
 *
 * Lebar dibatasi agar baris teks tetap nyaman dibaca. Tautan kembali ditaruh
 * di atas kartu, bukan di dalamnya, supaya tidak tertukar dengan tombol kirim
 * formulir dan tetap terlihat saat formulir berganti langkah.
 *
 * Tautan kembali memakai anchor biasa: beranda dilayani bundel publik yang
 * terpisah, jadi navigasi client-side tidak akan menemukan rutenya.
 */
export function AuthLayout({ children, footer }: { children: ReactNode; footer: ReactNode }) {
  return (
    <div className="flex min-h-full items-center justify-center px-4 py-10">
      <div className="w-full max-w-sm">
        <a href="/" className="mb-6 flex items-baseline justify-center gap-1.5">
          <span className="text-base font-bold tracking-tight">ILA</span>
          <span className="text-[10px] uppercase tracking-[0.2em] text-muted-foreground">Bot</span>
        </a>

        <div className="mb-3 flex items-center justify-between">
          <a
            href="/"
            className="inline-flex items-center gap-1.5 rounded-md px-1.5 py-1 text-[12px] text-muted-foreground transition-colors hover:text-foreground"
          >
            <ArrowLeft className="h-3.5 w-3.5" aria-hidden="true" />
            Kembali ke beranda
          </a>
        </div>

        <div className="border border-border bg-card p-6 shadow-sm">{children}</div>

        <div className="mt-4 text-center text-[13px] text-muted-foreground">{footer}</div>
      </div>
    </div>
  )
}
