import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'

/**
 * AuthLayout memusatkan formulir masuk dan daftar.
 * Lebar dibatasi agar baris teks tetap nyaman dibaca.
 */
export function AuthLayout({ children, footer }: { children: ReactNode; footer: ReactNode }) {
  return (
    <div className="flex min-h-full items-center justify-center px-4 py-10">
      <div className="w-full max-w-sm">
        <Link to="/" className="mb-8 flex items-baseline justify-center gap-1.5">
          <span className="text-base font-bold tracking-tight">ILA</span>
          <span className="text-[10px] uppercase tracking-[0.2em] text-muted-foreground">Bot</span>
        </Link>

        <div className="border border-border p-6">{children}</div>

        <div className="mt-4 text-center text-[13px] text-muted-foreground">{footer}</div>
      </div>
    </div>
  )
}
