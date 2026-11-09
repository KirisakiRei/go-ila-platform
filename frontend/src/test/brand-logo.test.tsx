import { describe, expect, it } from 'vitest'
import { render } from '@testing-library/react'
import { Logo } from '@/components/brand/Logo'

/**
 * Logo memakai turunan kecil dari `/brand/`, bukan berkas asli 660 KB.
 *
 * Uji ini menjaga regresi yang pernah terjadi: berkas logo 711×971 (≈660 KB)
 * dipakai langsung sebagai `src`, sehingga setiap halaman mengunduh gambar
 * raksasa untuk ditampilkan seukuran 28–64px.
 */
describe('Logo', () => {
  it('memakai turunan /brand/ dengan srcset, bukan berkas asli', () => {
    const { container } = render(<Logo className="h-7 w-auto" />)
    const images = container.querySelectorAll('img')

    expect(images.length).toBe(2)
    for (const img of images) {
      const src = img.getAttribute('src') ?? ''
      expect(src.startsWith('/brand/')).toBe(true)
      expect(src).toMatch(/-(64|128)\.png$/)
      // Berkas asli tidak boleh dipakai langsung.
      expect(src).not.toBe('/logo-light.png')
      expect(src).not.toBe('/logo-dark.png')
      // srcset wajib ada agar layar retina tidak memakai varian 64px.
      expect(img.getAttribute('srcset')).toContain('/brand/')
    }
  })

  it('hanya mengunduh satu tema: yang lain display:none lewat dark:', () => {
    const { container } = render(<Logo />)
    const [light, dark] = Array.from(container.querySelectorAll('img'))

    expect(light?.getAttribute('src')).toContain('logo-light')
    expect(light?.className).toContain('dark:hidden')
    expect(dark?.getAttribute('src')).toContain('logo-dark')
    expect(dark?.className).toContain('dark:block')
    // Lapis gelap murni hiasan; jangan dibacakan pembaca layar.
    expect(dark?.getAttribute('aria-hidden')).toBe('true')
  })

  it('meneruskan alt agar logo berdiri sendiri tetap terbaca', () => {
    const { container } = render(<Logo alt="Imaginary Liquidity" theme="light" />)
    const img = container.querySelector('img')

    expect(img?.getAttribute('alt')).toBe('Imaginary Liquidity')
    expect(img?.getAttribute('aria-hidden')).toBeNull()
  })

  it('memaksa satu tema saat tema halaman sudah pasti', () => {
    const { container } = render(<Logo theme="dark" />)
    const images = container.querySelectorAll('img')

    expect(images.length).toBe(1)
    expect(images[0]?.getAttribute('src')).toContain('logo-dark')
  })
})
