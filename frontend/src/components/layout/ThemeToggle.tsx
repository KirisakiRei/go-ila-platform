import { Monitor, Moon, Sun } from 'lucide-react'
import { useUserTheme, type ThemePreference } from '@/lib/theme'
import { cn } from '@/lib/utils'

const THEME_OPTIONS: ReadonlyArray<{
  value: ThemePreference
  label: string
  icon: typeof Sun
}> = [
  { value: 'light', label: 'Terang', icon: Sun },
  { value: 'dark', label: 'Gelap', icon: Moon },
  { value: 'system', label: 'Sistem', icon: Monitor },
]

type ThemeIconButtonProps = {
  className?: string
}

/**
 * Tombol cepat ganti tema untuk header desktop dan MobileTopBar.
 * Memakai transisi properti eksplisit (tanpa `transition-all`) dan scale tekan 0.97.
 */
export function ThemeIconButton({ className }: ThemeIconButtonProps) {
  const { isDark, toggle } = useUserTheme()
  const label = isDark ? 'Ubah ke mode terang' : 'Ubah ke mode gelap'

  return (
    <button
      type="button"
      onClick={toggle}
      aria-label={label}
      title={label}
      className={cn(
        'relative flex h-8 w-8 items-center justify-center rounded-lg text-muted-foreground transition-[transform,background-color,color] duration-150 [transition-timing-function:var(--ease-out)] hover:bg-secondary hover:text-foreground active:scale-[0.97]',
        className,
      )}
    >
      <Sun
        className={cn(
          'h-[17px] w-[17px] transition-[transform,opacity] duration-150 [transition-timing-function:var(--ease-out)]',
          isDark ? 'scale-100 rotate-0 opacity-100' : 'pointer-events-none absolute scale-90 -rotate-12 opacity-0',
        )}
      />
      <Moon
        className={cn(
          'h-[17px] w-[17px] transition-[transform,opacity] duration-150 [transition-timing-function:var(--ease-out)]',
          isDark ? 'pointer-events-none absolute scale-90 rotate-12 opacity-0' : 'scale-100 rotate-0 opacity-100',
        )}
      />
    </button>
  )
}

type ThemeSegmentedControlProps = {
  className?: string
}

/**
 * Pemilih 3 mode tema (Terang / Gelap / Sistem) untuk halaman Pengaturan.
 */
export function ThemeSegmentedControl({ className }: ThemeSegmentedControlProps) {
  const { preference, setPreference } = useUserTheme()

  return (
    <div
      role="radiogroup"
      aria-label="Tema tampilan dashboard"
      className={cn('grid grid-cols-3 gap-1 rounded-xl border border-border bg-secondary/50 p-1', className)}
    >
      {THEME_OPTIONS.map(({ value, label, icon: Icon }) => {
        const active = preference === value
        return (
          <button
            key={value}
            type="button"
            role="radio"
            aria-checked={active}
            onClick={() => setPreference(value)}
            className={cn(
              'flex items-center justify-center gap-1.5 rounded-lg px-2.5 py-1.5 text-[12px] font-medium transition-[transform,background-color,color,box-shadow] duration-150 [transition-timing-function:var(--ease-out)] active:scale-[0.97]',
              active
                ? 'bg-card font-semibold text-foreground shadow-xs'
                : 'text-muted-foreground hover:text-foreground',
            )}
          >
            <Icon className="h-3.5 w-3.5 shrink-0" />
            <span>{label}</span>
          </button>
        )
      })}
    </div>
  )
}
