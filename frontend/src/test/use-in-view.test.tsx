/* global IntersectionObserverCallback */
import { describe, expect, it, vi, afterEach } from 'vitest'
import { act, render } from '@testing-library/react'
import { useInView } from '@/pages/public/components/use-in-view'

afterEach(() => {
  vi.unstubAllGlobals()
})

function Contoh() {
  const [ref, terlihat] = useInView<HTMLDivElement>()
  return (
    <div ref={ref} data-testid="kotak">
      {terlihat ? 'terlihat' : 'belum'}
    </div>
  )
}

/** Observer tiruan yang bisa dipicu manual dari dalam uji. */
function buatObserverPalsu() {
  const tercatat: { callback: IntersectionObserverCallback; disconnect: () => void }[] = []

  class ObserverPalsu {
    private readonly entri: { callback: IntersectionObserverCallback; disconnect: () => void }
    constructor(callback: IntersectionObserverCallback) {
      this.entri = { callback, disconnect: () => {} }
      tercatat.push(this.entri)
    }
    observe() {}
    unobserve() {}
    takeRecords() {
      return []
    }
    disconnect() {
      this.entri.disconnect()
    }
  }

  vi.stubGlobal('IntersectionObserver', ObserverPalsu)

  return {
    picu: (masuk: boolean) => {
      tercatat.forEach(({ callback }) => {
        callback(
          [{ isIntersecting: masuk } as IntersectionObserverEntry],
          {} as IntersectionObserver,
        )
      })
    },
  }
}

describe('useInView', () => {
  it('langsung dianggap terlihat bila IntersectionObserver tidak tersedia', () => {
    vi.stubGlobal('IntersectionObserver', undefined)

    const { getByTestId } = render(<Contoh />)

    // Isi halaman tidak boleh tertahan di balik animasi yang tak pernah jalan.
    expect(getByTestId('kotak').textContent).toBe('terlihat')
  })

  it('menandai terlihat saat elemen masuk area pandang', () => {
    const observer = buatObserverPalsu()
    const { getByTestId } = render(<Contoh />)

    expect(getByTestId('kotak').textContent).toBe('belum')

    act(() => {
      observer.picu(true)
    })

    expect(getByTestId('kotak').textContent).toBe('terlihat')
  })

  it('tidak mengulang animasi saat elemen keluar dari area pandang', () => {
    const observer = buatObserverPalsu()
    const { getByTestId } = render(<Contoh />)

    act(() => {
      observer.picu(true)
    })
    expect(getByTestId('kotak').textContent).toBe('terlihat')

    // Keluar dari layar tidak boleh mengembalikan keadaan ke semula, supaya
    // pengunjung yang menggulir naik-turun tidak melihat animasi berulang.
    act(() => {
      observer.picu(false)
    })
    expect(getByTestId('kotak').textContent).toBe('terlihat')
  })
})
