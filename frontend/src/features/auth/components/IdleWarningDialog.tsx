import { useEffect, useRef } from 'react'
import { Button } from '@/components/ui/button'

type Props = {
  remainingSeconds: number
  onStay: () => void
  onLeaveNow: () => void
}

/**
 * IdleWarningDialog memberi tahu bahwa sesi akan berakhir karena tidak ada
 * aktivitas. Dua tombol disediakan supaya pengguna tidak mengklik refleks
 * (plans/23 §4.2).
 */
export function IdleWarningDialog({ remainingSeconds, onStay, onLeaveNow }: Props) {
  const stayRef = useRef<HTMLButtonElement>(null)

  // Fokus diarahkan ke "masih di sini" agar aman bila pengguna kembali.
  useEffect(() => {
    stayRef.current?.focus()
  }, [])

  const minutes = Math.floor(remainingSeconds / 60)
  const seconds = remainingSeconds % 60
  const label = `00:${String(seconds).padStart(2, '0')}`
  const readable = minutes > 0 ? `${minutes} menit ${seconds} detik` : `${seconds} detik`

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 p-4 backdrop-blur-sm"
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="idle-title"
      aria-describedby="idle-desc"
      // Klik di luar sengaja tidak menutup: mencegah kehilangan sesi tak sengaja.
    >
      <div className="w-full max-w-sm rounded-lg border border-border bg-card p-6 shadow-lg">
        <h2 id="idle-title" className="text-lg font-semibold tracking-tight">
          Masih di sini?
        </h2>
        <p id="idle-desc" className="mt-2 text-sm text-muted-foreground">
          Karena tidak ada aktivitas, kami akan mengeluarkan Anda dalam{' '}
          <span className="font-medium text-foreground">{readable}</span>.
        </p>

        <p
          className="mt-4 text-center font-mono text-3xl tabular-nums tracking-tight"
          aria-live="polite"
          aria-label={`Sisa waktu ${readable}`}
        >
          {label}
        </p>

        <p className="mt-4 text-xs text-muted-foreground">
          Ini demi keamanan akun dan dana Anda.
        </p>

        <div className="mt-6 flex flex-col gap-2 sm:flex-row-reverse">
          <Button ref={stayRef} className="sm:flex-1" onClick={onStay}>
            Aku masih di sini
          </Button>
          <Button variant="outline" className="sm:flex-1" onClick={onLeaveNow}>
            Keluar sekarang
          </Button>
        </div>
      </div>
    </div>
  )
}
