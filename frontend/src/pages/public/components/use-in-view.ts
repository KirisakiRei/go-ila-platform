import { useCallback, useRef, useState } from 'react'

type UseInViewOptions = {
  /** Jarak dari tepi bawah layar sebelum elemen dianggap terlihat. */
  margin?: string
}

/**
 * Menandai sebuah elemen saat pertama kali masuk area pandang.
 *
 * Hanya terpicu sekali. Animasi yang berulang setiap kali pengunjung
 * menggulir ke atas dan ke bawah membuat halaman terasa gelisah, sedangkan
 * animasi yang menggambar sesuatu dari nol cukup dilihat satu kali.
 *
 * Bila IntersectionObserver tidak tersedia (peramban lama atau lingkungan
 * uji), nilai langsung dianggap terlihat supaya isi halaman tidak pernah
 * tersembunyi di balik animasi yang tidak pernah jalan.
 */
export function useInView<T extends Element>({ margin = '0px 0px -100px 0px' }: UseInViewOptions = {}) {
  const [terlihat, setTerlihat] = useState(false)
  const observerRef = useRef<IntersectionObserver | null>(null)
  const sudahRef = useRef(false)

  const ref = useCallback(
    (node: T | null) => {
      observerRef.current?.disconnect()
      observerRef.current = null

      if (!node || sudahRef.current) return

      if (typeof IntersectionObserver === 'undefined') {
        sudahRef.current = true
        setTerlihat(true)
        return
      }

      const observer = new IntersectionObserver(
        (entries) => {
          if (entries.some((entry) => entry.isIntersecting)) {
            sudahRef.current = true
            setTerlihat(true)
            observer.disconnect()
            observerRef.current = null
          }
        },
        { rootMargin: margin },
      )

      observer.observe(node)
      observerRef.current = observer
    },
    [margin],
  )

  return [ref, terlihat] as const
}
