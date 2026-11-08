import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

/**
 * PublicHero adalah kepala halaman situs publik: judul besar bernada editorial,
 * satu kalimat penjelas, dan deretan aksi.
 *
 * Dipakai bersama oleh halaman Fitur, Harga, Bantuan, dan Tentang agar semua
 * halaman punya irama yang sama dan tidak berbeda-beda sendiri-sendiri.
 */
export function PublicHero({
  eyebrow,
  title,
  description,
  actions,
  aside,
  tone = 'light',
}: {
  eyebrow: string
  title: ReactNode
  description: string
  actions?: ReactNode
  aside?: ReactNode
  tone?: 'light' | 'dark'
}) {
  const isDark = tone === 'dark'

  return (
    <header
      className={cn(
        'relative overflow-hidden',
        isDark ? 'bg-[#050507] text-white' : 'bg-[#f5f5f7] text-[#09090b]',
      )}
    >
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 opacity-[0.5]"
        style={{
          background: isDark
            ? 'radial-gradient(60% 50% at 18% 0%, rgba(255,255,255,0.12) 0%, rgba(255,255,255,0) 100%)'
            : 'radial-gradient(70% 60% at 85% 0%, rgba(255,255,255,0.9) 0%, rgba(255,255,255,0) 100%)',
        }}
      />

      <div className="relative mx-auto w-full max-w-6xl px-4 pt-10 pb-16 sm:px-6 sm:pt-14 sm:pb-20">
        <div className="grid gap-10 lg:grid-cols-12 lg:items-end">
          <div className="lg:col-span-7">
            <p
              className={cn(
                'text-[11px] font-semibold uppercase tracking-[0.18em]',
                isDark ? 'text-zinc-400' : 'text-[#71717a]',
              )}
            >
              {eyebrow}
            </p>
            <h1 className="mt-4 text-[2rem] font-semibold leading-[1.08] tracking-tight sm:text-[2.9rem] lg:text-[3.2rem]">
              {title}
            </h1>
            <p
              className={cn(
                'mt-5 max-w-xl text-[15px] leading-relaxed',
                isDark ? 'text-zinc-300' : 'text-[#52525b]',
              )}
            >
              {description}
            </p>
            {actions && <div className="mt-8 flex flex-wrap items-center gap-3">{actions}</div>}
          </div>

          {aside && <div className="lg:col-span-5">{aside}</div>}
        </div>
      </div>
    </header>
  )
}

/**
 * SectionHeading adalah judul bagian di dalam halaman publik.
 *
 * Nomor bagian ditulis kecil di atas judul agar pembaca dapat menautkan
 * dirinya ke bagian tertentu ketika berdiskusi dengan dukungan.
 */
export function SectionHeading({
  index,
  title,
  description,
  align = 'left',
  tone = 'light',
}: {
  index?: string
  title: string
  description?: string
  align?: 'left' | 'center'
  tone?: 'light' | 'dark'
}) {
  return (
    <div className={cn('max-w-2xl', align === 'center' && 'mx-auto text-center')}>
      {index && (
        <p
          className={cn(
            'text-[11px] font-semibold uppercase tracking-[0.18em] tabular',
            tone === 'dark' ? 'text-zinc-500' : 'text-[#a1a1aa]',
          )}
        >
          {index}
        </p>
      )}
      <h2
        className={cn(
          'mt-2 text-[24px] font-semibold leading-[1.15] tracking-tight sm:text-[30px]',
          tone === 'dark' ? 'text-white' : 'text-[#09090b]',
        )}
      >
        {title}
      </h2>
      {description && (
        <p
          className={cn(
            'mt-3 text-[14px] leading-relaxed',
            tone === 'dark' ? 'text-zinc-400' : 'text-[#52525b]',
          )}
        >
          {description}
        </p>
      )}
    </div>
  )
}

/**
 * ActionLink adalah tombol situs publik.
 *
 * Rute `/app/*` wajib memakai `<a href>` karena panel pelanggan adalah bundel
 * terpisah; navigasi client-side tidak akan menemukannya di router publik.
 */
export function ActionLink({
  href,
  children,
  variant = 'primary',
  className,
}: {
  href: string
  children: ReactNode
  variant?: 'primary' | 'secondary' | 'ghost'
  className?: string
}) {
  const isExternalBundle = href.startsWith('/app/')
  const classes = cn(
    'inline-flex items-center justify-center gap-2 rounded-lg px-5 py-2.5 text-[13px] transition-colors',
    variant === 'primary' && 'bg-[#09090b] font-semibold text-white hover:bg-zinc-800',
    variant === 'secondary' &&
      'border border-zinc-300 bg-white font-medium text-[#09090b] hover:bg-zinc-100',
    variant === 'ghost' && 'font-medium text-[#09090b] underline underline-offset-4',
    className,
  )

  if (isExternalBundle) {
    return (
      <a href={href} className={classes}>
        {children}
      </a>
    )
  }

  return (
    <a href={href} className={classes}>
      {children}
    </a>
  )
}

/** Callout adalah kotak penekanan untuk pernyataan yang tidak boleh terlewat. */
export function Callout({
  tone = 'note',
  title,
  children,
}: {
  tone?: 'note' | 'warning'
  title: string
  children: ReactNode
}) {
  return (
    <div
      className={cn(
        'rounded-xl border p-5 sm:p-6',
        tone === 'warning'
          ? 'border-[#09090b]/85 bg-[#09090b] text-zinc-200'
          : 'border-zinc-300 bg-[#fafafa] text-[#3f3f46]',
      )}
    >
      <p
        className={cn(
          'text-[11px] font-semibold uppercase tracking-[0.16em]',
          tone === 'warning' ? 'text-white' : 'text-[#09090b]',
        )}
      >
        {title}
      </p>
      <div className="mt-2 text-[13.5px] leading-relaxed">{children}</div>
    </div>
  )
}
