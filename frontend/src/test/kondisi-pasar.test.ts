import { describe, expect, it } from 'vitest'
import { kondisiPasar, jamTutupMenyusul, tanggalWIB } from '@/features/signals/lib/kondisi-pasar'
import type { MarketWindow } from '@/types/system'

const jadwal = (open: boolean, next_label = 'Pasar tutup'): MarketWindow => ({
  open,
  label: open ? 'Pasar buka' : 'Pasar tutup',
  next_change: '2026-11-14T05:00:00+07:00',
  next_label,
  next_change_pair: { wib: '14 Nov 05:00', ny: '13 Nov 17:00', utc: '2026-11-13T22:00:00Z' },
})

describe('kondisiPasar', () => {
  it('laporan engine dipakai lebih dulu bila jelas', () => {
    const hasil = kondisiPasar({ state: 'OPEN', label: 'Pasar buka' }, jadwal(false))
    // Engine tahu Rollover/OPEN yang sesungguhnya; jadwal hanya jam buka-tutup.
    expect(hasil.state).toBe('OPEN')
    expect(hasil.disimpulkan).toBe(false)
  })

  it('UNKNOWN dari engine tidak lagi menampilkan "belum diketahui"', () => {
    // Bug produksi: satu layar menampilkan "Tidak diketahui" sekaligus
    // "Pasar buka". Dengan jadwal publik, jawabannya menjadi pasti.
    const hasil = kondisiPasar({ state: 'UNKNOWN', label: 'Tidak diketahui' }, jadwal(true))
    expect(hasil.state).toBe('OPEN')
    expect(hasil.label).toBe('Pasar Buka')
    expect(hasil.disimpulkan).toBe(true)
  })

  it('jadwal tertutup menghasilkan Pasar Tutup', () => {
    const hasil = kondisiPasar({ state: 'UNKNOWN', label: '' }, jadwal(false))
    expect(hasil.state).toBe('CLOSED')
  })

  it('rollover tidak ditimpa jadwal', () => {
    const hasil = kondisiPasar({ state: 'ROLLOVER', label: 'Rollover' }, jadwal(true))
    expect(hasil.state).toBe('ROLLOVER')
  })

  it('tanpa kedua sumber tetap jujur menyebut belum diketahui', () => {
    const hasil = kondisiPasar(null, null)
    expect(hasil.state).toBe('UNKNOWN')
    expect(hasil.disimpulkan).toBe(false)
  })
})

describe('tanggalWIB', () => {
  it('transaksi dini hari WIB masuk hari akuntansi yang benar', () => {
    // Kasus nyata produksi: posisi ditutup 17:43 UTC tanggal 12 Nov, yang
    // sebenarnya 00:43 WIB tanggal 13 Nov. Potongan string UTC akan
    // menempatkannya di tanggal 12 dan membuat P/L hari ini hilang.
    expect(tanggalWIB('2026-11-12T17:43:06.947Z')).toBe('2026-11-13')
  })

  it('transaksi siang WIB tetap di tanggal yang sama', () => {
    expect(tanggalWIB('2026-11-13T05:00:00Z')).toBe('2026-11-13')
  })

  it('masukan kosong atau rusak menghasilkan string kosong', () => {
    expect(tanggalWIB(null)).toBe('')
    expect(tanggalWIB('bukan tanggal')).toBe('')
  })
})

describe('jamTutupMenyusul', () => {
  it('mengenali label tutup berbahasa Indonesia dan Inggris', () => {
    expect(jamTutupMenyusul(jadwal(true, 'Pasar tutup'))).toBe(true)
    expect(jamTutupMenyusul(jadwal(false, 'Market close'))).toBe(true)
  })

  it('label buka bukan penanda tutup', () => {
    expect(jamTutupMenyusul(jadwal(false, 'Pasar buka'))).toBe(false)
    expect(jamTutupMenyusul(null)).toBe(false)
  })
})
