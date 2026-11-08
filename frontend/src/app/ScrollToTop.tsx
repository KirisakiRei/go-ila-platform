import { useEffect } from 'react'
import { useLocation } from 'react-router-dom'

/**
 * ScrollToTop mengembalikan posisi gulir ke puncak setiap kali alamat berubah.
 *
 * React Router tidak menyentuh posisi gulir sama sekali: tanpa komponen ini,
 * pengunjung yang membaca sampai kaki halaman lalu menekan menu akan mendarat
 * di tengah halaman tujuan. Perpindahan ke anchor (`#pasal`) sengaja
 * dibiarkan, karena di situlah tujuannya.
 */
export function ScrollToTop() {
  const { pathname, hash } = useLocation()

  useEffect(() => {
    if (hash) return

    // `instant` belum masuk daftar perilaku gulir bawaan TypeScript, sehingga
    // diberi anotasi di objeknya. Nilai `auto` tidak dipakai karena ia justru
    // menghormati `scroll-behavior: smooth` di stylesheet, dan perpindahan
    // halaman akan terlihat seperti animasi turun yang panjang.
    window.scrollTo({ top: 0, left: 0, behavior: 'instant' as ScrollBehavior })
  }, [pathname, hash])

  return null
}

/** ScrollBehavior adalah perilaku gulir yang dikenal peramban. */
type ScrollBehavior = 'auto' | 'smooth' | 'instant'
