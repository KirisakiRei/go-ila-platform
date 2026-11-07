import { useEffect, useState } from 'react'

/**
 * useDebouncedValue menunda nilai sampai pengguna berhenti mengetik.
 * Tanpa ini, setiap huruf memicu permintaan ke server.
 */
export function useDebouncedValue<T>(value: T, delayMs = 300): T {
  const [debounced, setDebounced] = useState(value)

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delayMs)
    return () => clearTimeout(timer)
  }, [value, delayMs])

  return debounced
}
