import { useEffect, useState } from 'react'

function readViewportBelow(maxWidth: number): boolean {
  if (typeof window === 'undefined') return false
  return window.innerWidth <= maxWidth
}

/**
 * useViewportBelow mendeteksi apakah lebar layar berada pada atau di bawah `maxWidth`.
 *
 * Di lingkungan uji JSDOM bawaan (`window.innerWidth === 1024`), hook ini
 * bernilai `false` sehingga pengujian desktop tidak melihat elemen ganda di DOM,
 * sementara perangkat ponsel nyata (`360px–430px`) atau pengujian mobile
 * (`window.innerWidth = 375`) merender komponen khusus layar sempit secara murni.
 */
export function useViewportBelow(maxWidth: number): boolean {
  const [matches, setMatches] = useState<boolean>(() => readViewportBelow(maxWidth))

  useEffect(() => {
    const update = () => setMatches(readViewportBelow(maxWidth))
    update()

    window.addEventListener('resize', update)
    const mql =
      typeof window.matchMedia === 'function'
        ? window.matchMedia(`(max-width: ${maxWidth}px)`)
        : null
    mql?.addEventListener?.('change', update)

    return () => {
      window.removeEventListener('resize', update)
      mql?.removeEventListener?.('change', update)
    }
  }, [maxWidth])

  return matches
}

/** Layar ponsel (`<= 767px`, setara `< md`). */
export function useIsMobile(): boolean {
  return useViewportBelow(767)
}

/** Layar ponsel & tablet vertikal (`<= 1023px`, setara `< lg`). */
export function useIsTabletOrMobile(): boolean {
  return useViewportBelow(1023)
}
