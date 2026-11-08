import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join } from 'node:path'

/**
 * Panel pelanggan dan situs publik adalah dua bundel Vite terpisah: publik
 * dilayani index.html di akar, panel dilayani app.html di /app/*.
 *
 * Akibatnya `<Link to="/app/...">` dari situs publik TIDAK bisa dipakai:
 * React Router akan mencarinya di router publik, tidak menemukannya, lalu
 * menampilkan halaman tidak ditemukan. Yang benar adalah anchor biasa agar
 * browser memuat bundel panel dari server.
 *
 * Bug ini mudah terulang setiap ada tautan baru, jadi diperiksa otomatis.
 */

const SUMBER = join(process.cwd(), 'src')

/** Hanya berkas di bawah pages/public yang memakai bundel publik. */
function berkasPublik(): string[] {
  const hasil: string[] = []

  function jelajah(direktori: string) {
    for (const nama of readdirSync(direktori)) {
      const jalur = join(direktori, nama)
      if (statSync(jalur).isDirectory()) {
        jelajah(jalur)
        continue
      }
      if (nama.endsWith('.tsx') || nama.endsWith('.ts')) hasil.push(jalur)
    }
  }

  jelajah(join(SUMBER, 'pages', 'public'))
  return hasil
}

describe('tautan lintas bundel', () => {
  it('situs publik tidak memakai <Link> untuk rute /app/*', () => {
    const pelanggaran: string[] = []

    for (const berkas of berkasPublik()) {
      const isi = readFileSync(berkas, 'utf-8')
      // <Link ...> dan <NavLink ...> sama-sama memakai navigasi client-side.
      const pola = /<(Link|NavLink)\b[^>]*\bto=["']\/app\//g
      for (const cocok of isi.matchAll(pola)) {
        const baris = isi.slice(0, cocok.index).split('\n').length
        pelanggaran.push(`${berkas}:${baris} -> ${cocok[0]}`)
      }
    }

    expect(pelanggaran, `pakai <a href> agar bundel panel dimuat ulang:\n${pelanggaran.join('\n')}`).toEqual([])
  })
})
