import { useEffect, useMemo, useState } from 'react'

/** OFFSET adalah ruang untuk header lengket yang tidak boleh tertutup pasal. */
const OFFSET = 120

/**
 * useScrollSpy menyorot pasal yang sedang dibaca pengguna.
 *
 * Alih-alih memasang observer untuk setiap judul, satu pembacaan posisi pada
 * setiap frame gulir dipakai bersama: seluruh judul diukur dari posisi
 * dokumen, lalu pasal aktif adalah yang terakhir sudah melewati ambang atas.
 * Cara ini juga benar saat pengguna melompat ke anchor lewat daftar isi.
 *
 * Daftar id diterima sebagai string gabungan karena pemanggil biasanya
 * menyusunnya dari hasil parse markdown pada setiap render; array baru seperti
 * itu akan memicu pemasangan ulang listener tanpa henti.
 */
export function useScrollSpy(ids: string[], enabled = true) {
  const joined = ids.join('|')
  const list = useMemo(() => (joined === '' ? [] : joined.split('|')), [joined])
  const [activeId, setActiveId] = useState<string>(() => list[0] ?? '')

  useEffect(() => {
    if (!enabled || list.length === 0) return

    let frame = 0

    const measure = () => {
      frame = 0
      const offsets: { id: string; top: number }[] = []

      for (const id of list) {
        const element = document.getElementById(id)
        if (!element) continue
        offsets.push({ id, top: element.getBoundingClientRect().top })
      }

      if (offsets.length === 0) return

      // Sampai di dasar halaman: pasal terakhir yang aktif, karena pasal pendek
      // di kaki dokumen tidak akan pernah melewati ambang atas.
      const atBottom =
        window.innerHeight + window.scrollY >= document.documentElement.scrollHeight - 8
      if (atBottom) {
        const last = offsets[offsets.length - 1]
        if (last) setActiveId(last.id)
        return
      }

      const passed = offsets.filter((item) => item.top <= OFFSET)
      const current = passed[passed.length - 1] ?? offsets[0]
      if (current) setActiveId(current.id)
    }

    const schedule = () => {
      if (frame !== 0) return
      frame = window.requestAnimationFrame(measure)
    }

    schedule()
    window.addEventListener('scroll', schedule, { passive: true, capture: true })
    window.addEventListener('resize', schedule, { passive: true })
    return () => {
      if (frame !== 0) window.cancelAnimationFrame(frame)
      window.removeEventListener('scroll', schedule, { capture: true })
      window.removeEventListener('resize', schedule)
    }
  }, [list, enabled])

  return activeId
}
